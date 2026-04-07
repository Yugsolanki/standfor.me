package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/Yugsolanki/standfor-me/internal/pkg/logger"
	"github.com/Yugsolanki/standfor-me/internal/pkg/requestid"
)

const maxStackSize = 1024 * 32 // 32KB

// Recoverer catches panics from downstream handlers and converts them into
// a 500 Internal Server Error response instead of crashing the entire process.
//
// It logs the panic with full stack trace and enriches the canonical log line
// with panic details. This middleware should sit just inside RequestID and
// just outside the canonical logger (or inside it — both work, but inside
// means the canonical logger's own defer also sees the panic fields).
//
// Design notes:
//   - In production, the response body is intentionally vague ("Internal Server Error")
//     to avoid leaking implementation details.
//   - The full panic value and stack trace are captured in structured logs
//     for debugging.
//   - The request ID is included in the error response so users can report it
//     to support.
func Recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					ctx := r.Context()
					requestID := requestid.GetRequestID(ctx)
					stack := string(debug.Stack())

					if len(stack) > maxStackSize {
						stack = stack[:maxStackSize]
					}

					log.Error("panic recovered",
						"panic", fmt.Sprintf("%v", rec),
						"request_id", requestID,
						"http_method", r.Method,
						"http_path", r.URL.Path,
						"stack_trace", stack)

					// Enrich the canonical log line if it's available
					logger.AddField(ctx, "panic", true)
					logger.AddField(ctx, "panic_error", fmt.Sprintf("%v", rec))
					logger.AddField(ctx, "stack_trace", stack)

					// Respond with a safe error message
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set(requestid.RequestIDHeader, requestID)
					w.WriteHeader(http.StatusInternalServerError)

					body := fmt.Sprintf(
						`
						{
							"error":"Internal Server Error",
							"request_id":%q,
							"message":"Something went wrong. Please reference this request ID when contacting support."
						}`,
						requestID,
					)
					_, _ = w.Write([]byte(body))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
