package middleware

import (
	"net/http"

	"github.com/Yugsolanki/standfor-me/internal/pkg/requestid"
	"github.com/google/uuid"
)

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
		id := r.Header.Get(requestid.RequestIDHeader)
		if id == "" {
			id = uuid.New().String()
		}

		// Set on response so client can correlate
		w.Header().Set(requestid.RequestIDHeader, id)

		// Store in context for downstream access
		ctx := requestid.SetRequestID(r.Context(), id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
