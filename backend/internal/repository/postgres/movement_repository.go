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

type MovementRepository struct {
	db *sqlx.DB
}

func NewMovementRepository(db *sqlx.DB) *MovementRepository {
	return &MovementRepository{db: db}
}

// Create creates a new movement
func (r *MovementRepository) Create(ctx context.Context, params domain.CreateMovementParams) (*domain.Movement, error) {
	const op = "MovementRepository.Create"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		INSERT INTO movements (slug, name, short_description, long_description, image_url, icon_url, website_url, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING *`

	var movement domain.Movement
	err := r.db.QueryRowxContext(ctx, query,
		params.Slug,
		params.Name,
		params.ShortDescription,
		params.LongDescription,
		params.ImageURL,
		params.IconURL,
		params.WebsiteURL,
		params.CreatedByUserID,
	).StructScan(&movement)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.NewConflictError(op, "a movement with this slug already exists")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &movement, nil
}

// FindByID finds a movement by its ID
func (r *MovementRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Movement, error) {
	const op = "MovementRepository.FindByID"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT * FROM movements 
		WHERE id = $1 AND deleted_at IS NULL`

	var movement domain.Movement
	if err := r.db.QueryRowxContext(ctx, query, id).StructScan(&movement); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "movement not found")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &movement, nil
}

// FindBySlug finds a movement by its slug
func (r *MovementRepository) FindBySlug(ctx context.Context, slug string) (*domain.Movement, error) {
	const op = "MovementRepository.FindBySlug"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT * FROM movements 
		WHERE slug = $1 AND deleted_at IS NULL`

	var movement domain.Movement
	if err := r.db.QueryRowxContext(ctx, query, slug).StructScan(&movement); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "movement not found")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &movement, nil
}

// Update updates a movement
func (r *MovementRepository) Update(ctx context.Context, id uuid.UUID, params domain.UpdateMovementParams) (*domain.Movement, error) {
	const op = "MovementRepository.Update"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE movements
		SET
			name = COALESCE($2, name),
			short_description = COALESCE($3, short_description),
			long_description = COALESCE($4, long_description),
			image_url = COALESCE($5, image_url),
			icon_url = COALESCE($6, icon_url),
			website_url = COALESCE($7, website_url)
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING *`

	var movement domain.Movement
	err := r.db.QueryRowxContext(ctx, query,
		id,
		params.Name,
		params.ShortDescription,
		params.LongDescription,
		params.ImageURL,
		params.IconURL,
		params.WebsiteURL,
	).StructScan(&movement)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.NewConflictError(op, "a movement with this slug already exists")
		}
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "movement not found")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &movement, nil
}

// UpdateStatus updates the status of a movement
func (r *MovementRepository) UpdateStatus(ctx context.Context, id uuid.UUID, params domain.UpdateMovementStatusParams) (*domain.Movement, error) {
	const op = "MovementRepository.UpdateStatus"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE movements
		SET status = $2
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING *`

	var movement domain.Movement
	err := r.db.QueryRowxContext(ctx, query, id, params.Status).StructScan(&movement)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "movement not found")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &movement, nil
}

// SubmitForReview submits a movement for review
func (r *MovementRepository) SubmitForReview(ctx context.Context, id uuid.UUID, params domain.SubmitForReviewParams) (*domain.Movement, error) {
	const op = "MovementRepository.SubmitForReview"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE movements
		SET status = 'pending_review'
		WHERE id = $1 
			AND deleted_at IS NULL
			AND status = 'draft'
		RETURNING *`

	var movement domain.Movement
	err := r.db.QueryRowxContext(ctx, query, id).StructScan(&movement)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewConflictError(op, "movement cannot be submitted for review - only draft movements can be submitted")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &movement, nil
}

// ReviewMovement reviews a movement
func (r *MovementRepository) ReviewMovement(ctx context.Context, id uuid.UUID, params domain.ReviewMovementParams) (*domain.Movement, error) {
	const op = "MovementRepository.ReviewMovement"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	newStatus := "rejected"
	if params.Approved {
		newStatus = "active"
	}

	const query = `
		UPDATE movements
		SET 
			status = $2,
			reviewed_by_user_id = $3,
			reviewed_at = NOW()
		WHERE id = $1 
			AND deleted_at IS NULL
			AND status = 'pending_review'
		RETURNING *`

	var movement domain.Movement
	err := r.db.QueryRowxContext(ctx, query, id, newStatus, params.ReviewedByUserID).StructScan(&movement)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewConflictError(op, "movement cannot be reviewed - only pending_review movements can be reviewed")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &movement, nil
}

func (r *MovementRepository) IncrementSupporters(ctx context.Context, id uuid.UUID) error {
	const op = "MovementRepository.IncrementSupporters"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE movements 
		SET supporter_count = supporter_count + 1 
		WHERE id = $1 
			AND status = 'active'
			AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return domain.NewInternalError(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewInternalError(op, err)
	}
	if rows == 0 {
		return domain.NewNotFoundError(op, "movement not found")
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return nil
}

func (r *MovementRepository) DecrementSupporters(ctx context.Context, id uuid.UUID) error {
	const op = "MovementRepository.DecrementSupporters"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE movements 
		SET 
			supporter_count = GREATEST(0, supporter_count - 1)
		WHERE id = $1 
			AND status = 'active'
			AND deleted_at IS NULL 
			AND supporter_count > 0`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return domain.NewInternalError(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewInternalError(op, err)
	}
	if rows == 0 {
		return domain.NewNotFoundError(op, "movement not found")
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return nil
}

// UpdateTrendingScore updates the trending score of a movement
func (r *MovementRepository) UpdateTrendingScore(ctx context.Context, id uuid.UUID, score float64) error {
	const op = "MovementRepository.UpdateTrendingScore"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE movements 
		SET trending_score = $2
		WHERE id = $1 
			AND status = 'active'
			AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id, score)
	if err != nil {
		return domain.NewInternalError(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewInternalError(op, err)
	}
	if rows == 0 {
		return domain.NewNotFoundError(op, "movement not found")
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return nil
}

// SoftDelete sets the status of a movement to archived and sets the deleted_at timestamp
func (r *MovementRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const op = "MovementRepository.SoftDelete"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE movements
		SET
			status = 'archived',
			deleted_at = NOW()
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
		return domain.NewNotFoundError(op, "movement not found")
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return nil
}

// Restore restores a movement that was soft deleted (within 30 days)
func (r *MovementRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.Movement, error) {
	const op = "MovementRepository.Restore"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE movements
		SET
			status = 'draft',
			deleted_at = NULL
		WHERE id = $1
			AND status = 'archived'
			AND deleted_at IS NOT NULL
			AND deleted_at >= NOW() - INTERVAL '30 days'
		RETURNING *`

	var movement domain.Movement
	if err := r.db.QueryRowxContext(ctx, query, id).StructScan(&movement); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError(op, "movement not found or restore window has expired")
		}
		return nil, domain.NewInternalError(op, err)
	}

	return &movement, nil
}

// HardDelete permanently deletes a movement that was soft deleted (must be archived first)
func (r *MovementRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	const op = "MovementRepository.HardDelete"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		DELETE FROM movements 
		WHERE id = $1 
			AND status = 'archived'
			AND deleted_at IS NOT NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return domain.NewInternalError(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewInternalError(op, err)
	}
	if rows == 0 {
		return domain.NewNotFoundError(op, "movement not found")
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return nil
}

// List lists all movements with optional filtering and pagination
func (r *MovementRepository) List(ctx context.Context, params domain.ListMovementsParams) ([]domain.Movement, int, error) {
	const op = "MovementRepository.List"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	total, err := r.Count(ctx, params.Status)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []domain.Movement{}, 0, nil
	}

	orderBy := "created_at DESC"
	if params.SortBy != "" {
		orderDir := "DESC"
		if params.Order == "asc" {
			orderDir = "ASC"
		}
		switch params.SortBy {
		case "created_at":
			orderBy = "created_at " + orderDir
		case "updated_at":
			orderBy = "updated_at " + orderDir
		case "supporter_count":
			orderBy = "supporter_count " + orderDir
		case "trending_score":
			orderBy = "trending_score " + orderDir
		}
	}

	whereClause := "deleted_at IS NULL"
	var args []interface{}
	argIndex := 1

	if params.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, params.Status)
		argIndex++
	}

	if params.Search != "" {
		whereClause += fmt.Sprintf(" AND name ILIKE $%d", argIndex)
		args = append(args, "%"+params.Search+"%")
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT * FROM movements
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderBy, argIndex, argIndex+1,
	)
	args = append(args, params.Limit, params.Offset)

	var movements []domain.Movement
	if err := r.db.SelectContext(ctx, &movements, query, args...); err != nil {
		return nil, 0, domain.NewInternalError(op, err)
	}

	return movements, total, nil
}

// ListActive lists all active movements with optional filtering and pagination
func (r *MovementRepository) ListActive(ctx context.Context, params domain.ListMovementsParams) ([]domain.Movement, int, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	params.Status = domain.MovementStatusActive
	return r.List(ctx, params)
}

// ListPendingReview lists all movements pending review with optional filtering and pagination
func (r *MovementRepository) ListPendingReview(ctx context.Context, params domain.ListMovementsParams) ([]domain.Movement, int, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	params.Status = domain.MovementStatusPendingReview
	return r.List(ctx, params)
}

// Count returns the count of movements with optional status filtering
func (r *MovementRepository) Count(ctx context.Context, status string) (int, error) {
	const op = "MovementRepository.Count"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	query := `
		SELECT COUNT(*) FROM movements 
		WHERE deleted_at IS NULL`
	args := []interface{}{}

	if status != "" {
		query += " AND status = $1"
		args = append(args, status)
	}

	var total int
	if err := r.db.QueryRowxContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, domain.NewInternalError(op, err)
	}

	return total, nil
}

// CountActive returns the count of active movements
func (r *MovementRepository) CountActive(ctx context.Context) (int, error) {
	const op = "MovementRepository.CountActive"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT COUNT(*) FROM movements 
		WHERE deleted_at IS NULL AND status = 'active'`

	var total int
	if err := r.db.QueryRowxContext(ctx, query).Scan(&total); err != nil {
		return 0, domain.NewInternalError(op, err)
	}

	return total, nil
}

// CountPendingReview returns the count of movements pending review
func (r *MovementRepository) CountPendingReview(ctx context.Context) (int, error) {
	const op = "MovementRepository.CountPendingReview"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT COUNT(*) FROM movements 
		WHERE deleted_at IS NULL AND status = 'pending_review'`

	var total int
	if err := r.db.QueryRowxContext(ctx, query).Scan(&total); err != nil {
		return 0, domain.NewInternalError(op, err)
	}

	return total, nil
}

// SlugExists checks if a movement with the given slug exists
func (r *MovementRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	const op = "MovementRepository.SlugExists"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT EXISTS (
			SELECT 1 FROM movements
			WHERE slug = $1 
			AND deleted_at IS NULL
		)
	`

	var exists bool
	err := r.db.QueryRowxContext(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, domain.NewInternalError(op, err)
	}

	return exists, nil
}

// Exists checks if a movement with the given ID exists
func (r *MovementRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	const op = "MovementRepository.Exists"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT EXISTS (
			SELECT 1 FROM movements
			WHERE id = $1 
			AND deleted_at IS NULL
		)
	`

	var exists bool
	err := r.db.QueryRowxContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, domain.NewInternalError(op, err)
	}

	return exists, nil
}

// BatchGetByIDs retrieves multiple movements by their IDs
func (r *MovementRepository) BatchGetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Movement, error) {
	const op = "MovementRepository.BatchGetByIDs"

	if len(ids) == 0 {
		return []domain.Movement{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT * FROM movements 
		WHERE id = ANY($1) AND deleted_at IS NULL`

	var movements []domain.Movement
	if err := r.db.SelectContext(ctx, &movements, query, ids); err != nil {
		return nil, domain.NewInternalError(op, err)
	}

	return movements, nil
}

// GetTrending returns the top N trending movements
func (r *MovementRepository) GetTrending(ctx context.Context, limit int) ([]domain.Movement, error) {
	const op = "MovementRepository.GetTrending"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT * FROM movements
		WHERE status = 'active' AND deleted_at IS NULL
		ORDER BY trending_score DESC
		LIMIT $1`

	var movements []domain.Movement
	if err := r.db.SelectContext(ctx, &movements, query, limit); err != nil {
		return nil, domain.NewInternalError(op, err)
	}

	return movements, nil
}

// GetPopular returns the top N popular movements
func (r *MovementRepository) GetPopular(ctx context.Context, limit int) ([]domain.Movement, error) {
	const op = "MovementRepository.GetPopular"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT * FROM movements
		WHERE status = 'active' AND deleted_at IS NULL
		ORDER BY supporter_count DESC
		LIMIT $1`

	var movements []domain.Movement
	if err := r.db.SelectContext(ctx, &movements, query, limit); err != nil {
		return nil, domain.NewInternalError(op, err)
	}

	return movements, nil
}

// GetRecent returns the top N recent movements
func (r *MovementRepository) GetRecent(ctx context.Context, limit int) ([]domain.Movement, error) {
	const op = "MovementRepository.GetRecent"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT * FROM movements
		WHERE status = 'active' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1`

	var movements []domain.Movement
	if err := r.db.SelectContext(ctx, &movements, query, limit); err != nil {
		return nil, domain.NewInternalError(op, err)
	}

	return movements, nil
}

// GetUserMovements returns all movements created by a specific user
func (r *MovementRepository) GetUserMovements(ctx context.Context, userID uuid.UUID) ([]domain.Movement, error) {
	const op = "MovementRepository.GetUserMovements"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		SELECT * FROM movements
		WHERE created_by_user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	var movements []domain.Movement
	if err := r.db.SelectContext(ctx, &movements, query, userID); err != nil {
		return nil, domain.NewInternalError(op, err)
	}

	return movements, nil
}

// Archive archives a movement (status -> 'archived')
// Only active or pending_review movements can be archived
// This is effectively the same as deleting the movement but we keep the record
func (r *MovementRepository) Archive(ctx context.Context, id uuid.UUID) error {
	const op = "MovementRepository.Archive"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	const query = `
		UPDATE movements
		SET status = 'archived'
		WHERE id = $1 AND deleted_at IS NULL
			AND status IN ('draft', 'active', 'rejected')
		RETURNING *`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return domain.NewInternalError(op, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewInternalError(op, err)
	}
	if rows == 0 {
		return domain.NewNotFoundError(op, "movement not found or cannot be archived")
	}

	logger.AddField(ctx, "result_rows_affected", rows)

	return nil
}

// CalculateTrendingScore calculates the trending score for a movement
// Score = (Recent Supporters / Total Supporters) * 100
// Higher score = more trending
func (r *MovementRepository) CalculateTrendingScore(ctx context.Context, id uuid.UUID) (float64, error) {
	const op = "MovementRepository.CalculateTrendingScore"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	m, err := r.FindByID(ctx, id)
	if err != nil {
		return 0, err
	}

	const query = `
		SELECT COUNT(*) FROM movement_supporters
		WHERE movement_id = $1 
			AND created_at >= NOW() - INTERVAL '7 days'`

	var recentSupporters int
	if err := r.db.QueryRowxContext(ctx, query, id).Scan(&recentSupporters); err != nil {
		return 0, domain.NewInternalError(op, err)
	}

	score := float64(recentSupporters)*1.0 + float64(m.SupporterCount)*0.1 + float64(m.TrendingScore)*0.5
	return score, nil
}

// UpdateAllTrendingScores updates the trending score for all active movements
func (r *MovementRepository) UpdateAllTrendingScores(ctx context.Context) (int64, error) {
	const op = "MovementRepository.UpdateAllTrendingScores"

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	const listQuery = `
		SELECT id FROM movements
		WHERE status = 'active' AND deleted_at IS NULL`

	rows, err := r.db.QueryxContext(ctx, listQuery)
	if err != nil {
		return 0, domain.NewInternalError(op, err)
	}
	defer rows.Close()

	var updated int64
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return 0, domain.NewInternalError(op, err)
		}

		score, err := r.CalculateTrendingScore(ctx, id)
		if err != nil {
			slog.Error("failed to calculate trending score", "movement_id", id, "error", err)
			continue
		}

		if err := r.UpdateTrendingScore(ctx, id, score); err != nil {
			slog.Error("failed to update trending score", "movement_id", id, "error", err)
			continue
		}

		updated++
	}

	if err := rows.Err(); err != nil {
		return 0, domain.NewInternalError(op, err)
	}

	logger.AddField(ctx, "updated_movements", updated)

	return updated, nil
}

// SearchMovements searches for active movements by name
func (r *MovementRepository) SearchMovements(ctx context.Context, params domain.SearchMovementsParams) ([]domain.Movement, int, error) {
	const op = "MovementRepository.SearchMovements"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	countQuery := `
		SELECT COUNT(*) FROM movements
		WHERE status = 'active' 
			AND deleted_at IS NULL
			AND name ILIKE $1`

	var total int
	if err := r.db.QueryRowxContext(ctx, countQuery, "%"+params.Query+"%").Scan(&total); err != nil {
		return nil, 0, domain.NewInternalError(op, err)
	}

	if total == 0 {
		return []domain.Movement{}, 0, nil
	}

	const query = `
		SELECT * FROM movements
		WHERE status = 'active' 
			AND deleted_at IS NULL
			AND name ILIKE $1
		ORDER BY 
			similarity(name, $2) DESC,
			supporter_count DESC
		LIMIT $3 OFFSET $4`

	var movements []domain.Movement
	if err := r.db.SelectContext(ctx, &movements, query, "%"+params.Query+"%", params.Query, params.Limit, params.Offset); err != nil {
		return nil, 0, domain.NewInternalError(op, err)
	}

	return movements, total, nil
}

// GetMovementSupporters retrieves supporters for a specific movement
func (r *MovementRepository) GetMovementSupporters(ctx context.Context, movementID uuid.UUID, params domain.ListSupportersParams) ([]domain.User, int, error) {
	const op = "MovementRepository.GetMovementSupporters"

	ctx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	countQuery := `
		SELECT COUNT(*) FROM movement_supporters
		WHERE movement_id = $1 AND deleted_at IS NULL`

	var total int
	if err := r.db.QueryRowxContext(ctx, countQuery, movementID).Scan(&total); err != nil {
		return nil, 0, domain.NewInternalError(op, err)
	}

	if total == 0 {
		return []domain.User{}, 0, nil
	}

	const query = `
		SELECT u.* FROM users u
		INNER JOIN movement_supporters ms ON u.id = ms.user_id
		WHERE ms.movement_id = $1 
			AND ms.deleted_at IS NULL
			AND u.deleted_at IS NULL
			AND u.status = 'active'
		ORDER BY ms.created_at DESC
		LIMIT $2 OFFSET $3`

	var users []domain.User
	if err := r.db.SelectContext(ctx, &users, query, movementID, params.Limit, params.Offset); err != nil {
		return nil, 0, domain.NewInternalError(op, err)
	}

	return users, total, nil
}
