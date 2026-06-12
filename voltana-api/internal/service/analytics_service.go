package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"

	"github.com/google/uuid"
)

const (
	// chargingEfficiency accounts for AC→battery losses (charger + BMS). kwh_charged
	// is wall energy, not energy into the pack; without this factor the estimated
	// capacity — and thus SOH — would read 10–15% too high (often > 100%).
	chargingEfficiency = 0.88
	// minQualifyingSessions is the floor below which we refuse to estimate SOH
	// (returns insufficient-data rather than a noisy number).
	minQualifyingSessions = 5
	// minQualifyingDeltaSOC filters out small swings: start/end SOC are whole-percent
	// integers, so a small delta carries a large relative rounding error.
	minQualifyingDeltaSOC = 25
	// highConfidenceSessions / highConfidenceAvgDelta gate the "high" confidence band.
	highConfidenceSessions = 8
	highConfidenceAvgDelta = 40.0
	// recomputeTimeout bounds a background recompute so a stuck query can't leak a goroutine.
	recomputeTimeout = 30 * time.Second

	// dashboardCacheTTL is how long the per-user dashboard aggregate is cached. A
	// charging write busts the key early; this is the staleness backstop.
	dashboardCacheTTL = 5 * time.Minute
	// defaultHistoryLimit / maxHistoryLimit bound the battery-history series.
	defaultHistoryLimit = 30
	maxHistoryLimit     = 100
)

// CacheStore is the small key/value cache the dashboard uses (cache-aside). The
// concrete implementation lives in repository (Redis) to keep the service testable.
type CacheStore interface {
	CacheGet(ctx context.Context, key string) (val string, ok bool, err error)
	CacheSet(ctx context.Context, key, val string, ttl time.Duration) error
	CacheDel(ctx context.Context, key string) error
}

// BatteryHealthResult is the outcome of a SOH estimate. Snapshot is nil when there
// is not enough qualifying data to estimate (QualifyingSessions reports how many
// sessions did qualify, so the UI can show progress toward the minimum).
type BatteryHealthResult struct {
	Snapshot           *domain.BatteryHealthSnapshot
	QualifyingSessions int
}

// AnalyticsService estimates battery State of Health from charging history (the
// delta-SOC method) and serves chemistry-aware recommendations. Ownership is
// enforced via CarRepository (cross-user/unknown car → ErrCarNotFound → 404).
// SOHAlertNotifier sends a push alert when a car's battery health crosses
// below the alert threshold (TASK-0039). Implemented by PushService; nil-safe.
type SOHAlertNotifier interface {
	NotifySOHDrop(userID uuid.UUID, carName string, sohPct float64)
}

// sohAlertThreshold — alert fires when SOH crosses from ≥ threshold to < threshold.
const sohAlertThreshold = 80.0

type AnalyticsService struct {
	cars     repository.CarRepository
	evModels repository.EVModelRepository
	catalog  repository.CatalogRepository
	sessions repository.ChargingRepository
	battery  repository.BatteryRepository
	cache    CacheStore
	notifier SOHAlertNotifier // optional (nil in tests / when push unconfigured)

	// per-car recompute coalescing: at most one recompute runs per car; a trigger
	// that arrives while one is in flight marks the car pending so the latest data
	// is picked up by a follow-up run.
	mu       sync.Mutex
	inflight map[uuid.UUID]bool
	pending  map[uuid.UUID]bool
}

func NewAnalyticsService(
	cars repository.CarRepository,
	evModels repository.EVModelRepository,
	catalog repository.CatalogRepository,
	sessions repository.ChargingRepository,
	battery repository.BatteryRepository,
	cache CacheStore,
) *AnalyticsService {
	return &AnalyticsService{
		cars:     cars,
		evModels: evModels,
		catalog:  catalog,
		sessions: sessions,
		battery:  battery,
		cache:    cache,
		inflight: make(map[uuid.UUID]bool),
		pending:  make(map[uuid.UUID]bool),
	}
}

func dashboardCacheKey(userID uuid.UUID) string { return "analytics:dashboard:" + userID.String() }

// SetSOHAlertNotifier wires the push notifier. Separate from the constructor
// (same pattern as ChargingService.SetHealthRecomputer) — push is optional.
func (s *AnalyticsService) SetSOHAlertNotifier(n SOHAlertNotifier) {
	s.notifier = n
}

// GetDashboard returns lifetime aggregate stats for a user (cache-aside, 5 min TTL).
// Cache errors are non-fatal — the stats are recomputed from the source of truth.
func (s *AnalyticsService) GetDashboard(ctx context.Context, userID uuid.UUID) (*domain.DashboardStats, error) {
	key := dashboardCacheKey(userID)
	if val, ok, err := s.cache.CacheGet(ctx, key); err != nil {
		log.Printf("analytics: dashboard cache get user=%s: %v", userID, err)
	} else if ok {
		var cached domain.DashboardStats
		if jErr := json.Unmarshal([]byte(val), &cached); jErr == nil {
			return &cached, nil
		}
		log.Printf("analytics: dashboard cache decode user=%s: corrupt, recomputing", userID)
	}

	totalKWh, totalCost, count, err := s.sessions.AggregateByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	stats := &domain.DashboardStats{
		TotalKWh:     round2(totalKWh),
		TotalCost:    round2(totalCost),
		TotalKM:      0, // derived below from session odometer deltas
		SessionCount: count,
	}
	// Both TotalKM and AvgKWhPer100KM are derived from the same session-odometer
	// aggregate (TASK-0018 / TASK-0023): sum of consecutive odometer-pair deltas.
	// TotalKM is therefore the distance *tracked in the app*, not the car's static
	// odometer field (which only changes on manual edits and never reflects new sessions).
	effKWh, effKM, err := s.sessions.EfficiencyAggregateByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if effKM > 0 {
		stats.TotalKM = int(effKM + 0.5) // nearest int, positive-safe without math import
		eff := round2(effKWh / (effKM / 100))
		stats.AvgKWhPer100KM = &eff
	}

	if blob, jErr := json.Marshal(stats); jErr == nil {
		if err := s.cache.CacheSet(ctx, key, string(blob), dashboardCacheTTL); err != nil {
			log.Printf("analytics: dashboard cache set user=%s: %v", userID, err)
		}
	}
	return stats, nil
}

// GetBatteryHistory returns a user's car's SOH snapshots oldest→newest for charting.
func (s *AnalyticsService) GetBatteryHistory(ctx context.Context, userID, carID uuid.UUID, limit int) ([]domain.BatteryHealthSnapshot, error) {
	if _, err := s.ownedCar(ctx, userID, carID); err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = defaultHistoryLimit
	}
	if limit > maxHistoryLimit {
		limit = maxHistoryLimit
	}
	return s.battery.ListByCar(ctx, userID, carID, limit)
}

// GetBattery returns the latest SOH estimate for a user's car. If no snapshot
// exists yet it computes one on demand (and persists it when there is enough
// data), so the endpoint is self-sufficient even before any write trigger fires.
func (s *AnalyticsService) GetBattery(ctx context.Context, userID, carID uuid.UUID) (*BatteryHealthResult, error) {
	if _, err := s.ownedCar(ctx, userID, carID); err != nil {
		return nil, err
	}

	if snap, err := s.battery.GetLatest(ctx, userID, carID); err == nil {
		return &BatteryHealthResult{Snapshot: snap, QualifyingSessions: snap.SampleSessionCount}, nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// No snapshot yet — compute (and persist if valid) on demand.
	return s.compute(ctx, userID, carID, true)
}

// GetRecommendations returns chemistry-aware battery-care advice for a user's car.
func (s *AnalyticsService) GetRecommendations(ctx context.Context, userID, carID uuid.UUID) (*domain.BatteryRecommendation, error) {
	car, err := s.ownedCar(ctx, userID, carID)
	if err != nil {
		return nil, err
	}
	chem := s.chemistryOf(ctx, car)
	rec := recommendFor(chem)
	return &rec, nil
}

// RecomputeAsync schedules a SOH recompute for a car off the request path. It
// coalesces concurrent triggers per car and runs on a detached context so it
// survives the originating HTTP request. Safe to call with a nil receiver-less
// dependency (no-op only if the service was never wired).
func (s *AnalyticsService) RecomputeAsync(userID, carID uuid.UUID) {
	// Bust the user's dashboard cache promptly so the next GET reflects the write
	// (the aggregate totals just changed). Best-effort; the TTL is the backstop.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := s.cache.CacheDel(ctx, dashboardCacheKey(userID)); err != nil {
		log.Printf("analytics: dashboard cache invalidate user=%s: %v", userID, err)
	}
	cancel()

	s.mu.Lock()
	if s.inflight[carID] {
		s.pending[carID] = true // a run is active; ensure a follow-up sees the new write
		s.mu.Unlock()
		return
	}
	s.inflight[carID] = true
	s.mu.Unlock()

	go s.recomputeLoop(userID, carID)
}

func (s *AnalyticsService) recomputeLoop(userID, carID uuid.UUID) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), recomputeTimeout)
		if _, err := s.compute(ctx, userID, carID, true); err != nil && !errors.Is(err, ErrCarNotFound) {
			log.Printf("analytics: recompute car=%s: %v", carID, err)
		}
		cancel()

		s.mu.Lock()
		if s.pending[carID] {
			delete(s.pending, carID) // another write landed; loop once more
			s.mu.Unlock()
			continue
		}
		delete(s.inflight, carID)
		s.mu.Unlock()
		return
	}
}

// compute runs the delta-SOC estimate. When persist is true and the estimate is
// valid (not insufficient-data), the snapshot is saved as new history.
func (s *AnalyticsService) compute(ctx context.Context, userID, carID uuid.UUID, persist bool) (*BatteryHealthResult, error) {
	car, err := s.ownedCar(ctx, userID, carID)
	if err != nil {
		return nil, err
	}

	// No linked model → no nominal capacity → cannot estimate SOH.
	nominal, ok := s.nominalCapacity(ctx, car)
	if !ok {
		return &BatteryHealthResult{}, nil
	}

	sessions, _, err := s.sessions.ListByUser(ctx, userID, domain.ChargingFilter{CarID: &carID}, 100, 0)
	if err != nil {
		return nil, err
	}

	var weightSum, weightedCapSum, deltaSum float64
	qualifying := 0
	for _, sess := range sessions {
		delta, cap, ok := sessionCapacity(sess)
		if !ok {
			continue
		}
		qualifying++
		weightSum += delta
		weightedCapSum += cap * delta
		deltaSum += delta
	}

	if qualifying < minQualifyingSessions {
		return &BatteryHealthResult{QualifyingSessions: qualifying}, nil
	}

	estimated := weightedCapSum / weightSum
	if estimated > nominal { // clamp: measured capacity can't exceed nameplate (SOH ≤ 100%)
		estimated = nominal
	}
	soh := 100 * estimated / nominal
	if soh < 0.01 { // floor: a pathologically tiny estimate must not round to 0.00 and
		soh = 0.01 // trip the DB CHECK (soh_pct > 0) on Save. Keeps SOH ∈ (0,100].
	}

	snap := &domain.BatteryHealthSnapshot{
		CarID:                carID,
		UserID:               userID,
		SOHPct:               round2(soh),
		EstimatedCapacityKWh: round2(estimated),
		NominalCapacityKWh:   round2(nominal),
		SampleSessionCount:   qualifying,
		Confidence:           confidenceFor(qualifying, deltaSum/float64(qualifying)),
		Method:               "delta_soc",
		ComputedAt:           time.Now(),
	}

	if persist {
		// Fetch the previous snapshot BEFORE saving so the ≥80→<80 cross can be
		// detected (TASK-0039). Missing previous = first estimate → no alert.
		var prev *domain.BatteryHealthSnapshot
		if s.notifier != nil {
			prev, _ = s.battery.GetLatest(ctx, userID, carID)
		}
		if err := s.battery.Save(ctx, snap); err != nil {
			return nil, err
		}
		if s.notifier != nil && prev != nil &&
			prev.SOHPct >= sohAlertThreshold && snap.SOHPct < sohAlertThreshold {
			s.notifier.NotifySOHDrop(userID, car.Name, snap.SOHPct)
		}
	}
	return &BatteryHealthResult{Snapshot: snap, QualifyingSessions: qualifying}, nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

// ownedCar fetches the car scoped to the user, translating a missing/!owned car
// into ErrCarNotFound (→ 404) so cross-user probing is indistinguishable.
func (s *AnalyticsService) ownedCar(ctx context.Context, userID, carID uuid.UUID) (*domain.Car, error) {
	car, err := s.cars.GetByID(ctx, userID, carID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCarNotFound
		}
		return nil, err
	}
	return car, nil
}

// nominalCapacity resolves the nameplate pack size (TASK-0034 fallback chain):
// user spec override → linked catalog car → linked ev_model.
func (s *AnalyticsService) nominalCapacity(ctx context.Context, car *domain.Car) (float64, bool) {
	if v, ok := car.SpecOverrides["battery_capacity_kwh"].(float64); ok && v > 0 {
		return v, true
	}
	if car.CatalogCarID != nil {
		cat, err := s.catalog.GetByID(ctx, *car.CatalogCarID)
		if err == nil && cat.BatteryCapacityKWh != nil && *cat.BatteryCapacityKWh > 0 {
			return *cat.BatteryCapacityKWh, true
		}
	}
	if car.EVModelID == nil {
		return 0, false
	}
	model, err := s.evModels.GetByID(ctx, *car.EVModelID)
	if err != nil || model.BatteryCapacityKWh == nil || *model.BatteryCapacityKWh <= 0 {
		return 0, false
	}
	return *model.BatteryCapacityKWh, true
}

// chemistryOf resolves cell chemistry with the same override → catalog →
// ev_model order as nominalCapacity.
func (s *AnalyticsService) chemistryOf(ctx context.Context, car *domain.Car) *string {
	if v, ok := car.SpecOverrides["cell_type"].(string); ok && v != "" {
		return &v
	}
	if car.CatalogCarID != nil {
		cat, err := s.catalog.GetByID(ctx, *car.CatalogCarID)
		if err == nil && cat.CellType != nil && *cat.CellType != "" {
			return cat.CellType
		}
	}
	if car.EVModelID == nil {
		return nil
	}
	model, err := s.evModels.GetByID(ctx, *car.EVModelID)
	if err != nil {
		return nil
	}
	return model.Chemistry
}

// sessionCapacity returns the SOC delta and the implied full-pack capacity for a
// session, plus whether the session qualifies for the SOH estimate. A session
// qualifies only with positive energy, both SOC values present, an increasing SOC,
// and a swing of at least minQualifyingDeltaSOC.
func sessionCapacity(sess domain.ChargingSession) (delta, capacity float64, ok bool) {
	if sess.KWhCharged == nil || *sess.KWhCharged <= 0 || sess.StartSOC == nil || sess.EndSOC == nil {
		return 0, 0, false
	}
	d := *sess.EndSOC - *sess.StartSOC
	if d < minQualifyingDeltaSOC {
		return 0, 0, false
	}
	delta = float64(d)
	// cap = energy into the pack / fraction of pack filled.
	capacity = (*sess.KWhCharged * chargingEfficiency) / (delta / 100)
	return delta, capacity, true
}

func confidenceFor(qualifying int, avgDelta float64) string {
	if qualifying >= highConfidenceSessions && avgDelta >= highConfidenceAvgDelta {
		return "high"
	}
	return "medium"
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
