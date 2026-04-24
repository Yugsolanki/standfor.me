package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Yugsolanki/standfor-me/internal/service/search"
	"github.com/jmoiron/sqlx"
)

type MovementIndexingRepository struct {
	db *sqlx.DB
}

func NewMovementIndexingRepository(db *sqlx.DB) *MovementIndexingRepository {
	return &MovementIndexingRepository{db: db}
}

// GetMovementForIndexing executes the full denormalization query for a single
// movement. This JOIN across 5 tables is intentionally run only during indexing,
// not during normal read operations, to keep the hot path fast.
func (r *MovementIndexingRepository) GetMovementForIndexing(
	ctx context.Context,
	movementID string,
) (*search.MovementIndexData, error) {
	const movementQuery = `
		SELECT
			m.id,
			m.slug,
			m.name,
			m.short_description,
			COALESCE(m.long_description, '')      AS long_description,
			COALESCE(m.image_url, '')             AS image_url,
			COALESCE(m.icon_url, '')              AS icon_url,
			COALESCE(m.website_url, '')           AS website_url,
			m.status,
			m.supporter_count,
			m.trending_score,
			COALESCE(m.created_by_user_id::TEXT, '') AS created_by_user_id,
			m.created_at,
			m.updated_at,

			-- Organization fields (NULL if unclaimed)
			COALESCE(o.id::TEXT, '')              AS claimed_by_org_id,
			COALESCE(o.name, '')                  AS organization_name,
			COALESCE(o.is_verified, FALSE)       AS org_has_verified,

			-- Depth of Commitment 
			COALESCE(
				AVG(um.verification_tier) FILTER (
					WHERE um.is_public = TRUE AND um.removed_at IS NULL
				), 0
			)::FLOAT AS avg_verification_tier,

			COALESCE(
                COUNT(um.id) FILTER (
                    WHERE um.is_public = TRUE 
                      AND um.removed_at IS NULL 
                      AND um.verification_tier > 0
                ), 0
            )::INT AS verified_supporter_count,

			COALESCE(
                COUNT(um.id) FILTER (
                    WHERE um.is_public = TRUE 
                      AND um.removed_at IS NULL 
                      AND (um.verification_tier = 0 OR um.verification_tier IS NULL)
                ), 0
            )::INT AS unverified_supporter_count,

			COALESCE(
				MIN(um.verification_tier) FILTER (
					WHERE um.is_public = TRUE AND um.removed_at IS NULL
				), 0
			)::INT AS min_verification_tier,

			COALESCE(
				MAX(um.verification_tier) FILTER (
					WHERE um.is_public = TRUE AND um.removed_at IS NULL
				), 0
			)::INT AS max_verification_tier,

			-- Badge levels ordered by the custom badge enum/ordering
			COALESCE(
				(
					SELECT badge_level FROM user_movements
					WHERE movement_id = m.id
					  AND is_public = TRUE
					  AND removed_at IS NULL
					ORDER BY
						CASE badge_level
							WHEN 'diamond'	THEN 5
							WHEN 'platinum' THEN 4
							WHEN 'gold'     THEN 3
							WHEN 'silver'   THEN 2
							WHEN 'bronze'   THEN 1
							ELSE 0
						END ASC
					LIMIT 1
				), 'bronze'
			) AS min_badge_level,

			COALESCE(
				(
					SELECT badge_level FROM user_movements
					WHERE movement_id = m.id
					  AND is_public = TRUE
					  AND removed_at IS NULL
					ORDER BY
						CASE badge_level
							WHEN 'diamond'	THEN 5
							WHEN 'platinum' THEN 4
							WHEN 'gold'     THEN 3
							WHEN 'silver'   THEN 2
							WHEN 'bronze'   THEN 1
							ELSE 0
						END DESC
					LIMIT 1
				), 'bronze'
			) AS max_badge_level,

			-- Tier Distribution
			(
                SELECT COALESCE(jsonb_object_agg(tier::TEXT, tier_count), '{}'::jsonb)
                FROM (
                    SELECT verification_tier AS tier, COUNT(*) AS tier_count
                    FROM user_movements
                    WHERE movement_id = m.id 
                      AND is_public = TRUE 
                      AND removed_at IS NULL 
                      AND verification_tier IS NOT NULL
                    GROUP BY verification_tier
                ) t
            ) AS tier_distribution,

			-- Badge Distribution
			(
                SELECT COALESCE(jsonb_object_agg(badge::TEXT, badge_count), '{}'::jsonb)
                FROM (
                    SELECT badge_level AS badge, COUNT(*) AS badge_count
                    FROM user_movements
                    WHERE movement_id = m.id 
                      AND is_public = TRUE 
                      AND removed_at IS NULL 
                      AND badge_level IS NOT NULL
                    GROUP BY badge_level
                ) b
            ) AS badge_distribution

		FROM movements m
		LEFT JOIN organizations o ON o.id = m.claimed_by_org_id
			AND o.deleted_at IS NULL
		LEFT JOIN user_movements um ON um.movement_id = m.id
		WHERE m.id = $1
		  AND m.deleted_at IS NULL
		GROUP BY
			m.id, m.slug, m.name, m.short_description, m.long_description,
			m.image_url, m.icon_url, m.status, m.supporter_count,
			m.trending_score, m.created_by_user_id, m.created_at, m.updated_at,
			o.id, o.name, o.is_verified
	`

	data := &search.MovementIndexData{}
	var tierDistRaw, badgeDistRaw []byte
	err := r.db.QueryRowContext(ctx, movementQuery, movementID).Scan(
		&data.ID,
		&data.Slug,
		&data.Name,
		&data.ShortDescription,
		&data.LongDescription,
		&data.ImageURL,
		&data.IconURL,
		&data.WebsiteURL,
		&data.Status,
		&data.SupporterCount,
		&data.TrendingScore,
		&data.CreatedByUserID,
		&data.CreatedAt,
		&data.UpdatedAt,
		&data.ClaimedByOrgID,
		&data.OrganizationName,
		&data.HasVerifiedOrg,
		&data.AvgVerificationTier,
		&data.VerifiedSupporterCount,
		&data.UnverifiedSupporterCount,
		&data.MinVerificationTier,
		&data.MaxVerificationTier,
		&data.MinBadgeLevel,
		&data.MaxBadgeLevel,
		&tierDistRaw,
		&badgeDistRaw,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("movement %s not found for indexing", movementID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying movement for indexing: %w", err)
	}

	// Parse JSONB distributions (PostgreSQL returns jsonb as []byte).
	data.TierDistribution, err = parseTierDistribution(tierDistRaw)
	if err != nil {
		return nil, fmt.Errorf("parsing tier_distribution for %s: %w", movementID, err)
	}
	data.BadgeDistribution, err = parseBadgeDistribution(badgeDistRaw)
	if err != nil {
		return nil, fmt.Errorf("parsing badge_distribution for %s: %w", movementID, err)
	}

	// ── Category query (separate to avoid row multiplication with aggregates)
	const categoryQuery = `
		SELECT
			c.id::TEXT,
			c.slug,
			c.name
		FROM movement_categories mc
		JOIN categories c ON c.id = mc.category_id
		WHERE mc.movement_id = $1
		  AND c.is_active = TRUE
		ORDER BY c.display_order ASC, c.name ASC
	`

	rows, err := r.db.QueryContext(ctx, categoryQuery, movementID)
	if err != nil {
		return nil, fmt.Errorf("querying categories for movement %s: %w", movementID, err)
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

// GetAllMovementIDsForReindex returns all active movement IDs for full reindexing.
func (r *MovementIndexingRepository) GetAllMovementIDsForReindex(ctx context.Context) ([]string, error) {
	const query = `
		SELECT id::TEXT FROM movements
		WHERE deleted_at IS NULL
			AND status != 'draft'
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying movement IDs for reindex: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning movement id: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating user id rows: %w", err)
	}

	return ids, nil
}

// GetAllMovementsForBulkIndex returns fully populated documents for all
// active movements. Used during initial setup or full re-indexing.
// Fetches in batches to avoid loading everything into memory at once.
func (r *MovementIndexingRepository) GetAllMovementsForBulkIndex(
	ctx context.Context,
	batchSize int,
	processBatch func(ids []string) error,
) error {
	ids, err := r.GetAllMovementIDsForReindex(ctx)
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
			return fmt.Errorf("processing batch starting at %d: %w", i, err)
		}
	}
	return nil
}

// parseTierDistribution converts raw JSONB bytes from PostgreSQL into a
// map[int]int (tier → count). The JSONB stores string keys like {"0":"5"},
// so we first unmarshal to map[string]int then convert.
func parseTierDistribution(raw []byte) (map[int]int, error) {
	result := make(map[int]int)
	if len(raw) == 0 || string(raw) == "null" {
		return result, nil
	}
	var stringMap map[string]int
	if err := json.Unmarshal(raw, &stringMap); err != nil {
		return nil, fmt.Errorf("unmarshaling tier_distribution: %w", err)
	}
	for k, v := range stringMap {
		tier := 0
		fmt.Sscanf(k, "%d", &tier)
		result[tier] = v
	}
	return result, nil
}

// parseBadgeDistribution converts raw JSONB bytes from PostgreSQL into a
// map[int]int (badge_level_numeric → count). The JSONB stores string keys
// like {"1":"10"} where 1=bronze, 2=silver, etc.
func parseBadgeDistribution(raw []byte) (map[int]int, error) {
	result := make(map[int]int)
	if len(raw) == 0 || string(raw) == "null" {
		return result, nil
	}
	var stringMap map[string]int
	if err := json.Unmarshal(raw, &stringMap); err != nil {
		return nil, fmt.Errorf("unmarshaling badge_distribution: %w", err)
	}
	for k, v := range stringMap {
		level, err := strconv.Atoi(k)
		if err != nil {
			// Non-numeric keys (e.g., "bronze") — skip
			continue
		}
		result[level] = v
	}
	return result, nil
}
