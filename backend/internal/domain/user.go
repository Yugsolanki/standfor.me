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
	Username     string `db:"username"      validate:"required,min=3,max=30,alphanum"`
	Email        string `db:"email"         validate:"required,email,max=255"`
	PasswordHash string `db:"password_hash" validate:"required"`
	DisplayName  string `db:"display_name"  validate:"required,min=3,max=50"`
}

// UpdateUserParams holds optional fields for a partial user update.
// Pointer fields: nil means "don't change", non-nil means "set to this value".
type UpdateUserParams struct {
	DisplayName       *string `db:"display_name"`
	Bio               *string `db:"bio"`
	AvatarURL         *string `db:"avatar_url"`
	Location          *string `db:"location"`
	ProfileVisibility *string `db:"profile_visibility"`
	EmbedEnabled      *bool   `db:"embed_enabled"`
}

// UpdateUsernameParams holds the required fields for updating a username.
type UpdateUsernameParams struct {
	Username string `db:"username" validate:"required,min=3,max=30,alphanum"`
}

// UpdateRoleParams holds the required fields for updating a role.
type UpdateRoleParams struct {
	Role string `db:"role" validate:"required"`
}

// UpdateStatusParams holds the required fields for updating a status.
type UpdateStatusParams struct {
	Status string `db:"status" validate:"required"`
}

// ListUsersParams holds pagination parameters for listing users.
type ListUsersParams struct {
	Limit  int `validate:"required,min=1,max=100"`
	Offset int `validate:"min=0"`
}
