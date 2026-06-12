package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"

	"github.com/google/uuid"
)

// Service-level sentinel errors. Handlers depend only on this package (not on
// repository), so repository errors are translated here.
var (
	ErrCarNotFound        = errors.New("car not found")
	ErrInvalidEVModelRef  = errors.New("ev_model_id does not reference an existing model")
	ErrInvalidCatalogCar  = errors.New("catalog_car_id does not reference a catalog car")
	ErrInvalidOverride    = errors.New("invalid spec override")
	ErrValidation         = errors.New("validation failed")
)

// overrideKinds whitelists every spec_overrides key and its JSON type
// (TASK-0034). Keys mirror the CatalogCar JSON contract, plus the single-choice
// exterior_color / interior_color picks. Anything else → ErrInvalidOverride.
var overrideKinds = map[string]string{
	// numbers
	"battery_capacity_kwh": "number", "usable_kwh": "number", "range_km": "number",
	"consumption_kwh_per_100km": "number", "motor_power_kw": "number", "torque_nm": "number",
	"motor_count": "number", "acceleration_0_100_s": "number", "max_speed_kmh": "number",
	"ac_charge_kw": "number", "dc_charge_kw": "number", "fast_charge_to_80_min": "number",
	"radar_count": "number", "camera_count": "number", "weight_kg": "number", "trunk_liters": "number",
	// strings
	"name_fa": "string", "name_en": "string", "brand": "string", "body_style_fa": "string",
	"class": "string", "body_type": "string", "segment": "string", "country": "string",
	"importer": "string", "platform": "string", "battery_voltage": "string", "cell_brand": "string",
	"cell_type": "string", "cooling": "string", "range_standard": "string", "motor_type": "string",
	"drivetrain": "string", "ac_connector": "string", "dc_connector": "string",
	"fast_charge_window": "string", "v2l": "string", "v2g": "string", "ota": "string",
	"adas_level": "string", "notes": "string",
	"exterior_color": "string", "interior_color": "string",
}

// overrides that mirror an ev_catalog DB CHECK must stay strictly positive.
var positiveOverrides = map[string]bool{
	"battery_capacity_kwh": true, "usable_kwh": true, "range_km": true,
}

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

// CarService holds car business logic: validation, pagination clamping, and
// translation of repository errors.
type CarService struct {
	cars    repository.CarRepository
	catalog repository.CatalogRepository
}

func NewCarService(cars repository.CarRepository, catalog repository.CatalogRepository) *CarService {
	return &CarService{cars: cars, catalog: catalog}
}

func (s *CarService) Create(ctx context.Context, userID uuid.UUID, in repository.CarInput) (*domain.Car, error) {
	in, err := s.prepareCatalogInput(ctx, in)
	if err != nil {
		return nil, err
	}
	if err := validateCarInput(in); err != nil {
		return nil, err
	}
	car, err := s.cars.Create(ctx, userID, normalizeCarInput(in))
	return car, translateCarErr(err)
}

// prepareCatalogInput validates the catalog link + overrides and defaults the
// car name to the catalog's name_fa when the client sent none.
func (s *CarService) prepareCatalogInput(ctx context.Context, in repository.CarInput) (repository.CarInput, error) {
	if err := validateOverrides(in.SpecOverrides); err != nil {
		return in, err
	}
	if in.CatalogCarID == nil {
		return in, nil
	}
	cat, err := s.catalog.GetByID(ctx, *in.CatalogCarID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return in, ErrInvalidCatalogCar
		}
		return in, err
	}
	if strings.TrimSpace(in.Name) == "" {
		in.Name = cat.NameFA
	}
	return in, nil
}

func validateOverrides(overrides map[string]any) error {
	for key, val := range overrides {
		kind, ok := overrideKinds[key]
		if !ok {
			return fmt.Errorf("%w: unknown key %q", ErrInvalidOverride, key)
		}
		switch kind {
		case "number":
			n, ok := val.(float64) // JSON numbers decode to float64
			if !ok {
				return fmt.Errorf("%w: %q must be a number", ErrInvalidOverride, key)
			}
			if positiveOverrides[key] && n <= 0 {
				return fmt.Errorf("%w: %q must be > 0", ErrInvalidOverride, key)
			}
		case "string":
			s, ok := val.(string)
			if !ok {
				return fmt.Errorf("%w: %q must be a string", ErrInvalidOverride, key)
			}
			if len(s) > 500 {
				return fmt.Errorf("%w: %q must be <= 500 chars", ErrInvalidOverride, key)
			}
		}
	}
	return nil
}

func (s *CarService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Car, int, error) {
	limit, offset = ClampPagination(limit, offset)
	return s.cars.ListByUser(ctx, userID, limit, offset)
}

func (s *CarService) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Car, error) {
	car, err := s.cars.GetByID(ctx, userID, id)
	return car, translateCarErr(err)
}

func (s *CarService) Update(ctx context.Context, userID, id uuid.UUID, in repository.CarInput) (*domain.Car, error) {
	in, err := s.prepareCatalogInput(ctx, in)
	if err != nil {
		return nil, err
	}
	if err := validateCarInput(in); err != nil {
		return nil, err
	}
	car, err := s.cars.Update(ctx, userID, id, normalizeCarInput(in))
	return car, translateCarErr(err)
}

func (s *CarService) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return translateCarErr(s.cars.Delete(ctx, userID, id))
}

// ── helpers ─────────────────────────────────────────────────────────────────

func validateCarInput(in repository.CarInput) error {
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 255 {
		return fmt.Errorf("%w: name is required (1-255 chars)", ErrValidation)
	}
	if in.OdometerKM < 0 {
		return fmt.Errorf("%w: odometer_km must be >= 0", ErrValidation)
	}
	if in.LicensePlate != nil && len(*in.LicensePlate) > 50 {
		return fmt.Errorf("%w: license_plate must be <= 50 chars", ErrValidation)
	}
	return nil
}

func normalizeCarInput(in repository.CarInput) repository.CarInput {
	in.Name = strings.TrimSpace(in.Name)
	return in
}

// ClampPagination applies the default/min/max page rules. Exported so handlers
// can report the same effective values in the list envelope.
func ClampPagination(limit, offset int) (int, int) {
	if limit < 1 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func translateCarErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNotFound):
		return ErrCarNotFound
	case errors.Is(err, repository.ErrInvalidEVModel):
		return ErrInvalidEVModelRef
	case errors.Is(err, repository.ErrInvalidCatalogCar):
		return ErrInvalidCatalogCar
	default:
		return err
	}
}
