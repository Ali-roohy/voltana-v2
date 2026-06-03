package service

import (
	"context"
	"errors"
	"strings"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"

	"github.com/google/uuid"
)

// ErrEVModelNotFound is returned when an ev_model id has no match.
var ErrEVModelNotFound = errors.New("ev model not found")

// EVModelService is read-only catalog logic: pagination clamping + error
// translation.
type EVModelService struct {
	models repository.EVModelRepository
}

func NewEVModelService(models repository.EVModelRepository) *EVModelService {
	return &EVModelService{models: models}
}

func (s *EVModelService) List(ctx context.Context, q string, limit, offset int) ([]domain.EVModel, int, error) {
	limit, offset = ClampPagination(limit, offset)
	return s.models.List(ctx, strings.TrimSpace(q), limit, offset)
}

func (s *EVModelService) Get(ctx context.Context, id uuid.UUID) (*domain.EVModel, error) {
	m, err := s.models.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEVModelNotFound
	}
	return m, err
}
