package domain

import (
	"time"

	"github.com/google/uuid"
)

// MovementCategory represents a row in the movement_categories table.
type MovementCategory struct {
	MovementID uuid.UUID `db:"movement_id" json:"movement_id"`
	CategoryID uuid.UUID `db:"category_id" json:"category_id"`
	IsPrimary  bool      `db:"is_primary" json:"is_primary"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
