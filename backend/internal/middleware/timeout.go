package middleware

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/pkg/logger"
	"github.com/Yugsolanki/standfor-me/internal/pkg/requestid"
)

// Timeout wraps each request with a context deadline. If the handler
// does not complete within the specified duration, the context is cancelled
// and a 503 Service Unavailable (or 504 Gateway Timeout) is returned.
//
// This prevents:
//   - Slow database queries from holding connections indefinitely
//   - Slow clients from exhausting server goroutines
//   - Runaway processing from consuming resources
//
// Important: Your handlers and services MUST check ctx.Err() or use
// ctx.Done() for this to be effective. Database drivers (pgx, etc.)
// and HTTP clients already respect context cancellation.
func Timeout(duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), duration)
			defer cancel()

			// Replace the context with timeout-bound context
			r = r.WithContext(ctx)

			// Channel to signal handler completion
			done := make(chan struct{})

			// Use a response recorder so we can prevent writing after timeout
			tw := &timeoutWriter{
				ResponseWriter: w,
				header:         make(http.Header),
			}

			// Run the handler in a separate goroutine so we can select on the context timeout
			go func() {
				defer close(done)
				next.ServeHTTP(tw, r)
			}()

			select {
			case <-done:
				// Handler completed in time. Copy the headers and status code.
				tw.mu.Lock()
				defer tw.mu.Unlock()

				// Copy the headers from the buffered writer to the real writer
				dst := w.Header()
				maps.Copy(dst, tw.header)

				if tw.statusCode > 0 {
					w.WriteHeader(tw.statusCode)
				}

				if tw.body != nil {
					_, _ = w.Write(tw.body)
				}

			case <-ctx.Done():
				// Timeout exceeded
				tw.mu.Lock()
				tw.timedOut = true
				requestID := tw.header.Get(requestid.RequestIDHeader)
				tw.mu.Unlock()

				logger.AddField(r.Context(), "timeout", true)
				logger.AddField(r.Context(), "timeout_duration", duration.String())

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set(requestid.RequestIDHeader, requestID)
				w.Header().Set("Connection", "close")
				w.WriteHeader(http.StatusGatewayTimeout)

				body := fmt.Sprintf(
					`
					{
						"error":"Gateway Timeout",
						"request_id":%q,
						"message":"The server did not complete your request in time. Please try again."
					}`,
					requestID,
				)
				_, _ = w.Write([]byte(body))
			}
		})
	}
}

// timeoutWriter buffers the response so that if a timeout occurs,
// we don't write a partial response followed by a timeout error.
type timeoutWriter struct {
	http.ResponseWriter

	mu         sync.Mutex
	header     http.Header
	body       []byte
	statusCode int
	timedOut   bool
}

// Header returns the headers map. It is safe to call concurrently.
func (tw *timeoutWriter) Header() http.Header {
	return tw.header
}

// WriteHeader writes the status code. It is safe to call concurrently.
func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.timedOut && tw.statusCode == 0 {
		tw.statusCode = code
	}
}

// Write writes the response body. It is safe to call concurrently.
func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	tw.body = append(tw.body, b...)
	return len(b), nil
}

// Flush flushes the buffered response to the real ResponseWriter if not timed out.
func (tw *timeoutWriter) Flush() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return
	}
	if f, ok := tw.ResponseWriter.(http.Flusher); ok {
		// Flush buffered body to real writer
		if len(tw.body) > 0 {
			_, _ = tw.ResponseWriter.Write(tw.body)
			tw.body = nil // Clear buffer after flush
			f.Flush()
		}
	}
}
