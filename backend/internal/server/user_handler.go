package server

import (
	"net/http"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/middleware"
	"github.com/Yugsolanki/standfor-me/internal/pkg/response"
	"github.com/Yugsolanki/standfor-me/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// --- Request Bodies

type updateUserRequest struct {
	DisplayName       *string `json:"display_name" validate:"omitempty,min=3,max=50"`
	Bio               *string `json:"bio" validate:"omitempty,max=100"`
	Location          *string `json:"location" validate:"omitempty,max=100"`
	AvatarURL         *string `json:"avatar_url" validate:"omitempty,url,max=2048"`
	ProfileVisibility *string `json:"profile_visibility" validate:"omitempty,oneof=public private unlisted"`
	EmbedEnabled      *bool   `json:"embed_enabled"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=72"`
}

type updateRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=user moderator admin superadmin"`
}

type updateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active suspended banned deactivated"`
}

// --- Response Shapes ---

type publicUserResponse struct {
	ID                string  `json:"id"`
	Username          string  `json:"username"`
	DisplayName       string  `json:"display_name"`
	Bio               *string `json:"bio,omitempty"`
	AvatarURL         *string `json:"avatar_url,omitempty"`
	Location          *string `json:"location,omitempty"`
	ProfileVisibility string  `json:"profile_visibility"`
}

type adminUserResponse struct {
	publicUserResponse
	Email         string `json:"email"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	EmailVerified bool   `json:"email_verified"`
}

func toPublishUserResponse(u *domain.User) publicUserResponse {
	return publicUserResponse{
		ID:                u.ID.String(),
		Username:          u.Username,
		DisplayName:       u.DisplayName,
		Bio:               u.Bio,
		AvatarURL:         u.AvatarURL,
		Location:          u.Location,
		ProfileVisibility: u.ProfileVisibility,
	}
}

func toAdminUserResponse(u *domain.User) adminUserResponse {
	return adminUserResponse{
		publicUserResponse: toPublishUserResponse(u),
		Email:              u.Email,
		Role:               u.Role,
		Status:             u.Status,
		EmailVerified:      u.EmailVerifiedAt != nil,
	}
}

// --- Public Helpers

// getUserHandler handles GET /api/v1/users/{username}.
// Public — returns the public profile of a user by username.
// Private profiles are visible only to the owner or admin+.
func (s *Server) getUserHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.getUserHandler"

	username := chi.URLParam(r, "username")
	if username == "" {
		return domain.NewBadRequestError(op, "username is required")
	}

	userSvc := s.services.User

	user, err := userSvc.GetByUsername(r.Context(), username)
	if err != nil {
		return err
	}

	if user.ProfileVisibility == domain.ProfileVisibilityPrivate {
		claims := middleware.ClaimsFromContext(r.Context())
		isOwner := claims != nil && claims.UserID == user.ID
		isAdmin := claims != nil &&
			(claims.Role == domain.RoleAdmin || claims.Role == domain.RoleSuperAdmin)

		if !isOwner && !isAdmin {
			return domain.NewForbiddenError(op, "this profile is private")
		}
	}

	response.JSON(w, r, http.StatusOK, toPublishUserResponse(user))

	return nil
}

// --- Authenticated Self-Service Handlers ---

// updateMeHandler handles PATCH /api/v1/users/me.
// Authenticated — allows a user to update their own profile.
func (s *Server) updateMeHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.updateMeHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	body, err := Decode[updateUserRequest](r)
	if err != nil {
		return err
	}

	if errs := s.validator.Validate(body); errs != nil {
		response.JSONValidationError(w, r, errs)
	}

	userSvc := s.services.User

	user, err := userSvc.Update(r.Context(), claims.UserID, domain.UpdateUserParams{
		DisplayName:       body.DisplayName,
		Bio:               body.Bio,
		Location:          body.Location,
		AvatarURL:         body.AvatarURL,
		ProfileVisibility: body.ProfileVisibility,
		EmbedEnabled:      body.EmbedEnabled,
	})
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusOK, toPublishUserResponse(user))

	return nil
}

// changePasswordHandler handles POST /api/v1/users/me/password.
// Authenticated — changes the authenticated user's password.
func (s *Server) changePasswordHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.changePasswordHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	body, err := Decode[changePasswordRequest](r)
	if err != nil {
		return err
	}

	if errs := s.validator.Validate(body); errs != nil {
		response.JSONValidationError(w, r, errs)
	}

	userSvc := s.services.User

	if err = userSvc.ChangePassword(r.Context(), claims.UserID, service.ChangePasswordInput{
		CurrentPassword: body.CurrentPassword,
		NewPassword:     body.NewPassword,
	}); err != nil {
		return err
	}

	response.JSONMessage(w, r, http.StatusOK, "password changed successfully")

	return nil
}

// deleteMeHandler handles DELETE /api/v1/users/me.
// Authenticated — soft-deletes the authenticated user's own account.
func (s *Server) deleteMeHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.deleteMeHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	userSvc := s.services.User

	if err := userSvc.SoftDelete(r.Context(), claims.UserID); err != nil {
		return err
	}

	response.JSONMessage(w, r, http.StatusOK, "account deleted")

	return nil
}

// --- Admin Handler

// TODO: adminListUserHandler

// adminGetUserHandler handles GET /api/v1/admin/users/{id}.
// Moderator+ — returns the full admin view of any user by UUID.
func (s *Server) adminGetUserHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.adminGetUserHandler"

	id, err := parseUUIDParam(r, "id")
	if err != nil {
		return err
	}

	userSvc := s.services.User

	user, err := userSvc.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusOK, toAdminUserResponse(user))
	return nil
}

// adminUpdateRoleHandler handles PATCH /api/v1/admin/users/{id}/role.
// Admin+ — changes a user's role.
// Only superadmins may promote to admin or superadmin.
func (s *Server) adminUpdateRoleHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.adminUpdateRoleHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	id, err := parseUUIDParam(r, "id")
	if err != nil {
		return err
	}

	body, err := Decode[updateRoleRequest](r)
	if err != nil {
		return err
	}

	if errs := s.validator.Validate(body); errs != nil {
		response.JSONValidationError(w, r, errs)
	}

	// Privilege escalation guard: only superadmins may assign
	// admin-level or superadmin-level roles.
	if (body.Role == domain.RoleAdmin || body.Role == domain.RoleSuperAdmin) &&
		claims.Role != domain.RoleSuperAdmin {
		return domain.NewForbiddenError(
			op,
			"only superadmins may assign admin or superadmin roles",
		)
	}

	userSvc := s.services.User

	user, err := userSvc.UpdateRole(r.Context(), id, body.Role)
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusOK, toAdminUserResponse(user))

	return nil
}

// adminUpdateStatusHandler handles PATCH /api/v1/admin/users/{id}/status.
// Moderator+ — changes a user's account status.
func (s *Server) adminUpdateStatusHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.adminUpdateStatusHandler"

	id, err := parseUUIDParam(r, "id")
	if err != nil {
		return err
	}

	body, err := Decode[updateStatusRequest](r)
	if err != nil {
		return err
	}

	if errs := s.validator.Validate(body); errs != nil {
		response.JSONValidationError(w, r, errs)
	}

	userSvc := s.services.User

	user, err := userSvc.UpdateStatus(r.Context(), id, body.Status)
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusOK, toAdminUserResponse(user))
	return nil
}

// adminDeleteUserHandler handles DELETE /api/v1/admin/users/{id}.
// Admin+ — soft-deletes any user account.
func (s *Server) adminDeleteUserHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.adminDeleteUserHandler"

	id, err := parseUUIDParam(r, "id")
	if err != nil {
		return err
	}

	userSvc := s.services.User

	if err = userSvc.SoftDelete(r.Context(), id); err != nil {
		return err
	}

	response.JSONMessage(w, r, http.StatusOK, "user deleted")
	return nil
}

// --- Helpers ---

// parseUUIDParam extracts and validates a UUID path parameter.
func parseUUIDParam(r *http.Request, param string) (uuid.UUID, error) {
	const op = "server.parseUUIDParam"

	raw := chi.URLParam(r, param)
	if raw == "" {
		return uuid.Nil, domain.NewBadRequestError(
			op,
			param+" is required",
		)
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, domain.NewBadRequestError(
			op,
			param+" must be a valid UUID",
		)
	}

	return id, nil
}
