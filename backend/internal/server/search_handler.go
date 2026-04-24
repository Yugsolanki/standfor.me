package server

import (
	"net/http"
	"strconv"

	searchdomain "github.com/Yugsolanki/standfor-me/internal/domain/search"
	"github.com/Yugsolanki/standfor-me/internal/pkg/response"
)

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

// SearchUsers handles advocate discovery with commitment-level filtering.
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

// SearchOrganizations handles organization discovery.
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
