package service_test

import (
	"context"
	"errors"
	"testing"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"
	"voltana-api/internal/service"

	"github.com/google/uuid"
)

type mockEVRepo struct {
	models     map[uuid.UUID]domain.EVModel
	lastQ      string
	lastLimit  int
	lastOffset int
}

func newMockEVRepo() *mockEVRepo {
	return &mockEVRepo{models: make(map[uuid.UUID]domain.EVModel)}
}

func (m *mockEVRepo) List(_ context.Context, q string, limit, offset int) ([]domain.EVModel, int, error) {
	m.lastQ, m.lastLimit, m.lastOffset = q, limit, offset
	items := make([]domain.EVModel, 0, len(m.models))
	for _, e := range m.models {
		items = append(items, e)
	}
	return items, len(items), nil
}

func (m *mockEVRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.EVModel, error) {
	e, ok := m.models[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &e, nil
}

func TestEVModel_ListClampsAndPassesQuery(t *testing.T) {
	repo := newMockEVRepo()
	svc := service.NewEVModelService(repo)

	if _, _, err := svc.List(context.Background(), "  tesla  ", 500, 0); err != nil {
		t.Fatal(err)
	}
	if repo.lastQ != "tesla" {
		t.Errorf("query should be trimmed and passed through, got %q", repo.lastQ)
	}
	if repo.lastLimit != 100 {
		t.Errorf("limit should clamp to 100, got %d", repo.lastLimit)
	}
}

func TestEVModel_GetNotFound(t *testing.T) {
	svc := service.NewEVModelService(newMockEVRepo())
	_, err := svc.Get(context.Background(), uuid.New())
	if !errors.Is(err, service.ErrEVModelNotFound) {
		t.Errorf("want ErrEVModelNotFound, got %v", err)
	}
}
