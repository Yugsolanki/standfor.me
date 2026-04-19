package service

import (
	"context"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/google/uuid"
)

type MovementRepository interface {
	MovementReader

	FindBySlug(ctx context.Context, slug string) (*domain.Movement, error)
	Create(ctx context.Context, params domain.CreateMovementParams) (*domain.Movement, error)
	Update(ctx context.Context, id uuid.UUID, params domain.UpdateMovementParams) (*domain.Movement, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, params domain.UpdateMovementStatusParams) (*domain.Movement, error)
	SubmitForReview(ctx context.Context, id uuid.UUID, params domain.SubmitForReviewParams) (*domain.Movement, error)
	ReviewMovement(ctx context.Context, id uuid.UUID, params domain.ReviewMovementParams) (*domain.Movement, error)
	IncrementSupporters(ctx context.Context, id uuid.UUID) error
	DecrementSupporters(ctx context.Context, id uuid.UUID) error
	UpdateTrendingScore(ctx context.Context, id uuid.UUID, score float64) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (*domain.Movement, error)
	HardDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params domain.ListMovementsParams) ([]domain.Movement, int, error)
	ListActive(ctx context.Context, params domain.ListMovementsParams) ([]domain.Movement, int, error)
	ListPendingReview(ctx context.Context, params domain.ListMovementsParams) ([]domain.Movement, int, error)
	Count(ctx context.Context, status string) (int, error)
	CountActive(ctx context.Context) (int, error)
	CountPendingReview(ctx context.Context) (int, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	BatchGetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Movement, error)
	GetTrending(ctx context.Context, limit int) ([]domain.Movement, error)
	GetPopular(ctx context.Context, limit int) ([]domain.Movement, error)
	GetRecent(ctx context.Context, limit int) ([]domain.Movement, error)
	GetUserMovements(ctx context.Context, userID uuid.UUID) ([]domain.Movement, error)
	Archive(ctx context.Context, id uuid.UUID) error
	CalculateTrendingScore(ctx context.Context, id uuid.UUID) (float64, error)
	UpdateAllTrendingScores(ctx context.Context) (int64, error)
	SearchMovements(ctx context.Context, params domain.SearchMovementsParams) ([]domain.Movement, int, error)
	GetMovementSupporters(ctx context.Context, movementID uuid.UUID, params domain.ListSupportersParams) ([]domain.User, int, error)
}

type MovementReader interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Movement, error)
}

type MovementService struct {
	movements MovementRepository
}

func NewMovementService(movements MovementRepository) *MovementService {
	return &MovementService{
		movements: movements,
	}
}

func (s *MovementService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Movement, error) {
	movement, err := s.movements.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return movement, nil
}

func (s *MovementService) GetBySlug(ctx context.Context, slug string) (*domain.Movement, error) {
	movement, err := s.movements.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return movement, nil
}

func (s *MovementService) Create(ctx context.Context, params domain.CreateMovementParams) (*domain.Movement, error) {
	movement, err := s.movements.Create(ctx, params)
	if err != nil {
		return nil, err
	}

	return movement, nil
}

func (s *MovementService) Update(ctx context.Context, id uuid.UUID, params domain.UpdateMovementParams) (*domain.Movement, error) {
	movement, err := s.movements.Update(ctx, id, params)
	if err != nil {
		return nil, err
	}

	return movement, nil
}

func (s *MovementService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*domain.Movement, error) {
	movement, err := s.movements.UpdateStatus(ctx, id, domain.UpdateMovementStatusParams{Status: status})
	if err != nil {
		return nil, err
	}

	return movement, nil
}

func (s *MovementService) SubmitForReview(ctx context.Context, id uuid.UUID, submittedByUserID uuid.UUID) (*domain.Movement, error) {
	movement, err := s.movements.SubmitForReview(ctx, id, domain.SubmitForReviewParams{SubmittedByUserID: submittedByUserID})
	if err != nil {
		return nil, err
	}

	return movement, nil
}

func (s *MovementService) Review(ctx context.Context, id uuid.UUID, reviewedByUserID uuid.UUID, approved bool) (*domain.Movement, error) {
	movement, err := s.movements.ReviewMovement(ctx, id, domain.ReviewMovementParams{ReviewedByUserID: reviewedByUserID, Approved: approved})
	if err != nil {
		return nil, err
	}

	return movement, nil
}

func (s *MovementService) IncrementSupporters(ctx context.Context, id uuid.UUID) error {
	if err := s.movements.IncrementSupporters(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *MovementService) DecrementSupporters(ctx context.Context, id uuid.UUID) error {
	if err := s.movements.DecrementSupporters(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *MovementService) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if err := s.movements.SoftDelete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *MovementService) Restore(ctx context.Context, id uuid.UUID) (*domain.Movement, error) {
	movement, err := s.movements.Restore(ctx, id)
	if err != nil {
		return nil, err
	}

	return movement, nil
}

func (s *MovementService) HardDelete(ctx context.Context, id uuid.UUID) error {
	if err := s.movements.HardDelete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *MovementService) List(ctx context.Context, params domain.ListMovementsParams) ([]domain.Movement, int, error) {
	movements, total, err := s.movements.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	return movements, total, nil
}

func (s *MovementService) ListActive(ctx context.Context, params domain.ListMovementsParams) ([]domain.Movement, int, error) {
	movements, total, err := s.movements.ListActive(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	return movements, total, nil
}

func (s *MovementService) ListPendingReview(ctx context.Context, params domain.ListMovementsParams) ([]domain.Movement, int, error) {
	movements, total, err := s.movements.ListPendingReview(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	return movements, total, nil
}

func (s *MovementService) Count(ctx context.Context, status string) (int, error) {
	total, err := s.movements.Count(ctx, status)
	if err != nil {
		return -1, err
	}

	return total, nil
}

func (s *MovementService) CountActive(ctx context.Context) (int, error) {
	total, err := s.movements.CountActive(ctx)
	if err != nil {
		return -1, err
	}

	return total, nil
}

func (s *MovementService) CountPendingReview(ctx context.Context) (int, error) {
	total, err := s.movements.CountPendingReview(ctx)
	if err != nil {
		return -1, err
	}

	return total, nil
}

func (s *MovementService) SlugExists(ctx context.Context, slug string) (bool, error) {
	exists, err := s.movements.SlugExists(ctx, slug)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (s *MovementService) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	exists, err := s.movements.Exists(ctx, id)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (s *MovementService) BatchGetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Movement, error) {
	if len(ids) == 0 {
		return []domain.Movement{}, nil
	}

	movements, err := s.movements.BatchGetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	return movements, nil
}

func (s *MovementService) GetTrending(ctx context.Context, limit int) ([]domain.Movement, error) {
	if limit <= 0 {
		limit = 10
	}

	movements, err := s.movements.GetTrending(ctx, limit)
	if err != nil {
		return nil, err
	}

	return movements, nil
}

func (s *MovementService) GetPopular(ctx context.Context, limit int) ([]domain.Movement, error) {
	if limit <= 0 {
		limit = 10
	}

	movements, err := s.movements.GetPopular(ctx, limit)
	if err != nil {
		return nil, err
	}

	return movements, nil
}

func (s *MovementService) GetRecent(ctx context.Context, limit int) ([]domain.Movement, error) {
	if limit <= 0 {
		limit = 10
	}

	movements, err := s.movements.GetRecent(ctx, limit)
	if err != nil {
		return nil, err
	}

	return movements, nil
}

func (s *MovementService) GetUserMovements(ctx context.Context, userID uuid.UUID) ([]domain.Movement, error) {
	movements, err := s.movements.GetUserMovements(ctx, userID)
	if err != nil {
		return nil, err
	}

	return movements, nil
}

func (s *MovementService) Archive(ctx context.Context, id uuid.UUID) error {
	if err := s.movements.Archive(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *MovementService) RecalculateTrendingScore(ctx context.Context, id uuid.UUID) error {
	score, err := s.movements.CalculateTrendingScore(ctx, id)
	if err != nil {
		return err
	}

	if err := s.movements.UpdateTrendingScore(ctx, id, score); err != nil {
		return err
	}

	return nil
}

func (s *MovementService) RecalculateAllTrendingScores(ctx context.Context) (int64, error) {
	updated, err := s.movements.UpdateAllTrendingScores(ctx)
	if err != nil {
		return -1, err
	}

	return updated, nil
}

func (s *MovementService) SearchMovements(ctx context.Context, params domain.SearchMovementsParams) ([]domain.Movement, int, error) {
	movements, total, err := s.movements.SearchMovements(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	return movements, total, nil
}

func (s *MovementService) GetMovementSupporters(ctx context.Context, movementID uuid.UUID, params domain.ListSupportersParams) ([]domain.User, int, error) {
	supporters, total, err := s.movements.GetMovementSupporters(ctx, movementID, params)
	if err != nil {
		return nil, 0, err
	}

	return supporters, total, nil
}
