package service

import (
	"context"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/pkg/crypto"
	"github.com/google/uuid"
)

type UserRepository interface {
	UserReader

	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	Update(ctx context.Context, id uuid.UUID, params domain.UpdateUserParams) (*domain.User, error)
	ChangePassword(ctx context.Context, id uuid.UUID, params domain.ChangePasswordParams) error
	UpdateRole(ctx context.Context, id uuid.UUID, params domain.UpdateRoleParams) (*domain.User, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, params domain.UpdateStatusParams) (*domain.User, error)
	VerifyEmail(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (*domain.User, error)
	AnonymizeExpired(ctx context.Context) (int64, error)
	HardDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params domain.ListUsersParams) ([]domain.User, int, error)
	Count(ctx context.Context) (int, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	EmailExists(ctx context.Context, email string) (bool, error)
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type UserService struct {
	users         UserRepository
	refreshTokens RefreshTokenStore
}

func NewUserService(users UserRepository, refreshTokens RefreshTokenStore) *UserService {
	return &UserService{
		users:         users,
		refreshTokens: refreshTokens,
	}
}

// GetByID fetches a user by their UUID.
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetByUsername fetches a user by their username.
func (s *UserService) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetByEmail fetches a user by their email.
func (s *UserService) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Update applies a partial update to a user record.
func (s *UserService) Update(ctx context.Context, id uuid.UUID, params domain.UpdateUserParams) (*domain.User, error) {
	user, err := s.users.Update(ctx, id, params)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// ChangePassword verifies the current password before replacing it
func (s *UserService) ChangePassword(ctx context.Context, id uuid.UUID, input ChangePasswordInput) error {
	const op = "UserService.ChangePassword"

	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if user.PasswordHash == nil || !crypto.CheckPassword(input.CurrentPassword, *user.PasswordHash) {
		return domain.NewInvalidCredentialsError(op)
	}

	newHash, err := crypto.HashPassword(input.NewPassword)
	if err != nil {
		return domain.NewInternalError(op, err)
	}

	if err = s.users.ChangePassword(ctx, id, domain.ChangePasswordParams{Password: newHash}); err != nil {
		return err
	}

	// Invalid all existing sessions so the user must re-authenticate
	if err = s.refreshTokens.RevokeAllForUser(ctx, id); err != nil {
		return err
	}

	return nil
}

// UpdateRole changes a user's role.
func (s *UserService) UpdateRole(ctx context.Context, id uuid.UUID, role string) (*domain.User, error) {
	user, err := s.users.UpdateRole(ctx, id, domain.UpdateRoleParams{Role: role})
	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateStatus changes a user's status.
func (s *UserService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*domain.User, error) {
	user, err := s.users.UpdateStatus(ctx, id, domain.UpdateStatusParams{Status: status})
	if err != nil {
		return nil, err
	}

	return user, nil
}

// VerifyEmail verifies a user's email address.
func (s *UserService) VerifyEmail(ctx context.Context, id uuid.UUID) error {
	if err := s.users.VerifyEmail(ctx, id); err != nil {
		return err
	}

	return nil
}

// SoftDelete marks a user as deleted and terminates all sessions
func (s *UserService) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if err := s.users.SoftDelete(ctx, id); err != nil {
		return err
	}

	_ = s.refreshTokens.RevokeAllForUser(ctx, id)

	return nil
}

// Restore restores a soft-deleted user.
func (s *UserService) Restore(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.users.Restore(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// AdminAnonymizeExpired anonymizes all expired users.
func (s *UserService) AdminAnonymizeExpired(ctx context.Context) (int64, error) {
	rows, err := s.users.AnonymizeExpired(ctx)
	if err != nil {
		return -1, err
	}

	return rows, nil
}

// AdminHardDelete permanently deletes a user.
func (s *UserService) AdminHardDelete(ctx context.Context, id uuid.UUID) error {
	if err := s.users.HardDelete(ctx, id); err != nil {
		return err
	}

	return nil
}

// List returns a paginated slice of non-deleted users ordered by creation date (newest first),
func (s *UserService) List(ctx context.Context, params domain.ListUsersParams) ([]domain.User, int, error) {
	users, total, err := s.users.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Count returns the total number of users.
func (s *UserService) Count(ctx context.Context) (int, error) {
	total, err := s.users.Count(ctx)
	if err != nil {
		return -1, err
	}
	return total, nil
}

// UsernameExists checks if a username already exists.
func (s *UserService) UsernameExists(ctx context.Context, username string) (bool, error) {
	exists, err := s.users.UsernameExists(ctx, username)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// EmailExists checks if an email address already exists.
func (s *UserService) EmailExists(ctx context.Context, email string) (bool, error) {
	exists, err := s.users.EmailExists(ctx, email)
	if err != nil {
		return false, err
	}

	return exists, nil
}
