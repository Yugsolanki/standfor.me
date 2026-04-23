package domain

import (
	"time"

	"github.com/google/uuid"
)

// Category represents a row in the categories table.
type Category struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	ParentID     *uuid.UUID `db:"parent_id" json:"parent_id"`
	Name         string     `db:"name" json:"name"`
	Slug         string     `db:"slug" json:"slug"`
	Description  string     `db:"description" json:"description"`
	IconURL      *string    `db:"icon_url" json:"icon_url"`
	DisplayOrder int        `db:"display_order" json:"display_order"`
	IsActive     bool       `db:"is_active" json:"is_active"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

func (c *Category) IsDeleted() bool {
	return c.DeletedAt != nil
}
