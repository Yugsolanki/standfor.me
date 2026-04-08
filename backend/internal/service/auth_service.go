package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/pkg/crypto"
	internaljwt "github.com/Yugsolanki/standfor-me/internal/pkg/jwt"
	"github.com/google/uuid"
)

// UserReader is the subset of the user repository that the
// AuthService needs. Keeping it minimal makes testing easier.
type UserReader interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	Create(ctx context.Context, params domain.CreateUserParams) (*domain.User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// RefreshTokenStore is the subset of the refresh token repository
// the AuthService requires.
type RefreshTokenStore interface {
	Create(ctx context.Context, params domain.CreateRefreshTokenParams) (*domain.RefreshToken, error)
	FindByTokenHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, hash string) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// RegisterInput carries the data submitted by a new user at the
// registration endpoint. Validation occurs in the service layer.
type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
}

// LoginInput carries credentials submitted at the login endpoint.
type LoginInput struct {
	Email    string
	Password string
}

// AuthService contains all authentication business logic:
// registration, login, token refresh, and logout.
type AuthService struct {
	users         UserReader
	refreshTokens RefreshTokenStore
	jwt           *internaljwt.Service
}

// NewAuthService constructs an AuthService with its dependencies.
func NewAuthService(
	users UserReader, refreshTokens RefreshTokenStore, jwt *internaljwt.Service,
) *AuthService {
	return &AuthService{
		users:         users,
		refreshTokens: refreshTokens,
		jwt:           jwt,
	}
}

// Register creates a new user account and returns a token pair.
// It enforces uniqueness of email and username before persisting.
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*domain.User, *domain.TokenPair, error) {
	const op = "AuthService.Register"

	emailTaken, err := s.users.EmailExists(ctx, input.Email)
	if err != nil {
		return nil, nil, domain.NewInternalError(op, err)
	}
	if emailTaken {
		return nil, nil, domain.NewConflictError(op, "an account with this email already exists")
	}

	usernameTaken, err := s.users.UsernameExists(ctx, input.Username)
	if err != nil {
		return nil, nil, domain.NewInternalError(op, err)
	}
	if usernameTaken {
		return nil, nil, domain.NewConflictError(op, "this username is already taken")
	}

	hash, err := crypto.HashPassword(input.Password)
	if err != nil {
		return nil, nil, domain.NewInternalError(op, fmt.Errorf("hashing password: %w", err))
	}

	user, err := s.users.Create(ctx, domain.CreateUserParams{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: hash,
		DisplayName:  input.DisplayName,
	})
	if err != nil {
		return nil, nil, domain.NewInternalError(op, err)
	}

	pair, err := s.issueTokenPair(ctx, op, user)
	if err != nil {
		return nil, nil, err
	}

	return user, pair, nil
}

// Login validates credentials and returns a new token pair.
// It intentionally returns ErrInvalidCredentials for both "not
// found" and "wrong password" to avoid user enumeration.
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*domain.User, *domain.TokenPair, error) {
	const op = "AuthService.Login"

	user, err := s.users.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.NewInvalidCredentialsError(op)
		}
		return nil, nil, domain.NewInternalError(op, err)
	}

	if user.Status != domain.StatusActive {
		return nil, nil, domain.NewForbiddenError(op, "account is not active")
	}

	if user.PasswordHash == nil || !crypto.CheckPassword(input.Password, *user.PasswordHash) {
		return nil, nil, domain.NewInvalidCredentialsError(op)
	}

	_ = s.users.UpdateLastLogin(ctx, user.ID)

	pair, err := s.issueTokenPair(ctx, op, user)
	if err != nil {
		return nil, nil, err
	}

	return user, pair, nil
}

// RefreshTokens validates an incoming refresh token, revokes it
// (one-time-use rotation), and issues a fresh pair.
func (s *AuthService) RefreshTokens(ctx context.Context, rawRefreshToken string) (*domain.TokenPair, error) {
	const op = "AuthService.RefreshTokens"

	hash := hashToken(rawRefreshToken)

	record, err := s.refreshTokens.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewUnauthorizedError(op, "refresh token is invalid or has expired")
		}
		return nil, domain.NewInternalError(op, err)
	}

	if time.Now().UTC().After(record.ExpiresAt) {
		return nil, domain.NewUnauthorizedError(op, "refresh token has expired")
	}

	if err = s.refreshTokens.Revoke(ctx, hash); err != nil {
		return nil, domain.NewInternalError(op, err)
	}

	user, err := s.users.FindByID(ctx, record.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewUnauthorizedError(op, "associated user no longer exists")
		}
		return nil, domain.NewInternalError(op, err)
	}

	if user.Status != domain.StatusActive {
		return nil, domain.NewForbiddenError(op, "account is not active")
	}

	pair, err := s.issueTokenPair(ctx, op, user)
	if err != nil {
		return nil, err
	}

	return pair, nil
}

// Logout revokes the single refresh token presented by the client.
func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	const op = "AuthService.Logout"

	if err := s.refreshTokens.Revoke(ctx, hashToken(rawRefreshToken)); err != nil {
		return domain.NewInternalError(op, err)
	}

	return nil
}

// LogoutAll revokes every refresh token for the given user,
// terminating all sessions across all devices.
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	const op = "AuthService.LogoutAll"

	if err := s.refreshTokens.RevokeAllForUser(ctx, userID); err != nil {
		return domain.NewInternalError(op, err)
	}

	return nil
}

// --- Private Helpers ---

func (s *AuthService) issueTokenPair(ctx context.Context, op string, user *domain.User) (*domain.TokenPair, error) {
	accessToken, err := s.jwt.IssueAccessToken(user)
	if err != nil {
		return nil, domain.NewInternalError(op, fmt.Errorf("issuing access token: %w", err))
	}

	rawRefresh, err := generateOpaqueToken()
	if err != nil {
		return nil, domain.NewInternalError(op, fmt.Errorf("generating refresh token: %w", err))
	}

	_, err = s.refreshTokens.Create(ctx, domain.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: time.Now().UTC().Add(s.jwt.RefreshTokenTTL()),
	})
	if err != nil {
		return nil, domain.NewInternalError(op, fmt.Errorf("persisting refresh token: %w", err))
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

// generateOpaqueToken produces a cryptographically random,
// base64-encoded token string suitable for use as a refresh token.
func generateOpaqueToken() (string, error) {
	b, err := crypto.GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return crypto.EncodeBase64(b), nil
}

// hashToken produces an SHA-256 hex digest of a raw token string.
// We never store raw refresh tokens — only their hashes — so that
// a database breach cannot be used to hijack active sessions.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
