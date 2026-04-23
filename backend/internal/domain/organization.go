package domain

import (
	"time"

	"github.com/google/uuid"
)

// OrganizationStatus represents the status of an organization.
type OrganizationStatus string

const (
	OrganizationStatusActive    OrganizationStatus = "active"
	OrganizationStatusInactive  OrganizationStatus = "inactive"
	OrganizationStatusSuspended OrganizationStatus = "suspended"
	OrganizationStatusRejected  OrganizationStatus = "rejected"
)

// VerificationStatus represents the verification status of an organization.
type VerificationStatus string

const (
	VerificationStatusUnverified VerificationStatus = "unverified"
	VerificationStatusPending    VerificationStatus = "pending"
	VerificationStatusVerified   VerificationStatus = "verified"
	VerificationStatusRejected   VerificationStatus = "rejected"
)

// SocialLinks holds URLs for various social media platforms.
type SocialLinks struct {
	X         string `json:"x,omitempty"`
	Bluesky   string `json:"bluesky,omitempty"`
	Instagram string `json:"instagram,omitempty"`
	Facebook  string `json:"facebook,omitempty"`
	LinkedIn  string `json:"linkedin,omitempty"`
	YouTube   string `json:"youtube,omitempty"`
	TikTok    string `json:"tiktok,omitempty"`
}

// Organization represents a row in the organizations table.
type Organization struct {
	// Identity
	ID               uuid.UUID `db:"id" json:"id"`
	Name             string    `db:"name" json:"name"`
	Slug             string    `db:"slug" json:"slug"`
	ShortDescription string    `db:"short_description" json:"short_description"`
	LongDescription  string    `db:"long_description" json:"long_description"`

	// Media & Contacts
	LogoURL       string `db:"logo_url" json:"logo_url"`
	CoverImageURL string `db:"cover_image_url" json:"cover_image_url"`
	WebsiteURL    string `db:"website_url" json:"website_url"`
	ContactEmail  string `db:"contact_email" json:"contact_email"`

	// Legal & Location
	EINTaxIDHash string `db:"ein_tax_id_hash" json:"ein_tax_id_hash"`
	CountryCode  string `db:"country_code" json:"country_code"`

	// Status & Verification
	Status             OrganizationStatus `db:"status" json:"status"`
	VerificationStatus VerificationStatus `db:"verification_status" json:"verification_status"`
	IsVerified         bool               `db:"is_verified" json:"is_verified"`
	VerifiedAt         *time.Time         `db:"verified_at" json:"verified_at,omitempty"`
	VerifiedByUserID   *uuid.UUID         `db:"verified_by_user_id" json:"verified_by_user_id,omitempty"`

	// Ownership
	CreatedByUserID uuid.UUID `db:"created_by_user_id" json:"created_by_user_id"`

	// Flexible Data
	SocialLinks SocialLinks `db:"social_links" json:"social_links"`

	// Timestamps
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

func (o *Organization) IsDeleted() bool {
	return o.DeletedAt != nil
}

func (o *Organization) IsActive() bool {
	return o.Status == OrganizationStatusActive && !o.IsDeleted()
}
