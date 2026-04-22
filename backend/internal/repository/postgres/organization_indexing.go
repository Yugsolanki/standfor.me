package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Yugsolanki/standfor-me/internal/service/search"
	"github.com/jmoiron/sqlx"
)

// OrganizationIndexingRepository fetches the data needed to build an
// OrganizationDocument for the Meilisearch "organizations" index.
type OrganizationIndexingRepository struct {
	db *sqlx.DB
}

// NewOrganizationIndexingRepository constructs the repository.
func NewOrganizationIndexingRepository(db *sqlx.DB) *OrganizationIndexingRepository {
	return &OrganizationIndexingRepository{db: db}
}

// GetOrgForIndexing fetches all fields required to build an OrganizationDocument.
//
// Sensitive fields that must never appear in a search document:
//   - ein_tax_id_hash  (hashed tax ID — still sensitive)
//   - contact_email    (private org contact)
//   - social_links     (not indexed, not filterable — too noisy)
//
// movement_count is computed here via a subquery rather than being stored
// on the organizations table, because it needs to reflect only active,
// non-deleted movements that reference this org.
func (r *OrganizationIndexingRepository) GetOrgForIndexing(
	ctx context.Context,
	orgID string,
) (*search.OrgIndexData, error) {
	const query = `
		SELECT
			o.id::TEXT,
			o.slug,
			o.name,
			COALESCE(o.short_description, '') AS short_description,
			COALESCE(o.long_description, '') AS long_description,
			COALESCE(o.logo_url, '') AS logo_url,
			COALESCE(o.website_url, '') AS website_url,
			COALESCE(o.country_code, '') AS country_code,
			o.is_verified,
			o.status,
			o.created_at,
			o.updated_at,

			-- supporter_count: The total number of unique users who have
			-- publicly declared support for ANY movement claimed by this org.
			-- This is a denormalized metric maintained by a background job
			-- (see cmd/worker). We read it directly here for efficiency.
			--
			-- TODO: If you haven't implemented the background job yet, you can
			-- temporarily compute it inline:

			(
			  SELECT COUNT(DISTINCT um.user_id)
			  FROM   user_movements um
			  JOIN   movements m ON m.id = um.movement_id
			  WHERE  m.claimed_by_org_id = o.id
			    AND  um.is_public   = TRUE
			    AND  um.removed_at IS NULL
			    AND  m.deleted_at  IS NULL
			)
			-- For now we read the pre-computed column for performance.
			COALESCE(o.supporter_count, 0) AS supporter_count,

			-- movement_count: How many active movements is this org associated with?
			-- Computed inline to always be accurate at index time.
			(
				SELECT COUNT(*)::INT
				FROM   movements m
				WHERE  m.claimed_by_org_id = o.id
				  AND  m.deleted_at       IS NULL
				  AND  m.status            = 'active'
			) AS movement_count

		FROM  organizations o
		WHERE o.id         = $1
		  AND o.deleted_at IS NULL
	`

	data := &search.OrgIndexData{}

	err := r.db.QueryRowContext(ctx, query, orgID).Scan(
		&data.ID,
		&data.Slug,
		&data.Name,
		&data.ShortDescription,
		&data.LongDescription,
		&data.LogoURL,
		&data.WebsiteURL,
		&data.CountryCode,
		&data.IsVerified,
		&data.Status,
		&data.CreatedAt,
		&data.UpdatedAt,
		&data.SupporterCount,
		&data.MovementCount,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization %s not found for indexing", orgID)
	}
	if err != nil {
		return nil, fmt.Errorf("querying organization %s for indexing: %w", orgID, err)
	}

	return data, nil
}

// GetAllOrgIDsForReindex returns all non-deleted organization IDs in
// chronological order. Both verified and unverified orgs are included —
// the search layer's is_verified filter handles visibility at query time.
func (r *OrganizationIndexingRepository) GetAllOrgIDsForReindex(ctx context.Context) ([]string, error) {
	const query = `
		SELECT id::TEXT
		FROM organizations
		WHERE deleted_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying org IDs for reindex: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning org ID: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating org id rows: %w", err)
	}

	return ids, nil
}

// GetAllOrgsForBulkIndex pages through all org IDs and calls processBatch
// for each chunk. Mirrors the exact same pattern used by movements and users
// to keep the reindex command consistent across all three entity types.
func (r *OrganizationIndexingRepository) GetAllOrgsForBulkIndex(
	ctx context.Context,
	batchSize int,
	processBatch func(orgIDs []string) error,
) error {
	ids, err := r.GetAllOrgIDsForReindex(ctx)
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
			return fmt.Errorf("processing org batch starting at index %d: %w", i, err)
		}
	}

	return nil
}
