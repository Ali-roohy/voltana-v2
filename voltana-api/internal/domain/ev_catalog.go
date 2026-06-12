package domain

import (
	"time"

	"github.com/google/uuid"
)

// CatalogCar is one entry of the rich EV catalog (TASK-0033) — shared reference
// data with no owner, read-only via the API, written only by migrations. It is
// deliberately separate from EVModel (the slim car-creation autocomplete) and
// from Car (a user's own vehicle). Nullable specs are pointers so missing data
// serializes as null, never 0.
type CatalogCar struct {
	ID                     uuid.UUID `json:"id"`
	NameFA                 string    `json:"name_fa"`
	NameEN                 string    `json:"name_en"`
	Brand                  *string   `json:"brand"`
	BodyStyleFA            *string   `json:"body_style_fa"`
	Class                  *string   `json:"class"`
	BodyType               *string   `json:"body_type"`
	Segment                *string   `json:"segment"`
	Country                *string   `json:"country"`
	Importer               *string   `json:"importer"`
	Platform               *string   `json:"platform"`
	BatteryCapacityKWh     *float64  `json:"battery_capacity_kwh"`
	BatteryVoltage         *string   `json:"battery_voltage"`
	UsableKWh              *float64  `json:"usable_kwh"`
	CellBrand              *string   `json:"cell_brand"`
	CellType               *string   `json:"cell_type"`
	Cooling                *string   `json:"cooling"`
	RangeKM                *int      `json:"range_km"`
	RangeStandard          *string   `json:"range_standard"`
	ConsumptionKWhPer100KM *float64  `json:"consumption_kwh_per_100km"`
	MotorPowerKW           *float64  `json:"motor_power_kw"`
	TorqueNM               *int      `json:"torque_nm"`
	MotorCount             *int      `json:"motor_count"`
	MotorType              *string   `json:"motor_type"`
	Acceleration0100S      *float64  `json:"acceleration_0_100_s"`
	MaxSpeedKMh            *int      `json:"max_speed_kmh"`
	Drivetrain             *string   `json:"drivetrain"`
	ACChargeKW             *float64  `json:"ac_charge_kw"`
	ACConnector            *string   `json:"ac_connector"`
	DCChargeKW             *float64  `json:"dc_charge_kw"`
	DCConnector            *string   `json:"dc_connector"`
	FastChargeWindow       *string   `json:"fast_charge_window"`
	FastChargeMin          *int      `json:"fast_charge_to_80_min"`
	V2L                    *string   `json:"v2l"`
	V2G                    *string   `json:"v2g"`
	OTA                    *string   `json:"ota"`
	ADASLevel              *string   `json:"adas_level"`
	RadarCount             *int      `json:"radar_count"`
	CameraCount            *int      `json:"camera_count"`
	WeightKG               *int      `json:"weight_kg"`
	TrunkLiters            *int      `json:"trunk_liters"`
	Notes                  *string   `json:"notes"`
	ExteriorColors         []string  `json:"exterior_colors"`
	InteriorColors         []string  `json:"interior_colors"`
	ImgURL                 *string   `json:"img_url"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}
