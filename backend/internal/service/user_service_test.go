package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/pkg/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type userRepoMock struct {
	mock.Mock
}

func (m *userRepoMock) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *userRepoMock) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *userRepoMock) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *userRepoMock) UsernameExists(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

func (m *userRepoMock) EmailExists(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *userRepoMock) Update(ctx context.Context, id uuid.UUID, params domain.UpdateUserParams) (*domain.User, error) {
	args := m.Called(ctx, id, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *userRepoMock) ChangeUsername(ctx context.Context, id uuid.UUID, params domain.ChangeUsernameParams) (*domain.User, error) {
	args := m.Called(ctx, id, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *userRepoMock) ChangePassword(ctx context.Context, id uuid.UUID, params domain.ChangePasswordParams) error {
	args := m.Called(ctx, id, params)
	return args.Error(0)
}

func (m *userRepoMock) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *userRepoMock) UpdateRole(ctx context.Context, id uuid.UUID, params domain.UpdateRoleParams) (*domain.User, error) {
	args := m.Called(ctx, id, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *userRepoMock) UpdateStatus(ctx context.Context, id uuid.UUID, params domain.UpdateStatusParams) (*domain.User, error) {
	args := m.Called(ctx, id, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *userRepoMock) Create(ctx context.Context, params domain.CreateUserParams) (*domain.User, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *userRepoMock) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *userRepoMock) List(ctx context.Context, params domain.ListUsersParams) ([]domain.User, int, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]domain.User), args.Int(1), args.Error(2)
}

type userSvcRefreshTokenMock struct {
	mock.Mock
}

func (m *userSvcRefreshTokenMock) Create(ctx context.Context, params domain.CreateRefreshTokenParams) (*domain.RefreshToken, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RefreshToken), args.Error(1)
}

func (m *userSvcRefreshTokenMock) FindByTokenHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RefreshToken), args.Error(1)
}

func (m *userSvcRefreshTokenMock) Revoke(ctx context.Context, hash string) error {
	args := m.Called(ctx, hash)
	return args.Error(0)
}

func (m *userSvcRefreshTokenMock) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func newTestUser(sts string) *domain.User {
	id := uuid.New()
	hash, _ := crypto.HashPassword("oldpassword123")
	return &domain.User{
		ID:           id,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: &hash,
		DisplayName:  "Test User",
		Role:         domain.RoleUser,
		Status:       sts,
	}
}

func TestUserService_GetByID_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	testUser := newTestUser(domain.StatusActive)
	userRepo.On("FindByID", ctx, testUser.ID).Return(testUser, nil)

	result, err := svc.GetByID(ctx, testUser.ID)

	require.NoError(t, err)
	assert.Equal(t, testUser.ID, result.ID)
	assert.Equal(t, testUser.Username, result.Username)
	userRepo.AssertExpectations(t)
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	id := uuid.New()
	userRepo.On("FindByID", ctx, id).Return(nil, domain.ErrNotFound)

	result, err := svc.GetByID(ctx, id)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	userRepo.AssertExpectations(t)
}

func TestUserService_GetByUsername_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	testUser := newTestUser(domain.StatusActive)
	userRepo.On("FindByUsername", ctx, "testuser").Return(testUser, nil)

	result, err := svc.GetByUsername(ctx, "testuser")

	require.NoError(t, err)
	assert.Equal(t, testUser.Username, result.Username)
	assert.Equal(t, testUser.Email, result.Email)
	userRepo.AssertExpectations(t)
}

func TestUserService_GetByUsername_NotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userRepo.On("FindByUsername", ctx, "nonexistent").Return(nil, domain.ErrNotFound)

	result, err := svc.GetByUsername(ctx, "nonexistent")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	userRepo.AssertExpectations(t)
}

func TestUserService_GetByEmail_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	testUser := newTestUser(domain.StatusActive)
	userRepo.On("FindByEmail", ctx, testUser.Email).Return(testUser, nil)

	result, err := svc.GetByEmail(ctx, testUser.Email)

	require.NoError(t, err)
	assert.Equal(t, testUser.Email, result.Email)
	assert.Equal(t, testUser.DisplayName, result.DisplayName)
	userRepo.AssertExpectations(t)
}

func TestUserService_GetByEmail_NotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userRepo.On("FindByEmail", ctx, "notfound@example.com").Return(nil, domain.ErrNotFound)

	result, err := svc.GetByEmail(ctx, "notfound@example.com")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	userRepo.AssertExpectations(t)
}

func TestUserService_Update_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	newBio := "Updated bio"
	params := domain.UpdateUserParams{Bio: &newBio}
	updatedUser := &domain.User{
		ID:          userID,
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Bio:         &newBio,
		Role:        domain.RoleUser,
		Status:      domain.StatusActive,
	}

	userRepo.On("Update", ctx, userID, params).Return(updatedUser, nil)

	result, err := svc.Update(ctx, userID, params)

	require.NoError(t, err)
	assert.Equal(t, newBio, *result.Bio)
	userRepo.AssertExpectations(t)
}

func TestUserService_Update_NotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	params := domain.UpdateUserParams{DisplayName: ptr("New Name")}
	userRepo.On("Update", ctx, userID, params).Return(nil, domain.ErrNotFound)

	result, err := svc.Update(ctx, userID, params)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	userRepo.AssertExpectations(t)
}

func TestUserService_Update_Partial(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	params := domain.UpdateUserParams{DisplayName: ptr("Only DisplayName")}
	updatedUser := &domain.User{
		ID:          userID,
		Username:    "testuser",
		DisplayName: "Only DisplayName",
		Role:        domain.RoleUser,
		Status:      domain.StatusActive,
	}

	userRepo.On("Update", ctx, userID, params).Return(updatedUser, nil)

	result, err := svc.Update(ctx, userID, params)

	require.NoError(t, err)
	assert.Equal(t, "Only DisplayName", result.DisplayName)
	userRepo.AssertExpectations(t)
}

func TestUserService_ChangeUsername_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	newUsername := "newusername"
	params := domain.ChangeUsernameParams{Username: newUsername}
	updatedUser := &domain.User{
		ID:       userID,
		Username: newUsername,
		Role:     domain.RoleUser,
		Status:   domain.StatusActive,
	}

	userRepo.On("UsernameExists", ctx, newUsername).Return(false, nil)
	userRepo.On("ChangeUsername", ctx, userID, params).Return(updatedUser, nil)

	result, err := svc.ChangeUsername(ctx, userID, newUsername)

	require.NoError(t, err)
	assert.Equal(t, newUsername, result.Username)
	userRepo.AssertExpectations(t)
}

func TestUserService_ChangeUsername_Duplicate(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	takenUsername := "takenuser"

	userRepo.On("UsernameExists", ctx, takenUsername).Return(true, nil)

	result, err := svc.ChangeUsername(ctx, userID, takenUsername)

	assert.Nil(t, result)
	assert.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok, "expected AppError")
	assert.Equal(t, domain.ErrConflict, appErr.Err)
	userRepo.AssertExpectations(t)
}

func TestUserService_ChangeUsername_EmptySkipsCheck(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	params := domain.ChangeUsernameParams{Username: ""}
	updatedUser := &domain.User{
		ID:       userID,
		Username: "",
		Role:     domain.RoleUser,
		Status:   domain.StatusActive,
	}

	userRepo.On("ChangeUsername", ctx, userID, params).Return(updatedUser, nil)

	result, err := svc.ChangeUsername(ctx, userID, "")

	require.NoError(t, err)
	userRepo.AssertNotCalled(t, "UsernameExists")
	assert.Equal(t, "", result.Username)
}

func TestUserService_ChangeUsername_RepoError(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	params := domain.ChangeUsernameParams{Username: "newuser"}

	userRepo.On("UsernameExists", ctx, "newuser").Return(false, nil)
	userRepo.On("ChangeUsername", ctx, userID, params).Return(nil, domain.ErrInternal)

	result, err := svc.ChangeUsername(ctx, userID, "newuser")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrInternal)
	userRepo.AssertExpectations(t)
}

func TestUserService_ChangePassword_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	oldHash, _ := crypto.HashPassword("oldpassword123")
	testUser := &domain.User{
		ID:           userID,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: &oldHash,
		DisplayName:  "Test User",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
	}

	input := ChangePasswordInput{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword456",
	}

	userRepo.On("FindByID", ctx, userID).Return(testUser, nil)
	userRepo.On("ChangePassword", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	refreshTokens.On("RevokeAllForUser", ctx, userID).Return(nil)

	err := svc.ChangePassword(ctx, userID, input)

	require.NoError(t, err)
	userRepo.AssertExpectations(t)
	refreshTokens.AssertExpectations(t)
}

func TestUserService_ChangePassword_InvalidCurrent(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	wrongHash, _ := crypto.HashPassword("someotherpassword")
	testUser := &domain.User{
		ID:           userID,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: &wrongHash,
		DisplayName:  "Test User",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
	}

	input := ChangePasswordInput{
		CurrentPassword: "wrongpassword",
		NewPassword:     "newpassword456",
	}

	userRepo.On("FindByID", ctx, userID).Return(testUser, nil)

	err := svc.ChangePassword(ctx, userID, input)

	assert.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok, "expected AppError")
	assert.Equal(t, domain.ErrInvalidCredentials, appErr.Err)
	userRepo.AssertExpectations(t)
}

func TestUserService_ChangePassword_NilPasswordHash(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	testUser := &domain.User{
		ID:           userID,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: nil,
		DisplayName:  "Test User",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
	}

	input := ChangePasswordInput{
		CurrentPassword: "anypassword",
		NewPassword:     "newpassword456",
	}

	userRepo.On("FindByID", ctx, userID).Return(testUser, nil)

	err := svc.ChangePassword(ctx, userID, input)

	assert.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok, "expected AppError")
	assert.Equal(t, domain.ErrInvalidCredentials, appErr.Err)
}

func TestUserService_ChangePassword_HashError(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	oldHash, _ := crypto.HashPassword("oldpassword123")
	testUser := &domain.User{
		ID:           userID,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: &oldHash,
		DisplayName:  "Test User",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
	}

	badInput := ChangePasswordInput{
		CurrentPassword: "oldpassword123",
		NewPassword:     strings.Repeat("a", 100),
	}

	userRepo.On("FindByID", ctx, userID).Return(testUser, nil)

	err := svc.ChangePassword(ctx, userID, badInput)

	assert.Error(t, err)
	appErr, ok := err.(*domain.AppError)
	require.True(t, ok, "expected AppError")
	assert.Equal(t, domain.ErrInternal, appErr.Err)
}

func TestUserService_ChangePassword_UserNotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	input := ChangePasswordInput{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword456",
	}

	userRepo.On("FindByID", ctx, userID).Return(nil, domain.ErrNotFound)

	err := svc.ChangePassword(ctx, userID, input)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserService_ChangePassword_RefreshRevokeFails(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	oldHash, _ := crypto.HashPassword("oldpassword123")
	testUser := &domain.User{
		ID:           userID,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: &oldHash,
		DisplayName:  "Test User",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
	}

	input := ChangePasswordInput{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword456",
	}

	userRepo.On("FindByID", mock.Anything, mock.Anything).Return(testUser, nil)
	userRepo.On("ChangePassword", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	refreshTokens.On("RevokeAllForUser", mock.Anything, mock.Anything).Return(errors.New("redis error"))

	err := svc.ChangePassword(ctx, userID, input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis error")
}

func TestUserService_UpdateRole_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	params := domain.UpdateRoleParams{Role: domain.RoleAdmin}
	updatedUser := &domain.User{
		ID:     userID,
		Role:   domain.RoleAdmin,
		Status: domain.StatusActive,
	}

	userRepo.On("UpdateRole", ctx, userID, params).Return(updatedUser, nil)

	result, err := svc.UpdateRole(ctx, userID, domain.RoleAdmin)

	require.NoError(t, err)
	assert.Equal(t, domain.RoleAdmin, result.Role)
	userRepo.AssertExpectations(t)
}

func TestUserService_UpdateRole_NotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	params := domain.UpdateRoleParams{Role: domain.RoleAdmin}

	userRepo.On("UpdateRole", ctx, userID, params).Return(nil, domain.ErrNotFound)

	result, err := svc.UpdateRole(ctx, userID, domain.RoleAdmin)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserService_UpdateRole_AllRoles(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	roles := []string{domain.RoleUser, domain.RoleModerator, domain.RoleAdmin, domain.RoleSuperAdmin}

	for _, role := range roles {
		params := domain.UpdateRoleParams{Role: role}
		updatedUser := &domain.User{ID: userID, Role: role}
		userRepo.On("UpdateRole", ctx, userID, params).Return(updatedUser, nil)

		result, err := svc.UpdateRole(ctx, userID, role)

		require.NoError(t, err, "role %s should succeed", role)
		assert.Equal(t, role, result.Role)
	}
}

func TestUserService_UpdateStatus_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	params := domain.UpdateStatusParams{Status: domain.StatusBanned}
	updatedUser := &domain.User{
		ID:     userID,
		Role:   domain.RoleUser,
		Status: domain.StatusBanned,
	}

	userRepo.On("UpdateStatus", ctx, userID, params).Return(updatedUser, nil)

	result, err := svc.UpdateStatus(ctx, userID, domain.StatusBanned)

	require.NoError(t, err)
	assert.Equal(t, domain.StatusBanned, result.Status)
	userRepo.AssertExpectations(t)
}

func TestUserService_UpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	params := domain.UpdateStatusParams{Status: domain.StatusSuspended}

	userRepo.On("UpdateStatus", ctx, userID, params).Return(nil, domain.ErrNotFound)

	result, err := svc.UpdateStatus(ctx, userID, domain.StatusSuspended)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserService_UpdateStatus_AllStatuses(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	statuses := []string{domain.StatusActive, domain.StatusSuspended, domain.StatusBanned, domain.StatusDeactivated}

	for _, status := range statuses {
		params := domain.UpdateStatusParams{Status: status}
		updatedUser := &domain.User{ID: userID, Status: status}
		userRepo.On("UpdateStatus", ctx, userID, params).Return(updatedUser, nil)

		result, err := svc.UpdateStatus(ctx, userID, status)

		require.NoError(t, err, "status %s should succeed", status)
		assert.Equal(t, status, result.Status)
	}
}

func TestUserService_SoftDelete_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()

	userRepo.On("SoftDelete", ctx, userID).Return(nil)
	refreshTokens.On("RevokeAllForUser", ctx, userID).Return(nil)

	err := svc.SoftDelete(ctx, userID)

	require.NoError(t, err)
	userRepo.AssertExpectations(t)
	refreshTokens.AssertExpectations(t)
}

func TestUserService_SoftDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()

	userRepo.On("SoftDelete", ctx, userID).Return(domain.ErrNotFound)

	err := svc.SoftDelete(ctx, userID)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserService_SoftDelete_RefreshTokensRevoked(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()

	userRepo.On("SoftDelete", ctx, userID).Return(nil)
	refreshTokens.On("RevokeAllForUser", ctx, userID).Return(errors.New("redis error"))

	err := svc.SoftDelete(ctx, userID)

	require.NoError(t, err)
}

func TestUserService_ChangePassword_EmptyNewPassword(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	userID := uuid.New()
	oldHash, _ := crypto.HashPassword("oldpassword123")
	testUser := &domain.User{
		ID:           userID,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: &oldHash,
		DisplayName:  "Test User",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
	}

	input := ChangePasswordInput{
		CurrentPassword: "oldpassword123",
		NewPassword:     "",
	}

	userRepo.On("FindByID", ctx, userID).Return(testUser, nil)
	userRepo.On("ChangePassword", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	refreshTokens.On("RevokeAllForUser", ctx, userID).Return(nil)

	err := svc.ChangePassword(ctx, userID, input)

	require.NoError(t, err)
}

func ptr[T any](v T) *T {
	return &v
}

func TestUserService_List_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	params := domain.ListUsersParams{Limit: 10, Offset: 0}
	users := []domain.User{
		{ID: uuid.New(), Username: "user1", DisplayName: "User One"},
		{ID: uuid.New(), Username: "user2", DisplayName: "User Two"},
	}
	total := 2

	userRepo.On("List", ctx, params).Return(users, total, nil)

	result, gotTotal, err := svc.List(ctx, params)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, total, gotTotal)
	assert.Equal(t, "user1", result[0].Username)
	userRepo.AssertExpectations(t)
}

func TestUserService_List_Empty(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	params := domain.ListUsersParams{Limit: 10, Offset: 0}
	users := []domain.User{}
	total := 0

	userRepo.On("List", ctx, params).Return(users, total, nil)

	result, gotTotal, err := svc.List(ctx, params)

	require.NoError(t, err)
	assert.Len(t, result, 0)
	assert.Equal(t, 0, gotTotal)
}

func TestUserService_List_WithPagination(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	params := domain.ListUsersParams{Limit: 5, Offset: 10}
	users := []domain.User{
		{ID: uuid.New(), Username: "user11"},
		{ID: uuid.New(), Username: "user12"},
	}
	total := 100

	userRepo.On("List", ctx, params).Return(users, total, nil)

	result, gotTotal, err := svc.List(ctx, params)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 100, gotTotal)
}

func TestUserService_List_RepoError(t *testing.T) {
	ctx := context.Background()
	userRepo := new(userRepoMock)
	refreshTokens := new(userSvcRefreshTokenMock)
	svc := NewUserService(userRepo, refreshTokens)

	params := domain.ListUsersParams{Limit: 10, Offset: 0}

	userRepo.On("List", ctx, params).Return(nil, 0, domain.ErrInternal)

	result, gotTotal, err := svc.List(ctx, params)

	assert.Nil(t, result)
	assert.Equal(t, 0, gotTotal)
	assert.ErrorIs(t, err, domain.ErrInternal)
}
