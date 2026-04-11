package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestUserForRefresh(t *testing.T, db *sqlx.DB) *domain.User {
	user, err := NewUserRepository(db).Create(context.Background(), domain.CreateUserParams{
		Username:     "rtuser_" + uuid.New().String()[:8],
		Email:        "rt_" + uuid.New().String()[:8] + "@example.com",
		PasswordHash: "hashedpassword",
		DisplayName:  "Refresh Test User",
	})
	require.NoError(t, err)
	return user
}

func cleanupRefreshToken(t *testing.T, db *sqlx.DB, tokenID uuid.UUID) {
	_, err := db.ExecContext(context.Background(), "DELETE FROM refresh_tokens WHERE id = $1", tokenID)
	if err != nil {
		t.Logf("cleanup failed: %v", err)
	}
}

func cleanupTestUserForRefresh(t *testing.T, db *sqlx.DB, userID uuid.UUID) {
	_, err := db.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		t.Logf("cleanup failed: %v", err)
	}
}

func TestRefreshTokenRepository_Create(t *testing.T) {
	

	db := getTestDB(t)
	repo := NewRefreshTokenRepository(db)

	user := setupTestUserForRefresh(t, db)
	defer cleanupTestUserForRefresh(t, db, user.ID)

	token, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: "test_token_hash_" + uuid.New().String()[:8],
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, user.ID, token.UserID)
	assert.NotEqual(t, uuid.Nil, token.ID)
	assert.NotZero(t, token.CreatedAt)

	defer cleanupRefreshToken(t, db, token.ID)
}

func TestRefreshTokenRepository_Create_InternalError(t *testing.T) {
	

	db := getTestDB(t)
	repo := NewRefreshTokenRepository(db)

	_, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    uuid.Nil,
		TokenHash: "invalid_user_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInternal)
}

func TestRefreshTokenRepository_FindByTokenHash(t *testing.T) {
	

	db := getTestDB(t)
	repo := NewRefreshTokenRepository(db)

	user := setupTestUserForRefresh(t, db)
	defer cleanupTestUserForRefresh(t, db, user.ID)

	tokenHash := "find_token_hash_" + uuid.New().String()[:8]
	createdToken, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)
	defer cleanupRefreshToken(t, db, createdToken.ID)

	foundToken, err := repo.FindByTokenHash(context.Background(), tokenHash)

	require.NoError(t, err)
	require.NotNil(t, foundToken)
	assert.Equal(t, createdToken.ID, foundToken.ID)
	assert.Equal(t, tokenHash, foundToken.TokenHash)
}

func TestRefreshTokenRepository_FindByTokenHash_NotFound(t *testing.T) {
	

	db := getTestDB(t)
	repo := NewRefreshTokenRepository(db)

	_, err := repo.FindByTokenHash(context.Background(), "nonexistent_hash_12345")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRefreshTokenRepository_FindByTokenHash_Expired(t *testing.T) {
	

	db := getTestDB(t)
	repo := NewRefreshTokenRepository(db)

	user := setupTestUserForRefresh(t, db)
	defer cleanupTestUserForRefresh(t, db, user.ID)

	tokenHash := "expired_token_hash_" + uuid.New().String()[:8]
	createdToken, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Second),
	})
	require.NoError(t, err)
	defer cleanupRefreshToken(t, db, createdToken.ID)

	time.Sleep(2 * time.Second)

	_, err = repo.FindByTokenHash(context.Background(), tokenHash)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRefreshTokenRepository_FindByTokenHash_Revoked(t *testing.T) {
	

	db := getTestDB(t)
	repo := NewRefreshTokenRepository(db)

	user := setupTestUserForRefresh(t, db)
	defer cleanupTestUserForRefresh(t, db, user.ID)

	tokenHash := "revoked_token_hash_" + uuid.New().String()[:8]
	createdToken, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	err = repo.Revoke(context.Background(), tokenHash)
	require.NoError(t, err)

	_, err = repo.FindByTokenHash(context.Background(), tokenHash)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	cleanupRefreshToken(t, db, createdToken.ID)
}

func TestRefreshTokenRepository_Revoke(t *testing.T) {
	

	db := getTestDB(t)
	repo := NewRefreshTokenRepository(db)

	user := setupTestUserForRefresh(t, db)
	defer cleanupTestUserForRefresh(t, db, user.ID)

	tokenHash := "revoke_token_hash_" + uuid.New().String()[:8]
	createdToken, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)
	defer cleanupRefreshToken(t, db, createdToken.ID)

	err = repo.Revoke(context.Background(), tokenHash)

	require.NoError(t, err)

	foundToken, err := repo.FindByTokenHash(context.Background(), tokenHash)
	require.Error(t, err)
	assert.Nil(t, foundToken)
}

func TestRefreshTokenRepository_Revoke_NonExistent(t *testing.T) {
	

	db := getTestDB(t)
	repo := NewRefreshTokenRepository(db)

	err := repo.Revoke(context.Background(), "nonexistent_hash_12345")

	require.NoError(t, err)
}

func TestRefreshTokenRepository_RevokeAllForUser(t *testing.T) {
	

	db := getTestDB(t)
	repo := NewRefreshTokenRepository(db)

	user := setupTestUserForRefresh(t, db)
	defer cleanupTestUserForRefresh(t, db, user.ID)

	tokenHash1 := "revoke_all_1_" + uuid.New().String()[:8]
	tokenHash2 := "revoke_all_2_" + uuid.New().String()[:8]

	token1, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash1,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	token2, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash2,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	defer cleanupRefreshToken(t, db, token1.ID)
	defer cleanupRefreshToken(t, db, token2.ID)

	err = repo.RevokeAllForUser(context.Background(), user.ID)

	require.NoError(t, err)

	_, err = repo.FindByTokenHash(context.Background(), tokenHash1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = repo.FindByTokenHash(context.Background(), tokenHash2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRefreshTokenRepository_DeleteExpired(t *testing.T) {
	

	db := getTestDB(t)
	repo := NewRefreshTokenRepository(db)

	user := setupTestUserForRefresh(t, db)
	defer cleanupTestUserForRefresh(t, db, user.ID)

	expiredTokenHash := "expired_delete_" + uuid.New().String()[:8]
	expiredToken, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: expiredTokenHash,
		ExpiresAt: time.Now().Add(1 * time.Second),
	})
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	revokedTokenHash := "revoked_delete_" + uuid.New().String()[:8]
	revokedToken, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: revokedTokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	err = repo.Revoke(context.Background(), revokedTokenHash)
	require.NoError(t, err)

	activeTokenHash := "active_delete_" + uuid.New().String()[:8]
	activeToken, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: activeTokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	deletedCount, err := repo.DeleteExpired(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(2), deletedCount)

	_, err = repo.FindByTokenHash(context.Background(), expiredTokenHash)
	require.Error(t, err)

	_, err = repo.FindByTokenHash(context.Background(), revokedTokenHash)
	require.Error(t, err)

	_, err = repo.FindByTokenHash(context.Background(), activeTokenHash)
	require.NoError(t, err)

	cleanupRefreshToken(t, db, activeToken.ID)
	_ = expiredToken
	_ = revokedToken
}

func TestRefreshTokenRepository_DeleteExpired_NothingToDelete(t *testing.T) {
	

	db := getTestDB(t)
	repo := NewRefreshTokenRepository(db)

	user := setupTestUserForRefresh(t, db)
	defer cleanupTestUserForRefresh(t, db, user.ID)

	tokenHash := "active_only_" + uuid.New().String()[:8]
	activeToken, err := repo.Create(context.Background(), domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)
	defer cleanupRefreshToken(t, db, activeToken.ID)

	deletedCount, err := repo.DeleteExpired(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(0), deletedCount)
}
