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

// OrganizationRepository handles all Meilisearch operations for the organizations index.
type OrganizationRepository struct {
	client *Client
}

// NewOrganizationRepository constructs an OrganizationRepository.
func NewOrganizationRepository(client *Client) *OrganizationRepository {
	return &OrganizationRepository{client: client}
}

// Search executes a full-text search against the organizations index.
func (r *OrganizationRepository) Search(
	ctx context.Context,
	req *search.OrganizationSearchRequest,
) (*search.SearchResult[search.OrganizationDocument], error) {
	page, perPage := normalizePagination(req.Page, req.PerPage)
	offset := (page - 1) * perPage

	fb := newFilterBuilder()

	// Optional filters
	fb.addOptionalBool("is_verfied", req.IsVerified)
	fb.addEquals("country_code", req.CountryCode)

	sort := buildSortExpression(req.SortBy, req.SortOrder, r.client.cfg.Indexes["organizations"].SearchableAttributes)

	params := &meilisearch.SearchRequest{
		Query:  req.Query,
		Filter: fb.build(),
		Sort:   sort,
		Limit:  int64(perPage),
		Offset: int64(offset),
	}

	raw, err := r.client.OrganizationsIndex.SearchWithContext(ctx, req.Query, params)
	if err != nil {
		return nil, fmt.Errorf("organizations search: %w", err)
	}

	// Decode hits
	hits, err := decodeHits[search.OrganizationDocument](raw.Hits)
	if err != nil {
		return nil, fmt.Errorf("decoding organization hits: %w", err)
	}

	totalPages := 1
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(raw.TotalHits) / float64(perPage)))
	}

	return &search.SearchResult[search.OrganizationDocument]{
		Hits:             hits,
		Query:            req.Query,
		TotalHits:        raw.TotalHits,
		TotalPages:       totalPages,
		Page:             page,
		PerPage:          perPage,
		ProcessingTimeMs: raw.ProcessingTimeMs,
	}, nil
}

// UpsertDocument adds or replaces a single organization document in the index.
func (r *OrganizationRepository) UpsertDocument(ctx context.Context, doc *search.OrganizationDocument) error {
	task, err := r.client.OrganizationsIndex.AddDocumentsWithContext(ctx, []search.OrganizationDocument{*doc}, &meilisearch.DocumentOptions{
		PrimaryKey: ptr.String("id"),
	})
	if err != nil {
		return fmt.Errorf("upserting organization document %s: %w", doc.ID, err)
	}
	r.client.logger.InfoContext(ctx, "organization document queued for indexing",
		"document_id", doc.ID,
		"task_uid", task.TaskUID,
	)
	return nil
}

// UpsertDocuments adds or replaces a batch of organization documents.
func (r *OrganizationRepository) UpsertDocuments(ctx context.Context, docs []search.OrganizationDocument) error {
	if len(docs) == 0 {
		return nil
	}
	task, err := r.client.OrganizationsIndex.AddDocumentsWithContext(ctx, docs, &meilisearch.DocumentOptions{
		PrimaryKey: ptr.String("id"),
	})
	if err != nil {
		return fmt.Errorf("bulk upserting %d organization documents: %w", len(docs), err)
	}
	r.client.logger.InfoContext(ctx, "organization documents queued for bulk indexing",
		"count", len(docs),
		"task_uid", task.TaskUID,
	)
	return nil
}

// DeleteDocument removes an organization from the index by its ID.
func (r *OrganizationRepository) DeleteDocument(ctx context.Context, id string) error {
	task, err := r.client.OrganizationsIndex.DeleteDocumentWithContext(ctx, id, &meilisearch.DocumentOptions{
		PrimaryKey: ptr.String("id"),
	})
	if err != nil {
		return fmt.Errorf("deleting organization document %s: %w", id, err)
	}
	r.client.logger.InfoContext(ctx, "organization document queued for deletion",
		"document_id", id,
		"task_uid", task.TaskUID,
	)
	return nil
}

// WaitForTask waits for a specific Meilisearch task to complete.
// Useful in tests and admin scripts that need synchronous behavior.
func (r *OrganizationRepository) WaitForTask(ctx context.Context, taskUID int64) error {
	result, err := r.client.ms.WaitForTaskWithContext(ctx, taskUID, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("waiting for task %d: %w", taskUID, err)
	}
	if result.Status == meilisearch.TaskStatusFailed {
		return fmt.Errorf("task %d failed: %s", taskUID, result.Error.Message)
	}
	return nil
}
