package repository

import (
	"context"

	"voltana-api/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CatalogRepository is read-only access to the rich EV catalog (TASK-0033).
type CatalogRepository interface {
	ListCatalog(ctx context.Context) ([]domain.CatalogCar, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.CatalogCar, error)
}

type pgxCatalogRepository struct {
	db *pgxpool.Pool
}

func NewCatalogRepository(db *pgxpool.Pool) CatalogRepository {
	return &pgxCatalogRepository{db: db}
}

// NUMERIC columns are cast to float8 so they scan cleanly into *float64
// (same convention as evModelCols).
const catalogCols = `id, name_fa, name_en, brand, body_style_fa, class, body_type, segment,
	country, importer, platform, battery_capacity_kwh::float8, battery_voltage,
	usable_kwh::float8, cell_brand, cell_type, cooling, range_km, range_standard,
	consumption_kwh_per_100km::float8, motor_power_kw::float8, torque_nm, motor_count,
	motor_type, acceleration_0_100_s::float8, max_speed_kmh, drivetrain,
	ac_charge_kw::float8, ac_connector, dc_charge_kw::float8, dc_connector,
	fast_charge_window, fast_charge_min, v2l, v2g, ota, adas_level, radar_count,
	camera_count, weight_kg, trunk_liters, notes, exterior_colors, interior_colors,
	img_url, created_at, updated_at`

func (r *pgxCatalogRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.CatalogCar, error) {
	rows, err := r.db.Query(ctx, `SELECT `+catalogCols+` FROM ev_catalog WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := collectCatalogRows(rows)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return &items[0], nil
}

func (r *pgxCatalogRepository) ListCatalog(ctx context.Context) ([]domain.CatalogCar, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+catalogCols+` FROM ev_catalog ORDER BY brand NULLS LAST, name_en`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectCatalogRows(rows)
}

func collectCatalogRows(rows pgx.Rows) ([]domain.CatalogCar, error) {
	items := make([]domain.CatalogCar, 0)
	for rows.Next() {
		m := domain.CatalogCar{}
		var id pgtype.UUID
		if err := rows.Scan(&id, &m.NameFA, &m.NameEN, &m.Brand, &m.BodyStyleFA, &m.Class,
			&m.BodyType, &m.Segment, &m.Country, &m.Importer, &m.Platform,
			&m.BatteryCapacityKWh, &m.BatteryVoltage, &m.UsableKWh, &m.CellBrand,
			&m.CellType, &m.Cooling, &m.RangeKM, &m.RangeStandard,
			&m.ConsumptionKWhPer100KM, &m.MotorPowerKW, &m.TorqueNM, &m.MotorCount,
			&m.MotorType, &m.Acceleration0100S, &m.MaxSpeedKMh, &m.Drivetrain,
			&m.ACChargeKW, &m.ACConnector, &m.DCChargeKW, &m.DCConnector,
			&m.FastChargeWindow, &m.FastChargeMin, &m.V2L, &m.V2G, &m.OTA,
			&m.ADASLevel, &m.RadarCount, &m.CameraCount, &m.WeightKG, &m.TrunkLiters,
			&m.Notes, &m.ExteriorColors, &m.InteriorColors, &m.ImgURL,
			&m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		m.ID = uuid.UUID(id.Bytes)
		items = append(items, m)
	}
	return items, rows.Err()
}
