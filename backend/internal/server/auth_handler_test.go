package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/pkg/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeJSONBody(body any) *bytes.Reader {
	b, _ := json.Marshal(body)
	return bytes.NewReader(b)
}

func TestRegisterHandler_EmptyBody(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.registerHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
	assert.Contains(t, appErr.Message, "empty")
}

func TestRegisterHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{invalid`)))
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.registerHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
}

func TestRegisterHandler_UnknownField(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"username":"u","email":"a@b.com","password":"pass123","display_name":"U","bad":1}`)))
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.registerHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
}

func TestRegisterHandler_InvalidJSONType(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"username":123}`)))
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.registerHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
}

func TestLoginHandler_EmptyBody(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.loginHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
}

func TestLoginHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{invalid`)))
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.loginHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
}

func TestLoginHandler_UnknownField(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"email":"a@b.com","password":"pass","bad":true}`)))
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.loginHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
}

func TestRefreshHandler_EmptyBody(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.refreshHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
}

func TestRefreshHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{invalid`)))
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.refreshHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
}

func TestRefreshHandler_UnknownField(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"refreshToken":"token","bad":true}`)))
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.refreshHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
}

func TestLogoutHandler_EmptyBody(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.logoutHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
}

func TestLogoutHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{invalid`)))
	r.Header.Set("Content-Type", "application/json")

	s := &Server{}
	err := s.logoutHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrBadRequest))
}

func TestLogoutAllHandler_NoClaims(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)

	s := &Server{}
	err := s.logoutAllHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrUnauthorized))
	assert.Contains(t, appErr.Message, "authentication required")
}

func TestMeHandler_NoClaims(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	s := &Server{}
	err := s.meHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrUnauthorized))
	assert.Contains(t, appErr.Message, "authentication required")
}

func TestToAuthUserResponse(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	h, _ := crypto.HashPassword("test")

	user := &domain.User{
		ID:                uuid.New(),
		Username:          "testuser",
		Email:             "test@example.com",
		DisplayName:       "Test User",
		PasswordHash:      &h,
		Role:              domain.RoleUser,
		Status:            domain.StatusActive,
		ProfileVisibility: domain.ProfileVisibilityPublic,
		EmbedEnabled:      true,
		AvatarURL:         stringPtr("https://example.com/avatar.png"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	resp := toAuthUserResponse(user)

	assert.Equal(t, user.ID.String(), resp.ID)
	assert.Equal(t, user.Username, resp.Username)
	assert.Equal(t, user.Email, resp.Email)
	assert.Equal(t, user.DisplayName, resp.DisplayName)
	assert.Equal(t, user.Role, resp.Role)
	assert.Equal(t, *user.AvatarURL, *resp.AvatarURL)
}

func TestToAuthUserResponse_NilAvatar(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	h, _ := crypto.HashPassword("test")

	user := &domain.User{
		ID:                uuid.New(),
		Username:          "testuser",
		Email:             "test@example.com",
		DisplayName:       "Test User",
		PasswordHash:      &h,
		Role:              domain.RoleUser,
		Status:            domain.StatusActive,
		ProfileVisibility: domain.ProfileVisibilityPublic,
		EmbedEnabled:      true,
		AvatarURL:         nil,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	resp := toAuthUserResponse(user)

	assert.Nil(t, resp.AvatarURL)
}

func stringPtr(s string) *string {
	return &s
}

func TestRegisterRequest_ValidationTags(t *testing.T) {
	t.Parallel()
	req := registerRequest{}

	tags := getStructTags(req)

	assert.Contains(t, tags["Username"], "required")
	assert.Contains(t, tags["Username"], "username")
	assert.Contains(t, tags["Username"], "min=3")
	assert.Contains(t, tags["Username"], "max=30")

	assert.Contains(t, tags["Email"], "required")
	assert.Contains(t, tags["Email"], "email")
	assert.Contains(t, tags["Email"], "max=255")

	assert.Contains(t, tags["Password"], "required")
	assert.Contains(t, tags["Password"], "min=8")
	assert.Contains(t, tags["Password"], "max=72")

	assert.Contains(t, tags["DisplayName"], "required")
	assert.Contains(t, tags["DisplayName"], "min=3")
	assert.Contains(t, tags["DisplayName"], "max=50")
}

func TestLoginRequest_ValidationTags(t *testing.T) {
	t.Parallel()
	req := loginRequest{}

	tags := getStructTags(req)

	assert.Contains(t, tags["Email"], "required")
	assert.Contains(t, tags["Email"], "email")
	assert.Contains(t, tags["Email"], "max=255")

	assert.Contains(t, tags["Password"], "required")
}

func TestRefreshRequest_ValidationTags(t *testing.T) {
	t.Parallel()
	req := refreshRequest{}

	tags := getStructTags(req)

	assert.Contains(t, tags["RefreshToken"], "required")
}

func TestLogoutRequest_ValidationTags(t *testing.T) {
	t.Parallel()
	req := logoutRequest{}

	tags := getStructTags(req)

	assert.Contains(t, tags["RefreshToken"], "required")
}

func getStructTags(v any) map[string]string {
	result := make(map[string]string)
	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		result[f.Name] = string(f.Tag)
	}
	return result
}

func TestAuthResponse_Fields(t *testing.T) {
	t.Parallel()
	h, _ := crypto.HashPassword("test")
	user := &domain.User{
		ID:           uuid.New(),
		Username:     "testuser",
		Email:        "test@example.com",
		DisplayName:  "Test User",
		Role:         domain.RoleUser,
		PasswordHash: &h,
		Status:       domain.StatusActive,
	}

	resp := authResponse{
		User:         toAuthUserResponse(user),
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}

	assert.Equal(t, "access-token", resp.AccessToken)
	assert.Equal(t, "refresh-token", resp.RefreshToken)
	assert.Equal(t, user.ID.String(), resp.User.ID)
}

func TestAuthUserResponse_Fields(t *testing.T) {
	t.Parallel()
	avatarURL := "https://example.com/avatar.png"

	resp := authUserResponse{
		ID:          uuid.New().String(),
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Role:        domain.RoleUser,
		AvatarURL:   &avatarURL,
	}

	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "testuser", resp.Username)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "Test User", resp.DisplayName)
	assert.Equal(t, domain.RoleUser, resp.Role)
	assert.NotNil(t, resp.AvatarURL)
}

func TestContextCanceled_Register(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := &Server{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", makeJSONBody(registerRequest{
		Username:    "user",
		Email:       "a@b.com",
		Password:    "password123",
		DisplayName: "User",
	})).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	err := s.registerHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrCanceled))
}

func TestContextCanceled_Login(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := &Server{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", makeJSONBody(loginRequest{
		Email:    "a@b.com",
		Password: "password123",
	})).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	err := s.loginHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrCanceled))
}

func TestContextCanceled_Refresh(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := &Server{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", makeJSONBody(refreshRequest{
		RefreshToken: "token",
	})).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	err := s.refreshHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrCanceled))
}

func TestContextCanceled_Logout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := &Server{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", makeJSONBody(logoutRequest{
		RefreshToken: "token",
	})).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")

	err := s.logoutHandler(w, r)

	require.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok)
	assert.True(t, errors.Is(appErr, domain.ErrCanceled))
}
