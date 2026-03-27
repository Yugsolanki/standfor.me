package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type requestIDKey struct{}

// RequestIDHeader is the canonical header name used to propagate request IDs
// across service boundaries.
const RequestIDHeader = "X-Request-ID"

// RequestID injects a unique identifier into every request. It first checks
// for an existing X-Request-ID header (set by an upstream load balancer or
// API gateway). If none exists, it generates a new UUID v4.
//
// The ID is:
//  1. Stored in the request context (accessible via GetRequestID)
//  2. Set on the response header (so clients/frontends can reference it)
//  3. Available to downstream middleware like the canonical logger
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = uuid.New().String()
		}

		// Set on response so client can correlate
		w.Header().Set(RequestIDHeader, id)

		// Store in context for downstream access
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from the context.
// Returns an empty string if no request ID is present.
//
// Usage:
//
//	requestID := GetRequestID(r.Context())
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
