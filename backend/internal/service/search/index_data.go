package search

import "time"

// MovementIndexData is the full set of data required to build a MovementDocument.
// The Postgres query that populates this should JOIN:
//   - movements
//   - organizations (via claimed_by_org_id)
//   - movement_categories → categories
//   - user_movements (aggregated)
type MovementIndexData struct {
	// Core movement fields
	ID               string
	Slug             string
	Name             string
	ShortDescription string
	LongDescription  string
	ImageURL         string
	IconURL          string
	WebsiteURL       string
	Status           string
	SupporterCount   int64
	TrendingScore    float64
	CreatedByUserID  string
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Organization (may be nil if not claimed)
	ClaimedByOrgID   string
	OrganizationName string
	HasVerifiedOrg   bool

	// Categories (multiple rows joined)
	CategoryIDs   []string
	CategorySlugs []string
	CategoryNames []string

	// ── Depth-of-Commitment Verification Metrics ──────────────────────────
	// These are computed by the Postgres layer via aggregate queries on
	// user_movements WHERE removed_at IS NULL AND is_public = TRUE.
	AvgVerificationTier      float64
	VerifiedSupporterCount   int
	UnverifiedSupporterCount int
	MinVerificationTier      int
	MaxVerificationTier      int

	// Badge levels as strings from user_movements.badge_level
	// The service converts these to numerics for the document.
	MinBadgeLevel string
	MaxBadgeLevel string

	// Distribution of supporters across tiers and badges.
	TierDistribution  map[int]int
	BadgeDistribution map[int]int
}

// UserIndexData is the full set of data required to build a UserDocument.
type UserIndexData struct {
	ID                string
	Username          string
	DisplayName       string
	Bio               string
	Location          string
	ProfileVisibility string
	Status            string
	IsDeleted         bool
	CreatedAt         time.Time
	UpdatedAt         time.Time

	// Denormalized from user_movements → movements
	MovementIDs   []string
	MovementNames []string
	MovementSlugs []string

	// Categories
	// * REVISIT: do we need this?
	CategoryIDs   []string
	CategorySlugs []string
	CategoryNames []string

	// Aggregated from user_movements
	TotalMovementsSupported int
	AvgVerificationTier     float64
	MinVerificationTier     int
	MaxBadgeLevel           string // converted to numeric by service
	VerifiedMovementCount   int    // count where verification_tier >= 2
}

// OrgIndexData is the full set of data required to build an OrganizationDocument.
type OrgIndexData struct {
	ID               string
	Slug             string
	Name             string
	ShortDescription string
	LongDescription  string
	LogoURL          string
	WebsiteURL       string
	CountryCode      string
	IsVerified       bool
	Status           string
	SupporterCount   int64
	MovementCount    int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
