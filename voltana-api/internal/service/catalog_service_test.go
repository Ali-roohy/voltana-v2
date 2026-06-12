package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"voltana-api/internal/domain"
	"voltana-api/internal/service"

	"github.com/google/uuid"
)

// ── mock CatalogRepository ────────────────────────────────────────────────────

type mockCatalogRepo struct {
	mu    sync.Mutex
	items []domain.CatalogCar
	calls int
	err   error
}

func (m *mockCatalogRepo) ListCatalog(_ context.Context) ([]domain.CatalogCar, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}

func catalogFixture(n int) []domain.CatalogCar {
	items := make([]domain.CatalogCar, 0, n)
	for i := 0; i < n; i++ {
		brand := "Brand"
		items = append(items, domain.CatalogCar{
			ID:             uuid.New(),
			NameFA:         "خودرو",
			NameEN:         "Car",
			Brand:          &brand,
			ExteriorColors: []string{"سفید", "مشکی"},
			InteriorColors: []string{"مشکی"},
		})
	}
	return items
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCatalogList_CacheAside(t *testing.T) {
	repo := &mockCatalogRepo{items: catalogFixture(23)}
	cache := newMockCache()
	svc := service.NewCatalogService(repo, cache)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("first List: %v", err)
	}
	if len(got) != 23 {
		t.Fatalf("first List: want 23 cars, got %d", len(got))
	}
	if repo.calls != 1 {
		t.Fatalf("first List: want 1 repo call, got %d", repo.calls)
	}
	if !cache.has("catalog:cars") {
		t.Fatal("first List: cache not populated")
	}

	// Second call must be served from the cache — repo untouched.
	got, err = svc.List(context.Background())
	if err != nil {
		t.Fatalf("second List: %v", err)
	}
	if len(got) != 23 {
		t.Fatalf("second List: want 23 cars, got %d", len(got))
	}
	if repo.calls != 1 {
		t.Fatalf("second List: want repo calls to stay 1, got %d", repo.calls)
	}
	if got[0].ExteriorColors[0] != "سفید" {
		t.Fatalf("second List: colors lost through the cache round-trip: %v", got[0].ExteriorColors)
	}
}

func TestCatalogList_CorruptCacheRecomputes(t *testing.T) {
	repo := &mockCatalogRepo{items: catalogFixture(3)}
	cache := newMockCache()
	cache.store["catalog:cars"] = "{not json["
	svc := service.NewCatalogService(repo, cache)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List with corrupt cache: %v", err)
	}
	if len(got) != 3 || repo.calls != 1 {
		t.Fatalf("corrupt cache must fall through to the repo: cars=%d calls=%d", len(got), repo.calls)
	}
}

func TestCatalogList_RepoErrorPropagates(t *testing.T) {
	repo := &mockCatalogRepo{err: errors.New("boom")}
	svc := service.NewCatalogService(repo, newMockCache())

	if _, err := svc.List(context.Background()); err == nil {
		t.Fatal("want repo error to propagate, got nil")
	}
}
