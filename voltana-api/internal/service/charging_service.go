package service

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	// ErrOdometerNotIncreasing is returned when a session's odometer is not greater
	// than the previous session's for the same car. The odometer is cumulative
	// (like the car's dashboard), so it must strictly increase (BUG-4). The message
	// is user-facing (surfaced verbatim in the API error envelope).
	ErrOdometerNotIncreasing = errors.New("کیلومترشمار باید از جلسه قبلی بیشتر باشد (کیلومترشمار تجمعی است)")
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
	// Snapshot the owner's rates as they are RIGHT NOW (TASK-0037 FEAT-6).
	// GetOrCreate also seeds a missing settings row from the admin defaults.
	st, err := s.settings.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}
	rates := repository.Rates{Peak: st.PeakRate, Mid: st.MidRate, Offpeak: st.OffpeakRate}
	in.RatePeakAtTime = &rates.Peak
	in.RateMidAtTime = &rates.Mid
	in.RateOffpeakAtTime = &rates.Offpeak

	prepared, err := s.prepare(ctx, userID, in, rates)
	if err != nil {
		return nil, err
	}
	if err := s.checkOdometerMonotonic(ctx, userID, &prepared, uuid.Nil); err != nil {
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
	items, total, err := s.sessions.ListByUser(ctx, userID, f, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		setSessionEfficiency(&items[i])
		setSessionWarnings(&items[i])
	}
	return items, total, nil
}

// Suspicious-data thresholds (FEAT-5). The efficiency band mirrors the aggregate
// sanity guard (BUG-4); duration is "implausible" when the energy/power estimate
// is off by more than 2× either way from the entered duration.
const (
	warnEffMin       = 5.0
	warnEffMax       = 40.0
	durationMismatch = 2.0
)

// setSessionWarnings computes the non-blocking data-quality flags for a session
// (FEAT-5). Always assigns a (possibly empty) slice so the JSON is `[]`, not null.
func setSessionWarnings(s *domain.ChargingSession) {
	w := []domain.SessionWarning{}

	if s.EfficiencyKWhPer100km != nil && (*s.EfficiencyKWhPer100km > warnEffMax || *s.EfficiencyKWhPer100km < warnEffMin) {
		w = append(w, domain.SessionWarning{Code: "efficiency_out_of_band", Message: "مصرف غیرعادی (خارج از محدوده ۵ تا ۴۰ کیلووات‌ساعت در ۱۰۰ کیلومتر)"})
	}
	if s.StartSOC != nil && s.EndSOC != nil && *s.StartSOC > *s.EndSOC {
		w = append(w, domain.SessionWarning{Code: "soc_decreasing", Message: "درصد شارژ پایان کمتر از شروع است"})
	}
	socChanged := s.StartSOC != nil && s.EndSOC != nil && *s.StartSOC != *s.EndSOC
	if socChanged && (s.KWhCharged == nil || *s.KWhCharged == 0) {
		w = append(w, domain.SessionWarning{Code: "zero_energy_soc_changed", Message: "انرژی صفر ثبت شده اما درصد شارژ تغییر کرده است"})
	}
	// Duration vs energy/power plausibility: predicted hours = energy / power.
	if s.EndedAt != nil && s.ChargePowerKW != nil && *s.ChargePowerKW > 0 && s.KWhCharged != nil && *s.KWhCharged > 0 {
		actualH := s.EndedAt.Sub(s.StartedAt).Hours()
		predictedH := *s.KWhCharged / *s.ChargePowerKW
		if actualH > 0 && (predictedH > actualH*durationMismatch || predictedH < actualH/durationMismatch) {
			w = append(w, domain.SessionWarning{Code: "duration_implausible", Message: "مدت زمان شارژ با انرژی و توان شارژ همخوانی ندارد"})
		}
	}

	s.Warnings = w
}

// setSessionEfficiency derives kWh/100km for a session from its odometer and the
// previous session's odometer (supplied by the repo). Set only when both readings
// exist, energy is known, and the distance is positive; otherwise left nil.
func setSessionEfficiency(s *domain.ChargingSession) {
	if s.OdometerKM == nil || s.PrevOdometerKM == nil || s.KWhCharged == nil {
		return
	}
	km := *s.OdometerKM - *s.PrevOdometerKM
	if km <= 0 {
		return
	}
	eff := math.Round((*s.KWhCharged/(float64(km)/100))*100) / 100
	s.EfficiencyKWhPer100km = &eff
}

func (s *ChargingService) Get(ctx context.Context, userID, id uuid.UUID) (*domain.ChargingSession, error) {
	sess, err := s.sessions.GetByID(ctx, userID, id)
	return sess, translateChargingErr(err)
}

func (s *ChargingService) Update(ctx context.Context, userID, id uuid.UUID, in domain.ChargingInput) (*domain.ChargingSession, error) {
	// Confirm the session belongs to the caller first, so a cross-user update is a
	// 404 regardless of the request body (don't let a bad car_id surface as 422 and
	// reveal that the session check was even reached).
	existing, err := s.sessions.GetByID(ctx, userID, id)
	if err != nil {
		return nil, translateChargingErr(err)
	}
	// Recompute against the session's FROZEN snapshot rates (FEAT-6) — editing a
	// session must never re-price it with the user's current rates. Legacy rows
	// without a snapshot fall back to current rates.
	var rates repository.Rates
	if existing.RatePeakAtTime != nil && existing.RateMidAtTime != nil && existing.RateOffpeakAtTime != nil {
		rates = repository.Rates{Peak: *existing.RatePeakAtTime, Mid: *existing.RateMidAtTime, Offpeak: *existing.RateOffpeakAtTime}
	} else if rates, err = s.settings.GetRates(ctx, userID); err != nil {
		return nil, err
	}
	prepared, err := s.prepare(ctx, userID, in, rates)
	if err != nil {
		return nil, err
	}
	if err := s.checkOdometerMonotonic(ctx, userID, &prepared, id); err != nil {
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
// the cost (server-side if not supplied) from the given rates, and normalizes
// free-text fields. It is the shared front-half of Create and Update; the caller
// chooses the rates (Create: current → snapshot; Update: the frozen snapshot).
func (s *ChargingService) prepare(ctx context.Context, userID uuid.UUID, in domain.ChargingInput, rates repository.Rates) (domain.ChargingInput, error) {
	if err := validateChargingInput(in); err != nil {
		return in, err
	}
	if err := s.ensureCarOwned(ctx, userID, in.CarID); err != nil {
		return in, err
	}
	in.Cost = resolveCost(in, rates)
	return normalizeChargingInput(in), nil
}

// checkOdometerMonotonic enforces the cumulative-odometer invariant (BUG-4): a
// session's odometer must be strictly greater than the nearest earlier session's
// for the same car. No odometer on the new session, or no earlier reading, → pass.
// excludeID skips the session being updated so it isn't compared against itself.
// As a side effect it derives TripDistanceKM (= odometer − previous odometer) and
// sets it on the input (TASK-0042 migration slice); left nil when not derivable.
func (s *ChargingService) checkOdometerMonotonic(ctx context.Context, userID uuid.UUID, in *domain.ChargingInput, excludeID uuid.UUID) error {
	in.TripDistanceKM = nil
	if in.OdometerKM == nil {
		return nil
	}
	prev, err := s.sessions.PreviousOdometer(ctx, userID, in.CarID, in.StartedAt, excludeID)
	if err != nil {
		return err
	}
	if prev != nil {
		if *in.OdometerKM <= *prev {
			return ErrOdometerNotIncreasing
		}
		trip := float64(*in.OdometerKM - *prev)
		in.TripDistanceKM = &trip
	}
	return nil
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
// time-of-use cost from the per-period energy and the GIVEN rates (creation
// snapshot or a session's frozen snapshot — never fetched here). When no period
// energy is provided there is nothing to compute, so cost stays nil.
func resolveCost(in domain.ChargingInput, rates repository.Rates) *float64 {
	if in.Cost != nil {
		return in.Cost
	}
	if in.EnergyPeakKWh == nil && in.EnergyMidKWh == nil && in.EnergyOffpeakKWh == nil {
		return nil
	}
	c := orZero(in.EnergyPeakKWh)*rates.Peak +
		orZero(in.EnergyMidKWh)*rates.Mid +
		orZero(in.EnergyOffpeakKWh)*rates.Offpeak
	return &c
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
	if in.OdometerKM != nil && *in.OdometerKM < 0 {
		return fmt.Errorf("%w: odometer_km must be >= 0", ErrValidation)
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
