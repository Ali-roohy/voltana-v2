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

// ── mock CarRepository ──────────────────────────────────────────────────────

type mockCarRepo struct {
	store      map[uuid.UUID]domain.Car
	invalidEV  bool // simulate FK violation when an ev_model_id is supplied
	lastLimit  int
	lastOffset int
}

func newMockCarRepo() *mockCarRepo {
	return &mockCarRepo{store: make(map[uuid.UUID]domain.Car)}
}

func (m *mockCarRepo) Create(_ context.Context, userID uuid.UUID, in repository.CarInput) (*domain.Car, error) {
	if m.invalidEV && in.EVModelID != nil {
		return nil, repository.ErrInvalidEVModel
	}
	c := domain.Car{ID: uuid.New(), UserID: userID, Name: in.Name, LicensePlate: in.LicensePlate,
		OdometerKM: in.OdometerKM, EVModelID: in.EVModelID}
	m.store[c.ID] = c
	return &c, nil
}

func (m *mockCarRepo) ListByUser(_ context.Context, userID uuid.UUID, limit, offset int) ([]domain.Car, int, error) {
	m.lastLimit, m.lastOffset = limit, offset
	items := make([]domain.Car, 0)
	for _, c := range m.store {
		if c.UserID == userID {
			items = append(items, c)
		}
	}
	return items, len(items), nil
}

func (m *mockCarRepo) GetByID(_ context.Context, userID, id uuid.UUID) (*domain.Car, error) {
	c, ok := m.store[id]
	if !ok || c.UserID != userID {
		return nil, repository.ErrNotFound
	}
	return &c, nil
}

func (m *mockCarRepo) Update(_ context.Context, userID, id uuid.UUID, in repository.CarInput) (*domain.Car, error) {
	c, ok := m.store[id]
	if !ok || c.UserID != userID {
		return nil, repository.ErrNotFound
	}
	if m.invalidEV && in.EVModelID != nil {
		return nil, repository.ErrInvalidEVModel
	}
	c.Name, c.OdometerKM, c.LicensePlate, c.EVModelID = in.Name, in.OdometerKM, in.LicensePlate, in.EVModelID
	m.store[id] = c
	return &c, nil
}

func (m *mockCarRepo) Delete(_ context.Context, userID, id uuid.UUID) error {
	c, ok := m.store[id]
	if !ok || c.UserID != userID {
		return repository.ErrNotFound
	}
	delete(m.store, id)
	return nil
}

// ── tests ───────────────────────────────────────────────────────────────────

func TestCar_CreateSuccess(t *testing.T) {
	svc := service.NewCarService(newMockCarRepo())
	uid := uuid.New()
	car, err := svc.Create(context.Background(), uid, repository.CarInput{Name: "  Daily  ", OdometerKM: 100})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if car.Name != "Daily" {
		t.Errorf("name should be trimmed, got %q", car.Name)
	}
	if car.UserID != uid {
		t.Error("car must belong to creating user")
	}
}

func TestCar_CreateValidation(t *testing.T) {
	svc := service.NewCarService(newMockCarRepo())
	cases := []repository.CarInput{
		{Name: ""},                                   // empty name
		{Name: "ok", OdometerKM: -1},                 // negative odometer
		{Name: "ok", LicensePlate: ptr(longString())}, // license too long
	}
	for i, in := range cases {
		if _, err := svc.Create(context.Background(), uuid.New(), in); !errors.Is(err, service.ErrValidation) {
			t.Errorf("case %d: want ErrValidation, got %v", i, err)
		}
	}
}

func TestCar_CreateInvalidEVModel(t *testing.T) {
	repo := newMockCarRepo()
	repo.invalidEV = true
	svc := service.NewCarService(repo)
	evID := uuid.New()
	_, err := svc.Create(context.Background(), uuid.New(), repository.CarInput{Name: "x", EVModelID: &evID})
	if !errors.Is(err, service.ErrInvalidEVModelRef) {
		t.Errorf("want ErrInvalidEVModelRef, got %v", err)
	}
}

func TestCar_OwnershipIsolation(t *testing.T) {
	repo := newMockCarRepo()
	svc := service.NewCarService(repo)
	owner, attacker := uuid.New(), uuid.New()

	car, err := svc.Create(context.Background(), owner, repository.CarInput{Name: "Owned"})
	if err != nil {
		t.Fatal(err)
	}

	// attacker must not be able to read / update / delete the owner's car
	if _, err := svc.Get(context.Background(), attacker, car.ID); !errors.Is(err, service.ErrCarNotFound) {
		t.Errorf("Get cross-user: want ErrCarNotFound, got %v", err)
	}
	if _, err := svc.Update(context.Background(), attacker, car.ID, repository.CarInput{Name: "Hijack"}); !errors.Is(err, service.ErrCarNotFound) {
		t.Errorf("Update cross-user: want ErrCarNotFound, got %v", err)
	}
	if err := svc.Delete(context.Background(), attacker, car.ID); !errors.Is(err, service.ErrCarNotFound) {
		t.Errorf("Delete cross-user: want ErrCarNotFound, got %v", err)
	}

	// owner still can
	if _, err := svc.Get(context.Background(), owner, car.ID); err != nil {
		t.Errorf("owner Get: %v", err)
	}
}

func TestCar_PaginationClamp(t *testing.T) {
	repo := newMockCarRepo()
	svc := service.NewCarService(repo)
	uid := uuid.New()

	if _, _, err := svc.List(context.Background(), uid, 1000, -5); err != nil {
		t.Fatal(err)
	}
	if repo.lastLimit != 100 {
		t.Errorf("limit should clamp to 100, got %d", repo.lastLimit)
	}
	if repo.lastOffset != 0 {
		t.Errorf("negative offset should clamp to 0, got %d", repo.lastOffset)
	}

	if _, _, err := svc.List(context.Background(), uid, 0, 0); err != nil {
		t.Fatal(err)
	}
	if repo.lastLimit != 20 {
		t.Errorf("missing limit should default to 20, got %d", repo.lastLimit)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func ptr[T any](v T) *T { return &v }

func longString() string {
	b := make([]byte, 51)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}
