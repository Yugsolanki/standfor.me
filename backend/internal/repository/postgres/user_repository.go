package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/pkg/logger"
)

const defaultQueryTimeout = 5 * time.Second

// UserRepository provides CRUD operations against the users table.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository returns a new UserRepository backed by the given connection pool.
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// --- Create ---

// Create inserts a new user and returns the created row.
func (r *UserRepository) Create(ctx context.Context, params domain.CreateUserParams) (*domain.User, error) {
	const op = "UserRepository.Create"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		INSERT INTO users (username, email, password_hash, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING *`

	var user domain.User
	err := r.db.QueryRowxContext(ctx, query,
		params.Username,
		params.Email,
		params.PasswordHash,
		params.DisplayName,
	).StructScan(&user)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.NewConflictError(op, "a user with this username or email already exists")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &user, nil
}

// --- Finders ---

// FindByID returns a non-deleted user by primary key.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const op = "UserRepository.FindByID"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT * FROM users 
		WHERE id = $1 AND deleted_at IS NULL`

	var user domain.User
	if err := r.db.QueryRowxContext(ctx, query, id).StructScan(&user); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "user not found")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &user, nil
}

// FindByUsername returns a non-deleted user by username.
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	const op = "UserRepository.FindByUsername"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT * FROM users 
		WHERE username = $1 AND deleted_at IS NULL`

	var user domain.User
	if err := r.db.QueryRowxContext(ctx, query, username).StructScan(&user); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "user not found")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &user, nil
}

// FindByEmail returns a non-deleted user by email address.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const op = "UserRepository.FindByEmail"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT * FROM users 
		WHERE email = $1 AND deleted_at IS NULL`

	var user domain.User
	if err := r.db.QueryRowxContext(ctx, query, email).StructScan(&user); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "user not found")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &user, nil
}

// --- Updates ---

// Update applies a partial update to a user using COALESCE to preserve unchanged fields.
func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, params domain.UpdateUserParams) (*domain.User, error) {
	const op = "UserRepository.Update"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE users
		SET
			display_name = COALESCE($2, display_name),
			bio = COALESCE($3, bio),
			avatar_url = COALESCE($4, avatar_url),
			location = COALESCE($5, location),
			profile_visibility = COALESCE($6, profile_visibility),
			embed_enabled = COALESCE($7, embed_enabled)
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING *`

	var user domain.User
	err := r.db.QueryRowxContext(ctx, query,
		id,
		params.DisplayName,
		params.Bio,
		params.AvatarURL,
		params.Location,
		params.ProfileVisibility,
		params.EmbedEnabled,
	).StructScan(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "user not found")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &user, nil
}

// UpdateUsername updates the username of a user.
func (r *UserRepository) UpdateUsername(ctx context.Context, id uuid.UUID, params domain.UpdateUsernameParams) (*domain.User, error) {
	const op = "UserRepository.UpdateUsername"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE users
		SET
			username = $2
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING *`

	var user domain.User
	err := r.db.QueryRowxContext(ctx, query,
		id,
		params.Username,
	).StructScan(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "user not found")
		}
		if isUniqueViolation(err) {
			return nil, domain.NewConflictError(op, "a user with this username or email already exists")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &user, nil
}

// ChangePassword updates only the password hash.
func (r *UserRepository) ChangePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	const op = "UserRepository.ChangePassword"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE users
		SET password_hash = $2
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id, passwordHash)
	if err != nil {
		return domain.NewInternalError(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewInternalError(op, err)
	}
	if rows == 0 {
		return domain.NewNotFoundError(op, "user not found")
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return nil
}

// UpdateRole updates the role of a user.
func (r *UserRepository) UpdateRole(ctx context.Context, id uuid.UUID, params domain.UpdateRoleParams) (*domain.User, error) {
	const op = "UserRepository.UpdateRole"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE users
		SET
			role = $2
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING *`

	var user domain.User
	err := r.db.QueryRowxContext(ctx, query,
		id,
		params.Role,
	).StructScan(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "user not found")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &user, nil
}

// UpdateStatus updates the status of a user.
func (r *UserRepository) UpdateStatus(ctx context.Context, id uuid.UUID, params domain.UpdateStatusParams) (*domain.User, error) {
	const op = "UserRepository.UpdateStatus"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE users
		SET
			status = $2
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING *`

	var user domain.User
	err := r.db.QueryRowxContext(ctx, query,
		id,
		params.Status,
	).StructScan(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "user not found")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &user, nil
}

// UpdateLastLogin sets the last_login_at timestamp to now.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	const op = "UserRepository.UpdateLastLogin"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE users 
		SET 
			last_login_at = NOW() 
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return domain.NewInternalError(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewInternalError(op, err)
	}
	if rows == 0 {
		return domain.NewNotFoundError(op, "user not found")
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return nil
}

// VerifyEmail marks the user's email as verified.
func (r *UserRepository) VerifyEmail(ctx context.Context, id uuid.UUID) error {
	const op = "UserRepository.VerifyEmail"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE users
		SET email_verified_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return domain.NewInternalError(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewInternalError(op, err)
	}
	if rows == 0 {
		return domain.NewNotFoundError(op, "user not found")
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return nil
}

// --- Deletion & Restoration ---

// SoftDelete marks a user as deleted and scrubs public PII for GDPR compliance.
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const op = "UserRepository.SoftDelete"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE users
		SET
			deleted_at = NOW(),
			status = 'deactivated',
			profile_visibility = 'private',
			display_name = 'Deleted User ' || id::text,
			bio = NULL,
			avatar_url = NULL,
			location = NULL
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return domain.NewInternalError(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewInternalError(op, err)
	}
	if rows == 0 {
		return domain.NewNotFoundError(op, "user not found")
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return nil
}

// Restore re-activates a soft-deleted user within the 30-day recovery window.
func (r *UserRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const op = "UserRepository.Restore"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE users
		SET
			deleted_at = NULL,
			status = 'active',
			profile_visibility = 'public'
		WHERE id = $1
		AND deleted_at IS NOT NULL
		AND deleted_at >= NOW() - INTERVAL '30 days'
		RETURNING *`

	var user domain.User
	if err := r.db.QueryRowxContext(ctx, query, id).StructScan(&user); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "user not found or restore window has expired")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &user, nil
}

// AnonymizeExpired permanently scrubs PII for users deleted more than 30 days ago.
// Returns the number of users anonymized.
func (r *UserRepository) AnonymizeExpired(ctx context.Context) (int64, error) {
	const op = "UserRepository.AnonymizeExpired"

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	const query = `
		UPDATE users
		SET
			password_hash = NULL,
			email = 'deleted_' || id::text || '@deleted.invalid',
			email_verified_at = NULL,
			status = 'deactivated'
		WHERE deleted_at IS NOT NULL
		AND deleted_at <= NOW() - INTERVAL '30 days'
		AND email NOT LIKE 'deleted_%@deleted.invalid'`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, domain.NewInternalError(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, domain.NewInternalError(op, err)
	}

	if rows > 0 {
		slog.Info("anonymized expired users", "count", rows)
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return rows, nil
}

// HardDelete permanently removes a user from the database.
// This is intended for admin use only.
func (r *UserRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	const op = "UserRepository.HardDelete"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		DELETE FROM users 
		WHERE id = $1 AND deleted_at IS NOT NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return domain.NewInternalError(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewInternalError(op, err)
	}
	if rows == 0 {
		return domain.NewNotFoundError(op, "user not found")
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return nil
}

// --- Listing ---

// List returns a paginated slice of non-deleted users ordered by creation date (newest first),
// along with the total count.
func (r *UserRepository) List(ctx context.Context, params domain.ListUsersParams) ([]domain.User, int, error) {
	const op = "UserRepository.List"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	// Count total
	total, err := r.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []domain.User{}, 0, nil
	}

	// Fetch page
	const query = `
		SELECT * FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	var users []domain.User
	if err := r.db.SelectContext(ctx, &users, query, params.Limit, params.Offset); err != nil {
		return nil, 0, domain.NewInternalError(op, err)
	}

	return users, total, nil
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	const op = "UserRepository.Count"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT COUNT(*) FROM users 
		WHERE deleted_at IS NULL`

	var total int
	if err := r.db.QueryRowxContext(ctx, query).Scan(&total); err != nil {
		return 0, domain.NewInternalError(op, err)
	}

	return total, nil
}

// --- Existence Checks ---

// UsernameExists returns true if a non-deleted user with the given username exists.
func (r *UserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	const op = "UserRepository.UsernameExists"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	query := `
		SELECT EXISTS (SELECT 1 FROM users WHERE username = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRowxContext(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, domain.NewInternalError(op, err)
	}

	return exists, nil
}

// EmailExists returns true if a non-deleted user with the given email exists.
func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	const op = "UserRepository.EmailExists"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	var exists bool
	err := r.db.QueryRowxContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`,
		email,
	).Scan(&exists)
	if err != nil {
		return false, domain.NewInternalError(op, err)
	}

	return exists, nil
}

// --- Helpers ---

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation (23505).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// pgx wraps constraint violations with code "23505"
	return fmt.Sprintf("%v", err) != "" && contains(err.Error(), "23505")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
