package server

import (
	"net/http"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/middleware"
	"github.com/Yugsolanki/standfor-me/internal/pkg/response"
	"github.com/Yugsolanki/standfor-me/internal/service"
)

// --- Request Bodies ---

type registerRequest struct {
	Username    string `json:"username"     validate:"required,username,min=3,max=30"`
	Email       string `json:"email"        validate:"required,email,max=255"`
	Password    string `json:"password"     validate:"required,min=8,max=72"`
	DisplayName string `json:"display_name" validate:"required,min=3,max=50"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// --- Request Helpers ---

type authUserResponse struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

type authResponse struct {
	User         authUserResponse `json:"user"`
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
}

func toAuthUserResponse(u *domain.User) authUserResponse {
	return authUserResponse{
		ID:          u.ID.String(),
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		AvatarURL:   u.AvatarURL,
	}
}

// --- Handlers ---

// registerHandler handles POST /api/v1/auth/register.
// Public — creates a new account and returns a token pair.
func (s *Server) registerHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.registerHandler"

	body, err := Decode[registerRequest](r)
	if err != nil {
		return err
	}

	if errs := s.validator.Validate(body); errs != nil {
		return domain.NewValidationError(op, errs)
	}

	authSvc := s.services.Auth

	user, pair, err := authSvc.Register(r.Context(), service.RegisterInput{
		Username:    body.Username,
		Email:       body.Email,
		Password:    body.Password,
		DisplayName: body.DisplayName,
	})
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusCreated, authResponse{
		User:         toAuthUserResponse(user),
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})

	return nil
}

// loginHandler handles POST /api/v1/auth/login.
// Public — validates credentials and returns a token pair.
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.loginHandler"

	body, err := Decode[loginRequest](r)
	if err != nil {
		return err
	}

	if errs := s.validator.Validate(body); errs != nil {
		return domain.NewValidationError(op, errs)
	}

	authSvc := s.services.Auth

	user, pair, err := authSvc.Login(r.Context(), service.LoginInput{
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusOK, authResponse{
		User:         toAuthUserResponse(user),
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})

	return nil
}

// refreshHandler handles POST /api/v1/auth/refresh.
// Public — rotates a refresh token and returns a new token pair.
func (s *Server) refreshHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.refreshHandler"

	body, err := Decode[refreshRequest](r)
	if err != nil {
		return err
	}

	if errs := s.validator.Validate(body); errs != nil {
		return domain.NewValidationError(op, errs)
	}

	authSvc := s.services.Auth

	pair, err := authSvc.RefreshTokens(r.Context(), body.RefreshToken)
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusOK, pair)

	return nil
}

// logoutHandler handles POST /api/v1/auth/logout.
// Authenticated — revokes the presented refresh token.
func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.logoutHandler"

	body, err := Decode[logoutRequest](r)
	if err != nil {
		return err
	}

	if errs := s.validator.Validate(body); errs != nil {
		return domain.NewValidationError(op, errs)
	}

	authSvc := s.services.Auth

	if err = authSvc.Logout(r.Context(), body.RefreshToken); err != nil {
		return err
	}

	response.JSONMessage(w, r, http.StatusOK, "logged out successfully")

	return nil
}

// logoutAllHandler handles POST /api/v1/auth/logout-all.
// Authenticated — revokes all refresh tokens for the caller.
func (s *Server) logoutAllHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.logoutAllHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	authSvc := s.services.Auth

	if err := authSvc.LogoutAll(r.Context(), claims.UserID); err != nil {
		return err
	}

	response.JSONMessage(w, r, http.StatusOK, "all sessions terminated")

	return nil
}

// meHandler handles GET /api/v1/auth/me.
// Authenticated — returns the profile of the authenticated user.
func (s *Server) meHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.meHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	userSvc := s.services.User

	user, err := userSvc.GetByID(r.Context(), claims.UserID)
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusOK, toAuthUserResponse(user))

	return nil
}
