package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserSettings is the one-per-user row of electricity rates + default car.
// ID and UserID are never serialized to API responses.
type UserSettings struct {
	ID           uuid.UUID  `json:"-"`
	UserID       uuid.UUID  `json:"-"`
	DefaultCarID *uuid.UUID `json:"default_car_id"`
	PeakRate     float64    `json:"peak_rate"`
	MidRate      float64    `json:"mid_rate"`
	OffpeakRate  float64    `json:"offpeak_rate"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// SettingsInput is the full-replace payload for PUT /v1/settings. It lives in
// domain so the handler builds it without importing the repository layer.
type SettingsInput struct {
	DefaultCarID *uuid.UUID
	PeakRate     float64
	MidRate      float64
	OffpeakRate  float64
}
