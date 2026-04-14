package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	internaljwt "github.com/Yugsolanki/standfor-me/internal/pkg/jwt"
	"github.com/Yugsolanki/standfor-me/internal/pkg/response"
)

type contextKey string

const (
	claimsKey contextKey = "auth_claims"
)

// --- Context Helpers ---

// ClaimsFromContext retrieves the AccessTokenClaims stored by the
// Authenticate middleware. Returns nil if the key is absent.
func ClaimsFromContext(ctx context.Context) *domain.AccessTokenClaims {
	claims, _ := ctx.Value(claimsKey).(*domain.AccessTokenClaims)
	return claims
}

// --- Middleware ---

// Authenticate extracts the Bearer token from the Authorization
// header, validates it, and stores the resulting claims on the
// request context. Requests without a valid token are rejected
// with 401 Unauthorized.
func Authenticate(jwtSvc *internaljwt.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const op = "middleware.Authenticate"

			raw, ok := extractBearerToken(r)
			if !ok {
				response.JSONError(w, r, domain.NewUnauthorizedError(
					op,
					"missing or malformed Authorization header",
				))
				return
			}

			claims, err := jwtSvc.ValidateAccessToken(raw)
			if err != nil {
				response.JSONError(w, r, domain.NewUnauthorizedError(op, "invalid or expired access token"))
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth is an alias for Authenticate that reads more clearly
// when stacked in route definitions alongside role checks.
func RequireAuth(jwtSvc *internaljwt.Service) func(http.Handler) http.Handler {
	return Authenticate(jwtSvc)
}

// RequireRole builds a middleware that permits access only to
// users whose role exactly matches one of the provided roles.
// Must be used after Authenticate so claims are present.
//
// Example:
//
//	r.With(RequireRole(domain.RoleAdmin, domain.RoleSuperAdmin)).Get("/admin", h)
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const op = "middleware.RequireRole"

			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				response.JSONError(w, r, domain.NewUnauthorizedError(op, "authenticate required"))
				return
			}

			if _, ok := allowed[claims.Role]; !ok {
				response.JSONError(w, r, domain.NewForbiddenError(op, "you do not have permission to access this resource"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireMinRole builds a middleware that permits access to users
// whose role is at or above the specified minimum in the hierarchy:
//
//	user < moderator < admin < superadmin
//
// Must be used after Authenticate so claims are present.
func RequireMinRole(minRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const op = "middleware.RequireMinRole"

			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				response.JSONError(w, r, domain.NewUnauthorizedError(op, "authentication required"))
				return
			}

			if !hasMinRole(claims.Role, minRole) {
				response.JSONError(w, r, domain.NewForbiddenError(op, "you do not have permission to access this resource"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// --- Helper ---

// roleRank maps each role to a numeric value so we can compare
var roleRank = map[string]int{
	domain.RoleUser:       0,
	domain.RoleModerator:  1,
	domain.RoleAdmin:      2,
	domain.RoleSuperAdmin: 3,
}

// hasMinRole returns true when actualRole has a rank >= minRole
func hasMinRole(actualRole, minRole string) bool {
	actual, aok := roleRank[actualRole]
	minR, mok := roleRank[minRole]
	if !aok || !mok {
		return false
	}
	return actual >= minR
}

// extractBearerToken pulls the raw JWT from the Authorization
// header. Returns ("", false) when the header is absent or does
// not follow the "Bearer <token>" format.
func extractBearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", false
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}
