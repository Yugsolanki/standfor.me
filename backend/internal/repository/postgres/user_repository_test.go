package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestUser(t *testing.T, db *sqlx.DB) *domain.User {
	user, err := NewUserRepository(db).Create(context.Background(), domain.CreateUserParams{
		Username:     "testuser_" + uuid.New().String()[:8],
		Email:        "test_" + uuid.New().String()[:8] + "@example.com",
		PasswordHash: "hashedpassword",
		DisplayName:  "Test User",
	})
	require.NoError(t, err)
	return user
}

func cleanupTestUser(t *testing.T, db *sqlx.DB, userID uuid.UUID) {
	_, err := db.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		t.Logf("cleanup failed: %v", err)
	}
}

func TestUserRepository_Create(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	ctx := context.Background()
	username := "createuser_" + uuid.New().String()[:8]
	email := "create_" + uuid.New().String()[:8] + "@example.com"

	user, err := repo.Create(ctx, domain.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: "hashedpassword123",
		DisplayName:  "Create Test User",
	})

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, "Create Test User", user.DisplayName)
	assert.Equal(t, domain.RoleUser, user.Role)
	assert.Equal(t, domain.StatusActive, user.Status)
	assert.NotZero(t, user.CreatedAt)

	defer cleanupTestUser(t, db, user.ID)
}

func TestUserRepository_Create_DuplicateUsername(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	ctx := context.Background()
	username := "duplicate_" + uuid.New().String()[:8]
	email1 := "dup1_" + uuid.New().String()[:8] + "@example.com"
	email2 := "dup2_" + uuid.New().String()[:8] + "@example.com"

	user1, err := repo.Create(ctx, domain.CreateUserParams{
		Username:     username,
		Email:        email1,
		PasswordHash: "hash1",
		DisplayName:  "User One",
	})
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user1.ID)

	_, err = repo.Create(ctx, domain.CreateUserParams{
		Username:     username,
		Email:        email2,
		PasswordHash: "hash2",
		DisplayName:  "User Two",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	ctx := context.Background()
	email := "samemail_" + uuid.New().String()[:8] + "@example.com"
	username1 := "user1_" + uuid.New().String()[:8]
	username2 := "user2_" + uuid.New().String()[:8]

	user1, err := repo.Create(ctx, domain.CreateUserParams{
		Username:     username1,
		Email:        email,
		PasswordHash: "hash1",
		DisplayName:  "User One",
	})
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user1.ID)

	_, err = repo.Create(ctx, domain.CreateUserParams{
		Username:     username2,
		Email:        email,
		PasswordHash: "hash2",
		DisplayName:  "User Two",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestUserRepository_FindByID(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	foundUser, err := repo.FindByID(context.Background(), createdUser.ID)

	require.NoError(t, err)
	require.NotNil(t, foundUser)
	assert.Equal(t, createdUser.ID, foundUser.ID)
	assert.Equal(t, createdUser.Username, foundUser.Username)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_FindByUsername(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	foundUser, err := repo.FindByUsername(context.Background(), createdUser.Username)

	require.NoError(t, err)
	require.NotNil(t, foundUser)
	assert.Equal(t, createdUser.Username, foundUser.Username)
}

func TestUserRepository_FindByUsername_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	_, err := repo.FindByUsername(context.Background(), "nonexistent_user_12345")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	foundUser, err := repo.FindByEmail(context.Background(), createdUser.Email)

	require.NoError(t, err)
	require.NotNil(t, foundUser)
	assert.Equal(t, createdUser.Email, foundUser.Email)
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	_, err := repo.FindByEmail(context.Background(), "nonexistent@example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_Update(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	newDisplayName := "Updated Name"
	newBio := "This is my new bio"
	newLocation := "New York"

	updatedUser, err := repo.Update(context.Background(), createdUser.ID, domain.UpdateUserParams{
		DisplayName: &newDisplayName,
		Bio:         &newBio,
		Location:    &newLocation,
	})

	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, newDisplayName, updatedUser.DisplayName)
	assert.Equal(t, newBio, *updatedUser.Bio)
	assert.Equal(t, newLocation, *updatedUser.Location)
}

func TestUserRepository_Update_PartialUpdate(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	originalDisplayName := createdUser.DisplayName
	newBio := "Partial update bio"

	updatedUser, err := repo.Update(context.Background(), createdUser.ID, domain.UpdateUserParams{
		Bio: &newBio,
	})

	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, originalDisplayName, updatedUser.DisplayName)
	assert.Equal(t, newBio, *updatedUser.Bio)
}

func TestUserRepository_Update_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	newDisplayName := "Updated"
	_, err := repo.Update(context.Background(), uuid.New(), domain.UpdateUserParams{
		DisplayName: &newDisplayName,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_ChangeUsername(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	newUsername := "newusername_" + uuid.New().String()[:8]

	updatedUser, err := repo.ChangeUsername(context.Background(), createdUser.ID, domain.ChangeUsernameParams{
		Username: newUsername,
	})

	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, newUsername, updatedUser.Username)
}

func TestUserRepository_ChangeUsername_Duplicate(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	ctx := context.Background()
	user1 := setupTestUser(t, db)
	user2 := setupTestUser(t, db)
	defer cleanupTestUser(t, db, user1.ID)
	defer cleanupTestUser(t, db, user2.ID)

	_, err := repo.ChangeUsername(ctx, user1.ID, domain.ChangeUsernameParams{
		Username: user2.Username,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestUserRepository_ChangePassword(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	err := repo.ChangePassword(context.Background(), createdUser.ID, domain.ChangePasswordParams{
		Password: "newpasswordhash",
	})

	require.NoError(t, err)

	foundUser, err := repo.FindByID(context.Background(), createdUser.ID)
	require.NoError(t, err)
	assert.Equal(t, "newpasswordhash", *foundUser.PasswordHash)
}

func TestUserRepository_ChangePassword_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	err := repo.ChangePassword(context.Background(), uuid.New(), domain.ChangePasswordParams{
		Password: "somepassword",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_UpdateRole(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	updatedUser, err := repo.UpdateRole(context.Background(), createdUser.ID, domain.UpdateRoleParams{
		Role: domain.RoleModerator,
	})

	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, domain.RoleModerator, updatedUser.Role)
}

func TestUserRepository_UpdateRole_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	_, err := repo.UpdateRole(context.Background(), uuid.New(), domain.UpdateRoleParams{
		Role: domain.RoleAdmin,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_UpdateStatus(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	updatedUser, err := repo.UpdateStatus(context.Background(), createdUser.ID, domain.UpdateStatusParams{
		Status: domain.StatusSuspended,
	})

	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, domain.StatusSuspended, updatedUser.Status)
}

func TestUserRepository_UpdateStatus_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	_, err := repo.UpdateStatus(context.Background(), uuid.New(), domain.UpdateStatusParams{
		Status: domain.StatusBanned,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	beforeTime := time.Now()
	time.Sleep(1 * time.Second)

	err := repo.UpdateLastLogin(context.Background(), createdUser.ID)

	require.NoError(t, err)

	foundUser, err := repo.FindByID(context.Background(), createdUser.ID)
	require.NoError(t, err)
	require.NotNil(t, foundUser.LastLoginAt)
	assert.True(t, foundUser.LastLoginAt.After(beforeTime))
	assert.WithinDuration(t, beforeTime, *foundUser.LastLoginAt, 2*time.Second)
}

func TestUserRepository_UpdateLastLogin_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	err := repo.UpdateLastLogin(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_VerifyEmail(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	err := repo.VerifyEmail(context.Background(), createdUser.ID)

	require.NoError(t, err)

	foundUser, err := repo.FindByID(context.Background(), createdUser.ID)
	require.NoError(t, err)
	require.NotNil(t, foundUser.EmailVerifiedAt)
	assert.NotZero(t, *foundUser.EmailVerifiedAt)
}

func TestUserRepository_VerifyEmail_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	err := repo.VerifyEmail(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_SoftDelete(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	userID := createdUser.ID

	err := repo.SoftDelete(context.Background(), userID)

	require.NoError(t, err)

	deletedUser, err := repo.FindByID(context.Background(), userID)
	require.Error(t, err)
	assert.Nil(t, deletedUser)
}

func TestUserRepository_SoftDelete_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	err := repo.SoftDelete(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_Restore(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	err := repo.SoftDelete(context.Background(), createdUser.ID)
	require.NoError(t, err)

	restoredUser, err := repo.Restore(context.Background(), createdUser.ID)

	require.NoError(t, err)
	require.NotNil(t, restoredUser)
	assert.Nil(t, restoredUser.DeletedAt)
	assert.Equal(t, domain.StatusActive, restoredUser.Status)
}

func TestUserRepository_Restore_Expired(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	_, err := db.ExecContext(context.Background(),
		"UPDATE users SET deleted_at = NOW() - INTERVAL '31 days' WHERE id = $1",
		createdUser.ID)
	require.NoError(t, err)

	_, err = repo.Restore(context.Background(), createdUser.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_Restore_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	_, err := repo.Restore(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_HardDelete(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	userID := createdUser.ID

	err := repo.SoftDelete(context.Background(), userID)
	require.NoError(t, err)

	err = repo.HardDelete(context.Background(), userID)

	require.NoError(t, err)

	_, err = repo.FindByID(context.Background(), userID)
	require.Error(t, err)
}

func TestUserRepository_HardDelete_NotSoftDeleted(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	err := repo.HardDelete(context.Background(), createdUser.ID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_List(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	user1 := setupTestUser(t, db)
	user2 := setupTestUser(t, db)
	defer cleanupTestUser(t, db, user1.ID)
	defer cleanupTestUser(t, db, user2.ID)

	users, total, err := repo.List(context.Background(), domain.ListUsersParams{
		Limit:  10,
		Offset: 0,
	})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 2)
	assert.GreaterOrEqual(t, len(users), 2)
}

func TestUserRepository_List_Empty(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	_, err := db.ExecContext(context.Background(), "DELETE FROM users")
	if err != nil {
		t.Logf("could not clean users table: %v", err)
	}

	users, total, err := repo.List(context.Background(), domain.ListUsersParams{
		Limit:  10,
		Offset: 0,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, users)
}

func TestUserRepository_List_Pagination(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	for range 3 {
		_, err := repo.Create(context.Background(), domain.CreateUserParams{
			Username:     "paguser" + uuid.New().String()[:8],
			Email:        "pag" + uuid.New().String()[:8] + "@example.com",
			PasswordHash: "hash",
			DisplayName:  "Page User",
		})
		if err != nil {
			t.Logf("create failed: %v", err)
		}
	}

	users, total, err := repo.List(context.Background(), domain.ListUsersParams{
		Limit:  2,
		Offset: 0,
	})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 3)
	assert.Len(t, users, 2)
}

func TestUserRepository_Count(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	user := setupTestUser(t, db)
	defer cleanupTestUser(t, db, user.ID)

	count, err := repo.Count(context.Background())

	require.NoError(t, err)
	assert.Greater(t, count, 0)
}

func TestUserRepository_UsernameExists(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	exists, err := repo.UsernameExists(context.Background(), createdUser.Username)

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUserRepository_UsernameExists_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	exists, err := repo.UsernameExists(context.Background(), "nonexistent123456")

	require.NoError(t, err)
	assert.False(t, exists)
}

func TestUserRepository_EmailExists(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	exists, err := repo.EmailExists(context.Background(), createdUser.Email)

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUserRepository_EmailExists_NotFound(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	exists, err := repo.EmailExists(context.Background(), "nonexistent@example.com")

	require.NoError(t, err)
	assert.False(t, exists)
}

func TestUserRepository_AnonymizeExpired(t *testing.T) {
	db := getTestDB(t)
	repo := NewUserRepository(db)

	createdUser := setupTestUser(t, db)
	defer cleanupTestUser(t, db, createdUser.ID)

	_, err := db.ExecContext(context.Background(),
		"UPDATE users SET deleted_at = NOW() - INTERVAL '31 days' WHERE id = $1",
		createdUser.ID)
	require.NoError(t, err)

	count, err := repo.AnonymizeExpired(context.Background())

	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(1))
}
