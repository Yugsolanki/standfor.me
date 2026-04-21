package search

import (
	"context"

	"github.com/Yugsolanki/standfor-me/internal/domain/search"
)

// MovementSearchRepo defines the search-related operations on movements.
type MovementSearchRepo interface {
	Search(ctx context.Context, req *search.MovementSearchRequest) (*search.SearchResult[search.MovementDocument], error)
	UpsertDocument(ctx context.Context, doc *search.MovementDocument) error
	UpsertDocuments(ctx context.Context, docs []search.MovementDocument) error
	DeleteDocument(ctx context.Context, id string) error
}

// UserSearchRepo defines the search-related operations on users.
type UserSearchRepo interface {
	Search(ctx context.Context, req *search.UserSearchRequest) (*search.SearchResult[search.UserDocument], error)
	UpsertDocument(ctx context.Context, doc *search.UserDocument) error
	UpsertDocuments(ctx context.Context, docs []search.UserDocument) error
	DeleteDocument(ctx context.Context, id string) error
}

// OrganizationSearchRepo defines the search-related operations on organizations.
type OrganizationSearchRepo interface {
	Search(ctx context.Context, req *search.OrganizationSearchRequest) (*search.SearchResult[search.OrganizationDocument], error)
	UpsertDocument(ctx context.Context, doc *search.OrganizationDocument) error
	UpsertDocuments(ctx context.Context, docs []search.OrganizationDocument) error
	DeleteDocument(ctx context.Context, id string) error
}

// MovementDataRepo defines the database queries needed to build a
// MovementDocument for indexing. This is implemented by the Postgres layer.
type MovementDataRepo interface {
	// GetMovementForIndexing returns all data needed to build a complete
	// MovementDocument, including aggregated verification metrics.
	GetMovementForIndexing(ctx context.Context, movementID string) (*MovementIndexData, error)
}

// UserDataRepo defines the database queries needed to build a UserDocument.
type UserDataRepo interface {
	GetUserForIndexing(ctx context.Context, userID string) (*UserIndexData, error)
}

// OrgDataRepo defines the database queries needed to build an OrganizationDocument.
type OrgDataRepo interface {
	GetOrgForIndexing(ctx context.Context, orgID string) (*OrgIndexData, error)
}
