package domain

import (
	"time"

	"github.com/google/uuid"
)

// Car is a user-owned vehicle. UserID is never serialized to API responses.
// CatalogCarID/SpecOverrides (TASK-0034) link the car to an ev_catalog entry;
// SpecOverrides holds only the user's diff from the catalog specs and is echoed
// verbatim — merging happens client-side.
type Car struct {
	ID            uuid.UUID      `json:"id"`
	UserID        uuid.UUID      `json:"-"`
	EVModelID     *uuid.UUID     `json:"ev_model_id"`
	CatalogCarID  *uuid.UUID     `json:"catalog_car_id"`
	SpecOverrides map[string]any `json:"spec_overrides"`
	Name          string         `json:"name"`
	LicensePlate  *string        `json:"license_plate"`
	OdometerKM    int            `json:"odometer_km"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
