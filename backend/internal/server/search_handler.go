package server

import (
	"net/http"
	"strconv"

	searchdomain "github.com/Yugsolanki/standfor-me/internal/domain/search"
	"github.com/Yugsolanki/standfor-me/internal/pkg/response"
)

// SearchMovements searches for movements with advanced filtering and sorting.
//
//	@Summary		Search movements
//	@Description	Search and filter movements by query, categories, verification tiers, organization status, and popularity metrics. Returns paginated results with engagement and depth-of-commitment metrics.
//	@Tags			Search
//	@Accept			json
//	@Produce		json
//	@Param			q							query		string						false	"Full-text search query (matches name, description)"
//	@Param			status						query		string						false	"Filter by movement status"	Enums(draft, active, archived, rejected, pending_review)
//	@Param			org_id						query		string						false	"Filter by organization ID"
//	@Param			sort_by						query		string						false	"Sort field"													Enums(name, supporter_count, trending_score, avg_verification_tier, created_at)
//	@Param			sort_order					query		string						false	"Sort order"													Enums(asc, desc)
//	@Param			page						query		int							false	"Page number"													default(1)
//	@Param			per_page					query		int							false	"Results per page"												default(20)
//	@Param			category_ids				query		[]string					false	"Filter by category IDs (comma-separated or multiple params)"	collectionFormat(multi)
//	@Param			category_slugs				query		[]string					false	"Filter by category slugs (comma-separated or multiple params)"	collectionFormat(multi)
//	@Param			category_names				query		[]string					false	"Filter by category names (comma-separated or multiple params)"	collectionFormat(multi)
//	@Param			has_verified_org			query		bool						false	"Filter movements claimed by verified organizations"			true,false
//	@Param			verified_supporter_count	query		int							false	"Filter by exact verified supporter count"
//	@Param			unverified_supporters		query		int							false	"Filter by exact unverified supporter count"
//	@Param			min_avg_vt					query		number						false	"Minimum average verification tier (0-5)"
//	@Param			max_avg_vt					query		number						false	"Maximum average verification tier (0-5)"
//	@Param			min_min_vt					query		int							false	"Minimum minimum verification tier (0-5)"
//	@Param			min_max_vt					query		int							false	"Minimum maximum verification tier (0-5)"
//	@Param			min_max_badge				query		int							false	"Minimum maximum badge level (0-5, 1=bronze,5=diamond)"
//	@Param			min_supporters				query		int							false	"Minimum total supporter count"
//	@Param			min_trending				query		number						false	"Minimum trending score"
//	@Success		200							{object}	response.SuccessResponse	"Successfully retrieved search results"
//	@Failure		400							{object}	response.ErrorResponse		"Invalid query parameters"
//	@Failure		500							{object}	response.ErrorResponse		"Internal server error"
//	@Router			/api/v1/search/movements [get]
func (s *Server) SearchMovements(w http.ResponseWriter, r *http.Request) error {
	req := &searchdomain.MovementSearchRequest{
		Query:   r.URL.Query().Get("q"),
		Status:  r.URL.Query().Get("status"),
		OrgID:   r.URL.Query().Get("org_id"),
		SortBy:  r.URL.Query().Get("sort_by"),
		Page:    parseIntParam(r, "page", 1),
		PerPage: parseIntParam(r, "per_page", 20),
	}

	req.SortOrder = parseSortOrder(r.URL.Query().Get("sort_order"))

	req.CategoryIDs = r.URL.Query()["category_ids"]
	req.CategorySlugs = r.URL.Query()["category_slugs"]
	req.CategoryNames = r.URL.Query()["category_names"]

	req.HasVerifiedOrg = parseBoolParam(r, "has_verified_org")

	req.VerifiedSupporterCount = parseInt64PtrParam(r, "verified_supporter_count")
	req.UnverifiedSupporterCount = parseInt64PtrParam(r, "unverified_supporters")
	req.MinAvgVerificationTier = parseFloat64Param(r, "min_avg_vt")
	req.MaxAvgVerificationTier = parseFloat64Param(r, "max_avg_vt")
	req.MinMinVerificationTier = parseIntPtrParam(r, "min_min_vt")
	req.MinMaxVerificationTier = parseIntPtrParam(r, "min_max_vt")
	req.MinMaxBadgeLevelNumeric = parseIntPtrParam(r, "min_max_badge")

	req.MinSupporterCount = parseInt64PtrParam(r, "min_supporters")
	req.MinTrendingScore = parseFloat64Param(r, "min_trending")

	result, err := s.services.Search.SearchMovements(r.Context(), req)
	if err != nil {
		response.JSONError(w, r, err)
		return nil
	}

	response.JSON(w, r, http.StatusOK, result)
	return nil
}

// SearchUsers searches for users/advocates with advanced filtering.
//
//	@Summary		Search users (advocates)
//	@Description	Search and filter user profiles by query, location, visibility, categories, and depth-of-commitment metrics. Returns paginated results with verification tier and movement support data. Only returns users with profile_visibility matching the filter (public by default).
//	@Tags			Search
//	@Accept			json
//	@Produce		json
//	@Param			q					query		string						false	"Full-text search query (matches display_name, username, bio)"
//	@Param			location			query		string						false	"Filter by location"
//	@Param			visibility			query		string						false	"Filter by profile visibility"															Enums(public, private, unlisted)
//	@Param			sort_by				query		string						false	"Sort field"																			Enums(display_name, username, avg_verification_tier, verified_movement_count, created_at)
//	@Param			sort_order			query		string						false	"Sort order"																			Enums(asc, desc)
//	@Param			page				query		int							false	"Page number"																			default(1)
//	@Param			per_page			query		int							false	"Results per page"																		default(20)
//	@Param			category_ids		query		[]string					false	"Filter by category IDs of supported movements (comma-separated or multiple params)"	collectionFormat(multi)
//	@Param			category_slugs		query		[]string					false	"Filter by category slugs of supported movements (comma-separated or multiple params)"	collectionFormat(multi)
//	@Param			category_names		query		[]string					false	"Filter by category names of supported movements (comma-separated or multiple params)"	collectionFormat(multi)
//	@Param			min_vt				query		int							false	"Minimum verification tier (0-5)"
//	@Param			min_max_badge		query		int							false	"Minimum maximum badge level achieved (0-5, 1=bronze,5=diamond)"
//	@Param			min_verified_count	query		int							false	"Minimum number of verified movements supported"
//	@Success		200					{object}	response.SuccessResponse	"Successfully retrieved search results"
//	@Failure		400					{object}	response.ErrorResponse		"Invalid query parameters"
//	@Failure		500					{object}	response.ErrorResponse		"Internal server error"
//	@Router			/api/v1/search/users [get]
func (s *Server) SearchUsers(w http.ResponseWriter, r *http.Request) error {
	req := &searchdomain.UserSearchRequest{
		Query:             r.URL.Query().Get("q"),
		Location:          r.URL.Query().Get("location"),
		ProfileVisibility: r.URL.Query().Get("visibility"),
		SortBy:            r.URL.Query().Get("sort_by"),
		Page:              parseIntParam(r, "page", 1),
		PerPage:           parseIntParam(r, "per_page", 20),
	}

	req.SortOrder = parseSortOrder(r.URL.Query().Get("sort_order"))

	req.CategoryIDs = r.URL.Query()["category_ids"]
	req.CategorySlugs = r.URL.Query()["category_slugs"]
	req.CategoryNames = r.URL.Query()["category_names"]

	req.MinVerificationTier = parseIntPtrParam(r, "min_vt")
	req.MinMaxBadgeLevelNumeric = parseIntPtrParam(r, "min_max_badge")
	req.MinVerifiedMovementCount = parseIntPtrParam(r, "min_verified_count")

	result, err := s.services.Search.SearchUsers(r.Context(), req)
	if err != nil {
		response.JSONError(w, r, err)
		return nil
	}

	response.JSON(w, r, http.StatusOK, result)
	return nil
}

// SearchOrganizations searches for organizations with filtering.
//
//	@Summary		Search organizations
//	@Description	Search and filter organizations (NGOs, non-profits, advocacy groups) by query, country, and verification status. Returns paginated results with supporter counts and movement associations.
//	@Tags			Search
//	@Accept			json
//	@Produce		json
//	@Param			q			query		string						false	"Full-text search query (matches name, short_description, long_description)"
//	@Param			country		query		string						false	"Filter by country code (ISO 3166-1 alpha-2, e.g., 'US', 'GB')"
//	@Param			sort_by		query		string						false	"Sort field"					Enums(name, supporter_count, movement_count, created_at)
//	@Param			sort_order	query		string						false	"Sort order"					Enums(asc, desc)
//	@Param			page		query		int							false	"Page number"					default(1)
//	@Param			per_page	query		int							false	"Results per page"				default(20)
//	@Param			verified	query		bool						false	"Filter by verification status"	true,false
//	@Success		200			{object}	response.SuccessResponse	"Successfully retrieved search results"
//	@Failure		400			{object}	response.ErrorResponse		"Invalid query parameters"
//	@Failure		500			{object}	response.ErrorResponse		"Internal server error"
//	@Router			/api/v1/search/organizations [get]
func (s *Server) SearchOrganizations(w http.ResponseWriter, r *http.Request) error {
	req := &searchdomain.OrganizationSearchRequest{
		Query:       r.URL.Query().Get("q"),
		CountryCode: r.URL.Query().Get("country"),
		SortBy:      r.URL.Query().Get("sort_by"),
		Page:        parseIntParam(r, "page", 1),
		PerPage:     parseIntParam(r, "per_page", 20),
	}

	req.SortOrder = parseSortOrder(r.URL.Query().Get("sort_order"))
	req.IsVerified = parseBoolParam(r, "verified")

	result, err := s.services.Search.SearchOrganizations(r.Context(), req)
	if err != nil {
		response.JSONError(w, r, err)
		return nil
	}

	response.JSON(w, r, http.StatusOK, result)
	return nil
}

// -----------------------------------------
// Parameter Parsing Helpers
// -----------------------------------------

func parseIntParam(r *http.Request, key string, defaultVal int) int {
	if val := r.URL.Query().Get(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func parseIntPtrParam(r *http.Request, key string) *int {
	if val := r.URL.Query().Get(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return &i
		}
	}
	return nil
}

func parseInt64PtrParam(r *http.Request, key string) *int64 {
	if val := r.URL.Query().Get(key); val != "" {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return &i
		}
	}
	return nil
}

func parseFloat64Param(r *http.Request, key string) *float64 {
	if val := r.URL.Query().Get(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return &f
		}
	}
	return nil
}

func parseBoolParam(r *http.Request, key string) *bool {
	if val := r.URL.Query().Get(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return &b
		}
	}
	return nil
}

func parseSortOrder(val string) searchdomain.SortOrder {
	if val == "asc" {
		return searchdomain.SortAsc
	}
	return searchdomain.SortDesc
}
