package domain

import (
	"encoding/json"
	"time"
)

// UserBackup is the export/import envelope for a single user's data
// (TASK-0037 FEAT-4). IDs inside are the EXPORTING account's ids and are used
// only to re-link sessions/snapshots to cars during import — the importer
// always mints fresh ids scoped to the authenticated user.
type UserBackup struct {
	SchemaVersion int              `json:"schema_version"`
	ExportedAt    time.Time        `json:"exported_at"`
	Settings      *BackupSettings  `json:"settings"`
	Cars          []BackupCar      `json:"cars"`
	Sessions      []BackupSession  `json:"charging_sessions"`
	Snapshots     []BackupSnapshot `json:"battery_snapshots"`
}

const BackupSchemaVersion = 1

type BackupSettings struct {
	DefaultCarID *string `json:"default_car_id"` // old car id, re-mapped on import
	PeakRate     float64 `json:"peak_rate"`
	MidRate      float64 `json:"mid_rate"`
	OffpeakRate  float64 `json:"offpeak_rate"`
	Currency     string  `json:"currency"`
}

type BackupCar struct {
	ID            string          `json:"id"` // export-side id (re-mapping key only)
	EVModelID     *string         `json:"ev_model_id"`
	CatalogCarID  *string         `json:"catalog_car_id"`
	Name          string          `json:"name"`
	LicensePlate  *string         `json:"license_plate"`
	OdometerKM    int             `json:"odometer_km"`
	SpecOverrides json.RawMessage `json:"spec_overrides"`
}

type BackupSession struct {
	CarID            string     `json:"car_id"` // references BackupCar.ID
	StartedAt        time.Time  `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at"`
	Location         *string    `json:"location"`
	KwhCharged       *float64   `json:"kwh_charged"`
	StartSoc         *int       `json:"start_soc"`
	EndSoc           *int       `json:"end_soc"`
	Cost             *float64   `json:"cost"`
	Notes            *string    `json:"notes"`
	EnergyPeakKwh    *float64   `json:"energy_peak_kwh"`
	EnergyMidKwh     *float64   `json:"energy_mid_kwh"`
	EnergyOffpeakKwh *float64   `json:"energy_offpeak_kwh"`
	OdometerKM       *int       `json:"odometer_km"`
	// Rate snapshot (FEAT-6) — preserved through export/import so a restore
	// never re-prices historical sessions.
	RatePeakAtTime    *float64 `json:"rate_peak_at_time"`
	RateMidAtTime     *float64 `json:"rate_mid_at_time"`
	RateOffpeakAtTime *float64 `json:"rate_offpeak_at_time"`
}

type BackupSnapshot struct {
	CarID                string    `json:"car_id"` // references BackupCar.ID
	SohPct               float64   `json:"soh_pct"`
	EstimatedCapacityKwh float64   `json:"estimated_capacity_kwh"`
	NominalCapacityKwh   float64   `json:"nominal_capacity_kwh"`
	SampleSessionCount   int       `json:"sample_session_count"`
	Confidence           string    `json:"confidence"`
	Method               string    `json:"method"`
	ComputedAt           time.Time `json:"computed_at"`
}

// ImportStats summarizes what an import wrote.
type ImportStats struct {
	Cars      int `json:"cars"`
	Sessions  int `json:"sessions"`
	Snapshots int `json:"snapshots"`
}
