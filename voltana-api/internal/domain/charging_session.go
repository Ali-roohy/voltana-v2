package domain

import (
	"time"

	"github.com/google/uuid"
)

// ChargingSession is a user-owned charging log entry. UserID is never serialized
// to API responses. The per-period energy columns (peak/mid/offpeak) back the
// server-side time-of-use cost calculation; kwh_charged is the grand total.
type ChargingSession struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"-"`
	CarID            uuid.UUID  `json:"car_id"`
	StartedAt        time.Time  `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at"`
	Location         *string    `json:"location"`
	KWhCharged       *float64   `json:"kwh_charged"`
	EnergyPeakKWh    *float64   `json:"energy_peak_kwh"`
	EnergyMidKWh     *float64   `json:"energy_mid_kwh"`
	EnergyOffpeakKWh *float64   `json:"energy_offpeak_kwh"`
	StartSOC         *int       `json:"start_soc"`
	EndSOC           *int       `json:"end_soc"`
	Cost             *float64   `json:"cost"`
	Notes            *string    `json:"notes"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ChargingInput carries the mutable fields of a charging session for create/update.
// It lives in domain (not repository) so the handler can build it without importing
// the repository layer — handler → service → repository stays one-directional.
type ChargingInput struct {
	CarID            uuid.UUID
	StartedAt        time.Time
	EndedAt          *time.Time
	Location         *string
	KWhCharged       *float64
	EnergyPeakKWh    *float64
	EnergyMidKWh     *float64
	EnergyOffpeakKWh *float64
	StartSOC         *int
	EndSOC           *int
	Cost             *float64
	Notes            *string
}

// ChargingFilter narrows a session list. Nil fields are ignored.
type ChargingFilter struct {
	CarID *uuid.UUID
	From  *time.Time
	To    *time.Time
}
