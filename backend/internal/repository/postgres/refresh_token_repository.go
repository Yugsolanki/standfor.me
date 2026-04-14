package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RefreshTokenRepository handles all persistence operations for refresh tokens.
type RefreshTokenRepository struct {
	db *sqlx.DB
}

// NewRefreshTokenRepository constructs a RefreshTokenRepository.
func NewRefreshTokenRepository(db *sqlx.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create persists a new refresh token record.
func (r *RefreshTokenRepository) Create(ctx context.Context, params domain.CreateRefreshTokenParams) (*domain.RefreshToken, error) {
	const op = "RefreshTokenRepository.Create"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at`

	var record domain.RefreshToken
	err := r.db.QueryRowxContext(ctx, query,
		params.ID,
		params.UserID,
		params.TokenHash,
		params.ExpiresAt,
	).StructScan(&record)
	if err != nil {
		return nil, domain.NewInternalError(op, err)
	}

	return &record, nil
}

// FindByTokenHash looks up an active (non-revoked, non-expired)
// refresh token by its hash. Returns domain.ErrNotFound when no
// matching record exists.
func (r *RefreshTokenRepository) FindByTokenHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	const op = "RefreshTokenRepository.FindByTokenHash"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
			AND revoked_at IS NULL
			AND expires_at > NOW()`

	var row domain.RefreshToken
	if err := r.db.GetContext(ctx, &row, query, hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "refresh token not found or expired")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &row, nil
}

// Revoke marks a single token as revoked using its hash.
// It is a no-op (no error) when the token does not exist.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, hash string) error {
	const op = "RefreshTokenRepository.Revoke"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1
			AND revoked_at IS NULL`

	if _, err := r.db.ExecContext(ctx, query, hash); err != nil {
		return domain.NewInternalError(op, err)
	}

	return nil
}

// RevokeAllForUser revokes every active refresh token belonging
// to a user. Used during logout-all and password changes.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	const op = "RefreshTokenRepository.RevokeAllForUser"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1
			AND revoked_at IS NULL`

	if _, err := r.db.ExecContext(ctx, query, userID); err != nil {
		return domain.NewInternalError(op, err)
	}

	return nil
}

// DeleteExpired hard-deletes all expired or revoked tokens.
// Intended to be called by the background worker on a schedule.
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	const op = "RefreshTokenRepository.DeleteExpired"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		DELETE FROM refresh_tokens
		WHERE expires_at < NOW()
			OR revoked_at IS NOT NULL`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, domain.NewInternalError(op, err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, domain.NewInternalError(op, err)
	}

	return n, nil
}
