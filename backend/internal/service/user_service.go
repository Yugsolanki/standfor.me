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
	ChangeUsername(ctx context.Context, id uuid.UUID, params domain.ChangeUsernameParams) (*domain.User, error)
	ChangePassword(ctx context.Context, id uuid.UUID, params domain.ChangePasswordParams) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	UpdateRole(ctx context.Context, id uuid.UUID, params domain.UpdateRoleParams) (*domain.User, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, params domain.UpdateStatusParams) (*domain.User, error)
	List(ctx context.Context, params domain.ListUsersParams) ([]domain.User, error)
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
	const op = "UserService.GetByID"

	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetByUsername fetches a user by their username.
func (s *UserService) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	const op = "UserService.GetByUsername"

	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetByEmail fetches a user by their email.
func (s *UserService) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const op = "UserService.GetByEmail"

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Update applies a partial update to a user record.
func (s *UserService) Update(ctx context.Context, id uuid.UUID, params domain.UpdateUserParams) (*domain.User, error) {
	const op = "UserService.Update"

	user, err := s.users.Update(ctx, id, params)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// ChangeUsername changes users username after verifying if it's available
func (s *UserService) ChangeUsername(ctx context.Context, id uuid.UUID, username string) (*domain.User, error) {
	const op = "UserService.ChangeUsername"

	if username != "" {
		taken, err := s.users.UsernameExists(ctx, username)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, domain.NewConflictError(op, "this username is already taken")
		}
	}

	user, err := s.users.ChangeUsername(ctx, id, domain.ChangeUsernameParams{Username: username})
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
	const op = "UserService.UpdateRole"

	user, err := s.users.UpdateRole(ctx, id, domain.UpdateRoleParams{Role: role})
	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateStatus changes a user's status.
func (s *UserService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*domain.User, error) {
	const op = "UserService.UpdateStatus"

	user, err := s.users.UpdateStatus(ctx, id, domain.UpdateStatusParams{Status: status})
	if err != nil {
		return nil, err
	}

	return user, nil
}

// SoftDelete marks a user as deleted and terminates all sessions
func (s *UserService) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const op = "UserService.SoftDelete"

	if err := s.users.SoftDelete(ctx, id); err != nil {
		return err
	}

	_ = s.refreshTokens.RevokeAllForUser(ctx, id)

	return nil
}
