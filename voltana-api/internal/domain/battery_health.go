package domain

import (
	"time"

	"github.com/google/uuid"
)

// BatteryHealthSnapshot is a point-in-time estimate of a car's battery State of
// Health (SOH), derived from charging history. UserID is never serialized.
type BatteryHealthSnapshot struct {
	ID                   uuid.UUID `json:"id"`
	CarID                uuid.UUID `json:"car_id"`
	UserID               uuid.UUID `json:"-"`
	SOHPct               float64   `json:"soh_pct"`
	EstimatedCapacityKWh float64   `json:"estimated_capacity_kwh"`
	NominalCapacityKWh   float64   `json:"nominal_capacity_kwh"`
	SampleSessionCount   int       `json:"sample_session_count"`
	Confidence           string    `json:"confidence"` // "low" | "medium" | "high"
	Method               string    `json:"method"`
	ComputedAt           time.Time `json:"computed_at"`
}

// DashboardStats are a user's lifetime analytics totals. AvgKWhPer100KM is nil
// when total distance is 0 (no odometer set yet) to avoid a meaningless ratio.
type DashboardStats struct {
	TotalKWh       float64  `json:"total_kwh"`
	TotalCost      float64  `json:"total_cost"`
	TotalKM        int      `json:"total_km"`
	AvgKWhPer100KM *float64 `json:"avg_kwh_per_100km"`
	SessionCount   int      `json:"session_count"`
}

// BatteryRecommendation is chemistry-aware battery-care advice for a car.
type BatteryRecommendation struct {
	Chemistry     *string  `json:"chemistry"`      // "LFP" | "NMC" | "NCA" | null (unknown)
	ChargeCeiling int      `json:"charge_ceiling"` // recommended daily charge ceiling %
	Tips          []string `json:"tips"`           // human-readable care tips
}
