package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/pkg/crypto"
	"github.com/Yugsolanki/standfor-me/internal/pkg/httputil"
	"github.com/Yugsolanki/standfor-me/internal/pkg/logger"
	"github.com/Yugsolanki/standfor-me/internal/pkg/requestid"
)

// responseRecorder is a wrapper around http.ResponseWriter that records
// the status code, bytes written, and whether the header has been written.
type responseRecorder struct {
	http.ResponseWriter
	statusCode    int
	bytesWritten  int
	headerWritten bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader records the status code and forwards the call to the underlying
// ResponseWriter. It only writes the header once, even if called multiple times.
func (r *responseRecorder) WriteHeader(statusCode int) {
	if !r.headerWritten {
		r.statusCode = statusCode
		r.headerWritten = true
		r.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write writes the response body and updates the bytes written count.
// It only writes the body once, even if called multiple times.
func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.headerWritten {
		r.headerWritten = true
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytesWritten += n
	return n, err
}

// Unwrap returns the underlying http.ResponseWriter.
func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func CanonicalLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Initialize canonical fields and seed with request-level data
			cf := logger.NewCanonicalFields()
			cf.Set("http_method", r.Method)
			cf.Set("http_path", r.URL.Path)
			cf.Set("request_id", requestid.GetRequestID(r.Context()))

			if r.URL.RawQuery != "" {
				cf.Set("http_query", r.URL.RawQuery)
			}

			cf.Set("client_ip", httputil.ExtractIP(r, true))
			cf.Set("user_agent", r.UserAgent())

			// Attach canonical fields to the context so any handler or service
			// deeper in the call stack can enrich the log line
			ctx := logger.WithCanonicalFields(r.Context(), cf)
			r = r.WithContext(ctx)

			// Set the request ID on the response headers for client correlation.
			w.Header().Set("X-Request-ID", requestid.GetRequestID(r.Context()))

			// wrap the response writer to capture status code and bytes
			rec := newResponseRecorder(w)

			defer func() {
				// --- Emit the canonical log line
				duration := time.Since(start)
				cf.Set("http_status", rec.statusCode)
				cf.Set("duration_ms", duration.Milliseconds())
				cf.Set("duration", duration.String())
				cf.Set("bytes_written", rec.bytesWritten)

				// Build the slog attribute from all collected fields
				allFields := cf.All()
				attrs := make([]slog.Attr, 0, len(allFields))
				for k, v := range allFields {
					attrs = append(attrs, slog.Any(k, v))
				}

				// Determine log level based on status code
				level := determineLogLevel(rec.statusCode, allFields)

				if shouldSkipLog(rec.statusCode, level, allFields) {
					return
				}

				// Emit the single canonical line
				log.LogAttrs(
					r.Context(),
					level,
					"canonical-log-line",
					attrs...,
				)
			}()

			// Call the next handler in the chain
			next.ServeHTTP(rec, r)
		})
	}
}

// determineLogLevel selects the appropriate log level based on the HTTP
// status code and whether a panic occurred.
func determineLogLevel(statusCode int, fields map[string]any) slog.Level {
	// Panics are always errors.
	if _, ok := fields["panic"]; ok {
		return slog.LevelError
	}

	switch {
	case statusCode >= 500:
		return slog.LevelError
	case statusCode >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

func shouldSkipLog(statusCode int, level slog.Level, fields map[string]any) bool {
	// Always log errors and warnings.
	if level >= slog.LevelWarn {
		return false
	}

	// Always log non-2xx responses and warnings/errors.
	// For successful 2xx responses, we can sample to reduce log volume,
	// especially for high-traffic endpoints like health checks.
	if statusCode < 200 || statusCode >= 300 {
		return false
	}

	// Skip logging for health checks and readiness probes to reduce noise.
	if path, ok := fields["http_path"].(string); ok {
		if path == "/healthz" || path == "/ready" {
			return true
		}
	}

	// Sample 10% of remaining 2xx responses
	f, err := crypto.CryptoFloat64()
	if err != nil {
		// In case of an error generating a random float,
		// default to logging the request
		return false
	}
	return f >= 0.1
}
