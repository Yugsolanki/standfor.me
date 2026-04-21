package search

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

type SortField struct {
	Field string    `json:"field"`
	Order SortOrder `json:"order"`
}

// ----------------------------------------------
// Movement Search
// ----------------------------------------------

type MovementSearchRequest struct {
	Query string `json:"query" form:"q"`

	// Pagination
	Page    int `json:"page" form:"page"`
	PerPage int `json:"per_page" form:"per_page"`

	// Category Filters
	CategoryIDs   []string `json:"category_ids" form:"category_ids"`
	CategoryNames []string `json:"category_names" form:"category_names"`
	CategorySlugs []string `json:"category_slugs" form:"category_slugs"`

	// Status
	Status string `json:"status" form:"status"`

	// Organization Filters
	HasVerifiedOrg *bool  `json:"has_verified_org" form:"has_verified_org"`
	OrgID          string `json:"org_id" form:"org_id"`

	// Depth of Commitment Verification Filters
	VerifiedSupporterCount   *int64   `json:"verified_supporter_count" form:"verified_supporters"`
	UnverifiedSupporterCount *int64   `json:"unverified_supporter_count" form:"unverified_supporters"`
	MinAvgVerificationTier   *float64 `json:"min_avg_verification_tier" form:"min_avg_vt"`
	MaxAvgVerificationTier   *float64 `json:"max_avg_verification_tier" form:"max_avg_vt"`
	MinMinVerificationTier   *int     `json:"min_min_verification_tier" form:"min_min_vt"`
	MinMaxVerificationTier   *int     `json:"min_max_verification_tier" form:"min_max_vt"`
	MinMaxBadgeLevelNumeric  *int     `json:"min_max_badge_level" form:"min_max_badge"`

	// Popularity Filters
	MinSupporterCount *int64   `json:"min_supporter_count" form:"min_supporters"`
	MinTrendingScore  *float64 `json:"min_trending_score" form:"min_trending"`

	// Sorting
	SortBy    string    `json:"sort_by" form:"sort_by"`
	SortOrder SortOrder `json:"sort_order" form:"sort_order"`
}

// ----------------------------------------------
// User Search
// ----------------------------------------------

type UserSearchRequest struct {
	Query   string `json:"query" form:"q"`
	Page    int    `json:"page" form:"page"`
	PerPage int    `json:"per_page" form:"per_page"`

	// Profile Filters
	ProfileVisibility string `json:"profile_visibility" form:"visibility"`
	Location          string `json:"location" form:"location"`

	// Category Filters
	CategoryIDs   []string `json:"category_ids" form:"category_ids"`
	CategoryNames []string `json:"category_names" form:"category_names"`
	CategorySlugs []string `json:"category_slugs" form:"category_slugs"`

	// Depth of Commitment Filters
	MinVerificationTier      *int `json:"min_verification_tier" form:"min_vt"`
	MinMaxBadgeLevelNumeric  *int `json:"min_max_badge_level" form:"min_max_badge"`
	MinVerifiedMovementCount *int `json:"min_verified_movement_count" form:"min_verified_count"`

	// Sorting
	SortBy    string    `json:"sort_by" form:"sort_by"`
	SortOrder SortOrder `json:"sort_order" form:"sort_order"`
}

// ----------------------------------------------
// Organization Search
// ----------------------------------------------

type OrganizationSearchRequest struct {
	Query       string `json:"query" form:"q"`
	Page        int    `json:"page" form:"page"`
	PerPage     int    `json:"per_page" form:"per_page"`
	IsVerified  *bool  `json:"is_verified" form:"verified"`
	CountryCode string `json:"country_code" form:"country"`

	// Sorting
	SortBy    string    `json:"sort_by" form:"sort_by"`
	SortOrder SortOrder `json:"sort_order" form:"sort_order"`
}

// ----------------------------------------------
// Generic Search Response
// ----------------------------------------------

// SearchResult is a generic wrapper returned by all search operations.
type SearchResult[T any] struct {
	Hits             []T    `json:"hits"`
	Query            string `json:"query"`
	TotalHits        int64  `json:"total_hits"`
	TotalPages       int    `json:"total_pages"`
	Page             int    `json:"page"`
	PerPage          int    `json:"per_page"`
	ProcessingTimeMs int64  `json:"processing_time_ms"`
}
