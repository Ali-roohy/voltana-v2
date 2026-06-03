package domain

import (
	"time"

	"github.com/google/uuid"
)

// ChargingStation is shared reference data (not user-owned): every authed user
// can read it, but only admins may mutate it. Nullable columns are pointers.
type ChargingStation struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	Address        *string   `json:"address"`
	ConnectorTypes *string   `json:"connector_types"`
	PowerKW        *int      `json:"power_kw"`
	Operator       *string   `json:"operator"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// StationMarker is the lightweight projection returned by the list endpoint —
// just enough to plot and label a marker on the map, keeping the payload small.
type StationMarker struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	ConnectorTypes *string   `json:"connector_types"`
	PowerKW        *int      `json:"power_kw"`
}

// StationInput carries the mutable fields of a station for create/update.
type StationInput struct {
	Name           string
	Latitude       float64
	Longitude      float64
	Address        *string
	ConnectorTypes *string
	PowerKW        *int
	Operator       *string
}

// StationBounds is an optional bounding-box filter for the list endpoint. When
// nil, the full set is returned; when set, only stations inside the box.
type StationBounds struct {
	MinLat, MaxLat float64
	MinLng, MaxLng float64
}
