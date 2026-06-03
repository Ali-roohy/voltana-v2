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

// ErrStationNotFound is returned when a station id does not exist. Reuses the
// shared ErrValidation sentinel (defined in car_service.go) for bad input.
var ErrStationNotFound = errors.New("station not found")

// StationService holds charging-station business logic: validation (incl.
// lat/lng bounds), and translation of repository errors. Stations are shared
// reference data, so nothing here is user-scoped — admin authorization is
// enforced at the middleware layer, above the handler.
type StationService struct {
	stations repository.StationRepository
}

func NewStationService(stations repository.StationRepository) *StationService {
	return &StationService{stations: stations}
}

func (s *StationService) Create(ctx context.Context, in domain.StationInput) (*domain.ChargingStation, error) {
	if err := validateStationInput(in); err != nil {
		return nil, err
	}
	return s.stations.Create(ctx, normalizeStationInput(in))
}

// List returns markers, optionally filtered to a bounding box. A nil bounds
// returns every station. When bounds are supplied they are validated first.
func (s *StationService) List(ctx context.Context, b *domain.StationBounds) ([]domain.StationMarker, error) {
	if b != nil {
		if err := validateBounds(*b); err != nil {
			return nil, err
		}
	}
	return s.stations.List(ctx, b)
}

func (s *StationService) Get(ctx context.Context, id uuid.UUID) (*domain.ChargingStation, error) {
	st, err := s.stations.GetByID(ctx, id)
	return st, translateStationErr(err)
}

func (s *StationService) Update(ctx context.Context, id uuid.UUID, in domain.StationInput) (*domain.ChargingStation, error) {
	if err := validateStationInput(in); err != nil {
		return nil, err
	}
	st, err := s.stations.Update(ctx, id, normalizeStationInput(in))
	return st, translateStationErr(err)
}

func (s *StationService) Delete(ctx context.Context, id uuid.UUID) error {
	return translateStationErr(s.stations.Delete(ctx, id))
}

// ── helpers ─────────────────────────────────────────────────────────────────

func validateStationInput(in domain.StationInput) error {
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 255 {
		return fmt.Errorf("%w: name is required (1-255 chars)", ErrValidation)
	}
	// Re-checked here (not only via gin binding) because binding's `required`
	// treats 0 as missing — latitude:0 (the equator) would otherwise be rejected
	// before the service ever sees it.
	if in.Latitude < -90 || in.Latitude > 90 {
		return fmt.Errorf("%w: latitude must be between -90 and 90", ErrValidation)
	}
	if in.Longitude < -180 || in.Longitude > 180 {
		return fmt.Errorf("%w: longitude must be between -180 and 180", ErrValidation)
	}
	if in.Address != nil && len(*in.Address) > 500 {
		return fmt.Errorf("%w: address must be <= 500 chars", ErrValidation)
	}
	if in.ConnectorTypes != nil && len(*in.ConnectorTypes) > 255 {
		return fmt.Errorf("%w: connector_types must be <= 255 chars", ErrValidation)
	}
	if in.PowerKW != nil && *in.PowerKW <= 0 {
		return fmt.Errorf("%w: power_kw must be > 0", ErrValidation)
	}
	if in.Operator != nil && len(*in.Operator) > 255 {
		return fmt.Errorf("%w: operator must be <= 255 chars", ErrValidation)
	}
	return nil
}

func validateBounds(b domain.StationBounds) error {
	if b.MinLat < -90 || b.MaxLat > 90 || b.MinLng < -180 || b.MaxLng > 180 {
		return fmt.Errorf("%w: bounding box out of range", ErrValidation)
	}
	if b.MinLat > b.MaxLat || b.MinLng > b.MaxLng {
		return fmt.Errorf("%w: bounding box min must be <= max", ErrValidation)
	}
	return nil
}

func normalizeStationInput(in domain.StationInput) domain.StationInput {
	in.Name = strings.TrimSpace(in.Name)
	return in
}

func translateStationErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNotFound):
		return ErrStationNotFound
	default:
		return err
	}
}
