package service_test

import (
	"context"
	"errors"
	"sort"
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
		OdometerKM: in.OdometerKM,
		RatePeakAtTime: in.RatePeakAtTime, RateMidAtTime: in.RateMidAtTime, RateOffpeakAtTime: in.RateOffpeakAtTime,
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
	// Mirror the SQL window: per car (by start time) set PrevOdometerKM to the
	// previous session's odometer, so the service's efficiency calc is exercised.
	order := append([]domain.ChargingSession(nil), items...)
	sort.Slice(order, func(i, j int) bool { return order[i].StartedAt.Before(order[j].StartedAt) })
	prevByCar := map[uuid.UUID]*int{}
	prevOf := map[uuid.UUID]*int{}
	for i := range order {
		car := order[i].CarID
		if p, ok := prevByCar[car]; ok {
			prevOf[order[i].ID] = p
		}
		prevByCar[car] = order[i].OdometerKM
	}
	for i := range items {
		if p, ok := prevOf[items[i].ID]; ok {
			items[i].PrevOdometerKM = p
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
	s.OdometerKM, s.KWhCharged = in.OdometerKM, in.KWhCharged
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

// EfficiencyAggregateByUser mirrors the SQL: per car, ordered by start time, sum
// energy + (odometer - prev odometer) over consecutive pairs with both readings,
// a positive delta, and known energy.
func (m *mockChargingRepo) EfficiencyAggregateByUser(_ context.Context, userID uuid.UUID) (float64, float64, error) {
	byCar := map[uuid.UUID][]domain.ChargingSession{}
	for _, s := range m.store {
		if s.UserID == userID {
			byCar[s.CarID] = append(byCar[s.CarID], s)
		}
	}
	var sumKWh, sumKM float64
	for _, list := range byCar {
		sort.Slice(list, func(i, j int) bool { return list[i].StartedAt.Before(list[j].StartedAt) })
		for i := 1; i < len(list); i++ {
			prev, cur := list[i-1], list[i]
			if prev.OdometerKM == nil || cur.OdometerKM == nil || cur.KWhCharged == nil {
				continue
			}
			km := *cur.OdometerKM - *prev.OdometerKM
			if km <= 0 {
				continue
			}
			sumKWh += *cur.KWhCharged
			sumKM += float64(km)
		}
	}
	return sumKWh, sumKM, nil
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
		// Mirror the real repo: a fresh row is seeded with the (admin default)
		// rates — here the harness rates double as those defaults.
		s = domain.UserSettings{ID: uuid.New(), UserID: userID,
			PeakRate: m.rates.Peak, MidRate: m.rates.Mid, OffpeakRate: m.rates.Offpeak}
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

func TestCharging_PerSessionEfficiency(t *testing.T) {
	owner := uuid.New()
	svc, carID, _ := newChargingSvc(t, owner, repository.Rates{})
	ctx := context.Background()

	// Session 1: odometer 1000, earlier.
	in1 := domain.ChargingInput{CarID: carID, StartedAt: time.Now().Add(-2 * time.Hour), OdometerKM: ptr(1000)}
	if _, err := svc.Create(ctx, owner, in1); err != nil {
		t.Fatalf("create s1: %v", err)
	}
	// Session 2: odometer 1300 (300 km later), 45 kWh → 45 / (300/100) = 15.0 kWh/100km.
	in2 := domain.ChargingInput{CarID: carID, StartedAt: time.Now(), OdometerKM: ptr(1300), KWhCharged: ptr(45.0)}
	if _, err := svc.Create(ctx, owner, in2); err != nil {
		t.Fatalf("create s2: %v", err)
	}

	items, _, err := svc.List(ctx, owner, domain.ChargingFilter{}, 100, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var s1, s2 *domain.ChargingSession
	for i := range items {
		switch *items[i].OdometerKM {
		case 1000:
			s1 = &items[i]
		case 1300:
			s2 = &items[i]
		}
	}
	if s1 == nil || s2 == nil {
		t.Fatal("expected both sessions in list")
	}
	if s1.EfficiencyKWhPer100km != nil {
		t.Errorf("s1 has no prior reading → efficiency should be nil, got %v", *s1.EfficiencyKWhPer100km)
	}
	if s2.EfficiencyKWhPer100km == nil || *s2.EfficiencyKWhPer100km != 15.0 {
		t.Errorf("s2 efficiency: want 15.0, got %v", s2.EfficiencyKWhPer100km)
	}
}

func TestCharging_EfficiencyNilWhenOdometerNotIncreasing(t *testing.T) {
	owner := uuid.New()
	svc, carID, _ := newChargingSvc(t, owner, repository.Rates{})
	ctx := context.Background()
	// Same odometer (km_driven = 0) → no efficiency.
	if _, err := svc.Create(ctx, owner, domain.ChargingInput{CarID: carID, StartedAt: time.Now().Add(-time.Hour), OdometerKM: ptr(500)}); err != nil {
		t.Fatalf("create s1: %v", err)
	}
	if _, err := svc.Create(ctx, owner, domain.ChargingInput{CarID: carID, StartedAt: time.Now(), OdometerKM: ptr(500), KWhCharged: ptr(20.0)}); err != nil {
		t.Fatalf("create s2: %v", err)
	}
	items, _, err := svc.List(ctx, owner, domain.ChargingFilter{}, 100, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for i := range items {
		if items[i].EfficiencyKWhPer100km != nil {
			t.Errorf("zero distance → efficiency must be nil, got %v", *items[i].EfficiencyKWhPer100km)
		}
	}
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


// ── rate snapshots (TASK-0037 FEAT-6) ─────────────────────────────────────────

func TestCreate_SnapshotsRatesAtCreation(t *testing.T) {
	owner := uuid.New()
	svc, carID, _ := newChargingSvc(t, owner, repository.Rates{Peak: 5000, Mid: 3000, Offpeak: 1000})

	in := baseInput(carID)
	e := 10.0
	in.EnergyPeakKWh = &e
	sess, err := svc.Create(context.Background(), owner, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess.RatePeakAtTime == nil || *sess.RatePeakAtTime != 5000 ||
		sess.RateMidAtTime == nil || *sess.RateMidAtTime != 3000 ||
		sess.RateOffpeakAtTime == nil || *sess.RateOffpeakAtTime != 1000 {
		t.Errorf("snapshot rates = %v/%v/%v, want 5000/3000/1000",
			sess.RatePeakAtTime, sess.RateMidAtTime, sess.RateOffpeakAtTime)
	}
	if sess.Cost == nil || *sess.Cost != 50000 {
		t.Errorf("cost = %v, want 50000 (10 kWh × 5000)", sess.Cost)
	}
}

func TestUpdate_RecomputesWithFrozenRates(t *testing.T) {
	owner := uuid.New()
	carRepo := newMockCarRepo()
	car, _ := carRepo.Create(context.Background(), owner, repository.CarInput{Name: "Daily"})
	chRepo := newMockChargingRepo()
	settings := &mockSettingsRepo{rates: repository.Rates{Peak: 5000, Mid: 3000, Offpeak: 1000}}
	svc := service.NewChargingService(chRepo, carRepo, settings)

	in := baseInput(car.ID)
	e := 10.0
	in.EnergyPeakKWh = &e
	sess, err := svc.Create(context.Background(), owner, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// The user's rates change AFTER the session was created.
	settings.rates = repository.Rates{Peak: 9000, Mid: 7000, Offpeak: 5000}
	if cur, ok := settings.store[owner]; ok {
		cur.PeakRate, cur.MidRate, cur.OffpeakRate = 9000, 7000, 5000
		settings.store[owner] = cur
	}

	// Edit the session (more energy, no manual cost) → cost must use the
	// FROZEN 5000 peak rate, not the new 9000.
	upd := baseInput(car.ID)
	e2 := 20.0
	upd.EnergyPeakKWh = &e2
	updated, err := svc.Update(context.Background(), owner, sess.ID, upd)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Cost == nil || *updated.Cost != 100000 {
		t.Errorf("updated cost = %v, want 100000 (20 kWh × frozen 5000, not 180000 with current 9000)", updated.Cost)
	}
	if updated.RatePeakAtTime == nil || *updated.RatePeakAtTime != 5000 {
		t.Errorf("snapshot must stay frozen at 5000, got %v", updated.RatePeakAtTime)
	}

	// A NEW session after the rate change snapshots the NEW rates.
	in2 := baseInput(car.ID)
	e3 := 10.0
	in2.EnergyPeakKWh = &e3
	sess2, err := svc.Create(context.Background(), owner, in2)
	if err != nil {
		t.Fatalf("Create 2nd: %v", err)
	}
	// owner's settings row already exists with the updated rates.
	if sess2.Cost == nil || *sess2.Cost != 90000 {
		t.Errorf("2nd session cost = %v, want 90000 (10 kWh × current 9000)", sess2.Cost)
	}
	if sess2.RatePeakAtTime == nil || *sess2.RatePeakAtTime != 9000 {
		t.Errorf("2nd session snapshot = %v, want 9000", sess2.RatePeakAtTime)
	}
}
