package service

import (
	"context"
	"errors"
	"fmt"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"

	"github.com/google/uuid"
)

// SettingsService holds user-settings business logic: auto-create-on-read,
// rate validation, and default-car ownership checks. It reuses ErrValidation and
// ErrInvalidCarRef from the package (shared with the car/charging services).
type SettingsService struct {
	settings repository.SettingsRepository
	cars     repository.CarRepository
}

func NewSettingsService(settings repository.SettingsRepository, cars repository.CarRepository) *SettingsService {
	return &SettingsService{settings: settings, cars: cars}
}

// Get returns the caller's settings, auto-creating a default row on first call.
func (s *SettingsService) Get(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error) {
	st, err := s.settings.GetOrCreate(ctx, userID)
	return st, translateSettingsErr(err)
}

// Update full-replaces the caller's rates + default_car_id (upsert). default_car_id
// must reference one of the caller's own cars, or be nil.
func (s *SettingsService) Update(ctx context.Context, userID uuid.UUID, in domain.SettingsInput) (*domain.UserSettings, error) {
	if err := validateSettingsInput(in); err != nil {
		return nil, err
	}
	if in.DefaultCarID != nil {
		if err := s.ensureCarOwned(ctx, userID, *in.DefaultCarID); err != nil {
			return nil, err
		}
	}
	st, err := s.settings.Update(ctx, userID, in)
	return st, translateSettingsErr(err)
}

// ── helpers ─────────────────────────────────────────────────────────────────

func validateSettingsInput(in domain.SettingsInput) error {
	for name, v := range map[string]float64{
		"peak_rate":    in.PeakRate,
		"mid_rate":     in.MidRate,
		"offpeak_rate": in.OffpeakRate,
	} {
		if v < 0 {
			return fmt.Errorf("%w: %s must be >= 0", ErrValidation, name)
		}
	}
	return nil
}

// ensureCarOwned confirms default_car_id belongs to the caller (reusing
// CarRepository); a missing/!owned car becomes ErrInvalidCarRef (→ 422).
func (s *SettingsService) ensureCarOwned(ctx context.Context, userID, carID uuid.UUID) error {
	if _, err := s.cars.GetByID(ctx, userID, carID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidCarRef
		}
		return err
	}
	return nil
}

func translateSettingsErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrInvalidCar):
		return ErrInvalidCarRef
	default:
		return err
	}
}
