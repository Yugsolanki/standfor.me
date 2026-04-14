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
	ID        uuid.UUID  `db:"id" json:"id" `
	UserID    uuid.UUID  `db:"user_id" json:"user_id" `
	TokenHash string     `db:"token_hash" json:"token_hash" `
	ExpiresAt time.Time  `db:"expires_at" json:"expires_at" `
	CreatedAt time.Time  `db:"created_at" json:"created_at" `
	RevokedAt *time.Time `db:"revoked_at" json:"revoked_at" `
}

// CreateRefreshTokenParams are the parameters required to
// persist a new refresh token record.
type CreateRefreshTokenParams struct {
	ID        uuid.UUID `db:"id" validate:"required"`
	UserID    uuid.UUID `db:"user_id" validate:"required"`
	TokenHash string    `db:"token_hash" validate:"required"`
	ExpiresAt time.Time `db:"expires_at" validate:"required,gt"`
}
