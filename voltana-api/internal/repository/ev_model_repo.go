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

// EVModelRepository is read-only access to the shared EV catalog.
type EVModelRepository interface {
	List(ctx context.Context, q string, limit, offset int) (items []domain.EVModel, total int, err error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.EVModel, error)
}

type pgxEVModelRepository struct {
	db *pgxpool.Pool
}

func NewEVModelRepository(db *pgxpool.Pool) EVModelRepository {
	return &pgxEVModelRepository{db: db}
}

// battery_capacity_kwh is cast to float8 so it scans cleanly into *float64.
const evModelCols = `id, name_fa, name_en, brand, battery_capacity_kwh::float8, range_km, chemistry, created_at`

func (r *pgxEVModelRepository) List(ctx context.Context, q string, limit, offset int) ([]domain.EVModel, int, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if q == "" {
		rows, err = r.db.Query(ctx,
			`SELECT `+evModelCols+`, COUNT(*) OVER() AS total
			 FROM ev_models ORDER BY name_en ASC LIMIT $1 OFFSET $2`,
			limit, offset,
		)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT `+evModelCols+`, COUNT(*) OVER() AS total
			 FROM ev_models
			 WHERE to_tsvector('simple', name_fa) @@ plainto_tsquery('simple', $1)
			    OR to_tsvector('simple', name_en) @@ plainto_tsquery('simple', $1)
			 ORDER BY name_en ASC LIMIT $2 OFFSET $3`,
			q, limit, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]domain.EVModel, 0)
	total := 0
	for rows.Next() {
		m := domain.EVModel{}
		var id pgtype.UUID
		if err := rows.Scan(&id, &m.NameFA, &m.NameEN, &m.Brand, &m.BatteryCapacityKWh,
			&m.RangeKM, &m.Chemistry, &m.CreatedAt, &total); err != nil {
			return nil, 0, err
		}
		m.ID = uuid.UUID(id.Bytes)
		items = append(items, m)
	}
	return items, total, rows.Err()
}

func (r *pgxEVModelRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.EVModel, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+evModelCols+` FROM ev_models WHERE id = $1`, id,
	)
	m := &domain.EVModel{}
	var mID pgtype.UUID
	err := row.Scan(&mID, &m.NameFA, &m.NameEN, &m.Brand, &m.BatteryCapacityKWh,
		&m.RangeKM, &m.Chemistry, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	m.ID = uuid.UUID(mID.Bytes)
	return m, nil
}
