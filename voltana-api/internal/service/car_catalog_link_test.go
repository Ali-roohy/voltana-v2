package service_test

// TASK-0034 — catalog link + spec_overrides on the cars service, and the
// analytics capacity/chemistry fallback chain (override → catalog → ev_model).

import (
	"context"
	"errors"
	"testing"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"
	"voltana-api/internal/service"

	"github.com/google/uuid"
)

func catalogWith(t *testing.T, capacity float64, cellType string) (*mockCatalogRepo, uuid.UUID) {
	t.Helper()
	id := uuid.New()
	return &mockCatalogRepo{items: []domain.CatalogCar{{
		ID:                 id,
		NameFA:             "تویوتا بی‌زد4ایکس",
		NameEN:             "Toyota bZ4X FWD",
		BatteryCapacityKWh: &capacity,
		CellType:           &cellType,
	}}}, id
}

func TestCar_CreateFromCatalog_DefaultsNameFA(t *testing.T) {
	catalog, catID := catalogWith(t, 66.7, "NMC")
	svc := service.NewCarService(newMockCarRepo(), catalog)

	car, err := svc.Create(context.Background(), uuid.New(), repository.CarInput{
		CatalogCarID:  &catID,
		SpecOverrides: map[string]any{"exterior_color": "آبی"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if car.Name != "تویوتا بی‌زد4ایکس" {
		t.Errorf("name should default to catalog name_fa, got %q", car.Name)
	}
	if car.CatalogCarID == nil || *car.CatalogCarID != catID {
		t.Error("catalog_car_id must round-trip")
	}
	if car.SpecOverrides["exterior_color"] != "آبی" {
		t.Errorf("overrides must round-trip, got %v", car.SpecOverrides)
	}
}

func TestCar_CreateBogusCatalogID(t *testing.T) {
	svc := service.NewCarService(newMockCarRepo(), &mockCatalogRepo{})
	bogus := uuid.New()
	_, err := svc.Create(context.Background(), uuid.New(), repository.CarInput{Name: "x", CatalogCarID: &bogus})
	if !errors.Is(err, service.ErrInvalidCatalogCar) {
		t.Errorf("want ErrInvalidCatalogCar, got %v", err)
	}
}

func TestCar_OverrideValidation(t *testing.T) {
	catalog, catID := catalogWith(t, 66.7, "NMC")
	svc := service.NewCarService(newMockCarRepo(), catalog)

	cases := []struct {
		name      string
		overrides map[string]any
	}{
		{"unknown key", map[string]any{"warp_drive": 9000}},
		{"string where number expected", map[string]any{"battery_capacity_kwh": "big"}},
		{"number where string expected", map[string]any{"exterior_color": 7}},
		{"capacity must be positive", map[string]any{"battery_capacity_kwh": -5.0}},
		{"range must be positive", map[string]any{"range_km": 0.0}},
	}
	for _, tc := range cases {
		_, err := svc.Create(context.Background(), uuid.New(), repository.CarInput{
			Name: "x", CatalogCarID: &catID, SpecOverrides: tc.overrides,
		})
		if !errors.Is(err, service.ErrInvalidOverride) {
			t.Errorf("%s: want ErrInvalidOverride, got %v", tc.name, err)
		}
	}
}

func TestCar_LegacyEVModelPathUnchanged(t *testing.T) {
	svc := service.NewCarService(newMockCarRepo(), &mockCatalogRepo{})
	evID := uuid.New()
	car, err := svc.Create(context.Background(), uuid.New(), repository.CarInput{Name: "Legacy", EVModelID: &evID})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if car.CatalogCarID != nil || len(car.SpecOverrides) != 0 {
		t.Error("legacy create must not gain catalog fields")
	}
}

// ── analytics fallback chain ──────────────────────────────────────────────────

// seedAnalyticsCar wires an analytics service around one car built from the
// given input, with 5 qualifying sessions so the SOH math actually runs.
func seedAnalyticsCar(t *testing.T, catalog *mockCatalogRepo, in repository.CarInput) (*service.AnalyticsService, uuid.UUID, uuid.UUID) {
	t.Helper()
	owner := uuid.New()
	carRepo := newMockCarRepo()
	car, err := carRepo.Create(context.Background(), owner, in)
	if err != nil {
		t.Fatalf("seed car: %v", err)
	}
	chRepo := newMockChargingRepo()
	for i := 0; i < 5; i++ {
		seedSession(t, chRepo, owner, car.ID, 20, 80, 36) // Δ60% · 36 kWh → ~52.8 kWh pack
	}
	svc := service.NewAnalyticsService(carRepo, newMockEVRepo(), catalog, chRepo, newMockBatteryRepo(), newMockCache())
	return svc, owner, car.ID
}

func TestAnalytics_CapacityFromOverride(t *testing.T) {
	catalog, catID := catalogWith(t, 66.7, "NMC")
	svc, owner, carID := seedAnalyticsCar(t, catalog, repository.CarInput{
		Name:          "mine",
		CatalogCarID:  &catID,
		SpecOverrides: map[string]any{"battery_capacity_kwh": 60.0}, // override beats catalog 66.7
	})

	res, err := svc.GetBattery(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetBattery: %v", err)
	}
	if res.Snapshot == nil {
		t.Fatal("want a snapshot (capacity resolvable), got insufficient-data")
	}
	if res.Snapshot.NominalCapacityKWh != 60 {
		t.Errorf("nominal should come from the override (60), got %v", res.Snapshot.NominalCapacityKWh)
	}
}

func TestAnalytics_CapacityFromCatalog(t *testing.T) {
	catalog, catID := catalogWith(t, 66.7, "NMC")
	svc, owner, carID := seedAnalyticsCar(t, catalog, repository.CarInput{
		Name: "mine", CatalogCarID: &catID, // no override, no ev_model
	})

	res, err := svc.GetBattery(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetBattery: %v", err)
	}
	if res.Snapshot == nil {
		t.Fatal("want a snapshot via the catalog capacity, got insufficient-data")
	}
	if res.Snapshot.NominalCapacityKWh != 66.7 {
		t.Errorf("nominal should come from the catalog (66.7), got %v", res.Snapshot.NominalCapacityKWh)
	}
}

func TestAnalytics_NoCapacityAnywhere(t *testing.T) {
	svc, owner, carID := seedAnalyticsCar(t, &mockCatalogRepo{}, repository.CarInput{Name: "bare"})

	res, err := svc.GetBattery(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetBattery: %v", err)
	}
	if res.Snapshot != nil {
		t.Error("no override, no catalog, no ev_model → must stay insufficient-data")
	}
}

func TestAnalytics_ChemistryFallback(t *testing.T) {
	catalog, catID := catalogWith(t, 66.7, "LFP")

	// override wins over catalog
	svc, owner, carID := seedAnalyticsCar(t, catalog, repository.CarInput{
		Name: "mine", CatalogCarID: &catID,
		SpecOverrides: map[string]any{"cell_type": "NMC"},
	})
	rec, err := svc.GetRecommendations(context.Background(), owner, carID)
	if err != nil {
		t.Fatalf("GetRecommendations: %v", err)
	}
	if rec.Chemistry == nil || *rec.Chemistry != "NMC" {
		t.Errorf("chemistry should come from the override (NMC), got %v", rec.Chemistry)
	}

	// catalog when no override
	svc2, owner2, carID2 := seedAnalyticsCar(t, catalog, repository.CarInput{
		Name: "mine", CatalogCarID: &catID,
	})
	rec2, err := svc2.GetRecommendations(context.Background(), owner2, carID2)
	if err != nil {
		t.Fatalf("GetRecommendations: %v", err)
	}
	if rec2.Chemistry == nil || *rec2.Chemistry != "LFP" {
		t.Errorf("chemistry should come from the catalog (LFP), got %v", rec2.Chemistry)
	}
}
