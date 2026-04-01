package middleware

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{
			"https://standfor.me",
			"https://www.standfor.me",
		},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Content-Type",
			"Authorization",
			"X-Requested-With",
			"Accept",
			"Origin",
			"X-CSRF-Token",
		},
		ExposedHeaders: []string{
			"X-Request-Id",
			"X-Total-Count",
			"X-Page-Number",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
			"RetryAfter",
		},
		AllowCredentials: true,
		MaxAge:           int((24 * time.Hour).Seconds()),
	}
}

func DevelopmentCORSConfig() CORSConfig {
	cfg := DefaultCORSConfig()
	cfg.AllowedOrigins = []string{
		"http://localhost:5173", // Vite default
		"http://127.0.0.1:5173", // Vite default
		"http://localhost:5174", // Vite fallback
		"http://localhost:3000", // Alternative dev server
		"http://127.0.0.1:3000", // Alternative dev server
	}
	return cfg
}

func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")
	exposed := strings.Join(cfg.ExposedHeaders, ", ")
	credentials := boolToString(cfg.AllowCredentials)
	maxAge := fmt.Sprintf("%d", cfg.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// If the Origin header is not present,
			// it's a same-origin request, so we can skip CORS headers
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// If the Origin header is present but not allowed,
			// we can either reject the request or serve it without CORS headers
			if !isOriginAllowed(origin, cfg.AllowedOrigins) {
				// Reject the request with 403 Forbidden
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Set CORS headers for allowed origins
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			w.Header().Set("Access-Control-Allow-Credentials", credentials)
			if len(cfg.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", exposed)
			}
			if cfg.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", maxAge)
			}
			w.Header().Set("Vary", "Origin")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isOriginAllowed(origin string, allowed []string) bool {
	return slices.Contains(allowed, origin)
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
