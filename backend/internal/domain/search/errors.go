package search

import "errors"

var (
	// ErrIndexNotFound is returned when an operation targets a non-existent index.
	ErrIndexNotFound = errors.New("search: index not found")

	// ErrInvalidQuery is returned when search parameters fail validation.
	ErrInvalidQuery = errors.New("search: invalid query parameters")

	// ErrDocumentNotFound is returned when a specific document cannot be located.
	ErrDocumentNotFound = errors.New("search: document not found")

	// ErrIndexingFailed is returned when a document cannot be written to the index.
	ErrIndexingFailed = errors.New("search: indexing operation failed")
)
