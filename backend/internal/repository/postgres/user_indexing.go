package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Yugsolanki/standfor-me/internal/service/search"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// UserIndexingRepository fetches the data needed to build a user search document.
type UserIndexingRepository struct {
	db *sqlx.DB
}

// NewUserIndexingRepository constructs the repository.
func NewUserIndexingRepository(db *sqlx.DB) *UserIndexingRepository {
	return &UserIndexingRepository{db: db}
}

// GetUserForIndexing fetches all data needed for a user search document.
// PII fields (email, password_hash) are explicitly excluded from this query.
func (r *UserIndexingRepository) GetUserForIndexing(
	ctx context.Context,
	userID string,
) (*search.UserIndexData, error) {
	const query = `
		SELECT
			u.id::TEXT,
			u.username,
			u.display_name,
			COALESCE(u.bio, '') AS bio,
			COALESCE(u.location, '') AS location,
			u.profile_visibility::TEXT,
			u.status::TEXT,
			(u.deleted_at IS NOT NULL) AS is_deleted,
			u.created_at,
			u.updated_at,

			-- Aggregated movement id, name and slug for full-text search
			COALESCE(
				ARRAY_AGG(
					m.id ORDER BY um.display_order ASC, um.created_at DESC
				) FILTER (
					WHERE m.id IS NOT NULL
					  AND um.is_public = TRUE
					  AND um.removed_at IS NULL
				),
				'{}'
			) AS movement_ids,

			COALESCE(
				ARRAY_AGG(
					m.name ORDER BY um.display_order ASC, um.created_at DESC
				) FILTER (
					WHERE m.id IS NOT NULL
					  AND um.is_public = TRUE
					  AND um.removed_at IS NULL
				),
				'{}'
			) AS movement_names,

			COALESCE(
				ARRAY_AGG(
					m.slug ORDER BY um.display_order ASC, um.created_at DESC
				) FILTER (
					WHERE m.id IS NOT NULL
					  AND um.is_public = TRUE
					  AND um.removed_at IS NULL
				),
				'{}'
			) AS movement_slugs,

			-- Total public, non-removed movements
			COUNT(um.id) FILTER (
				WHERE um.is_public = TRUE AND um.removed_at IS NULL
			)::INT AS total_movements_supported,

			-- Average verification tier across public movements
			COALESCE(
				AVG(um.verification_tier) FILTER (
					WHERE um.is_public = TRUE AND um.removed_at IS NULL
				), 0
			)::FLOAT AS avg_verification_tier,

			-- Minimum verification tier (their least committed cause)
			COALESCE(
				MIN(um.verification_tier) FILTER (
					WHERE um.is_public = TRUE AND um.removed_at IS NULL
				), 0
			)::INT AS min_verification_tier,

			-- Highest badge earned across all movements
			COALESCE(
				(
					SELECT badge_level FROM user_movements
					WHERE user_id = u.id
					  AND is_public = TRUE
					  AND removed_at IS NULL
					ORDER BY
						CASE badge_level
							WHEN 'platinum' THEN 4
							WHEN 'gold'     THEN 3
							WHEN 'silver'   THEN 2
							WHEN 'bronze'   THEN 1
							ELSE 0
						END DESC
					LIMIT 1
				), ''
			) AS max_badge_level,

			-- Count of movements where the user is "Engaged" or higher
			COUNT(um.id) FILTER (
				WHERE um.is_public = TRUE
				  AND um.removed_at IS NULL
				  AND um.verification_tier >= 2
			)::INT AS verified_movement_count

		FROM users u
		LEFT JOIN user_movements um ON um.user_id = u.id
		LEFT JOIN movements m ON m.id = um.movement_id
			AND m.deleted_at IS NULL
			AND m.status = 'active'
		WHERE u.id = $1
		GROUP BY u.id
	`

	data := &search.UserIndexData{}
	var movementIds, movementNames, movementSlugs []string

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&data.ID,
		&data.Username,
		&data.DisplayName,
		&data.Bio,
		&data.Location,
		&data.ProfileVisibility,
		&data.Status,
		&data.IsDeleted,
		&data.CreatedAt,
		&data.UpdatedAt,
		&movementIds,
		&movementNames,
		&movementSlugs,
		&data.TotalMovementsSupported,
		&data.AvgVerificationTier,
		&data.MinVerificationTier,
		&data.MaxBadgeLevel,
		&data.VerifiedMovementCount,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user %s not found for indexing", userID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying user for indexing: %w", err)
	}

	// Ensure nil slices become empty slices for consistent JSON output.
	if movementIds == nil {
		movementIds = []string{}
	}
	if movementNames == nil {
		movementNames = []string{}
	}
	if movementSlugs == nil {
		movementSlugs = []string{}
	}
	data.MovementIDs = movementIds
	data.MovementNames = movementNames
	data.MovementSlugs = movementSlugs

	// ── Category query (separate to avoid row multiplication with aggregates)
	const categoryQuery = `
		SELECT
            c.id::TEXT,
            c.slug,
            c.name
        FROM user_movements um
        JOIN movement_categories mc ON mc.movement_id = um.movement_id
        JOIN categories c ON c.id = mc.category_id
        WHERE um.user_id = $1
          AND um.removed_at IS NULL  -- Only count active supports
          AND c.is_active = TRUE
        GROUP BY 
            c.id, 
            c.slug, 
            c.name, 
            c.display_order
        ORDER BY c.display_order ASC, c.name ASC
	`

	rows, err := r.db.QueryContext(ctx, categoryQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("querying categories for user %s: %w", userID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var catID, catSlug, catName string
		if err := rows.Scan(&catID, &catSlug, &catName); err != nil {
			return nil, fmt.Errorf("scanning category row: %w", err)
		}
		data.CategoryIDs = append(data.CategoryIDs, catID)
		data.CategorySlugs = append(data.CategorySlugs, catSlug)
		data.CategoryNames = append(data.CategoryNames, catName)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating category rows: %w", err)
	}

	// Initialize slices so JSON never serializes as null.
	if data.CategoryIDs == nil {
		data.CategoryIDs = []string{}
		data.CategorySlugs = []string{}
		data.CategoryNames = []string{}
	}

	return data, nil
}

// GetAllUserIDsForReindex returns every non-deleted, public user ID.
// Private profiles are still returned here — the search repository layer
// enforces visibility filtering at query time, not at index time.
// Indexing private profiles allows admins to search across all profiles
// and allows the visibility filter to be applied consistently.
func (r *UserIndexingRepository) GetAllUserIDsForReindex(ctx context.Context) ([]string, error) {
	const query = `
		SELECT id::TEXT FROM users
		WHERE deleted_at IS NULL
			AND stauts != 'draft'
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying user IDs for reindex: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning user id: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating user id rows: %w", err)
	}

	return ids, nil
}

// GetAllUsersForBulkIndex is the mirror of GetAllMovementsForBulkIndex.
// It pages through all user IDs and invokes processBatch for each chunk.
//
// The caller (reindex command) is responsible for calling IndexUser on
// each ID within the batch callback. This design keeps the repository
// responsible only for data access, not for orchestration.
func (r *UserIndexingRepository) GetAllUsersForBulkIndex(
	ctx context.Context,
	batchSize int,
	processBatch func(ids []string) error,
) error {
	ids, err := r.GetAllUserIDsForReindex(ctx)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	for i := 0; i < len(ids); i += batchSize {
		end := min(i+batchSize, len(ids))

		batch := ids[i:end]

		if err := processBatch(batch); err != nil {
			return fmt.Errorf("processing user batch starting at index %d: %w", i, err)
		}
	}

	return nil
}

// ---------------------------------------
// Internal helpers
// ---------------------------------------

// pqStringArray is a local type alias that implements sql.Scanner for
// Postgres TEXT[] columns. The lib/pq package provides pq.Array() but
// using a named type here makes the Scan call more readable.
type pqStringArray []string

func (a *pqStringArray) Scan(src any) error {
	// Delegate to pq's array parsing logic.
	return (&pq.GenericArray{A: (*[]string)(a)}).Scan(src)
}

// Use unused
var _ pqStringArray
