package httputil

import (
	"net"
	"net/http"
	"strings"
)

// extractIP extracts client's IP address from the request
// If trustProxy is true, it checks for X-Forward-For and X-Real-IP headers first
func ExtractIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		// Checking for X-Real-IP first
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			if parsedIP := net.ParseIP(strings.TrimSpace(realIP)); parsedIP != nil {
				return parsedIP.String()
			}
		}

		// Checking for X-Forwarded-For
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				clientIP := strings.TrimSpace(parts[0])
				if parsedIP := net.ParseIP(clientIP); parsedIP != nil {
					return parsedIP.String()
				}
			}
		}
	}

	// Fallback to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		if parsedIP := net.ParseIP(r.RemoteAddr); parsedIP != nil {
			return parsedIP.String()
		}
		return r.RemoteAddr
	}
	return host
}
