package meilisearch

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/domain/search"
	"github.com/aws/smithy-go/ptr"
	"github.com/meilisearch/meilisearch-go"
)

// MovementRepository handles all Meilisearch operations for the movements index.
type MovementRepository struct {
	client *Client
}

// NewMovementRepository constructs a MovementRepository.
func NewMovementRepository(client *Client) *MovementRepository {
	return &MovementRepository{
		client: client,
	}
}

// ----------------------------------
// Search
// ----------------------------------

// Search executes a full-text search against the movements index, applying
// depth-of-commitment filters and returning a paginated result set.
func (r *MovementRepository) Search(
	ctx context.Context,
	req *search.MovementSearchRequest,
) (*search.SearchResult[search.MovementDocument], error) {
	// validate & normalize pagination
	page, perPage := normalizePagination(req.Page, req.PerPage)
	offset := (page - 1) * perPage

	// build filter expression
	fb := newFilterBuilder()

	// Status filter default to 'active' to prevent draft/archived leaking
	status := req.Status
	if status == "" {
		status = "active"
	}
	fb.addEquals("status", status)

	// Optional boolean filters
	fb.addOptionalBool("has_verified_org", req.HasVerifiedOrg)

	// Organization filter
	fb.addEquals("claimed_by_org_id", req.OrgID)

	// Category filters
	fb.addInStrings("category_ids", req.CategoryIDs)
	fb.addInStrings("category_names", req.CategoryNames)
	fb.addInStrings("category_slugs", req.CategorySlugs)

	// Depth of Commitment filters
	fb.addOptionalGTEInt64("verified_supporter_count", req.VerifiedSupporterCount)
	fb.addOptionalGTEInt64("unverified_supporter_count", req.UnverifiedSupporterCount)
	fb.addOptionalGTEFloat("avg_verification_tier", req.MinAvgVerificationTier)
	fb.addOptionalLTEFloat("avg_verification_tier", req.MaxAvgVerificationTier)
	fb.addOptionalGTE("min_verification_tier", req.MinMinVerificationTier)
	fb.addOptionalGTE("max_verification_tier", req.MinMaxVerificationTier)
	fb.addOptionalGTE("max_badge_level_numeric", req.MinMaxBadgeLevelNumeric)

	// Popularity filters
	fb.addOptionalGTEInt64("supporter_count", req.MinSupporterCount)
	fb.addOptionalGTEFloat64("trending_score", req.MinTrendingScore)

	// Build sort expression
	sort := buildSortExpression(req.SortBy, req.SortOrder, r.client.cfg.Indexes["movements"].SearchableAttributes)

	// Execute Search
	params := &meilisearch.SearchRequest{
		Query:  req.Query,
		Filter: fb.build(),
		Sort:   sort,
		Limit:  int64(perPage),
		Offset: int64(offset),
	}

	raw, err := r.client.MovementsIndex.SearchWithContext(ctx, req.Query, params)
	if err != nil {
		return nil, fmt.Errorf("movements search: %w", err)
	}

	// Decode hits
	hits, err := decodeHits[search.MovementDocument](raw.Hits)
	if err != nil {
		return nil, fmt.Errorf("decoding movement hits: %w", err)
	}

	totalPages := 1
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(raw.TotalHits) / float64(perPage)))
	}

	return &search.SearchResult[search.MovementDocument]{
		Hits:             hits,
		Query:            req.Query,
		TotalHits:        raw.TotalHits,
		TotalPages:       totalPages,
		Page:             page,
		PerPage:          perPage,
		ProcessingTimeMs: raw.ProcessingTimeMs,
	}, nil
}

// ----------------------------------
// Indexing
// ----------------------------------

// UpsertDocument adds or replaces a single movement document in the index.
// It is safe to call for both new movements and updates.
func (r *MovementRepository) UpsertDocument(
	ctx context.Context,
	doc *search.MovementDocument,
) error {
	task, err := r.client.MovementsIndex.AddDocumentsWithContext(ctx, []search.MovementDocument{*doc}, &meilisearch.DocumentOptions{
		PrimaryKey: ptr.String("id"),
	})
	if err != nil {
		return fmt.Errorf("upserting movement document %s: %w", doc.ID, err)
	}

	// We do NOT wait for the task here — indexing is asynchronous by design.
	r.client.logger.InfoContext(ctx, "movement document queued for indexing",
		"document_id", doc.ID,
		"task_uid", task.TaskUID,
	)

	return nil
}

// UpsertDocuments adds or replaces a batch of movement documents.
// Use this for bulk re-indexing operations.
func (r *MovementRepository) UpsertDocuments(
	ctx context.Context,
	docs []search.MovementDocument,
) error {
	if len(docs) == 0 {
		return nil
	}
	task, err := r.client.MovementsIndex.AddDocumentsWithContext(ctx, docs, &meilisearch.DocumentOptions{
		PrimaryKey: ptr.String("id"),
	})
	if err != nil {
		return fmt.Errorf("bulk upserting %d movement documents: %w", len(docs), err)
	}
	r.client.logger.InfoContext(ctx, "movement documents queued for bulk indexing",
		"count", len(docs),
		"task_uid", task.TaskUID,
	)
	return nil
}

// DeleteDocument removes a movement from the index by its ID.
// Call this when a movement is deleted or set to a non-searchable status.
func (r *MovementRepository) DeleteDocument(ctx context.Context, id string) error {
	task, err := r.client.MovementsIndex.DeleteDocumentWithContext(ctx, id, &meilisearch.DocumentOptions{
		PrimaryKey: ptr.String("id"),
	})
	if err != nil {
		return fmt.Errorf("deleting movement document %s: %w", id, err)
	}
	r.client.logger.InfoContext(ctx, "movement document queued for deletion",
		"document_id", id,
		"task_uid", task.TaskUID,
	)
	return nil
}

// WaitForTask waits for a specific Meilisearch task to complete.
// Useful in tests and admin scripts that need synchronous behavior.
func (r *MovementRepository) WaitForTask(ctx context.Context, taskUID int64) error {
	result, err := r.client.ms.WaitForTaskWithContext(ctx, taskUID, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("waiting for task %d: %w", taskUID, err)
	}
	if result.Status == meilisearch.TaskStatusFailed {
		return fmt.Errorf("task %d failed: %s", taskUID, result.Error.Message)
	}
	return nil
}
