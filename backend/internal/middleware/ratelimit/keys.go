package ratelimit

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Yugsolanki/standfor-me/internal/pkg/httputil"
)

// KeyFunc extracts the rate limiting key from the HTTP request
// The returned key is then identify the client for rate limiting purpose.
type KeyFunc func(r *http.Request) (string, error)

// KeyByIP returns a KeyFunc that extracts client's IP address
// If trustPoxy is true, it checks for X-Forward-For and X-Real-IP headers first
func KeyByIP(trustProxy bool) KeyFunc {
	return func(r *http.Request) (string, error) {
		ip := httputil.ExtractIP(r, trustProxy)
		if ip == "" {
			return "", fmt.Errorf("ratelimit: unable to extract IP from request")
		}
		return fmt.Sprintf("ip:%s", ip), nil
	}
}

// KeyByHeader returns a KeyFunc that extracts a key from header
// Useful for API key-based rate limiting
func KeyByHeader(header string) KeyFunc {
	return func(r *http.Request) (string, error) {
		val := r.Header.Get(header)
		if val == "" {
			return "", fmt.Errorf("ratelimit: header %q not found in request", header)
		}
		return fmt.Sprintf("header:%s:%s", strings.ToLower(header), val), nil
	}
}

// KeyByRoute returns a KeyFunc that combines client ip with request route
// This allows per-route rate limiting
func KeyByRoute(trustProxy bool) KeyFunc {
	return func(r *http.Request) (string, error) {
		ip := httputil.ExtractIP(r, trustProxy)
		if ip == "" {
			return "", fmt.Errorf("ratelimit: unable to extract IP from request")
		}
		return fmt.Sprintf("route:%s:%s:%s", r.Method, r.URL, ip), nil
	}
}

// KeyByUserID returns a KeyFunc that extracts user ID from request context
// Falls back to IP-based key if user ID is not found
func KeyByUserID(contextKey any, trustProxy bool) KeyFunc {
	return func(r *http.Request) (string, error) {
		if userID := r.Context().Value(contextKey); userID != nil {
			return fmt.Sprintf("user:%v", userID), nil
		}

		// Fall-Back to IP-based key for authentication requests
		ip := httputil.ExtractIP(r, trustProxy)
		if ip == "" {
			return "", fmt.Errorf("ratelimit: unable to extract identifier from request")
		}
		return fmt.Sprintf("ip:%s", ip), nil
	}
}

// ComposeKeys combines multiple key functions into a single key
// The resulting key is a concatenation of all individual keys separated by "|".
func ComposeKeys(funcs ...KeyFunc) KeyFunc {
	return func(r *http.Request) (string, error) {
		parts := make([]string, 0, len(funcs))
		for _, fn := range funcs {
			key, err := fn(r)
			if err != nil {
				return "", err
			}
			parts = append(parts, key)
		}
		return strings.Join(parts, "|"), nil
	}
}
