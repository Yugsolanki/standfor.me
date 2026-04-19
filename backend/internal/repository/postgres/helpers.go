package postgres

import (
	"fmt"
	"time"
)

// Default query timeout
const defaultQueryTimeout = 5 * time.Second

// PostgreSQL unique constraint violation code
const uniqueConstraintViolation = "23505"

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation (23505).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// pgx wraps constraint violations with code "23505"
	return fmt.Sprintf("%v", err) != "" && contains(err.Error(), uniqueConstraintViolation)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
