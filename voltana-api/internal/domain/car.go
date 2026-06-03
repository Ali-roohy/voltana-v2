package domain

import (
	"time"

	"github.com/google/uuid"
)

// Car is a user-owned vehicle. UserID is never serialized to API responses.
type Car struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"-"`
	EVModelID    *uuid.UUID `json:"ev_model_id"`
	Name         string     `json:"name"`
	LicensePlate *string    `json:"license_plate"`
	OdometerKM   int        `json:"odometer_km"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
