package main

import (
	"context"
	"log"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

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

	users := GenerateFakeUsers(count)

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
			log.Println("❌ Failed to seed org", "name", org.Name, "error", err)
			continue
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
			log.Println("❌ Failed to seed movement", "name", mov.Name, "error", err)
			continue
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
