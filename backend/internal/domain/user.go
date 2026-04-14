package domain

import (
	"time"

	"github.com/google/uuid"
)

// --- User Profile Visibility ---

const (
	ProfileVisibilityPublic   = "public"
	ProfileVisibilityPrivate  = "private"
	ProfileVisibilityUnlisted = "unlisted"
)

// --- User Roles ---

const (
	RoleUser       = "user"
	RoleModerator  = "moderator"
	RoleAdmin      = "admin"
	RoleSuperAdmin = "superadmin"
)

// --- User Status ---

const (
	StatusActive      = "active"
	StatusSuspended   = "suspended"
	StatusBanned      = "banned"
	StatusDeactivated = "deactivated"
)

// User represents a row in the users table.
// Nullable columns use pointer types so sqlx can scan NULL correctly.
type User struct {
	ID                uuid.UUID  `db:"id"                  json:"id"`
	Username          string     `db:"username"            json:"username"`
	Email             string     `db:"email"               json:"email"`
	EmailVerifiedAt   *time.Time `db:"email_verified_at"   json:"email_verified_at,omitempty"`
	PasswordHash      *string    `db:"password_hash"       json:"-"`
	DisplayName       string     `db:"display_name"        json:"display_name"`
	Bio               *string    `db:"bio"                 json:"bio,omitempty"`
	AvatarURL         *string    `db:"avatar_url"          json:"avatar_url,omitempty"`
	Location          *string    `db:"location"            json:"location,omitempty"`
	ProfileVisibility string     `db:"profile_visibility"  json:"profile_visibility"`
	EmbedEnabled      bool       `db:"embed_enabled"       json:"embed_enabled"`
	Role              string     `db:"role"                json:"role"`
	Status            string     `db:"status"              json:"status"`
	LastLoginAt       *time.Time `db:"last_login_at"       json:"last_login_at,omitempty"`
	DeletedAt         *time.Time `db:"deleted_at"          json:"-"`
	CreatedAt         time.Time  `db:"created_at"          json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"          json:"updated_at"`
}

// CreateUserParams holds the required fields for creating a new user.
type CreateUserParams struct {
	Username     string `db:"username"      validate:"required,min=3,max=30,username"`
	Email        string `db:"email"         validate:"required,email,max=255"`
	PasswordHash string `db:"password_hash" validate:"required"`
	DisplayName  string `db:"display_name"  validate:"required,min=3,max=50"`
}

// UpdateUserParams holds optional fields for a partial user update.
// Pointer fields: nil means "don't change", non-nil means "set to this value".
type UpdateUserParams struct {
	DisplayName       *string `db:"display_name" validate:"omitempty,min=3,max=50"`
	Bio               *string `db:"bio" validate:"omitempty,max=1000"`
	AvatarURL         *string `db:"avatar_url" validate:"omitempty,url,max=2048"`
	Location          *string `db:"location" validate:"omitempty,max=100"`
	ProfileVisibility *string `db:"profile_visibility" validate:"omitempty,oneof=public private unlisted"`
	EmbedEnabled      *bool   `db:"embed_enabled" validate:"omitempty"`
}

// ChangePasswordParams holds the required fields for updating a password.
type ChangePasswordParams struct {
	Password string `db:"password" validate:"required,min=8,max=72"`
}

// UpdateRoleParams holds the required fields for updating a role.
type UpdateRoleParams struct {
	Role string `db:"role" validate:"required,oneof=user moderator admin superadmin"`
}

// UpdateStatusParams holds the required fields for updating a status.
type UpdateStatusParams struct {
	Status string `db:"status" validate:"required,oneof=active suspended banned deactivated"`
}

// ListUsersParams holds pagination parameters for listing users.
type ListUsersParams struct {
	Limit  int `validate:"required,min=1,max=100"`
	Offset int `validate:"min=0"`
}
