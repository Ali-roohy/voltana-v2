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
	ErrCarNotFound       = errors.New("car not found")
	ErrInvalidEVModelRef = errors.New("ev_model_id does not reference an existing model")
	ErrValidation        = errors.New("validation failed")
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

// CarService holds car business logic: validation, pagination clamping, and
// translation of repository errors.
type CarService struct {
	cars repository.CarRepository
}

func NewCarService(cars repository.CarRepository) *CarService {
	return &CarService{cars: cars}
}

func (s *CarService) Create(ctx context.Context, userID uuid.UUID, in repository.CarInput) (*domain.Car, error) {
	if err := validateCarInput(in); err != nil {
		return nil, err
	}
	car, err := s.cars.Create(ctx, userID, normalizeCarInput(in))
	return car, translateCarErr(err)
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
	default:
		return err
	}
}
