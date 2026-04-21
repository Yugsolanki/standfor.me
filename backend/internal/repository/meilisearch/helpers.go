package meilisearch

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/Yugsolanki/standfor-me/internal/domain/search"
	"github.com/meilisearch/meilisearch-go"
)

const (
	defaultPage    = 1
	defaultPerPage = 20
	maxPerPage     = 100
)

// normalizePagination clamps page and perPage to sane values.
func normalizePagination(page, perPage int) (int, int) {
	if page <= 0 {
		page = defaultPage
	}
	if perPage <= 0 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return page, perPage
}

// buildSortExpression converts a SortBy + SortOrder pair into the
// Meilisearch sort string format: ["field:direction"]
// It validates that the field is in the allowlist to prevent injection.
func buildSortExpression(sortBy string, order search.SortOrder, allowlist []string) []string {
	if sortBy == "" {
		return nil
	}

	// Validate against allowlist
	allowed := slices.Contains(allowlist, sortBy)
	if !allowed {
		return nil
	}

	direction := "desc"
	if order == search.SortAsc {
		direction = "asc"
	}

	return []string{fmt.Sprintf("%s:%s", sortBy, direction)}
}

// decodeHits converts the raw []interface{} hits from Meilisearch into
// typed structs using JSON round-trip marshaling.
func decodeHits[T any](rawHits meilisearch.Hits) ([]T, error) {
	if len(rawHits) == 0 {
		return []T{}, nil
	}

	// Marshal raw hits back to JSON, then unmarshal into our typed slice.
	data, err := json.Marshal(rawHits)
	if err != nil {
		return nil, fmt.Errorf("marshaling raw hits: %w", err)
	}

	var results []T
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("unmarshaling hits into %T: %w", results, err)
	}

	return results, nil
}
