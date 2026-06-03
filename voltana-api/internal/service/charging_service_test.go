package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"
	"voltana-api/internal/service"

	"github.com/google/uuid"
)

// ── mock ChargingRepository ───────────────────────────────────────────────────

type mockChargingRepo struct {
	store      map[uuid.UUID]domain.ChargingSession
	lastFilter domain.ChargingFilter
	lastLimit  int
	lastOffset int
}

func newMockChargingRepo() *mockChargingRepo {
	return &mockChargingRepo{store: make(map[uuid.UUID]domain.ChargingSession)}
}

func (m *mockChargingRepo) Create(_ context.Context, userID uuid.UUID, in domain.ChargingInput) (*domain.ChargingSession, error) {
	s := domain.ChargingSession{
		ID: uuid.New(), UserID: userID, CarID: in.CarID, StartedAt: in.StartedAt, EndedAt: in.EndedAt,
		Location: in.Location, KWhCharged: in.KWhCharged, EnergyPeakKWh: in.EnergyPeakKWh,
		EnergyMidKWh: in.EnergyMidKWh, EnergyOffpeakKWh: in.EnergyOffpeakKWh,
		StartSOC: in.StartSOC, EndSOC: in.EndSOC, Cost: in.Cost, Notes: in.Notes,
	}
	m.store[s.ID] = s
	return &s, nil
}

func (m *mockChargingRepo) ListByUser(_ context.Context, userID uuid.UUID, f domain.ChargingFilter, limit, offset int) ([]domain.ChargingSession, int, error) {
	m.lastFilter, m.lastLimit, m.lastOffset = f, limit, offset
	items := make([]domain.ChargingSession, 0)
	for _, s := range m.store {
		if s.UserID == userID {
			items = append(items, s)
		}
	}
	return items, len(items), nil
}

func (m *mockChargingRepo) GetByID(_ context.Context, userID, id uuid.UUID) (*domain.ChargingSession, error) {
	s, ok := m.store[id]
	if !ok || s.UserID != userID {
		return nil, repository.ErrNotFound
	}
	return &s, nil
}

func (m *mockChargingRepo) Update(_ context.Context, userID, id uuid.UUID, in domain.ChargingInput) (*domain.ChargingSession, error) {
	s, ok := m.store[id]
	if !ok || s.UserID != userID {
		return nil, repository.ErrNotFound
	}
	s.CarID, s.StartedAt, s.EndedAt, s.Cost = in.CarID, in.StartedAt, in.EndedAt, in.Cost
	s.EnergyPeakKWh, s.EnergyMidKWh, s.EnergyOffpeakKWh = in.EnergyPeakKWh, in.EnergyMidKWh, in.EnergyOffpeakKWh
	m.store[id] = s
	return &s, nil
}

func (m *mockChargingRepo) Delete(_ context.Context, userID, id uuid.UUID) error {
	s, ok := m.store[id]
	if !ok || s.UserID != userID {
		return repository.ErrNotFound
	}
	delete(m.store, id)
	return nil
}

func (m *mockChargingRepo) AggregateByUser(_ context.Context, userID uuid.UUID) (float64, float64, int, error) {
	var totalKWh, totalCost float64
	count := 0
	for _, s := range m.store {
		if s.UserID != userID {
			continue
		}
		count++
		if s.KWhCharged != nil {
			totalKWh += *s.KWhCharged
		}
		if s.Cost != nil {
			totalCost += *s.Cost
		}
	}
	return totalKWh, totalCost, count, nil
}

// ── mock SettingsRepository ───────────────────────────────────────────────────

type mockSettingsRepo struct {
	rates repository.Rates                  // backs GetRates (charging cost tests)
	store map[uuid.UUID]domain.UserSettings // backs GetOrCreate/Update (settings tests)
}

func (m *mockSettingsRepo) GetRates(_ context.Context, _ uuid.UUID) (repository.Rates, error) {
	return m.rates, nil
}

func (m *mockSettingsRepo) GetOrCreate(_ context.Context, userID uuid.UUID) (*domain.UserSettings, error) {
	if m.store == nil {
		m.store = make(map[uuid.UUID]domain.UserSettings)
	}
	s, ok := m.store[userID]
	if !ok {
		s = domain.UserSettings{ID: uuid.New(), UserID: userID}
		m.store[userID] = s
	}
	return &s, nil
}

func (m *mockSettingsRepo) Update(_ context.Context, userID uuid.UUID, in domain.SettingsInput) (*domain.UserSettings, error) {
	if m.store == nil {
		m.store = make(map[uuid.UUID]domain.UserSettings)
	}
	s := domain.UserSettings{
		ID: uuid.New(), UserID: userID, DefaultCarID: in.DefaultCarID,
		PeakRate: in.PeakRate, MidRate: in.MidRate, OffpeakRate: in.OffpeakRate,
	}
	m.store[userID] = s
	return &s, nil
}

// ── test harness ──────────────────────────────────────────────────────────────

// newChargingSvc wires the charging service with a car already owned by `owner`,
// returning the service, the car id, and the charging-repo mock for assertions.
func newChargingSvc(t *testing.T, owner uuid.UUID, rates repository.Rates) (*service.ChargingService, uuid.UUID, *mockChargingRepo) {
	t.Helper()
	carRepo := newMockCarRepo()
	car, err := carRepo.Create(context.Background(), owner, repository.CarInput{Name: "Daily"})
	if err != nil {
		t.Fatalf("seed car: %v", err)
	}
	chRepo := newMockChargingRepo()
	svc := service.NewChargingService(chRepo, carRepo, &mockSettingsRepo{rates: rates})
	return svc, car.ID, chRepo
}

func baseInput(carID uuid.UUID) domain.ChargingInput {
	return domain.ChargingInput{CarID: carID, StartedAt: time.Now()}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCharging_CostComputedFromPeriods(t *testing.T) {
	owner := uuid.New()
	svc, carID, _ := newChargingSvc(t, owner, repository.Rates{Peak: 10, Mid: 5, Offpeak: 2})

	in := baseInput(carID)
	in.EnergyPeakKWh, in.EnergyMidKWh, in.EnergyOffpeakKWh = ptr(1.0), ptr(2.0), ptr(3.0)
	// 1*10 + 2*5 + 3*2 = 26
	sess, err := svc.Create(context.Background(), owner, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess.Cost == nil || *sess.Cost != 26 {
		t.Errorf("cost = %v, want 26", sess.Cost)
	}
}

func TestCharging_ProvidedCostWins(t *testing.T) {
	owner := uuid.New()
	svc, carID, _ := newChargingSvc(t, owner, repository.Rates{Peak: 10, Mid: 5, Offpeak: 2})

	in := baseInput(carID)
	in.EnergyPeakKWh = ptr(5.0) // would compute 50, but explicit cost must win
	in.Cost = ptr(99.0)
	sess, err := svc.Create(context.Background(), owner, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess.Cost == nil || *sess.Cost != 99 {
		t.Errorf("cost = %v, want 99 (client value preserved)", sess.Cost)
	}
}

func TestCharging_NoEnergyNoCost(t *testing.T) {
	owner := uuid.New()
	svc, carID, _ := newChargingSvc(t, owner, repository.Rates{Peak: 10})

	sess, err := svc.Create(context.Background(), owner, baseInput(carID))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess.Cost != nil {
		t.Errorf("cost = %v, want nil when no energy and no cost given", *sess.Cost)
	}
}

func TestCharging_Validation(t *testing.T) {
	owner := uuid.New()
	svc, carID, _ := newChargingSvc(t, owner, repository.Rates{})

	start := time.Now()
	cases := map[string]domain.ChargingInput{
		"missing car":     {StartedAt: start},
		"missing started": {CarID: carID},
		"start_soc high":  {CarID: carID, StartedAt: start, StartSOC: ptr(150)},
		"end_soc neg":     {CarID: carID, StartedAt: start, EndSOC: ptr(-1)},
		"end before start": {CarID: carID, StartedAt: start, EndedAt: ptr(start.Add(-time.Hour))},
		"negative kwh":    {CarID: carID, StartedAt: start, KWhCharged: ptr(-1.0)},
	}
	for name, in := range cases {
		if _, err := svc.Create(context.Background(), owner, in); !errors.Is(err, service.ErrValidation) {
			t.Errorf("%s: want ErrValidation, got %v", name, err)
		}
	}
}

func TestCharging_InvalidCarRef(t *testing.T) {
	owner := uuid.New()
	svc, _, _ := newChargingSvc(t, owner, repository.Rates{})

	in := baseInput(uuid.New()) // a car id the user does not own
	if _, err := svc.Create(context.Background(), owner, in); !errors.Is(err, service.ErrInvalidCarRef) {
		t.Errorf("want ErrInvalidCarRef, got %v", err)
	}
}

func TestCharging_OwnershipIsolation(t *testing.T) {
	owner, attacker := uuid.New(), uuid.New()
	svc, carID, _ := newChargingSvc(t, owner, repository.Rates{})

	sess, err := svc.Create(context.Background(), owner, baseInput(carID))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Get(context.Background(), attacker, sess.ID); !errors.Is(err, service.ErrChargingNotFound) {
		t.Errorf("Get cross-user: want ErrChargingNotFound, got %v", err)
	}
	// A cross-user update must be 404 (session-ownership) — not 422 — even when the
	// body references a car the attacker does not own.
	if _, err := svc.Update(context.Background(), attacker, sess.ID, baseInput(carID)); !errors.Is(err, service.ErrChargingNotFound) {
		t.Errorf("Update cross-user: want ErrChargingNotFound, got %v", err)
	}
	if err := svc.Delete(context.Background(), attacker, sess.ID); !errors.Is(err, service.ErrChargingNotFound) {
		t.Errorf("Delete cross-user: want ErrChargingNotFound, got %v", err)
	}
	// owner can still read it
	if _, err := svc.Get(context.Background(), owner, sess.ID); err != nil {
		t.Errorf("owner Get: %v", err)
	}
}

func TestCharging_ListFilterAndPaginationClamp(t *testing.T) {
	owner := uuid.New()
	svc, carID, repo := newChargingSvc(t, owner, repository.Rates{})

	f := domain.ChargingFilter{CarID: &carID}
	if _, _, err := svc.List(context.Background(), owner, f, 1000, -5); err != nil {
		t.Fatal(err)
	}
	if repo.lastLimit != 100 {
		t.Errorf("limit should clamp to 100, got %d", repo.lastLimit)
	}
	if repo.lastOffset != 0 {
		t.Errorf("negative offset should clamp to 0, got %d", repo.lastOffset)
	}
	if repo.lastFilter.CarID == nil || *repo.lastFilter.CarID != carID {
		t.Errorf("car_id filter not passed through to repository")
	}
}
