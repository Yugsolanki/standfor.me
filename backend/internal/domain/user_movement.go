package domain

import (
	"time"

	"github.com/google/uuid"
)

// AdvocacyStatus represents the current state of a user's movement support.
type AdvocacyStatus string

const (
	AdvocacyStatusActive  AdvocacyStatus = "active"
	AdvocacyStatusPaused  AdvocacyStatus = "paused"
	AdvocacyStatusRemoved AdvocacyStatus = "removed"
)

// BadgeLevel represents the earned recognition tier for a user's advocacy.
type BadgeLevel string

const (
	BadgeLevelBronze   BadgeLevel = "bronze"   // self
	BadgeLevelSilver   BadgeLevel = "silver"   // social
	BadgeLevelGold     BadgeLevel = "gold"     // financial
	BadgeLevelPlatinum BadgeLevel = "platinum" // action
	BadgeLevelDiamond  BadgeLevel = "diamond"  // org
)

// UserMovement represents a row in the user_movements table.
type UserMovement struct {
	// Identity
	ID uuid.UUID `db:"id" json:"id"`

	// Relationships
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	MovementID uuid.UUID `db:"movement_id" json:"movement_id"`

	// Advocacy Details
	PersonalStatement *string        `db:"personal_statement" json:"personal_statement,omitempty"`
	VerificationTier  int16          `db:"verification_tier" json:"verification_tier"`
	BadgeLevel        BadgeLevel     `db:"badge_level" json:"badge_level"`
	DisplayOrder      int16          `db:"display_order" json:"display_order"`
	IsPinned          bool           `db:"is_pinned" json:"is_pinned"`
	IsPublic          bool           `db:"is_public" json:"is_public"`
	Status            AdvocacyStatus `db:"status" json:"status"`

	// Timestamps
	SupportedSince time.Time  `db:"supported_since" json:"supported_since"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
	RemovedAt      *time.Time `db:"removed_at" json:"removed_at,omitempty"`
}

func (um *UserMovement) IsDeleted() bool {
	return um.RemovedAt != nil
}
