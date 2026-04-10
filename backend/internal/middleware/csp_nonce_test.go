package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSPNonce_MiddlewareAddsNonce(t *testing.T) {
	t.Parallel()

	handler := CSPNonce()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	csp := rec.Header().Get("Content-Security-Policy")
	assert.True(t, strings.Contains(csp, "nonce-"))
	assert.True(t, strings.Contains(csp, "script-src"))
	assert.True(t, strings.Contains(csp, "style-src"))
}

func TestCSPNonce_NonceInContext(t *testing.T) {
	t.Parallel()

	handler := CSPNonce()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := GetNonce(r)
		assert.NotEmpty(t, nonce)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCSPNonce_GetNonceNoNonce(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	nonce := GetNonce(req)
	assert.Empty(t, nonce)
}

func TestCSPNonce_GetNonceFromContext(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), CSPNonceKey, "test-nonce-123")
	req = req.WithContext(ctx)

	nonce := GetNonce(req)
	assert.Equal(t, "test-nonce-123", nonce)
}

func TestCSPNonce_BaseCSPPresent(t *testing.T) {
	t.Parallel()

	handler := CSPNonce()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")

	assert.True(t, strings.Contains(csp, "default-src 'self'"))
	assert.True(t, strings.Contains(csp, "img-src 'self' data: blob: https://cdn.standfor.me"))
	assert.True(t, strings.Contains(csp, "font-src 'self' https://fonts.gstatic.com"))
	assert.True(t, strings.Contains(csp, "media-src 'self' https://cdn.standfor.me"))
	assert.True(t, strings.Contains(csp, "connect-src 'self' https://api.standfor.me wss://api.standfor.me"))
	assert.True(t, strings.Contains(csp, "frame-ancestors 'none'"))
	assert.True(t, strings.Contains(csp, "base-uri 'self'"))
	assert.True(t, strings.Contains(csp, "form-action 'self'"))
	assert.True(t, strings.Contains(csp, "object-src 'none'"))
	assert.True(t, strings.Contains(csp, "worker-src 'self'"))
}

func TestCSPNonce_ScriptSrcWithNonce(t *testing.T) {
	t.Parallel()

	handler := CSPNonce()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	assert.True(t, strings.Contains(csp, "script-src 'self' 'nonce-"))
}

func TestCSPNonce_StyleSrcWithNonce(t *testing.T) {
	t.Parallel()

	handler := CSPNonce()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	assert.True(t, strings.Contains(csp, "style-src 'self' 'nonce-"))
}

func TestBuildCSPWithNonce(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nonce    string
		expected bool
	}{
		{"with nonce", "abc123", true},
		{"empty nonce", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			csp := buildCSPWithNonce(tt.nonce)
			assert.True(t, strings.Contains(csp, "default-src 'self'"))
			if tt.nonce != "" {
				assert.True(t, strings.Contains(csp, "'nonce-"+tt.nonce+"'"))
			} else {
				assert.True(t, strings.Contains(csp, "script-src 'self'"))
				assert.True(t, strings.Contains(csp, "unsafe-inline"))
			}
		})
	}
}

func TestGenerateNonce_DefaultLength(t *testing.T) {
	t.Parallel()

	nonce, err := generateNonce(0)
	assert.NoError(t, err)
	assert.NotEmpty(t, nonce)
	assert.Greater(t, len(nonce), 10)
}

func TestGenerateNonce_CustomLength(t *testing.T) {
	t.Parallel()

	nonce, err := generateNonce(32)
	assert.NoError(t, err)
	assert.NotEmpty(t, nonce)
	assert.Greater(t, len(nonce), 20)
}

func TestGenerateNonce_NegativeLength(t *testing.T) {
	t.Parallel()

	nonce, err := generateNonce(-5)
	assert.NoError(t, err)
	assert.NotEmpty(t, nonce)
	assert.Greater(t, len(nonce), 10)
}

func TestGenerateNonce_DifferentNonces(t *testing.T) {
	t.Parallel()

	nonce1, err := generateNonce(16)
	assert.NoError(t, err)

	nonce2, err := generateNonce(16)
	assert.NoError(t, err)

	assert.NotEqual(t, nonce1, nonce2, "nonces should be unique")
}

func TestCSPNonce_NonceKeyType(t *testing.T) {
	t.Parallel()

	key := CSPNonceKey
	assert.Equal(t, "csp-nonce", string(key))
}

func TestCSPNonce_MultipleRequestsDifferentNonces(t *testing.T) {
	t.Parallel()

	handler := CSPNonce()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	nonces := make(map[string]bool)

	for range 10 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		csp := rec.Header().Get("Content-Security-Policy")

		var nonce string
		for part := range strings.SplitSeq(csp, ";") {
			if strings.Contains(part, "nonce-") {
				nonce = strings.TrimSpace(strings.Split(part, "nonce-")[1])
				break
			}
		}

		assert.NotEmpty(t, nonce)
		nonces[nonce] = true
	}

	assert.GreaterOrEqual(t, len(nonces), 2, "should have at least 2 different nonces")
}

func TestCSPNonce_EmptyAllowedOrigins(t *testing.T) {
	t.Parallel()

	handler := CSPNonce()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
}

func TestCSPNonce_NonceFormat(t *testing.T) {
	t.Parallel()

	handler := CSPNonce()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := GetNonce(r)
		assert.NotEmpty(t, nonce)
		assert.Greater(t, len(nonce), 10)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
}

func TestCSPNonce_PostMethod(t *testing.T) {
	t.Parallel()

	handler := CSPNonce()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
}

func TestCSPNonce_OptionsMethod(t *testing.T) {
	t.Parallel()

	handler := CSPNonce()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
}
