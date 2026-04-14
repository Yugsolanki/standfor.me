package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Yugsolanki/standfor-me/internal/pkg/logger"
	"github.com/Yugsolanki/standfor-me/internal/pkg/requestid"
)

// Common payload limits
const (
	DefaultMaxBodySize = 1 << 20  // 1 MB
	MaxUploadSize      = 10 << 20 // 10 MB
	MaxAvatarSize      = 5 << 20  // 5 MB
)

// PayloadLimit restricts the maximum size of request bodies to prevent
// abuse, resource exhaustion, and out-of-memory conditions.
//
// It uses http.MaxBytesReader, which:
//   - Returns an error when the limit is exceeded (not silently truncate)
//   - Automatically closes the connection to stop the client from sending more data
//   - Works with streaming — doesn't buffer the entire body in memory first
func PayloadLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only limit methods that carry a body
			if r.Method == http.MethodPost ||
				r.Method == http.MethodPut ||
				r.Method == http.MethodPatch {

				if r.ContentLength > maxBytes {
					requestID := requestid.GetRequestID(r.Context())
					logger.AddField(r.Context(), "rejected", "payload_too_large")
					logger.AddField(r.Context(), "content_length", r.ContentLength)
					logger.AddField(r.Context(), "max_bytes", formatBytes(maxBytes))

					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Connection", "close")
					w.WriteHeader(http.StatusRequestEntityTooLarge)

					body := fmt.Sprintf(
						`
						{
							"error":"Payload Too Large",
							"request_id":%q,
							"message":"Request body must not exceed %s.","max_size":"%s"
						}`,
						requestID,
						formatBytes(maxBytes),
						formatBytes(maxBytes),
					)
					_, _ = w.Write([]byte(body))
					return
				}

				// Wrap the body with a max-bytes reader. This handles cases where
				// Content-Length is missing or incorrect (e.g., chunked encoding).
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
				logger.AddField(r.Context(), "payload_limit", formatBytes(maxBytes))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// PayloadLimitByRoute applies different payload limits based on URL path patterns.
// This is useful when you want a single middleware instance with route-aware limits.
//
// Example:
//
//	routeLimits := map[string]int64{
//		"/api/v1/upload": MaxUploadSize,
//		"/api/v1/avatar": MaxAvatarSize,
//	}
//	payloadLimit := middleware.PayloadLimitByRoute(routeLimits, DefaultMaxBodySize)
func PayloadLimitByRoute(routeLimits map[string]int64, defaultLimit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost ||
				r.Method == http.MethodPut ||
				r.Method == http.MethodPatch {

				limit := defaultLimit
				for pattern, l := range routeLimits {
					if strings.HasPrefix(r.URL.Path, pattern) {
						limit = l
						break // break after the first match
					}
				}

				if r.ContentLength > limit {
					requestID := requestid.GetRequestID(r.Context())
					logger.AddField(r.Context(), "rejected", "payload_too_large")
					logger.AddField(r.Context(), "content_length", r.ContentLength)
					logger.AddField(r.Context(), "max_bytes", formatBytes(limit))

					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Connection", "close")
					w.WriteHeader(http.StatusRequestEntityTooLarge)

					body := fmt.Sprintf(
						`
						{
							"error":"Payload Too Large",
							"request_id":%q,
							"message":"Request body must not exceed %s.","max_size":"%s"
						}`,
						requestID,
						formatBytes(limit),
						formatBytes(limit),
					)
					_, _ = w.Write([]byte(body))
					return
				}

				r.Body = http.MaxBytesReader(w, r.Body, limit)
				logger.AddField(r.Context(), "payload_limit", formatBytes(limit))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// formatBytes converts a byte count into a human-readable string.
func formatBytes(b int64) string {
	const (
		kb = 1 << 10
		mb = 1 << 20
		gb = 1 << 30
	)

	if b < 0 {
		return "0 B"
	}

	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
