package requestid

import "context"

// requestIDKey is a private type used to store the request ID in the context.
type requestIDKey struct{}

// RequestIDHeader is the canonical header name used to propagate request IDs
// across service boundaries.
const RequestIDHeader = "X-Request-ID"

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

// SetRequestID sets the request ID in the context.
//
// Usage:
//
//	ctx = SetRequestID(ctx, id)
func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}
