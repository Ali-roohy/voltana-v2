package domain

import (
	"time"

	"github.com/google/uuid"
)

// EVModel is shared reference data (the EV catalog). It has no owner — every
// authenticated user can read it; it is never written via the API.
type EVModel struct {
	ID                 uuid.UUID `json:"id"`
	NameFA             string    `json:"name_fa"`
	NameEN             string    `json:"name_en"`
	Brand              *string   `json:"brand"`
	BatteryCapacityKWh *float64  `json:"battery_capacity_kwh"`
	RangeKM            *int      `json:"range_km"`
	Chemistry          *string   `json:"chemistry"`
	CreatedAt          time.Time `json:"created_at"`
}
