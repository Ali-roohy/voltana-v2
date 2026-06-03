package repository

import (
	"context"
	"errors"

	"voltana-api/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StationRepository is the persistence boundary for charging stations. Unlike
// the car/charging repositories, stations are shared reference data — methods
// are NOT scoped by userID. Authorization (admin-only writes) is enforced above
// the repository, in the AdminOnly middleware.
type StationRepository interface {
	Create(ctx context.Context, in domain.StationInput) (*domain.ChargingStation, error)
	// List returns lightweight markers, optionally filtered to a bounding box
	// (nil = the full set). Bounded reference data, so no pagination.
	List(ctx context.Context, b *domain.StationBounds) ([]domain.StationMarker, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ChargingStation, error)
	Update(ctx context.Context, id uuid.UUID, in domain.StationInput) (*domain.ChargingStation, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgxStationRepository struct {
	db *pgxpool.Pool
}

func NewStationRepository(db *pgxpool.Pool) StationRepository {
	return &pgxStationRepository{db: db}
}

const stationCols = `id, name, latitude, longitude, address, connector_types, power_kw, operator, created_at, updated_at`

func (r *pgxStationRepository) Create(ctx context.Context, in domain.StationInput) (*domain.ChargingStation, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO charging_stations (name, latitude, longitude, address, connector_types, power_kw, operator)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+stationCols,
		in.Name, in.Latitude, in.Longitude, in.Address, in.ConnectorTypes, in.PowerKW, in.Operator,
	)
	return scanStation(row)
}

func (r *pgxStationRepository) List(ctx context.Context, b *domain.StationBounds) ([]domain.StationMarker, error) {
	var minLat, maxLat, minLng, maxLng any
	if b != nil {
		minLat, maxLat, minLng, maxLng = b.MinLat, b.MaxLat, b.MinLng, b.MaxLng
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, name, latitude, longitude, connector_types, power_kw
		 FROM charging_stations
		 WHERE ($1::float8 IS NULL OR latitude  >= $1)
		   AND ($2::float8 IS NULL OR latitude  <= $2)
		   AND ($3::float8 IS NULL OR longitude >= $3)
		   AND ($4::float8 IS NULL OR longitude <= $4)
		 ORDER BY name ASC`,
		minLat, maxLat, minLng, maxLng,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.StationMarker, 0)
	for rows.Next() {
		m := domain.StationMarker{}
		var id pgtype.UUID
		if err := rows.Scan(&id, &m.Name, &m.Latitude, &m.Longitude, &m.ConnectorTypes, &m.PowerKW); err != nil {
			return nil, err
		}
		m.ID = uuid.UUID(id.Bytes)
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *pgxStationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ChargingStation, error) {
	row := r.db.QueryRow(ctx, `SELECT `+stationCols+` FROM charging_stations WHERE id = $1`, id)
	return scanStation(row)
}

func (r *pgxStationRepository) Update(ctx context.Context, id uuid.UUID, in domain.StationInput) (*domain.ChargingStation, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE charging_stations SET
			name = $1, latitude = $2, longitude = $3, address = $4,
			connector_types = $5, power_kw = $6, operator = $7
		 WHERE id = $8 RETURNING `+stationCols,
		in.Name, in.Latitude, in.Longitude, in.Address, in.ConnectorTypes, in.PowerKW, in.Operator, id,
	)
	return scanStation(row)
}

func (r *pgxStationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM charging_stations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanStation(row pgx.Row) (*domain.ChargingStation, error) {
	s := &domain.ChargingStation{}
	var id pgtype.UUID
	err := row.Scan(&id, &s.Name, &s.Latitude, &s.Longitude, &s.Address,
		&s.ConnectorTypes, &s.PowerKW, &s.Operator, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	s.ID = uuid.UUID(id.Bytes)
	return s, nil
}
