package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/config"
	"github.com/brianvoe/gofakeit/v7"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
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

	users := seedUsers(context.Background(), db, 1000)

	categories := seedCategories(context.Background(), db)

	organizations := seedOrganizations(context.Background(), *db, 200, users)

	movements := seedMovements(context.Background(), db, 500, users, organizations)

	_ = seedMovementCategories(context.Background(), db, categories, movements)

	_ = seedUserMovements(context.Background(), db, 3000, users, movements)

	log.Println("✅ Seeding completed successfully!")
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

// makeUniqueSlug returns a unique slug by appending -N suffix on collision.
func makeUniqueSlug(slug string, used map[string]int) string {
	base := slug
	n, exists := used[slug]
	if exists {
		// find next available suffix
		for {
			n++
			candidate := fmt.Sprintf("%s-%d", base, n)
			if _, exists := used[candidate]; !exists {
				break
			}
		}
	}
	used[slug] = n + 1
	if n == 0 {
		return slug
	}
	return fmt.Sprintf("%s-%d", base, n)
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
