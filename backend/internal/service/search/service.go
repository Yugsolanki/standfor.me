package search

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Yugsolanki/standfor-me/internal/domain/search"
	"github.com/Yugsolanki/standfor-me/internal/pkg/validator"
)

// Service orchestrates search operations and document indexing.
// It sits between HTTP handlers and the Meilisearch repository layer,
// handling business logic like document assembly and validation.
type Service struct {
	movements     MovementSearchRepo
	users         UserSearchRepo
	organizations OrganizationSearchRepo

	// Data repo - used only during indexing to fetch DB data.
	movementData MovementDataRepo
	userData     UserDataRepo
	orgData      OrgDataRepo

	logger *slog.Logger
}

// NewService constructs a Service with all required dependencies.
func NewService(
	movements MovementSearchRepo,
	users UserSearchRepo,
	organizations OrganizationSearchRepo,
	movementData MovementDataRepo,
	userData UserDataRepo,
	orgData OrgDataRepo,
	logger *slog.Logger,
) *Service {
	return &Service{
		movements:     movements,
		users:         users,
		organizations: organizations,
		movementData:  movementData,
		userData:      userData,
		orgData:       orgData,
		logger:        logger,
	}
}

// -----------------------------------
// Search Operations
// -----------------------------------

// SearchMovements is the primary entry point for movement search.
func (s *Service) SearchMovements(
	ctx context.Context,
	req *search.MovementSearchRequest,
) (*search.SearchResult[search.MovementDocument], error) {
	v := validator.New()
	if err := v.Validate(req); err != nil {
		return nil, fmt.Errorf("validation error: %v", err)
	}

	result, err := s.movements.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("searching movements: %w", err)
	}

	return result, nil
}

// SearchUsers is the primary entry point for user/advocate search.
func (s *Service) SearchUsers(
	ctx context.Context,
	req *search.UserSearchRequest,
) (*search.SearchResult[search.UserDocument], error) {
	result, err := s.users.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("searching users: %w", err)
	}
	return result, nil
}

// SearchOrganizations is the primary entry point for organization search.
func (s *Service) SearchOrganizations(
	ctx context.Context,
	req *search.OrganizationSearchRequest,
) (*search.SearchResult[search.OrganizationDocument], error) {
	result, err := s.organizations.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("searching organizations: %w", err)
	}
	return result, nil
}

// -----------------------------------
// Indexing Operations
// -----------------------------------

// IndexMovement fetches the latest movement data from Postgres and upserts
// the document into Meilisearch. Call this after any movement-related write.
func (s *Service) IndexMovement(ctx context.Context, movementID string) error {
	data, err := s.movementData.GetMovementForIndexing(ctx, movementID)
	if err != nil {
		return fmt.Errorf("fetching movement data for indexing %s: %w", movementID, err)
	}

	doc := buildMovementDocument(data)

	if err := s.movements.UpsertDocument(ctx, doc); err != nil {
		return fmt.Errorf("%w: movement %s", search.ErrIndexingFailed, movementID)
	}

	s.logger.InfoContext(ctx, "movement indexed successfully", "movement_id", movementID)
	return nil
}

// RemoveMovement removes a movement document from the search index.
// Call this when a movement is deleted or permanently deactivated.
func (s *Service) RemoveMovement(ctx context.Context, movementID string) error {
	if err := s.movements.DeleteDocument(ctx, movementID); err != nil {
		return fmt.Errorf("removing movement from index: %w", err)
	}
	return nil
}

// IndexUser fetches the latest user data and upserts the search document.
func (s *Service) IndexUser(ctx context.Context, userID string) error {
	data, err := s.userData.GetUserForIndexing(ctx, userID)
	if err != nil {
		return fmt.Errorf("fetching user data for indexing %s: %w", userID, err)
	}

	doc := buildUserDocument(data)

	if err := s.users.UpsertDocument(ctx, doc); err != nil {
		return fmt.Errorf("%w: user %s", search.ErrIndexingFailed, userID)
	}

	s.logger.InfoContext(ctx, "user indexed successfully", "user_id", userID)
	return nil
}

// RemoveUser removes a user document from the search index.
func (s *Service) RemoveUser(ctx context.Context, userID string) error {
	if err := s.users.DeleteDocument(ctx, userID); err != nil {
		return fmt.Errorf("removing user from index: %w", err)
	}
	return nil
}

// IndexOrganization fetches the latest org data and upserts the search document.
func (s *Service) IndexOrganization(ctx context.Context, orgID string) error {
	data, err := s.orgData.GetOrgForIndexing(ctx, orgID)
	if err != nil {
		return fmt.Errorf("fetching org data for indexing %s: %w", orgID, err)
	}

	doc := buildOrgDocument(data)

	if err := s.organizations.UpsertDocument(ctx, doc); err != nil {
		return fmt.Errorf("%w: org %s", search.ErrIndexingFailed, orgID)
	}

	s.logger.InfoContext(ctx, "organization indexed successfully", "org_id", orgID)
	return nil
}

// RemoveOrganization removes an org document from the search index.
func (s *Service) RemoveOrganization(ctx context.Context, orgID string) error {
	if err := s.organizations.DeleteDocument(ctx, orgID); err != nil {
		return fmt.Errorf("removing org from index: %w", err)
	}
	return nil
}

// -----------------------------------
// Document Builders
// -----------------------------------

// buildMovementDocument transforms raw DB data into a flat search document.
// This is where the depth-of-commitment metrics are computed and normalized.
func buildMovementDocument(data *MovementIndexData) *search.MovementDocument {
	return &search.MovementDocument{
		ID:               data.ID,
		Slug:             data.Slug,
		Name:             data.Name,
		ShortDescription: data.ShortDescription,
		LongDescription:  data.LongDescription,
		ImageURL:         data.ImageURL,
		IconURL:          data.IconURL,
		WebsiteURL:       data.WebsiteURL,
		Status:           data.Status,
		SupporterCount:   int(data.SupporterCount),
		TrendingScore:    data.TrendingScore,
		CreatedByUserID:  data.CreatedByUserID,

		// Organization
		ClaimedByOrgID:   data.ClaimedByOrgID,
		OrganizationName: data.OrganizationName,
		HasVerifiedOrg:   data.HasVerifiedOrg,

		// Categories
		CategoryIDs:   data.CategoryIDs,
		CategorySlugs: data.CategorySlugs,
		CategoryNames: data.CategoryNames,

		// --- Depth of Commitment Metrics ---
		// Verification Tier metrics
		AvgVerificationTier: data.AvgVerificationTier,
		MinVerificationTier: data.MinVerificationTier,
		MaxVerificationTier: data.MaxVerificationTier,

		// Supporter counts
		VerifiedSupporterCount:   data.VerifiedSupporterCount,
		UnverifiedSupporterCount: data.UnverifiedSupporterCount,

		// Min/Max badge level
		MinBadgeLevelNumeric: search.BadgeLevelToNumeric(data.MinBadgeLevel),
		MaxBadgeLevelNumeric: search.BadgeLevelToNumeric(data.MaxBadgeLevel),

		// Supporter distributions
		TierDistribution:  data.TierDistribution,
		BadgeDistribution: data.BadgeDistribution,

		// Unix timestamps
		CreatedAtUnix: data.CreatedAt.Unix(),
		UpdatedAtUnix: data.UpdatedAt.Unix(),
	}
}

// buildUserDocument transforms raw DB data into a flat user search document.
func buildUserDocument(data *UserIndexData) *search.UserDocument {
	return &search.UserDocument{
		ID:                data.ID,
		Username:          data.Username,
		DisplayName:       data.DisplayName,
		Bio:               data.Bio,
		Location:          data.Location,
		ProfileVisibility: data.ProfileVisibility,
		Status:            data.Status,
		IsDeleted:         data.IsDeleted,

		// Movements
		MovementIDs:   data.MovementIDs,
		MovementNames: data.MovementNames,
		MovementSlugs: data.MovementSlugs,

		// Depth of Commitment Metrics
		AvgVerificationTier:     data.AvgVerificationTier,
		MinVerificationTier:     data.MinVerificationTier,
		MaxBadgeLevelNumeric:    search.BadgeLevelToNumeric(data.MaxBadgeLevel),
		VerifiedMovementCount:   data.VerifiedMovementCount,
		TotalMovementsSupported: data.TotalMovementsSupported,

		// Categories
		// * REVISIT : implement later with consideration
		CategoryIDs:   data.CategoryIDs,
		CategorySlugs: data.CategorySlugs,
		CategoryNames: data.CategoryNames,

		// Timestamps
		CreatedAtUnix: data.CreatedAt.Unix(),
		UpdatedAtUnix: data.UpdatedAt.Unix(),
	}
}

// buildOrgDocument transforms raw DB data into a flat org search document.
func buildOrgDocument(data *OrgIndexData) *search.OrganizationDocument {
	return &search.OrganizationDocument{
		ID:               data.ID,
		Slug:             data.Slug,
		Name:             data.Name,
		ShortDescription: data.ShortDescription,
		LongDescription:  data.LongDescription,
		LogoURL:          data.LogoURL,
		WebsiteURL:       data.WebsiteURL,
		CountryCode:      data.CountryCode,

		// Verification Status
		IsVerified: data.IsVerified,
		Status:     data.Status,

		// Engagement Metrics
		SupporterCount: int(data.SupporterCount),
		MovementCount:  data.MovementCount,

		// Timestamps
		CreatedAtUnix: data.CreatedAt.Unix(),
		UpdatedAtUnix: data.UpdatedAt.Unix(),
	}
}
