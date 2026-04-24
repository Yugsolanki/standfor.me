package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/config"
	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func main() {
	// Load Config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("❌ Failed to load config: ", err)
	}

	// --- Init Database ---
	db, err := sqlx.ConnectContext(context.Background(), "pgx", cfg.Database.DSN())
	if err != nil {
		log.Fatal("❌ Failed to connect to database: ", err)
	}
	defer db.Close()

	log.Println("Database connected", "db", cfg.Database.Host)

	_ = gofakeit.Seed(time.Now().UnixNano())
	ctx := context.Background()

	log.Println("🌱 Starting database seeding...")

	// Clear old data
	clearData(ctx, *db)

	users := seedUsers(context.Background(), db, 100)

	categories := seedCategories(context.Background(), db)

	organizations := seedOrganizations(context.Background(), *db, 20, users)

	movements := seedMovements(context.Background(), db, 50, users, organizations)

	_ = seedMovementCategories(context.Background(), db, categories, movements)

	_ = seedUserMovements(context.Background(), db, 200, users, movements)

	log.Println("✅ Seeding completed successfully!")
}

// --- SEEDING FUNCTIONS ---

// seedUsers seeds the database with fake users.
func seedUsers(ctx context.Context, db *sqlx.DB, count int) []uuid.UUID {
	log.Println("Seeding users", "count", count)
	var ids []uuid.UUID

	const query = `
		INSERT INTO users (
			id, username, email, email_verified_at, password_hash, 
			display_name, bio, avatar_url, location, 
			profile_visibility, embed_enabled, role, status, 
			last_login_at, created_at, updated_at, deleted_at
		)
		VALUES (
			:id, :username, :email, :email_verified_at, :password_hash, 
			:display_name, :bio, :avatar_url, :location, 
			:profile_visibility, :embed_enabled, :role, :status, 
			:last_login_at, :created_at, :updated_at, :deleted_at
		);
	`

	users := GenerateFakeUsers(100)

	for _, user := range users {
		_, err := db.NamedExecContext(ctx, query, user)
		if err != nil {
			log.Println("❌ Failed to seed user", "error", err)
		}
		ids = append(ids, user.ID)
	}

	log.Println("Users seeded", "count", count)
	return ids
}

// seedCategories seeds the database with fake categories.
func seedCategories(ctx context.Context, db *sqlx.DB) []uuid.UUID {
	log.Println("Seeding categories")
	var ids []uuid.UUID

	const query = `
		INSERT INTO categories (
			id, parent_id, name, slug, description, 
			icon_url, display_order, is_active, 
			created_at, updated_at, deleted_at
		)
		VALUES (
			:id, :parent_id, :name, :slug, :description, 
			:icon_url, :display_order, :is_active, 
			:created_at, :updated_at, :deleted_at
		);
	`

	for _, cat := range GenerateFakeCategories() {
		_, err := db.NamedExecContext(ctx, query, cat)
		if err != nil {
			log.Println("❌ Failed to seed category", "error", err)
		}
		ids = append(ids, cat.ID)
	}

	log.Println("Categories seeded", "count", len(ids))
	return ids
}

// seedOrganizations seeds the databased with fake organizations.
func seedOrganizations(ctx context.Context, db sqlx.DB, count int, userIDs []uuid.UUID) []uuid.UUID {
	log.Println("Seeding organizations", "count", count)
	var ids []uuid.UUID

	const query = `
		INSERT INTO organizations (
			id, name, slug, short_description, long_description, 
			logo_url, cover_image_url, website_url, contact_email, 
			ein_tax_id_hash, country_code, status, verification_status, 
			is_verified, verified_at, verified_by_user_id, 
			created_by_user_id, social_links, created_at, updated_at, deleted_at
		)
		VALUES (
			:id, :name, :slug, :short_description, :long_description, 
			:logo_url, :cover_image_url, :website_url, :contact_email, 
			:ein_tax_id_hash, :country_code, :status, :verification_status, 
			:is_verified, :verified_at, :verified_by_user_id, 
			:created_by_user_id, :social_links, :created_at, :updated_at, :deleted_at
		);
	`

	organizations := GenerateFakeOrganizations(count, userIDs)

	for _, org := range organizations {
		_, err := db.NamedExecContext(ctx, query, org)
		if err != nil {
			log.Println("❌ Failed to seed user", "error", err)
		}
		ids = append(ids, org.ID)
	}

	log.Println("Organizations seeded", "count", len(ids))
	return ids
}

// seedMovements seeds the database with fake movements.
func seedMovements(ctx context.Context, db *sqlx.DB, count int, userIDs []uuid.UUID, orgIDs []uuid.UUID) []uuid.UUID {
	log.Println("Seeding movements", "count", count)
	var ids []uuid.UUID

	const query = `
		INSERT INTO movements (
			id, slug, name, short_description, long_description, 
			image_url, icon_url, website_url, supporter_count, 
			trending_score, status, claimed_by_org_id, 
			created_by_user_id, reviewed_by_user_id, reviewed_at, 
			created_at, updated_at, deleted_at
		)
		VALUES (
			:id, :slug, :name, :short_description, :long_description, 
			:image_url, :icon_url, :website_url, :supporter_count, 
			:trending_score, :status, :claimed_by_org_id, 
			:created_by_user_id, :reviewed_by_user_id, :reviewed_at, 
			:created_at, :updated_at, :deleted_at
		);
	`

	movements := GenerateFakeMovements(count, userIDs, orgIDs)

	for _, mov := range movements {
		_, err := db.NamedExecContext(ctx, query, mov)
		if err != nil {
			log.Println("❌ Failed to seed movememt", "error", err)
		}
		ids = append(ids, mov.ID)
	}

	log.Println("Movements seeded", "count", len(ids))
	return ids
}

// seedMovementCategories seeds the database with fake movement_categories.
func seedMovementCategories(ctx context.Context, db *sqlx.DB, catIDs []uuid.UUID, movIDs []uuid.UUID) []domain.MovementCategory {
	log.Println("Seeding movement_categories")
	var mcs []domain.MovementCategory

	const query = `
		INSERT INTO movement_categories (
			movement_id, category_id, is_primary, created_at
		)
		VALUES (
			:movement_id, :category_id, :is_primary, :created_at
		);
	`

	movementCategories := GenerateFakeMovementCategories(movIDs, catIDs)

	for _, mc := range movementCategories {
		_, err := db.NamedExecContext(ctx, query, mc)
		if err != nil {
			log.Println("❌ Failed to seed movement_category", "error", err)
		}
		mcs = append(mcs, mc)
	}

	log.Println("Movement categories seeded", "count", len(mcs))
	return mcs
}

// seedUserMovements seeds the database with fake user_movements.
func seedUserMovements(ctx context.Context, db *sqlx.DB, count int, userIDs []uuid.UUID, movIDs []uuid.UUID) []uuid.UUID {
	log.Println("Seeding user_movements", "count", count)
	var ids []uuid.UUID

	const query = `
		INSERT INTO user_movements (
			id, user_id, movement_id, personal_statement, verification_tier, badge_level, display_order, is_pinned, is_public, status, supported_since, created_at, updated_at, removed_at
		)
		VALUES (
			:id, :user_id, :movement_id, :personal_statement, :verification_tier, :badge_level, :display_order, :is_pinned, :is_public, :status, :supported_since, :created_at, :updated_at, :removed_at
		);
	`

	userMovements := GenerateFakeUserMovements(count, userIDs, movIDs)

	for _, um := range userMovements {
		_, err := db.NamedExecContext(ctx, query, um)
		if err != nil {
			log.Println("❌ Failed to seed user_movement", "error", err)
		}
		ids = append(ids, um.ID)
	}

	log.Println("User movements seeded", "count", len(ids))
	return ids
}

// ----------------------------------------
// Fake Data Generating Functions
// ----------------------------------------

func GenerateFakeUsers(count int) []domain.User {
	var users []domain.User

	for range count {
		var user domain.User

		user.ID = uuid.New()
		user.Username = gofakeit.Regex("[a-z0-9_-]{5,30}")
		emailPrefix := strings.ReplaceAll(user.Username, "-", "")
		user.Email = emailPrefix + "@example.com"

		displayName := gofakeit.Name()
		if len(displayName) < 3 {
			displayName += " Doe" // Pad if too short
		} else if len(displayName) > 50 {
			displayName = displayName[:50] // Truncate if too long
		}
		user.DisplayName = displayName

		user.PasswordHash = ptr(fmt.Sprintf("$2a$10$%s", gofakeit.Password(true, true, true, false, false, 53)))

		user.ProfileVisibility = gofakeit.RandomString([]string{"public", "private", "unlisted"})
		user.Role = gofakeit.RandomString([]string{"user", "admin", "moderator"})
		user.Status = gofakeit.RandomString([]string{"active", "suspended", "banned", "deactivated"})

		// Optional Fields (Simulating realistic nullability - roughly 70% chance of existing)
		if gofakeit.Number(1, 100) > 30 {
			verifiedAt := gofakeit.PastDate()
			user.EmailVerifiedAt = &verifiedAt
		}

		if gofakeit.Number(1, 100) > 30 {
			// Bio: <= 1000 characters
			bio := gofakeit.Sentence(20)
			if len(bio) > 1000 {
				bio = bio[:1000]
			}
			user.Bio = &bio
		}

		if gofakeit.Number(1, 100) > 30 {
			// Avatar URL: <= 2048 characters
			url := gofakeit.URL()
			user.AvatarURL = &url
		}

		if gofakeit.Number(1, 100) > 30 {
			// Location: <= 100 characters
			loc := fmt.Sprintf("%s, %s", gofakeit.City(), gofakeit.State())
			if len(loc) > 100 {
				loc = loc[:100]
			}
			user.Location = &loc
		}

		if gofakeit.Number(1, 100) > 30 {
			lastLogin := gofakeit.PastDate()
			user.LastLoginAt = &lastLogin
		}

		// Deleted At (Rare, only 5% of users might be soft-deleted)
		if gofakeit.Number(1, 100) <= 5 {
			deletedAt := gofakeit.PastDate()
			user.DeletedAt = &deletedAt
		}

		createdAt := gofakeit.PastDate()
		user.CreatedAt = createdAt

		user.UpdatedAt = gofakeit.DateRange(createdAt, time.Now())

		lastLogin := gofakeit.DateRange(createdAt, time.Now())
		user.LastLoginAt = &lastLogin

		users = append(users, user)

	}

	return users
}

func GenerateFakeCategories() []domain.Category {
	var categories []domain.Category

	for i, name := range MovementCategories {
		now := time.Now()
		createdAt := gofakeit.PastDate()

		cat := domain.Category{
			ID:           uuid.New(),
			Name:         name,
			Slug:         generateSlug(name),
			Description:  gofakeit.Sentence(12),
			DisplayOrder: i + 1,
			IsActive:     gofakeit.Number(1, 100) > 10,
			CreatedAt:    createdAt,
			UpdatedAt:    gofakeit.DateRange(createdAt, now),
		}

		// 50% chance to have an Icon URL
		if gofakeit.Bool() {
			url := gofakeit.URL()
			cat.IconURL = &url
		}

		// Hierarchy Logic: 30% chance to be a subcategory (assign a ParentID)
		// We only pick from previously generated categories to prevent infinite loops / invalid states
		if len(categories) > 0 && gofakeit.Number(1, 100) > 70 {
			parentIndex := gofakeit.Number(0, len(categories)-1)
			parentID := categories[parentIndex].ID
			cat.ParentID = &parentID
		}

		// 5% chance the category is soft-delete
		if gofakeit.Number(1, 100) <= 5 {
			deletedAt := gofakeit.DateRange(createdAt, now)
			cat.DeletedAt = &deletedAt
			cat.IsActive = false
		}

		categories = append(categories, cat)
	}

	return categories
}

func GenerateFakeMovements(count int, userIDs []uuid.UUID, orgIDs []uuid.UUID) []domain.Movement {
	var movements []domain.Movement
	titleCaser := cases.Title(language.English)

	for i := range count {
		now := time.Now()
		createdAt := gofakeit.PastDate()
		updatedAt := gofakeit.DateRange(createdAt, now)

		// Generate a catchy movement name
		name := fmt.Sprintf("The %s %s Movement", titleCaser.String(gofakeit.Adjective()), titleCaser.String(gofakeit.Noun()))

		m := domain.Movement{
			ID:               uuid.New(),
			Name:             name,
			Slug:             generateSlug(name),
			ShortDescription: gofakeit.Sentence(20),
			SupporterCount:   gofakeit.Number(0, 50000),
			TrendingScore:    math.Round(gofakeit.Float64Range(0, 100)*100) / 100,
			Status: gofakeit.RandomString([]string{
				domain.MovementStatusDraft,
				domain.MovementStatusActive,
				domain.MovementStatusArchived,
				domain.MovementStatusRejected,
				domain.MovementStatusPendingReview,
			}),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		// 80% chance to have a long description
		if gofakeit.Number(1, 100) > 20 {
			m.LongDescription = ptr(gofakeit.Sentence(5000))
		}

		// URLs (Random probabilities)
		if gofakeit.Bool() {
			m.ImageURL = ptr(gofakeit.URL())
		}
		if gofakeit.Bool() {
			m.IconURL = ptr(gofakeit.URL())
		}
		if gofakeit.Number(1, 100) > 40 {
			webURL := gofakeit.URL()
			m.WebsiteURL = &webURL
		}

		// --- Relational Data ---

		// Assign a Creator
		if len(userIDs) > 0 {
			creator := userIDs[gofakeit.Number(0, len(userIDs)-1)]
			m.CreatedByUserID = &creator
		}

		// Logic for Reviewer: Only Active, Archived, or Rejected movements have been reviewed
		if m.Status == domain.MovementStatusActive || m.Status == domain.MovementStatusArchived || m.Status == domain.MovementStatusRejected {
			reviewedAt := gofakeit.DateRange(createdAt, updatedAt)
			m.ReviewedAt = &reviewedAt

			if len(userIDs) > 0 {
				reviewer := userIDs[gofakeit.Number(0, len(userIDs)-1)]
				m.ReviewedByUserID = &reviewer
			}
		}

		// 40% chance the movement is claimed by an Organization
		if len(orgIDs) > 0 && gofakeit.Number(1, 100) > 60 {
			orgID := orgIDs[gofakeit.Number(0, len(orgIDs)-1)]
			m.ClaimedByOrgID = &orgID
		}

		// 5% chance the movement is soft-deleted
		if gofakeit.Number(1, 100) <= 5 {
			deletedAt := gofakeit.DateRange(updatedAt, now)
			m.DeletedAt = &deletedAt
		}

		_ = i
		movements = append(movements, m)
	}

	return movements
}

// GenerateMovementCategories assigns random categories to movements.
func GenerateFakeMovementCategories(movementIDs []uuid.UUID, categoryIDs []uuid.UUID) []domain.MovementCategory {
	var movementCategories []domain.MovementCategory

	if len(movementIDs) == 0 || len(categoryIDs) == 0 {
		return movementCategories
	}

	for _, mID := range movementIDs {
		// Assign between 1 and 5 categories per movement
		numCategories := min(gofakeit.Number(1, 5), len(categoryIDs))

		usedCats := make(map[uuid.UUID]bool)

		for i := range numCategories {
			catID := categoryIDs[gofakeit.Number(0, len(categoryIDs)-1)]

			// Prevent assigning the same category twice to the same movement
			for usedCats[catID] {
				catID = categoryIDs[gofakeit.Number(0, len(categoryIDs)-1)]
			}
			usedCats[catID] = true

			mc := domain.MovementCategory{
				MovementID: mID,
				CategoryID: catID,
				IsPrimary:  i == 0,
				CreatedAt:  gofakeit.PastDate(),
			}

			movementCategories = append(movementCategories, mc)
		}
	}

	return movementCategories
}

// GenerateFakeUserMovements creates fake advocacy records tying users to movements.
func GenerateFakeUserMovements(count int, userIDs []uuid.UUID, movementIDs []uuid.UUID) []domain.UserMovement {
	var userMovements []domain.UserMovement

	if len(userIDs) == 0 || len(movementIDs) == 0 {
		return userMovements
	}

	usedPairs := make(map[string]bool)

	for i := range count {
		// Randomly pair a user and a movement
		uID := userIDs[gofakeit.Number(0, len(userIDs)-1)]
		mID := movementIDs[gofakeit.Number(0, len(movementIDs)-1)]
		pairKey := uID.String() + "_" + mID.String()

		// Prevent unique constraint violations (User supporting same movement twice)
		retries := 0
		for usedPairs[pairKey] && retries < 10 {
			uID = userIDs[gofakeit.Number(0, len(userIDs)-1)]
			mID = movementIDs[gofakeit.Number(0, len(movementIDs)-1)]
			pairKey = uID.String() + "_" + mID.String()
			retries++
		}
		if retries >= 10 {
			continue // Skip if we are running out of unique combinations
		}
		usedPairs[pairKey] = true

		// Timestamps
		now := time.Now()
		supportedSince := gofakeit.PastDate()
		createdAt := gofakeit.DateRange(supportedSince, now)
		updatedAt := gofakeit.DateRange(createdAt, now)

		// Constants mapping
		status := domain.AdvocacyStatus(gofakeit.RandomString([]string{
			string(domain.AdvocacyStatusActive),
			string(domain.AdvocacyStatusPaused),
			string(domain.AdvocacyStatusRemoved),
		}))

		um := domain.UserMovement{
			ID:         uuid.New(),
			UserID:     uID,
			MovementID: mID,
			// #nosec G115
			VerificationTier: int16(gofakeit.Number(0, 5)),
			BadgeLevel: domain.BadgeLevel(gofakeit.RandomString([]string{
				string(domain.BadgeLevelBronze),
				string(domain.BadgeLevelSilver),
				string(domain.BadgeLevelGold),
				string(domain.BadgeLevelPlatinum),
				string(domain.BadgeLevelDiamond),
			})),
			// #nosec G115
			DisplayOrder:   int16(gofakeit.Number(1, 10)),
			IsPinned:       gofakeit.Number(1, 100) <= 15, // Make pinning less common (15% chance).
			IsPublic:       gofakeit.Number(1, 100) > 10,  // Make public the default (90% chance).
			Status:         status,
			SupportedSince: supportedSince,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
		}

		// 40% chance of having a personal statement
		if gofakeit.Number(1, 100) <= 40 {
			statement := gofakeit.Sentence(20)
			um.PersonalStatement = &statement
		}

		// Important: Set RemovedAt only if the status is 'removed' for data consistency.
		if status == domain.AdvocacyStatusRemoved {
			removedAt := gofakeit.DateRange(createdAt, now)
			um.RemovedAt = &removedAt
		}

		_ = i
		userMovements = append(userMovements, um)
	}

	return userMovements
}

func GenerateFakeOrganizations(count int, userIDs []uuid.UUID) []domain.Organization {
	var orgs []domain.Organization

	if len(userIDs) == 0 {
		return orgs
	}

	for i := range count {
		now := time.Now()
		createdAt := gofakeit.PastDate()
		updatedAt := gofakeit.DateRange(createdAt, now)
		name := gofakeit.Company()

		org := domain.Organization{
			ID:               uuid.New(),
			Name:             name,
			Slug:             generateSlug(name),
			ShortDescription: gofakeit.Slogan(),
			LongDescription:  gofakeit.Paragraph(2),
			LogoURL:          gofakeit.URL(),
			CoverImageURL:    gofakeit.URL(),
			WebsiteURL:       gofakeit.URL(),
			ContactEmail:     gofakeit.Email(),
			EINTaxIDHash:     fmt.Sprintf("$2a$10$%s", gofakeit.Password(true, true, true, false, false, 53)), // Simulate a hash,
			CountryCode:      gofakeit.CountryAbr(),
			CreatedByUserID:  userIDs[gofakeit.Number(0, len(userIDs)-1)],
			CreatedAt:        createdAt,
			UpdatedAt:        updatedAt,
		}

		// --- Status & Verification Logic ---

		// This block ensures data consistency
		vStatus := domain.VerificationStatus(gofakeit.RandomString([]string{
			string(domain.VerificationStatusUnverified),
			string(domain.VerificationStatusPending),
			string(domain.VerificationStatusVerified),
			string(domain.VerificationStatusRejected),
		}))
		org.VerificationStatus = vStatus

		if vStatus == domain.VerificationStatusVerified {
			org.IsVerified = true
			verifiedAt := gofakeit.DateRange(createdAt, updatedAt)
			org.VerifiedAt = &verifiedAt
			org.VerifiedByUserID = &userIDs[gofakeit.Number(0, len(userIDs)-1)]

			// 90% chance of a verified org to be active
			if gofakeit.Number(1, 100) < 90 {
				org.Status = domain.OrganizationStatusActive
			} else {
				org.Status = domain.OrganizationStatusInactive
			}
		} else {
			org.IsVerified = false
			// Set a status that makes sense for a non-verified org.
			org.Status = domain.OrganizationStatus(gofakeit.RandomString([]string{
				string(domain.OrganizationStatusInactive),
				string(domain.OrganizationStatusSuspended),
				string(domain.OrganizationStatusRejected),
			}))
		}

		// --- Nested Social Link Struct

		// Randomly populate social links to make data more realistic.
		links := domain.SocialLinks{}
		if gofakeit.Bool() {
			links.X = "https://x.com/" + gofakeit.Username()
		}
		if gofakeit.Bool() {
			links.Facebook = "https://facebook.com/" + gofakeit.Username()
		}
		if gofakeit.Bool() {
			links.Instagram = "https://instagram.com/" + gofakeit.Username()
		}
		if gofakeit.Bool() {
			links.LinkedIn = "https://linkedin.com/company/" + gofakeit.Username()
		}
		org.SocialLinks = links

		// A small chance for an organization to be deleted.
		if gofakeit.Number(1, 100) <= 5 {
			deletedAt := gofakeit.DateRange(createdAt, now)
			org.DeletedAt = &deletedAt
			org.Status = domain.OrganizationStatusInactive // Deleted orgs should be inactive.
		}

		_ = i
		orgs = append(orgs, org)
	}

	return orgs
}

// ----------------------------------------
// Helper Functions
// ----------------------------------------

// clearData truncates all tables in the database, effectively resetting the DB to an empty state.
func clearData(ctx context.Context, db sqlx.DB) {
	log.Println("Clearing old data")
	tables := []string{
		"movement_categories", "user_movements", "movements",
		"organizations", "categories", "users",
	}
	for _, table := range tables {
		_, err := db.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil && !strings.Contains(err.Error(), "does not exist") {
			log.Printf("Warning: ❌ Failed to truncate %s: %v", table, err)
		}
	}
}

// generateSlug creates a URL-friendly slug from the category name
func generateSlug(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " & ", "-and-")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "'", "")
	// Remove any remaining non-alphanumeric chars except hyphens
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, s)

	if len(s) == 0 {
		return "a" // Fallback
	}

	return s
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

// Extracts IDs from a slice
// Extract User.ID, Movement.ID, Organization.ID, Category.ID, UserMovement.ID
// func extractIDs[T any](s []T, f func(T) uuid.UUID) []uuid.UUID {
// 	ids := make([]uuid.UUID, 0, len(s))
// 	for _, v := range s {
// 		ids = append(ids, f(v))
// 	}
// 	return ids
// }
