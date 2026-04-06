package domain

import (
	"time"

	"github.com/google/uuid"
)

// TokenPair holds both tokens returned to the client after
// a successful authentication or refresh operation.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AccessTokenClaims represents the validated, parsed claims
// extracted from a JWT access token. It is stored on the
// request context after successful authentication.
type AccessTokenClaims struct {
	UserID uuid.UUID
	Role   string
	Email  string
}

// RefreshToken is the domain model for a persisted refresh
// token record stored in the database.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

// CreateRefreshTokenParams are the parameters required to
// persist a new refresh token record.
type CreateRefreshTokenParams struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
}
