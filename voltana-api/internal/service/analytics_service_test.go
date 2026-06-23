package service_test

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"
	"voltana-api/internal/service"

	"github.com/google/uuid"
)

// ── mock BatteryRepository ────────────────────────────────────────────────────

type mockBatteryRepo struct {
	mu    sync.Mutex
	store map[uuid.UUID][]domain.BatteryHealthSnapshot // all snapshots by car_id, in save order
	saves int
}

func newMockBatteryRepo() *mockBatteryRepo {
	return &mockBatteryRepo{store: make(map[uuid.UUID][]domain.BatteryHealthSnapshot)}
}

func (m *mockBatteryRepo) Save(_ context.Context, snap *domain.BatteryHealthSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap.ID = uuid.New()
	m.store[snap.CarID] = append(m.store[snap.CarID], *snap)
	m.saves++
	return nil
}

func (m *mockBatteryRepo) GetLatest(_ context.Context, userID, carID uuid.UUID) (*domain.BatteryHealthSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var latest *domain.BatteryHealthSnapshot
	for i := range m.store[carID] {
		s := m.store[carID][i]
		if s.UserID != userID {
			continue
		}
		if latest == nil || s.ComputedAt.After(latest.ComputedAt) {
			cp := s
			latest = &cp
		}
	}
	if latest == nil {
		return nil, repository.ErrNotFound
	}
	return latest, nil
}

// ListByCar mirrors the pgx repo: the newest `limit` snapshots, returned in
// chronological (ASC) order.
func (m *mockBatteryRepo) ListByCar(_ context.Context, userID, carID uuid.UUID, limit int) ([]domain.BatteryHealthSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	owned := make([]domain.BatteryHealthSnapshot, 0)
	for _, s := range m.store[carID] {
		if s.UserID == userID {
			owned = append(owned, s)
		}
	}
	sort.Slice(owned, func(i, j int) bool { return owned[i].ComputedAt.Before(owned[j].ComputedAt) })
	if len(owned) > limit {
		owned = owned[len(owned)-limit:] // keep the newest `limit`, still ASC
	}
	return owned, nil
}

func (m *mockBatteryRepo) saveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saves
}

// ── mock CacheStore ───────────────────────────────────────────────────────────

type mockCache struct {
	mu    sync.Mutex
	store map[string]string
	gets  int
	sets  int
	dels  int
}

func newMockCache() *mockCache { return &mockCache{store: make(map[string]string)} }

func (m *mockCache) CacheGet(_ context.Context, key string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gets++
	v, ok := m.store[key]
	return v, ok, nil
}

func (m *mockCache) CacheSet(_ context.Context, key, val string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sets++
	m.store[key] = val
	return nil
}

func (m *mockCache) CacheDel(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dels++
	delete(m.store, key)
	return nil
}

func (m *mockCache) has(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.store[key]
	return ok
}

// ── harness ───────────────────────────────────────────────────────────────────

// newAnalyticsSvc wires the analytics service with a car owned by `owner` linked to
// an ev_model of the given capacity/chemistry (pass capacity=nil to seed a car with
// NO model). Returns the service, the car id, the charging repo (to seed sessions),
// and the battery repo (to assert persistence).
func newAnalyticsSvc(t *testing.T, owner uuid.UUID, capacity *float64, chemistry *string) (*service.AnalyticsService, uuid.UUID, *mockChargingRepo, *mockBatteryRepo) {
	t.Helper()
	carRepo := newMockCarRepo()
	evRepo := newMockEVRepo()

	in := repository.CarInput{Name: "Daily"}
	if capacity != nil {
		modelID := uuid.New()
		evRepo.models[modelID] = domain.EVModel{ID: modelID, BatteryCapacityKWh: capacity, Chemistry: chemistry}
		in.EVModelID = &modelID
	}
	car, err := carRepo.Create(context.Background(), owner, in)
	if err != nil {
		t.Fatalf("seed car: %v", err)
	}

	chRepo := newMockChargingRepo()
	batRepo := newMockBatteryRepo()
	svc := service.NewAnalyticsService(carRepo, evRepo, &mockCatalogRepo{}, chRepo, batRepo, newMockCache())
	return svc, car.ID, chRepo, batRepo
}

// newDashboardSvc wires an analytics service exposing the car repo (to seed odometer)
// and the cache (to assert hit/miss/invalidation) for the dashboard tests.
func newDashboardSvc(t *testing.T) (*service.AnalyticsService, *mockCarRepo, *mockChargingRepo, *mockCache) {
	t.Helper()
	carRepo := newMockCarRepo()
	chRepo := newMockChargingRepo()
	cache := newMockCache()
	svc := service.NewAnalyticsService(carRepo, newMockEVRepo(), &mockCatalogRepo{}, chRepo, newMockBatteryRepo(), cache)
	return svc, carRepo, chRepo, cache
}

func seedSession(t *testing.T, chRepo *mockChargingRepo, owner, carID uuid.UUID, startSOC, endSOC int, kwh float64) {
	t.Helper()
	_, err := chRepo.Create(context.Background(), owner, domain.ChargingInput{
		CarID: carID, StartedAt: time.Now(),
		StartSOC: ptr(startSOC), EndSOC: ptr(endSOC), KWhCharged: ptr(kwh),
	})
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestAnalytics_SOHFromDeltaSOC(t *testing.T) {
	owner := uuid.New()
	svc, carID, chRepo, batRepo := newAnalyticsSvc(t, owner, ptr(60.0), ptr("NMC"))

	// 5 identical qualifying sessions: Δsoc=50, kwh=30 →
	//   cap = (30 * 0.88) / 0.50 = 52.8 kWh ; SOH = 100 * 52.8 / 60 = 88.0%
	for i := 0; i < 5; i++ {
		seedSession(t, chRepo, owner, carID, 20, 70, 30)
	}

	res, err := svc.GetBattery(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetBattery: %v", err)
	}
	if res.Snapshot == nil {
		t.Fatalf("expected a snapshot, got insufficient-data (qualifying=%d)", res.QualifyingSessions)
	}
	if res.Snapshot.SOHPct != 88.0 {
		t.Errorf("soh = %v, want 88.0", res.Snapshot.SOHPct)
	}
	if res.Snapshot.EstimatedCapacityKWh != 52.8 {
		t.Errorf("estimated capacity = %v, want 52.8", res.Snapshot.EstimatedCapacityKWh)
	}
	if res.Snapshot.SampleSessionCount != 5 {
		t.Errorf("sample count = %d, want 5", res.Snapshot.SampleSessionCount)
	}
	if res.Snapshot.Confidence != "medium" { // 5 sessions < high threshold (8)
		t.Errorf("confidence = %q, want medium", res.Snapshot.Confidence)
	}
	if batRepo.saveCount() != 1 {
		t.Errorf("expected the on-demand estimate to be persisted once, got %d saves", batRepo.saveCount())
	}
}

// BUG-5: SOH is no longer clamped to 100% — a battery measuring above nameplate
// (regen/downhill/no-HVAC) legitimately estimates above 100%.
func TestAnalytics_SOHAllowedAbove100(t *testing.T) {
	owner := uuid.New()
	svc, carID, chRepo, _ := newAnalyticsSvc(t, owner, ptr(60.0), ptr("LFP"))

	// Δsoc=50, kwh=50 → raw cap = (50*0.88)/0.5 = 88 kWh > 60 nominal → SOH = 100*88/60 = 146.67.
	for i := 0; i < 5; i++ {
		seedSession(t, chRepo, owner, carID, 10, 60, 50)
	}
	res, err := svc.GetBattery(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetBattery: %v", err)
	}
	if res.Snapshot == nil || res.Snapshot.SOHPct != 146.67 {
		t.Fatalf("soh = %v, want 146.67 (unclamped)", res.Snapshot)
	}
	if res.Snapshot.EstimatedCapacityKWh != 88 {
		t.Errorf("estimated capacity = %v, want 88 (unclamped)", res.Snapshot.EstimatedCapacityKWh)
	}
}

func TestAnalytics_SOHLowerFloorAvoidsZero(t *testing.T) {
	owner := uuid.New()
	svc, carID, chRepo, batRepo := newAnalyticsSvc(t, owner, ptr(60.0), ptr("NMC"))

	// Pathologically tiny energy: Δsoc=25 qualifies, but cap = (0.0005*0.88)/0.25 ≈ 0.00176,
	// SOH = 100*0.00176/60 ≈ 0.0029 — which rounds to 0.00 WITHOUT the floor and would trip
	// the DB CHECK (soh_pct > 0). The floor must keep it at 0.01.
	for i := 0; i < 5; i++ {
		seedSession(t, chRepo, owner, carID, 0, 25, 0.0005)
	}
	res, err := svc.GetBattery(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetBattery: %v", err)
	}
	if res.Snapshot == nil {
		t.Fatalf("expected a snapshot (5 qualifying), got insufficient (qualifying=%d)", res.QualifyingSessions)
	}
	if res.Snapshot.SOHPct <= 0 {
		t.Errorf("soh_pct = %v, must be > 0 (DB CHECK); the lower floor should clamp it", res.Snapshot.SOHPct)
	}
	if res.Snapshot.SOHPct != 0.01 {
		t.Errorf("soh_pct = %v, want 0.01 (floored)", res.Snapshot.SOHPct)
	}
	if batRepo.saveCount() != 1 {
		t.Errorf("floored snapshot should still persist once, got %d saves", batRepo.saveCount())
	}
}

func TestAnalytics_HighConfidence(t *testing.T) {
	owner := uuid.New()
	svc, carID, chRepo, _ := newAnalyticsSvc(t, owner, ptr(60.0), ptr("NMC"))
	for i := 0; i < 8; i++ { // ≥8 sessions, avg Δsoc 50 ≥ 40 → high
		seedSession(t, chRepo, owner, carID, 20, 70, 30)
	}
	res, err := svc.GetBattery(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetBattery: %v", err)
	}
	if res.Snapshot == nil || res.Snapshot.Confidence != "high" {
		t.Errorf("confidence = %v, want high", res.Snapshot)
	}
}

func TestAnalytics_InsufficientWhenTooFewQualifying(t *testing.T) {
	owner := uuid.New()
	svc, carID, chRepo, batRepo := newAnalyticsSvc(t, owner, ptr(60.0), ptr("NMC"))

	seedSession(t, chRepo, owner, carID, 20, 70, 30) // 3 qualifying
	seedSession(t, chRepo, owner, carID, 20, 70, 30)
	seedSession(t, chRepo, owner, carID, 20, 70, 30)

	res, err := svc.GetBattery(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetBattery: %v", err)
	}
	if res.Snapshot != nil {
		t.Errorf("expected insufficient-data, got snapshot %+v", res.Snapshot)
	}
	if res.QualifyingSessions != 3 {
		t.Errorf("qualifying = %d, want 3", res.QualifyingSessions)
	}
	if batRepo.saveCount() != 0 {
		t.Errorf("insufficient data must not persist a snapshot, got %d saves", batRepo.saveCount())
	}
}

func TestAnalytics_DeltaSOCFilterExcludesSmallSwings(t *testing.T) {
	owner := uuid.New()
	svc, carID, chRepo, _ := newAnalyticsSvc(t, owner, ptr(60.0), ptr("NMC"))

	// 6 sessions but all Δsoc=20 (< 25 threshold) → none qualify → insufficient.
	for i := 0; i < 6; i++ {
		seedSession(t, chRepo, owner, carID, 40, 60, 12)
	}
	res, err := svc.GetBattery(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetBattery: %v", err)
	}
	if res.Snapshot != nil || res.QualifyingSessions != 0 {
		t.Errorf("small swings should not qualify; got snapshot=%v qualifying=%d", res.Snapshot, res.QualifyingSessions)
	}
}

func TestAnalytics_NullEVModelIsInsufficient(t *testing.T) {
	owner := uuid.New()
	svc, carID, chRepo, _ := newAnalyticsSvc(t, owner, nil, nil) // car with NO ev_model

	for i := 0; i < 6; i++ { // plenty of qualifying sessions, but no nominal capacity
		seedSession(t, chRepo, owner, carID, 20, 70, 30)
	}
	res, err := svc.GetBattery(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetBattery: %v", err)
	}
	if res.Snapshot != nil {
		t.Errorf("no ev_model → no nominal capacity → insufficient; got %+v", res.Snapshot)
	}
}

func TestAnalytics_OwnershipIsolation(t *testing.T) {
	owner, attacker := uuid.New(), uuid.New()
	svc, carID, chRepo, _ := newAnalyticsSvc(t, owner, ptr(60.0), ptr("NMC"))
	for i := 0; i < 5; i++ {
		seedSession(t, chRepo, owner, carID, 20, 70, 30)
	}

	if _, err := svc.GetBattery(context.Background(), attacker, carID); !errors.Is(err, service.ErrCarNotFound) {
		t.Errorf("battery cross-user: want ErrCarNotFound, got %v", err)
	}
	if _, err := svc.GetRecommendations(context.Background(), attacker, carID); !errors.Is(err, service.ErrCarNotFound) {
		t.Errorf("recommendations cross-user: want ErrCarNotFound, got %v", err)
	}
	if _, err := svc.GetBattery(context.Background(), owner, uuid.New()); !errors.Is(err, service.ErrCarNotFound) {
		t.Errorf("unknown car: want ErrCarNotFound, got %v", err)
	}
}

func TestAnalytics_RecommendationsByChemistry(t *testing.T) {
	owner := uuid.New()
	cases := []struct {
		name      string
		chemistry *string
		ceiling   int
	}{
		{"LFP", ptr("LFP"), 100},
		{"NMC", ptr("NMC"), 80},
		{"NCA", ptr("NCA"), 80},
		{"nil chemistry", nil, 80},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, carID, _, _ := newAnalyticsSvc(t, owner, ptr(60.0), tc.chemistry)
			rec, err := svc.GetRecommendations(context.Background(), owner, carID)
			if err != nil {
				t.Fatalf("GetRecommendations: %v", err)
			}
			if rec.ChargeCeiling != tc.ceiling {
				t.Errorf("%s: ceiling = %d, want %d", tc.name, rec.ChargeCeiling, tc.ceiling)
			}
			if len(rec.Tips) == 0 {
				t.Errorf("%s: expected at least one tip", tc.name)
			}
		})
	}
}

func TestAnalytics_RecommendationsNoModelIsGeneric(t *testing.T) {
	owner := uuid.New()
	svc, carID, _, _ := newAnalyticsSvc(t, owner, nil, nil) // car with no ev_model
	rec, err := svc.GetRecommendations(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetRecommendations: %v", err)
	}
	if rec.Chemistry != nil || rec.ChargeCeiling != 80 {
		t.Errorf("no model → generic advice (nil chemistry, ceiling 80), got %+v", rec)
	}
}

func TestAnalytics_DashboardAggregates(t *testing.T) {
	owner := uuid.New()
	svc, carRepo, chRepo, _ := newDashboardSvc(t)
	ctx := context.Background()

	// Car odometer values are intentionally set but must NOT affect TotalKM after
	// TASK-0023 fix (TotalKM now comes from session-odometer deltas, not car fields).
	c1, _ := carRepo.Create(ctx, owner, repository.CarInput{Name: "A", OdometerKM: 10000})
	carRepo.Create(ctx, owner, repository.CarInput{Name: "B", OdometerKM: 5000})

	// Three sessions with consecutive odometers: total 60 kWh, total cost 1500.
	//   s1→s2: 300 km, 25 kWh · s2→s3: 200 km, 15 kWh → 40 kWh over 500 km.
	base := time.Now().Add(-3 * time.Hour)
	for i, kc := range []struct {
		kwh, cost float64
		odo       int
	}{{20, 500, 1000}, {25, 600, 1300}, {15, 400, 1500}} {
		chRepo.Create(ctx, owner, domain.ChargingInput{
			CarID: c1.ID, StartedAt: base.Add(time.Duration(i) * time.Hour),
			KWhCharged: ptr(kc.kwh), Cost: ptr(kc.cost), OdometerKM: ptr(kc.odo),
		})
	}

	stats, err := svc.GetDashboard(ctx, owner)
	if err != nil {
		t.Fatalf("GetDashboard: %v", err)
	}
	// TotalKM must come from session deltas (300+200=500), NOT car odometers (10000+5000=15000).
	if stats.TotalKWh != 60 || stats.TotalCost != 1500 || stats.TotalKM != 500 || stats.SessionCount != 3 {
		t.Errorf("stats = %+v, want kwh=60 cost=1500 km=500 count=3", stats)
	}
	// avg = 40 kWh / (500 km / 100) = 40/5 = 8.0 kWh/100km
	if stats.AvgKWhPer100KM == nil || *stats.AvgKWhPer100KM != 8.0 {
		t.Errorf("avg = %v, want 8.0", stats.AvgKWhPer100KM)
	}
}

// TestAnalytics_DashboardTotalKMFromSessionDeltas is the regression test for
// TASK-0023: TotalKM must reflect session odometer deltas, not the car's static
// odometer field, so it updates in real time as new sessions are logged.
func TestAnalytics_DashboardTotalKMFromSessionDeltas(t *testing.T) {
	owner := uuid.New()
	svc, carRepo, chRepo, _ := newDashboardSvc(t)
	ctx := context.Background()

	// Car has a large static odometer that must NOT appear in TotalKM.
	car, _ := carRepo.Create(ctx, owner, repository.CarInput{Name: "Test", OdometerKM: 99999})
	base := time.Now().Add(-2 * time.Hour)

	// Seed two sessions: odometers 1000 and 1100 → delta 100 km.
	chRepo.Create(ctx, owner, domain.ChargingInput{
		CarID: car.ID, StartedAt: base, KWhCharged: ptr(10.0), OdometerKM: ptr(1000),
	})
	chRepo.Create(ctx, owner, domain.ChargingInput{
		CarID: car.ID, StartedAt: base.Add(time.Hour), KWhCharged: ptr(10.0), OdometerKM: ptr(1100),
	})

	stats, err := svc.GetDashboard(ctx, owner)
	if err != nil {
		t.Fatalf("GetDashboard (2 sessions): %v", err)
	}
	if stats.TotalKM != 100 {
		t.Errorf("TotalKM = %d after 2 sessions, want 100 (session delta); car odometer must be ignored", stats.TotalKM)
	}

	// Add a third session (odo 1200) — delta grows to 200 km total.
	chRepo.Create(ctx, owner, domain.ChargingInput{
		CarID: car.ID, StartedAt: base.Add(2 * time.Hour), KWhCharged: ptr(10.0), OdometerKM: ptr(1200),
	})
	// Bust the dashboard cache so the next call recomputes.
	svc.RecomputeAsync(owner, car.ID)

	stats2, err := svc.GetDashboard(ctx, owner)
	if err != nil {
		t.Fatalf("GetDashboard (3 sessions): %v", err)
	}
	if stats2.TotalKM != 200 {
		t.Errorf("TotalKM = %d after 3 sessions, want 200; dashboard must update with new sessions", stats2.TotalKM)
	}
}

func TestAnalytics_DashboardNullEfficiencyWhenNoDistance(t *testing.T) {
	owner := uuid.New()
	svc, carRepo, chRepo, _ := newDashboardSvc(t)
	ctx := context.Background()

	c, _ := carRepo.Create(ctx, owner, repository.CarInput{Name: "A", OdometerKM: 0}) // no distance yet
	chRepo.Create(ctx, owner, domain.ChargingInput{CarID: c.ID, StartedAt: time.Now(), KWhCharged: ptr(30.0)})

	stats, err := svc.GetDashboard(ctx, owner)
	if err != nil {
		t.Fatalf("GetDashboard: %v", err)
	}
	if stats.AvgKWhPer100KM != nil {
		t.Errorf("avg should be nil when total_km==0, got %v", *stats.AvgKWhPer100KM)
	}
}

func TestAnalytics_DashboardCacheHitAndInvalidation(t *testing.T) {
	owner := uuid.New()
	svc, carRepo, _, cache := newDashboardSvc(t)
	ctx := context.Background()
	carRepo.Create(ctx, owner, repository.CarInput{Name: "A", OdometerKM: 10000})

	// First call populates the cache.
	if _, err := svc.GetDashboard(ctx, owner); err != nil {
		t.Fatal(err)
	}
	key := "analytics:dashboard:" + owner.String()
	if !cache.has(key) {
		t.Fatal("first GetDashboard should populate the cache")
	}
	if cache.sets != 1 {
		t.Errorf("sets = %d, want 1", cache.sets)
	}

	// Second call is a cache hit (no additional set).
	if _, err := svc.GetDashboard(ctx, owner); err != nil {
		t.Fatal(err)
	}
	if cache.sets != 1 {
		t.Errorf("second call should hit cache, not re-set; sets = %d", cache.sets)
	}

	// A charging write (via RecomputeAsync hook) must invalidate the dashboard key.
	svc.RecomputeAsync(owner, uuid.New())
	if cache.has(key) {
		t.Error("RecomputeAsync should DEL the dashboard cache key")
	}
}

func TestAnalytics_BatteryHistoryOrderingAndIsolation(t *testing.T) {
	owner, attacker := uuid.New(), uuid.New()
	svc, carID, chRepo, _ := newAnalyticsSvc(t, owner, ptr(60.0), ptr("NMC"))
	for i := 0; i < 5; i++ {
		seedSession(t, chRepo, owner, carID, 20, 70, 30)
	}
	// Populate a snapshot via on-demand compute.
	if _, err := svc.GetBattery(context.Background(), owner, carID); err != nil {
		t.Fatal(err)
	}

	items, err := svc.GetBatteryHistory(context.Background(), owner, carID, 30)
	if err != nil {
		t.Fatalf("GetBatteryHistory: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected at least one snapshot in history")
	}
	// cross-user → 404
	if _, err := svc.GetBatteryHistory(context.Background(), attacker, carID, 30); !errors.Is(err, service.ErrCarNotFound) {
		t.Errorf("history cross-user: want ErrCarNotFound, got %v", err)
	}
}

func TestAnalytics_BatteryHistoryReturnsNewestChronological(t *testing.T) {
	owner := uuid.New()
	svc, carID, _, batRepo := newAnalyticsSvc(t, owner, ptr(60.0), ptr("NMC"))
	ctx := context.Background()

	// Seed 40 snapshots with increasing time and soh = 1..40 (older→newer).
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 40; i++ {
		batRepo.Save(ctx, &domain.BatteryHealthSnapshot{
			CarID: carID, UserID: owner, SOHPct: float64(i),
			ComputedAt: base.Add(time.Duration(i) * time.Hour),
			Confidence: "medium", Method: "delta_soc",
		})
	}

	items, err := svc.GetBatteryHistory(ctx, owner, carID, 30)
	if err != nil {
		t.Fatalf("GetBatteryHistory: %v", err)
	}
	if len(items) != 30 {
		t.Fatalf("len = %d, want 30 (clamped to the newest window)", len(items))
	}
	// Newest 30 of soh 1..40 are 11..40, returned chronologically (ASC).
	if items[0].SOHPct != 11 {
		t.Errorf("first soh = %v, want 11 (oldest of the newest-30 window)", items[0].SOHPct)
	}
	if items[len(items)-1].SOHPct != 40 {
		t.Errorf("last soh = %v, want 40 (most recent)", items[len(items)-1].SOHPct)
	}
	for i := 1; i < len(items); i++ {
		if !items[i-1].ComputedAt.Before(items[i].ComputedAt) {
			t.Fatalf("not chronological at %d: %v !before %v", i, items[i-1].ComputedAt, items[i].ComputedAt)
		}
	}
}

func TestAnalytics_RecomputeAsyncPersists(t *testing.T) {
	owner := uuid.New()
	svc, carID, chRepo, batRepo := newAnalyticsSvc(t, owner, ptr(60.0), ptr("NMC"))
	for i := 0; i < 5; i++ {
		seedSession(t, chRepo, owner, carID, 20, 70, 30)
	}

	svc.RecomputeAsync(owner, carID)

	// Async, on a detached context — poll briefly for the persisted snapshot.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if batRepo.saveCount() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if batRepo.saveCount() == 0 {
		t.Fatal("RecomputeAsync did not persist a snapshot")
	}
	snap, err := batRepo.GetLatest(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetLatest after recompute: %v", err)
	}
	if snap.SOHPct != 88.0 {
		t.Errorf("recomputed soh = %v, want 88.0", snap.SOHPct)
	}
}
