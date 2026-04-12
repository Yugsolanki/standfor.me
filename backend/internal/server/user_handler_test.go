package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	appvalidator "github.com/Yugsolanki/standfor-me/internal/pkg/validator"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contextKey string

const testClaimsKey contextKey = "auth_claims"

func contextWithClaims(ctx context.Context, claims *domain.AccessTokenClaims) context.Context {
	return context.WithValue(ctx, testClaimsKey, claims)
}

func TestGetUserHandler_MissingUsername(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodGet, "/api/v1/users/", nil)
	w := httptest.NewRecorder()

	err := s.getUserHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrBadRequest, appErr.Err)
}

func TestUpdateMeHandler_NoAuth(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", nil)
	r = r.WithContext(context.Background())
	w := httptest.NewRecorder()

	err := s.updateMeHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrUnauthorized, appErr.Err)
}

func TestChangePasswordHandler_NoAuth(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/password", nil)
	r = r.WithContext(context.Background())
	w := httptest.NewRecorder()

	err := s.changePasswordHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrUnauthorized, appErr.Err)
}

func TestDeleteMeHandler_NoAuth(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
	r = r.WithContext(context.Background())
	w := httptest.NewRecorder()

	err := s.deleteMeHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrUnauthorized, appErr.Err)
}

func TestAdminGetUserHandler_InvalidUUID(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/invalid-uuid", nil)
	w := httptest.NewRecorder()

	err := s.adminGetUserHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrBadRequest, appErr.Err)
}

func TestAdminUpdateRoleHandler_NoAuth(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	userID := uuid.New()
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+userID.String()+"/role", nil)
	r = r.WithContext(context.Background())
	w := httptest.NewRecorder()

	err := s.adminUpdateRoleHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrUnauthorized, appErr.Err)
}

func TestAdminUpdateRoleHandler_InvalidUUID(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/invalid-uuid/role", nil)
	w := httptest.NewRecorder()

	err := s.adminUpdateRoleHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrUnauthorized, appErr.Err)
}

func TestAdminUpdateRoleHandler_NonSuperadminAssigningAdmin(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	userID := uuid.New()
	body := updateRoleRequest{Role: domain.RoleAdmin}
	bodyBytes, _ := json.Marshal(body)

	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+userID.String()+"/role", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.Background())
	w := httptest.NewRecorder()

	err := s.adminUpdateRoleHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrUnauthorized, appErr.Err)
}

func TestAdminUpdateRoleHandler_NonSuperadminAssigningSuperadmin(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	userID := uuid.New()
	body := updateRoleRequest{Role: domain.RoleSuperAdmin}
	bodyBytes, _ := json.Marshal(body)

	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+userID.String()+"/role", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.Background())
	w := httptest.NewRecorder()

	err := s.adminUpdateRoleHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrUnauthorized, appErr.Err)
}

func TestAdminUpdateStatusHandler_InvalidUUID(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/invalid-uuid/status", nil)
	w := httptest.NewRecorder()

	err := s.adminUpdateStatusHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrBadRequest, appErr.Err)
}

func TestAdminDeleteUserHandler_InvalidUUID(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/invalid-uuid", nil)
	w := httptest.NewRecorder()

	err := s.adminDeleteUserHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrBadRequest, appErr.Err)
}

func TestParseUUIDParam_Missing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/", nil)

	id, err := parseUUIDParam(r, "id")

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, id)
}

func TestParseUUIDParam_Invalid(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/not-a-uuid", nil)

	id, err := parseUUIDParam(r, "id")

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, id)
}

func TestToPublicUserResponse(t *testing.T) {
	bio := "Test bio"
	avatar := "https://example.com/avatar.png"
	location := "NYC"

	testUser := &domain.User{
		ID:                uuid.New(),
		Username:          "testuser",
		DisplayName:       "Test User",
		Bio:               &bio,
		AvatarURL:         &avatar,
		Location:          &location,
		ProfileVisibility: domain.ProfileVisibilityPublic,
	}

	resp := toPublishUserResponse(testUser)

	assert.Equal(t, testUser.ID.String(), resp.ID)
	assert.Equal(t, testUser.Username, resp.Username)
	assert.Equal(t, testUser.DisplayName, resp.DisplayName)
	assert.Equal(t, &bio, resp.Bio)
	assert.Equal(t, &avatar, resp.AvatarURL)
	assert.Equal(t, &location, resp.Location)
	assert.Equal(t, domain.ProfileVisibilityPublic, resp.ProfileVisibility)
}

func TestToPublicUserResponse_NilPointers(t *testing.T) {
	testUser := &domain.User{
		ID:                uuid.New(),
		Username:          "testuser",
		DisplayName:       "Test User",
		Bio:               nil,
		AvatarURL:         nil,
		Location:          nil,
		ProfileVisibility: domain.ProfileVisibilityPublic,
	}

	resp := toPublishUserResponse(testUser)

	assert.Equal(t, testUser.Username, resp.Username)
	assert.Nil(t, resp.Bio)
	assert.Nil(t, resp.AvatarURL)
	assert.Nil(t, resp.Location)
}

func TestToAdminUserResponse(t *testing.T) {
	email := "test@example.com"
	testUser := &domain.User{
		ID:                uuid.New(),
		Username:          "testuser",
		Email:             email,
		DisplayName:       "Test User",
		Role:              domain.RoleAdmin,
		Status:            domain.StatusActive,
		ProfileVisibility: domain.ProfileVisibilityPublic,
	}

	resp := toAdminUserResponse(testUser)

	assert.Equal(t, testUser.ID.String(), resp.ID)
	assert.Equal(t, email, resp.Email)
	assert.Equal(t, domain.RoleAdmin, resp.Role)
	assert.Equal(t, domain.StatusActive, resp.Status)
}

func TestToAdminUserResponse_EmailVerified(t *testing.T) {
	now := time.Now()
	testUser := &domain.User{
		ID:                uuid.New(),
		Username:          "testuser",
		Email:             "test@example.com",
		EmailVerifiedAt:   &now,
		DisplayName:       "Test User",
		Role:              domain.RoleUser,
		Status:            domain.StatusActive,
		ProfileVisibility: domain.ProfileVisibilityPublic,
	}

	resp := toAdminUserResponse(testUser)

	assert.True(t, resp.EmailVerified)
}

func TestToAdminUserResponse_EmailNotVerified(t *testing.T) {
	testUser := &domain.User{
		ID:                uuid.New(),
		Username:          "testuser",
		Email:             "test@example.com",
		EmailVerifiedAt:   nil,
		DisplayName:       "Test User",
		Role:              domain.RoleUser,
		Status:            domain.StatusActive,
		ProfileVisibility: domain.ProfileVisibilityPublic,
	}

	resp := toAdminUserResponse(testUser)

	assert.False(t, resp.EmailVerified)
}

func TestAdminUpdateRoleHandler_InvalidRole(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	userID := uuid.New()
	body := updateRoleRequest{Role: "invalid_role"}
	bodyBytes, _ := json.Marshal(body)

	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+userID.String()+"/role", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")

	claims := &domain.AccessTokenClaims{UserID: uuid.New(), Role: domain.RoleSuperAdmin}
	ctx := contextWithClaims(context.Background(), claims)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	err := s.adminUpdateRoleHandler(w, r)

	assert.NotNil(t, err)
}

func TestAdminUpdateStatusHandler_InvalidStatus(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	userID := uuid.New()
	body := updateStatusRequest{Status: "invalid_status"}
	bodyBytes, _ := json.Marshal(body)

	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+userID.String()+"/status", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	err := s.adminUpdateStatusHandler(w, r)

	assert.NotNil(t, err)
}

func TestUpdateMeHandler_InvalidBody(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", bytes.NewReader([]byte("invalid json")))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.Background())
	w := httptest.NewRecorder()

	err := s.updateMeHandler(w, r)

	require.Error(t, err)
}

func TestChangePasswordHandler_InvalidBody(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/password", bytes.NewReader([]byte("invalid json")))
	r = r.WithContext(context.Background())
	w := httptest.NewRecorder()

	err := s.changePasswordHandler(w, r)

	require.Error(t, err)
}

func TestAdminListUsersHandler_NoAuth(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	r = r.WithContext(context.Background())
	w := httptest.NewRecorder()

	err := s.adminListUsersHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrUnauthorized, appErr.Err)
}

func TestAdminListUsersHandler_NonAdmin(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	r = r.WithContext(context.Background())
	w := httptest.NewRecorder()

	err := s.adminListUsersHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrUnauthorized, appErr.Err)
}

func TestAdminListUsersHandler_ModeratorForbidden(t *testing.T) {
	validator := appvalidator.New()
	s := &Server{
		services:  &Services{User: nil},
		validator: validator,
	}

	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	w := httptest.NewRecorder()

	err := s.adminListUsersHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrUnauthorized, appErr.Err)
}
