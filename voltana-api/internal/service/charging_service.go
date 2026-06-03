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

// Service-level sentinel errors for charging sessions. Handlers depend only on
// this package, so repository errors are translated here. ErrValidation is
// shared with the car service (same package).
var (
	ErrChargingNotFound = errors.New("charging session not found")
	ErrInvalidCarRef    = errors.New("car_id does not reference one of your cars")
)

// HealthRecomputer is notified when a car's charging history changes so battery
// SOH can be re-estimated off the request path. Implemented by AnalyticsService.
type HealthRecomputer interface {
	RecomputeAsync(userID, carID uuid.UUID)
}

// ChargingService holds charging-session business logic: validation, server-side
// time-of-use cost calculation, car-ownership checks, and error translation.
type ChargingService struct {
	sessions repository.ChargingRepository
	cars     repository.CarRepository
	settings repository.SettingsRepository
	health   HealthRecomputer // optional; nil in tests that don't exercise recompute
}

func NewChargingService(sessions repository.ChargingRepository, cars repository.CarRepository, settings repository.SettingsRepository) *ChargingService {
	return &ChargingService{sessions: sessions, cars: cars, settings: settings}
}

// SetHealthRecomputer wires the battery-health recompute trigger. Kept separate
// from the constructor to avoid an init-order cycle (the recomputer reads the same
// charging repository this service writes through).
func (s *ChargingService) SetHealthRecomputer(h HealthRecomputer) {
	s.health = h
}

// triggerRecompute schedules a SOH recompute for the car after a write. Nil-safe.
func (s *ChargingService) triggerRecompute(userID, carID uuid.UUID) {
	if s.health != nil {
		s.health.RecomputeAsync(userID, carID)
	}
}

func (s *ChargingService) Create(ctx context.Context, userID uuid.UUID, in domain.ChargingInput) (*domain.ChargingSession, error) {
	prepared, err := s.prepare(ctx, userID, in)
	if err != nil {
		return nil, err
	}
	sess, err := s.sessions.Create(ctx, userID, prepared)
	if err != nil {
		return nil, translateChargingErr(err)
	}
	s.triggerRecompute(userID, sess.CarID)
	return sess, nil
}

func (s *ChargingService) List(ctx context.Context, userID uuid.UUID, f domain.ChargingFilter, limit, offset int) ([]domain.ChargingSession, int, error) {
	limit, offset = ClampPagination(limit, offset)
	return s.sessions.ListByUser(ctx, userID, f, limit, offset)
}

func (s *ChargingService) Get(ctx context.Context, userID, id uuid.UUID) (*domain.ChargingSession, error) {
	sess, err := s.sessions.GetByID(ctx, userID, id)
	return sess, translateChargingErr(err)
}

func (s *ChargingService) Update(ctx context.Context, userID, id uuid.UUID, in domain.ChargingInput) (*domain.ChargingSession, error) {
	// Confirm the session belongs to the caller first, so a cross-user update is a
	// 404 regardless of the request body (don't let a bad car_id surface as 422 and
	// reveal that the session check was even reached).
	if _, err := s.sessions.GetByID(ctx, userID, id); err != nil {
		return nil, translateChargingErr(err)
	}
	prepared, err := s.prepare(ctx, userID, in)
	if err != nil {
		return nil, err
	}
	sess, err := s.sessions.Update(ctx, userID, id, prepared)
	if err != nil {
		return nil, translateChargingErr(err)
	}
	s.triggerRecompute(userID, sess.CarID)
	return sess, nil
}

func (s *ChargingService) Delete(ctx context.Context, userID, id uuid.UUID) error {
	// Fetch first so we know which car's history changed (and so a cross-user
	// delete is a 404 before we touch anything).
	sess, err := s.sessions.GetByID(ctx, userID, id)
	if err != nil {
		return translateChargingErr(err)
	}
	if err := s.sessions.Delete(ctx, userID, id); err != nil {
		return translateChargingErr(err)
	}
	s.triggerRecompute(userID, sess.CarID)
	return nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

// prepare validates the input, confirms the car belongs to the caller, resolves
// the cost (server-side if not supplied), and normalizes free-text fields. It is
// the shared front-half of Create and Update.
func (s *ChargingService) prepare(ctx context.Context, userID uuid.UUID, in domain.ChargingInput) (domain.ChargingInput, error) {
	if err := validateChargingInput(in); err != nil {
		return in, err
	}
	if err := s.ensureCarOwned(ctx, userID, in.CarID); err != nil {
		return in, err
	}
	cost, err := s.resolveCost(ctx, userID, in)
	if err != nil {
		return in, err
	}
	in.Cost = cost
	return normalizeChargingInput(in), nil
}

// ensureCarOwned reuses CarRepository so a session can only reference one of the
// caller's own cars; a missing/!owned car becomes ErrInvalidCarRef (→ 422).
func (s *ChargingService) ensureCarOwned(ctx context.Context, userID, carID uuid.UUID) error {
	if _, err := s.cars.GetByID(ctx, userID, carID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidCarRef
		}
		return err
	}
	return nil
}

// resolveCost keeps a client-supplied cost as-is; otherwise it computes the
// time-of-use cost from the per-period energy and the user's rates. When no
// period energy is provided there is nothing to compute, so cost stays nil.
func (s *ChargingService) resolveCost(ctx context.Context, userID uuid.UUID, in domain.ChargingInput) (*float64, error) {
	if in.Cost != nil {
		return in.Cost, nil
	}
	if in.EnergyPeakKWh == nil && in.EnergyMidKWh == nil && in.EnergyOffpeakKWh == nil {
		return nil, nil
	}
	rates, err := s.settings.GetRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	c := orZero(in.EnergyPeakKWh)*rates.Peak +
		orZero(in.EnergyMidKWh)*rates.Mid +
		orZero(in.EnergyOffpeakKWh)*rates.Offpeak
	return &c, nil
}

func validateChargingInput(in domain.ChargingInput) error {
	if in.CarID == uuid.Nil {
		return fmt.Errorf("%w: car_id is required", ErrValidation)
	}
	if in.StartedAt.IsZero() {
		return fmt.Errorf("%w: started_at is required", ErrValidation)
	}
	if in.EndedAt != nil && in.EndedAt.Before(in.StartedAt) {
		return fmt.Errorf("%w: ended_at must be at or after started_at", ErrValidation)
	}
	if err := validateSOC(in.StartSOC, "start_soc"); err != nil {
		return err
	}
	if err := validateSOC(in.EndSOC, "end_soc"); err != nil {
		return err
	}
	for name, v := range map[string]*float64{
		"kwh_charged":        in.KWhCharged,
		"energy_peak_kwh":    in.EnergyPeakKWh,
		"energy_mid_kwh":     in.EnergyMidKWh,
		"energy_offpeak_kwh": in.EnergyOffpeakKWh,
		"cost":               in.Cost,
	} {
		if v != nil && *v < 0 {
			return fmt.Errorf("%w: %s must be >= 0", ErrValidation, name)
		}
	}
	return nil
}

func validateSOC(v *int, name string) error {
	if v != nil && (*v < 0 || *v > 100) {
		return fmt.Errorf("%w: %s must be between 0 and 100", ErrValidation, name)
	}
	return nil
}

func normalizeChargingInput(in domain.ChargingInput) domain.ChargingInput {
	in.Location = trimPtr(in.Location)
	in.Notes = trimPtr(in.Notes)
	return in
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	return &t
}

func orZero(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func translateChargingErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNotFound):
		return ErrChargingNotFound
	case errors.Is(err, repository.ErrInvalidCar):
		return ErrInvalidCarRef
	default:
		return err
	}
}
