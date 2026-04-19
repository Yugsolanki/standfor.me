package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	MovementStatusDraft         = "draft"
	MovementStatusActive        = "active"
	MovementStatusArchived      = "archived"
	MovementStatusRejected      = "rejected"
	MovementStatusPendingReview = "pending_review"
)

type Movement struct {
	ID               uuid.UUID  `db:"id"                    json:"id"`
	Slug             string     `db:"slug"                  json:"slug"`
	Name             string     `db:"name"                  json:"name"`
	ShortDescription string     `db:"short_description"     json:"short_description"`
	LongDescription  *string    `db:"long_description"     json:"long_description,omitempty"`
	ImageURL         *string    `db:"image_url"             json:"image_url,omitempty"`
	IconURL          *string    `db:"icon_url"              json:"icon_url,omitempty"`
	WebsiteURL       *string    `db:"website_url"          json:"website_url,omitempty"`
	SupporterCount   int        `db:"supporter_count"       json:"supporter_count"`
	TrendingScore    float64    `db:"trending_score"       json:"trending_score"`
	Status           string     `db:"status"                json:"status"`
	CreatedByUserID  *uuid.UUID `db:"created_by_user_id"    json:"created_by_user_id,omitempty"`
	ReviewedByUserID *uuid.UUID `db:"reviewed_by_user_id"   json:"reviewed_by_user_id,omitempty"`
	ReviewedAt       *time.Time `db:"reviewed_at"           json:"reviewed_at,omitempty"`
	DeletedAt        *time.Time `db:"deleted_at"            json:"-"`
	CreatedAt        time.Time  `db:"created_at"            json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"            json:"updated_at"`
}

type CreateMovementParams struct {
	Slug             string     `db:"slug"               validate:"required,min=3,max=100,slug"`
	Name             string     `db:"name"               validate:"required,min=3,max=200"`
	ShortDescription string     `db:"short_description" validate:"required,min=10,max=500"`
	LongDescription  *string    `db:"long_description"  validate:"omitempty,max=5000"`
	ImageURL         *string    `db:"image_url"          validate:"omitempty,url,max=2048"`
	IconURL          *string    `db:"icon_url"           validate:"omitempty,url,max=2048"`
	WebsiteURL       *string    `db:"website_url"       validate:"omitempty,url,max=2048"`
	CreatedByUserID  *uuid.UUID `db:"created_by_user_id" validate:"omitempty"`
}

type UpdateMovementParams struct {
	Slug             *string `db:"slug"               validate:"omitempty,min=3,max=100,slug"`
	Name             *string `db:"name"              validate:"omitempty,min=3,max=200"`
	ShortDescription *string `db:"short_description" validate:"omitempty,min=10,max=500"`
	LongDescription  *string `db:"long_description"  validate:"omitempty,max=5000"`
	ImageURL         *string `db:"image_url"          validate:"omitempty,url,max=2048"`
	IconURL          *string `db:"icon_url"           validate:"omitempty,url,max=2048"`
	WebsiteURL       *string `db:"website_url"       validate:"omitempty,url,max=2048"`
}

type UpdateMovementStatusParams struct {
	Status string `db:"status" validate:"required,oneof=draft active archived rejected pending_review"`
}

type SubmitForReviewParams struct {
	SubmittedByUserID uuid.UUID `db:"submitted_by_user_id" validate:"required"`
}

type ReviewMovementParams struct {
	ReviewedByUserID uuid.UUID `db:"reviewed_by_user_id" validate:"required"`
	Approved         bool      `db:"approved"`
}

type ListMovementsParams struct {
	Limit  int    `validate:"required,min=1,max=100"`
	Offset int    `validate:"min=0"`
	Status string `validate:"omitempty,oneof=draft active archived rejected pending_review"`
	Search string `validate:"omitempty,max=200"`
	SortBy string `validate:"omitempty,oneof=created_at updated_at supporter_count trending_score"`
	Order  string `validate:"omitempty,oneof=asc desc"`
}

type IncrementSupportersParams struct {
	MovementID uuid.UUID `db:"movement_id" validate:"required"`
}

type SearchMovementsParams struct {
	Query  string `validate:"required,min=2"`
	Limit  int    `validate:"required,min=1,max=100"`
	Offset int    `validate:"min=0"`
}

type ListSupportersParams struct {
	Limit  int `validate:"required,min=1,max=100"`
	Offset int `validate:"min=0"`
}
