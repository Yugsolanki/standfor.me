package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	MaxCategoriesPerMovement = 10
)

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
	usedSlugs := make(map[string]int)

	for range count {
		now := time.Now()
		createdAt := gofakeit.PastDate()
		updatedAt := gofakeit.DateRange(createdAt, now)

		// Generate a catchy movement name
		name := fmt.Sprintf("The %s %s Movement", titleCaser.String(gofakeit.Adjective()), titleCaser.String(gofakeit.Noun()))

		m := domain.Movement{
			ID:               uuid.New(),
			Name:             name,
			Slug:             makeUniqueSlug(generateSlug(name), usedSlugs),
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
		numCategories := min(gofakeit.Number(1, MaxCategoriesPerMovement), len(categoryIDs))

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

	for range count {
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

		userMovements = append(userMovements, um)
	}

	return userMovements
}

func GenerateFakeOrganizations(count int, userIDs []uuid.UUID) []domain.Organization {
	var orgs []domain.Organization

	if len(userIDs) == 0 {
		return orgs
	}

	usedSlugs := make(map[string]int)

	for range count {
		now := time.Now()
		createdAt := gofakeit.PastDate()
		updatedAt := gofakeit.DateRange(createdAt, now)
		name := gofakeit.Company()

		org := domain.Organization{
			ID:               uuid.New(),
			Name:             name,
			Slug:             makeUniqueSlug(generateSlug(name), usedSlugs),
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

		orgs = append(orgs, org)
	}

	return orgs
}
