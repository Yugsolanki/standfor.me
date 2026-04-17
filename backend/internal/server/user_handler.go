package server

import (
	"net/http"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/middleware"
	"github.com/Yugsolanki/standfor-me/internal/pkg/pagination"
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
// Public — anyone can view a user profile, but private profiles are protected.
//
//	@Summary		Get user by username
//	@Description	Get user by username
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			username	path		string	true	"Username"
//	@Success		200			{object}	publicUserResponse
//	@Failure		400			{object}	response.ErrorResponse
//	@Failure		404			{object}	response.ErrorResponse
//	@Failure		500			{object}	response.ErrorResponse
//	@Router			/api/v1/users/{username} [get]
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

	// Check if the user is private or suspended or banned or deactivated
	// Only owner or admin can view private profile or unlisted profile
	// Only owner or admin can view suspended or banned or deactivated profile
	if user.ProfileVisibility != domain.ProfileVisibilityPublic && user.Status != domain.StatusActive {
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
//
//	@Summary		Update current user
//	@Description	Updates the authenticated user's profile settings
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		updateUserRequest	true	"Fields to update"
//	@Success		200		{object}	publicUserResponse
//	@Failure		401		{object}	response.ErrorResponse
//	@Failure		422		{object}	response.ErrorResponse
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/api/v1/users/me [patch]
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
//
//	@Summary		Change password
//	@Description	Changes the authenticated user's password
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		changePasswordRequest	true	"Current and new password"
//	@Success		200		{object}	response.SuccessResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		401		{object}	response.ErrorResponse
//	@Failure		422		{object}	response.ErrorResponse
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/api/v1/users/me/password [post]
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
//
//	@Summary		Delete current user
//	@Description	Soft-deletes the authenticated user's account
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.SuccessResponse
//	@Failure		401	{object}	response.ErrorResponse
//	@Failure		500	{object}	response.ErrorResponse
//	@Router			/api/v1/users/me [delete]
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

// adminListUserHandler handles GET /api/v1/admin/users.
//
//	@Summary		List all users (admin)
//	@Description	Returns a paginated list of all users with admin details
//	@Tags			admin-users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number (default 1)"
//	@Param			per_page	query		int	false	"Items per page (default 20, max 100)"
//	@Success		200			{object}	pagination.Response{data=[]adminUserResponse}
//	@Failure		401			{object}	response.ErrorResponse
//	@Failure		403			{object}	response.ErrorResponse
//	@Failure		500			{object}	response.ErrorResponse
//	@Router			/api/v1/admin/users [get]
func (s *Server) adminListUsersHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.adminListUsersHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	if claims.Role != domain.RoleAdmin && claims.Role != domain.RoleSuperAdmin {
		return domain.NewForbiddenError(op, "only admins and superadmins can list users")
	}

	p := pagination.FromRequest(r)

	userSvc := s.services.User

	users, total, err := userSvc.List(r.Context(), domain.ListUsersParams{
		Limit:  p.Limit(),
		Offset: p.Offset(),
	})
	if err != nil {
		return err
	}

	items := make([]adminUserResponse, len(users))
	for i := range users {
		items[i] = toAdminUserResponse(&users[i])
	}

	paginatedResponse := pagination.NewResponse(items, p, int64(total))

	response.JSON(w, r, http.StatusOK, paginatedResponse)

	return nil
}

// adminGetUserHandler handles GET /api/v1/admin/users/{id}.
// Moderator+ — returns the full admin view of any user by UUID.
//
//	@Summary		Get user by ID (admin)
//	@Description	Returns full admin details for a user by UUID
//	@Tags			admin-users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"User UUID"
//	@Success		200	{object}	adminUserResponse
//	@Failure		400	{object}	response.ErrorResponse
//	@Failure		401	{object}	response.ErrorResponse
//	@Failure		403	{object}	response.ErrorResponse
//	@Failure		404	{object}	response.ErrorResponse
//	@Failure		500	{object}	response.ErrorResponse
//	@Router			/api/v1/admin/users/{id} [get]
func (s *Server) adminGetUserHandler(w http.ResponseWriter, r *http.Request) error {
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
//
//	@Summary		Update user role (admin)
//	@Description	Changes a user's role. Only superadmins can assign admin/superadmin roles
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"User UUID"
//	@Param			request	body		updateRoleRequest	true	"New role"
//	@Success		200		{object}	adminUserResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		401		{object}	response.ErrorResponse
//	@Failure		403		{object}	response.ErrorResponse
//	@Failure		404		{object}	response.ErrorResponse
//	@Failure		422		{object}	response.ErrorResponse
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/api/v1/admin/users/{id}/role [patch]
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
//
//	@Summary		Update user status (admin)
//	@Description	Changes a user's account status (active, suspended, banned, deactivated)
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"User UUID"
//	@Param			request	body		updateStatusRequest	true	"New status"
//	@Success		200		{object}	adminUserResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		401		{object}	response.ErrorResponse
//	@Failure		403		{object}	response.ErrorResponse
//	@Failure		404		{object}	response.ErrorResponse
//	@Failure		422		{object}	response.ErrorResponse
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/api/v1/admin/users/{id}/status [patch]
func (s *Server) adminUpdateStatusHandler(w http.ResponseWriter, r *http.Request) error {
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
//
//	@Summary		Delete user (admin)
//	@Description	Soft-deletes a user account
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"User UUID"
//	@Success		200	{object}	response.SuccessResponse
//	@Failure		400	{object}	response.ErrorResponse
//	@Failure		401	{object}	response.ErrorResponse
//	@Failure		403	{object}	response.ErrorResponse
//	@Failure		404	{object}	response.ErrorResponse
//	@Failure		500	{object}	response.ErrorResponse
//	@Router			/api/v1/admin/users/{id} [delete]
func (s *Server) adminDeleteUserHandler(w http.ResponseWriter, r *http.Request) error {
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
