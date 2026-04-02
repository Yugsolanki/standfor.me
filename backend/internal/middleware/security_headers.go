package middleware

import "net/http"

// SecurityHeadersConfig allows customization of security headers per environment.
// Sane defaults are provided by DefaultSecurityHeadersConfig().
type SecurityHeadersConfig struct {
	// XFrameOptions prevents clickjacking by controlling iframe embedding.
	// Options: "DENY", "SAMEORIGIN"
	// Default: "DENY"
	XFrameOptions string

	// XContentTypeOptions prevents MIME-type sniffing.
	// Default: "nosniff"
	XContentTypeOptions string

	// ReferrerPolicy controls how much referrer info is sent.
	// Default: "strict-origin-when-cross-origin"
	ReferrerPolicy string

	// StrictTransportSecurity enforces HTTPS via HSTS.
	// Default: "max-age=63072000; includeSubDomains; preload" (2 years)
	StrictTransportSecurity string

	// PermissionsPolicy restricts browser features (camera, mic, geolocation, etc.)
	// Default denies most features since Standfor.me is a profile/advocacy platform.
	PermissionsPolicy string

	// CrossOriginOpenerPolicy isolates the browsing context.
	// Default: "same-origin"
	CrossOriginOpenerPolicy string

	// CrossOriginResourcePolicy controls cross-origin resource loading.
	// Default: "same-origin"
	CrossOriginResourcePolicy string

	// CrossOriginEmbedderPolicy controls cross-origin embedding.
	// Default: "require-corp"
	CrossOriginEmbedderPolicy string

	// CacheControl for API responses. Static assets should override this
	// at the handler or CDN level.
	// Default: "no-store, no-cache, must-revalidate, proxy-revalidate"
	CacheControl string

	// EnableHSTS controls whether the Strict-Transport-Security header is set.
	// Set to false in development (when using HTTP).
	// Default: true
	EnableHSTS bool
}

func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		XFrameOptions:       "DENY",
		XContentTypeOptions: "nosniff",
		ReferrerPolicy:      "strict-origin-when-cross-origin",
		// TODO: Ensure cdn.standfor.me AND api.standfor.me serve valid HTTPS
		StrictTransportSecurity: "max-age=63072000; includeSubDomains; preload",
		EnableHSTS:              true,
		PermissionsPolicy: "camera=(), " +
			"microphone=(), " +
			"geolocation=(), " +
			"payment=(), " +
			"usb=(), " +
			"magnetometer=(), " +
			"gyroscope=(), " +
			"accelerometer=(), " +
			"ambient-light-sensor=(), " +
			"autoplay=(), " +
			"encrypted-media=(), " +
			"interest-cohort=()",
		CrossOriginOpenerPolicy: "same-origin",
		// "same-origin" is the safest option, but may require CORS headers on some embedded resources. We can relax this to "same-site" or "unsafe-none" if we run into issues.
		CrossOriginResourcePolicy: "same-origin",
		// TODO: Make sure all embedded resources (images, media) have appropriate CORS headers to allow "require-corp"
		CrossOriginEmbedderPolicy: "require-corp",
		CacheControl:              "no-store, no-cache, must-revalidate, proxy-revalidate",
	}
}

func DevelopmentSecurityHeadersConfig() SecurityHeadersConfig {
	cfg := DefaultSecurityHeadersConfig()
	cfg.EnableHSTS = false
	cfg.CrossOriginEmbedderPolicy = "unsafe-none"
	return cfg
}

// SecurityHeaders (Helmet) injects security headers into every response.
func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			// Core security headers
			if cfg.XContentTypeOptions != "" {
				h.Set("X-Content-Type-Options", cfg.XContentTypeOptions)
			}

			if cfg.ReferrerPolicy != "" {
				h.Set("Referrer-Policy", cfg.ReferrerPolicy)
			}

			if cfg.XFrameOptions != "" {
				h.Set("X-Frame-Options", cfg.XFrameOptions)
			}

			if cfg.PermissionsPolicy != "" {
				h.Set("Permissions-Policy", cfg.PermissionsPolicy)
			}

			// HSTS — only set over HTTPS or when explicitly enabled.
			if cfg.EnableHSTS && cfg.StrictTransportSecurity != "" {
				h.Set("Strict-Transport-Security", cfg.StrictTransportSecurity)
			}

			// Cross-Origin isolation headers
			if cfg.CrossOriginOpenerPolicy != "" {
				h.Set("Cross-Origin-Opener-Policy", cfg.CrossOriginOpenerPolicy)
			}
			if cfg.CrossOriginResourcePolicy != "" {
				h.Set("Cross-Origin-Resource-Policy", cfg.CrossOriginResourcePolicy)
			}
			if cfg.CrossOriginEmbedderPolicy != "" {
				h.Set("Cross-Origin-Embedder-Policy", cfg.CrossOriginEmbedderPolicy)
			}

			// Cache control for API responses
			if cfg.CacheControl != "" {
				h.Set("Cache-Control", cfg.CacheControl)
			}

			// Remove the Server header to avoid leaking implementation details.
			h.Del("Server")
			h.Del("X-Powered-By")

			next.ServeHTTP(w, r)
		})
	}
}
