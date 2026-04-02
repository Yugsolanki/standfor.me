package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Yugsolanki/standfor-me/internal/pkg/crypto"
)

type nonceKey string

const (
	CSPNonceKey nonceKey = "csp-nonce"
	NonceLength int      = 16
	BaseCSP     string   = "default-src 'self'; " +
		"img-src 'self' data: blob: https://cdn.standfor.me; " +
		"font-src 'self' https://fonts.gstatic.com; " +
		"media-src 'self' https://cdn.standfor.me; " +
		"connect-src 'self' https://api.standfor.me wss://api.standfor.me; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"object-src 'none'; " +
		"worker-src 'self'"
	// Note: we add script-src and style-src in buildCSPWithNonce
)

// CSPNonce is a middleware that generates nonces for script and style tags and adds them to the request context.
func CSPNonce() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nonce, err := generateNonce(NonceLength)
			if err != nil {
				// TODO: Log this in production it indicates a serious entropy issue.
				w.Header().Set("Content-Security-Policy", BaseCSP)
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), CSPNonceKey, nonce)
			r = r.WithContext(ctx)

			w.Header().Set("Content-Security-Policy", buildCSPWithNonce(nonce))

			next.ServeHTTP(w, r)
		})
	}
}

func buildCSPWithNonce(nonce string) string {
	if nonce != "" {
		return BaseCSP +
			"; script-src 'self' 'nonce-" + nonce + "'" +
			"; style-src 'self' 'nonce-" + nonce + "'"
	}

	return BaseCSP +
		"; script-src 'self'" +
		"; style-src 'self' 'unsafe-inline'"
}

func generateNonce(length int) (string, error) {
	if length <= 0 {
		length = 16
	}

	b, err := crypto.GenerateRandomBytes(length)
	if err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	return crypto.EncodeBase64(b), nil
}

func GetNonce(r *http.Request) string {
	if nonce, ok := r.Context().Value(CSPNonceKey).(string); ok {
		return nonce
	}
	return ""
}
