package search

// Document represents the data structure for a document in the search index.
type MovementDocument struct {
	// --- Core Identity ---
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`

	// Searchable Text Fields
	ShortDescription string `json:"short_description"`
	LongDescription  string `json:"long_description,omitempty"`

	// Media
	ImageURL   string `json:"image_url,omitempty"`
	IconURL    string `json:"icon_url,omitempty"`
	WebsiteURL string `json:"website_url,omitempty"`

	// Organization Context
	ClaimedByOrgID   string `json:"claimed_by_org_id,omitempty"`
	OrganizationName string `json:"organization_name,omitempty"`
	HasVerifiedOrg   bool   `json:"has_verified_org"`

	// --- Engagement Metrics ---
	SupporterCount int     `json:"supporter_count"`
	TrendingScore  float64 `json:"trending_score"`

	// --- Depth of Commitment Metrics ---
	AvgVerificationTier      float64        `json:"avg_verification_tier"`
	VerifiedSupporterCount   int            `json:"verified_supporter_count"`
	UnverifiedSupporterCount int            `json:"unverified_supporter_count"`
	MinVerificationTier      int            `json:"min_verification_tier"`
	MaxVerificationTier      int            `json:"max_verification_tier"`
	MinBadgeLevelNumeric     int            `json:"min_badge_level_numeric"`
	MaxBadgeLevelNumeric     int            `json:"max_badge_level_numeric"`
	TierDistribution         map[string]int `json:"tier_distribution"`
	BadgeDistribution        map[string]int `json:"badge_distribution"`

	// --- Category Data ---
	CategoryIDs   []string `json:"category_ids"`
	CategorySlugs []string `json:"category_slugs"`
	CategoryNames []string `json:"category_names"`

	// --- Administrative ---
	Status          string `json:"status"`
	CreatedByUserID string `json:"created_by_user_id"`

	// --- Timestamps ---
	CreatedAtUnix int64 `json:"created_at_unix"`
	UpdatedAtUnix int64 `json:"updated_at_unix"`
}

// UserDocument represents the data structure for a user in the search index.
type UserDocument struct {
	// --- Core Identity ---
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`

	// Searchable Fields
	Bio      string `json:"bio"`
	Location string `json:"location"`

	// --- Denormalized from user_movements -> movements table
	MovementIDs   []string `json:"movement_ids"`
	MovementNames []string `json:"movement_names"`
	MovementSlugs []string `json:"movement_slugs"`

	// --- Depth of Commitment Metrics ---
	AvgVerificationTier     float64 `json:"avg_verification_tier"`
	MinVerificationTier     int     `json:"min_verification_tier"`
	MaxBadgeLevelNumeric    int     `json:"max_badge_level_numeric"`
	VerifiedMovementCount   int     `json:"verified_movement_count"`
	TotalMovementsSupported int     `json:"total_movements_supported"`

	// --- Category Affiliations ---
	CategoryIDs   []string `json:"category_ids"`
	CategorySlugs []string `json:"category_slugs"`
	CategoryNames []string `json:"category_names"`

	// --- Administrative ---
	Status            string `json:"status"`
	ProfileVisibility string `json:"profile_visibility"`
	IsDeleted         bool   `json:"is_deleted"`

	// --- Timestamps ---
	CreatedAtUnix   int64 `json:"created_at_unix"`
	UpdatedAtUnix   int64 `json:"updated_at_unix"`
	LastLoginAtUnix int64 `json:"last_login_at_unix"`
}

// OrganizationDocument represents the data structure for an organization in the search index.
type OrganizationDocument struct {
	// --- Core Identity ---
	ID               string `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	ShortDescription string `json:"short_description"`
	LongDescription  string `json:"long_description"`
	LogoURL          string `json:"logo_url"`
	WebsiteURL       string `json:"website_url"`
	CountryCode      string `json:"country_code"`

	// Verification Status
	IsVerified bool   `json:"is_verified"`
	Status     string `json:"status"`

	// --- Engagement Metrics ---
	SupporterCount int `json:"supporter_count"`
	MovementCount  int `json:"movement_count"`

	// --- Timestamps ---
	CreatedAtUnix int64 `json:"created_at_unix"`
	UpdatedAtUnix int64 `json:"updated_at_unix"`
}

// --- Helper Functions ---

// BadgeLevelToString converts a badge level to its string representation.
func BadgeLevelToString(badLevel int) string {
	badgeMap := map[int]string{
		1: "bronze",
		2: "silver",
		3: "gold",
		4: "platinum",
		5: "diamond",
	}
	if val, ok := badgeMap[badLevel]; ok {
		return val
	}
	return ""
}

// BadgeLevelToNumeric converts a badge level to its numeric representation.
func BadgeLevelToNumeric(badgeLevel string) int {
	badgeMap := map[string]int{
		"bronze":   1,
		"silver":   2,
		"gold":     3,
		"platinum": 4,
		"diamond":  5,
	}
	if val, ok := badgeMap[badgeLevel]; ok {
		return val
	}
	return 0
}
