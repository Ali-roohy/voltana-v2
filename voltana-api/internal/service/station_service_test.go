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

// ── mock StationRepository ──────────────────────────────────────────────────

type mockStationRepo struct {
	store      map[uuid.UUID]domain.ChargingStation
	lastBounds *domain.StationBounds
}

func newMockStationRepo() *mockStationRepo {
	return &mockStationRepo{store: make(map[uuid.UUID]domain.ChargingStation)}
}

func (m *mockStationRepo) Create(_ context.Context, in domain.StationInput) (*domain.ChargingStation, error) {
	s := domain.ChargingStation{
		ID: uuid.New(), Name: in.Name, Latitude: in.Latitude, Longitude: in.Longitude,
		Address: in.Address, ConnectorTypes: in.ConnectorTypes, PowerKW: in.PowerKW, Operator: in.Operator,
	}
	m.store[s.ID] = s
	return &s, nil
}

func (m *mockStationRepo) List(_ context.Context, b *domain.StationBounds) ([]domain.StationMarker, error) {
	m.lastBounds = b
	items := make([]domain.StationMarker, 0)
	for _, s := range m.store {
		items = append(items, domain.StationMarker{ID: s.ID, Name: s.Name, Latitude: s.Latitude, Longitude: s.Longitude})
	}
	return items, nil
}

func (m *mockStationRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.ChargingStation, error) {
	s, ok := m.store[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &s, nil
}

func (m *mockStationRepo) Update(_ context.Context, id uuid.UUID, in domain.StationInput) (*domain.ChargingStation, error) {
	s, ok := m.store[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	s.Name, s.Latitude, s.Longitude = in.Name, in.Latitude, in.Longitude
	s.Address, s.ConnectorTypes, s.PowerKW, s.Operator = in.Address, in.ConnectorTypes, in.PowerKW, in.Operator
	m.store[id] = s
	return &s, nil
}

func (m *mockStationRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.store[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.store, id)
	return nil
}

// ── tests ───────────────────────────────────────────────────────────────────

func TestStation_CreateSuccess(t *testing.T) {
	svc := service.NewStationService(newMockStationRepo())
	st, err := svc.Create(context.Background(), domain.StationInput{Name: "  Valiasr  ", Latitude: 35.7, Longitude: 51.4})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if st.Name != "Valiasr" {
		t.Errorf("name should be trimmed, got %q", st.Name)
	}
}

func TestStation_CreateAtEquatorAndPrimeMeridian(t *testing.T) {
	// latitude:0 / longitude:0 must be accepted — the service validates bounds
	// itself precisely because gin binding's `required` would reject a 0 here.
	svc := service.NewStationService(newMockStationRepo())
	if _, err := svc.Create(context.Background(), domain.StationInput{Name: "Null Island", Latitude: 0, Longitude: 0}); err != nil {
		t.Errorf("0,0 should be valid, got %v", err)
	}
}

func TestStation_CreateValidation(t *testing.T) {
	svc := service.NewStationService(newMockStationRepo())
	cases := []domain.StationInput{
		{Name: "", Latitude: 35, Longitude: 51},               // empty name
		{Name: "x", Latitude: 91, Longitude: 51},              // lat too high
		{Name: "x", Latitude: -91, Longitude: 51},             // lat too low
		{Name: "x", Latitude: 35, Longitude: 181},             // lng too high
		{Name: "x", Latitude: 35, Longitude: -181},            // lng too low
		{Name: "x", Latitude: 35, Longitude: 51, PowerKW: ptr(0)}, // non-positive power
	}
	for i, in := range cases {
		if _, err := svc.Create(context.Background(), in); !errors.Is(err, service.ErrValidation) {
			t.Errorf("case %d: want ErrValidation, got %v", i, err)
		}
	}
}

func TestStation_GetNotFound(t *testing.T) {
	svc := service.NewStationService(newMockStationRepo())
	if _, err := svc.Get(context.Background(), uuid.New()); !errors.Is(err, service.ErrStationNotFound) {
		t.Errorf("want ErrStationNotFound, got %v", err)
	}
}

func TestStation_UpdateDeleteNotFound(t *testing.T) {
	svc := service.NewStationService(newMockStationRepo())
	if _, err := svc.Update(context.Background(), uuid.New(), domain.StationInput{Name: "x", Latitude: 1, Longitude: 1}); !errors.Is(err, service.ErrStationNotFound) {
		t.Errorf("Update: want ErrStationNotFound, got %v", err)
	}
	if err := svc.Delete(context.Background(), uuid.New()); !errors.Is(err, service.ErrStationNotFound) {
		t.Errorf("Delete: want ErrStationNotFound, got %v", err)
	}
}

func TestStation_ListBounds(t *testing.T) {
	repo := newMockStationRepo()
	svc := service.NewStationService(repo)

	// nil bounds → full set, repo sees nil
	if _, err := svc.List(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if repo.lastBounds != nil {
		t.Error("nil bounds should pass through as nil")
	}

	// valid bounds pass through
	good := &domain.StationBounds{MinLat: 35, MaxLat: 36, MinLng: 51, MaxLng: 52}
	if _, err := svc.List(context.Background(), good); err != nil {
		t.Fatalf("valid bounds: %v", err)
	}

	// min > max is rejected
	bad := &domain.StationBounds{MinLat: 36, MaxLat: 35, MinLng: 51, MaxLng: 52}
	if _, err := svc.List(context.Background(), bad); !errors.Is(err, service.ErrValidation) {
		t.Errorf("inverted bounds: want ErrValidation, got %v", err)
	}

	// out-of-range box is rejected
	oob := &domain.StationBounds{MinLat: -100, MaxLat: 36, MinLng: 51, MaxLng: 52}
	if _, err := svc.List(context.Background(), oob); !errors.Is(err, service.ErrValidation) {
		t.Errorf("out-of-range bounds: want ErrValidation, got %v", err)
	}
}
