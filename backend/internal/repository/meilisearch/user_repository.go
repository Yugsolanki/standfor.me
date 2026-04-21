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

// UserRepository handles all Meilisearch operations for the users index.
type UserRepository struct {
	client *Client
}

// NewUserRepository constructs a UserRepository.
func NewUserRepository(client *Client) *UserRepository {
	return &UserRepository{client: client}
}

// Search executes a full-text search against the users index
func (r *UserRepository) Search(
	ctx context.Context,
	req *search.UserSearchRequest,
) (*search.SearchResult[search.UserDocument], error) {
	// validate & normalize pagination
	page, perPage := normalizePagination(req.Page, req.PerPage)
	offset := (page - 1) * perPage

	// build filter expression
	fb := newFilterBuilder()

	// Privacy: never return non-public profiles from the search index.
	// If the caller explicitly requests a visibility (for admin use), allow it.
	visibility := req.ProfileVisibility
	if visibility == "" {
		visibility = "public"
	}
	fb.addEquals("profile_visibility", visibility)

	// Always  exclude deleted users from search results
	fb.addEqualsBool("is_deleted", false)

	// Active users only
	fb.addEquals("status", "active")

	// Optional location filter
	fb.addEquals("location", req.Location)

	// * REVISIT - Are you sure? Do we need this?
	// Category filters
	// fb.addInStrings("category_ids", req.CategoryIDs)
	// fb.addInStrings("category_names", req.CategoryNames)
	// fb.addInStrings("category_slugs", req.CategorySlugs)

	// Depth of Commitment filters
	fb.addOptionalGTE("min_verification_tier", req.MinVerificationTier)
	fb.addOptionalGTE("max_badge_level_numeric", req.MinMaxBadgeLevelNumeric)
	fb.addOptionalGTE("verified_movement_count", req.MinVerifiedMovementCount)

	// Build sort expression
	sort := buildSortExpression(req.SortBy, req.SortOrder, r.client.cfg.Indexes["users"].SearchableAttributes)

	// Execute Search
	params := &meilisearch.SearchRequest{
		Query:  req.Query,
		Filter: fb.build(),
		Sort:   sort,
		Limit:  int64(perPage),
		Offset: int64(offset),
	}

	raw, err := r.client.UsersIndex.SearchWithContext(ctx, req.Query, params)
	if err != nil {
		return nil, fmt.Errorf("users search: %w", err)
	}

	// Decode hits
	hits, err := decodeHits[search.UserDocument](raw.Hits)
	if err != nil {
		return nil, fmt.Errorf("decoding user hits: %w", err)
	}

	totalPages := 1
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(raw.TotalHits) / float64(perPage)))
	}

	return &search.SearchResult[search.UserDocument]{
		Hits:             hits,
		Query:            req.Query,
		TotalHits:        raw.TotalHits,
		TotalPages:       totalPages,
		Page:             page,
		PerPage:          perPage,
		ProcessingTimeMs: raw.ProcessingTimeMs,
	}, nil
}

// UpsertDocument adds or replaces a single user document in the index.
func (r *UserRepository) UpsertDocument(ctx context.Context, doc *search.UserDocument) error {
	task, err := r.client.UsersIndex.AddDocumentsWithContext(ctx, []search.UserDocument{*doc}, &meilisearch.DocumentOptions{
		PrimaryKey: ptr.String("id"),
	})
	if err != nil {
		return fmt.Errorf("upserting user document %s: %w", doc.ID, err)
	}
	r.client.logger.InfoContext(ctx, "user document queued for indexing",
		"document_id", doc.ID,
		"task_uid", task.TaskUID,
	)
	return nil
}

// UpsertDocuments adds or replaces a batch of user documents.
func (r *UserRepository) UpsertDocuments(ctx context.Context, docs []search.UserDocument) error {
	if len(docs) == 0 {
		return nil
	}
	task, err := r.client.UsersIndex.AddDocumentsWithContext(ctx, docs, &meilisearch.DocumentOptions{
		PrimaryKey: ptr.String("id"),
	})
	if err != nil {
		return fmt.Errorf("bulk upserting %d user documents: %w", len(docs), err)
	}
	r.client.logger.InfoContext(ctx, "user documents queued for bulk indexing",
		"count", len(docs),
		"task_uid", task.TaskUID,
	)
	return nil
}

// DeleteDocument removes a user from the index by their ID.
func (r *UserRepository) DeleteDocument(ctx context.Context, id string) error {
	task, err := r.client.UsersIndex.DeleteDocumentWithContext(ctx, id, &meilisearch.DocumentOptions{
		PrimaryKey: ptr.String("id"),
	})
	if err != nil {
		return fmt.Errorf("deleting user document %s: %w", id, err)
	}
	r.client.logger.InfoContext(ctx, "user document queued for deletion",
		"document_id", id,
		"task_uid", task.TaskUID,
	)
	return nil
}

// WaitForTask waits for a specific Meilisearch task to complete.
// Useful in tests and admin scripts that need synchronous behavior.
func (r *UserRepository) WaitForTask(ctx context.Context, taskUID int64) error {
	result, err := r.client.ms.WaitForTaskWithContext(ctx, taskUID, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("waiting for task %d: %w", taskUID, err)
	}
	if result.Status == meilisearch.TaskStatusFailed {
		return fmt.Errorf("task %d failed: %s", taskUID, result.Error.Message)
	}
	return nil
}
