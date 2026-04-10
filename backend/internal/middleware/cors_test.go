package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORS_SameOriginRequest(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightAllowedOrigin(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://standfor.me")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "https://standfor.me", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", rec.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_PreflightDisallowedOrigin(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, "Forbidden", strings.TrimSpace(rec.Body.String()))
}

func TestCORS_RegularRequestAllowedOrigin(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://www.standfor.me")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "https://www.standfor.me", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", rec.Header().Get("Vary"))
}

func TestCORS_ExposedHeaders(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://standfor.me")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "X-Request-Id, X-Total-Count, X-Page-Number, X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset, RetryAfter", rec.Header().Get("Access-Control-Expose-Headers"))
}

func TestCORS_MaxAge(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	cfg.MaxAge = 3600
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://standfor.me")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "3600", rec.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_MaxAgeZero(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	cfg.MaxAge = 0
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://standfor.me")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_NoExposedHeaders(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://standfor.me"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           0,
	}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://standfor.me")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Expose-Headers"))
}

func TestIsOriginAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		origin   string
		allowed  []string
		expected bool
	}{
		{"exact match", "https://standfor.me", []string{"https://standfor.me"}, true},
		{"no match", "https://evil.com", []string{"https://standfor.me"}, false},
		{"empty allowed", "https://standfor.me", []string{}, false},
		{"one of many", "https://www.standfor.me", []string{"https://standfor.me", "https://www.standfor.me"}, true},
		{"case sensitive", "https://Standfor.me", []string{"https://standfor.me"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isOriginAllowed(tt.origin, tt.allowed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBoolToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    bool
		expected string
	}{
		{true, "true"},
		{false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			result := boolToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultCORSConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()

	assert.Contains(t, cfg.AllowedOrigins, "https://standfor.me")
	assert.Contains(t, cfg.AllowedOrigins, "https://www.standfor.me")
	assert.Contains(t, cfg.AllowedMethods, "GET")
	assert.Contains(t, cfg.AllowedMethods, "POST")
	assert.Contains(t, cfg.AllowedHeaders, "Content-Type")
	assert.Contains(t, cfg.AllowedHeaders, "Authorization")
	assert.Contains(t, cfg.ExposedHeaders, "X-Request-Id")
	assert.True(t, cfg.AllowCredentials)
	assert.Greater(t, cfg.MaxAge, 0)
}

func TestDevelopmentCORSConfig(t *testing.T) {
	t.Parallel()

	cfg := DevelopmentCORSConfig()

	assert.Contains(t, cfg.AllowedOrigins, "http://localhost:5173")
	assert.Contains(t, cfg.AllowedOrigins, "http://127.0.0.1:5173")
	assert.Contains(t, cfg.AllowedOrigins, "http://localhost:3000")
}

func TestCORS_VaryHeader(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://standfor.me")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "Origin", rec.Header().Get("Vary"))
}

func TestCORS_CredentialsFalse(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://standfor.me"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Content-Type"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           0,
	}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://standfor.me")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "false", rec.Header().Get("Access-Control-Allow-Credentials"))
}
