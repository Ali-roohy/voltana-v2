package repository

import (
	"context"
	"errors"

	"voltana-api/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInvalidEVModel is returned when a car references a non-existent ev_model_id
// (foreign-key violation).
var ErrInvalidEVModel = errors.New("ev_model_id does not reference an existing model")

// ErrInvalidCatalogCar is returned when a car references a non-existent
// catalog_car_id (foreign-key violation).
var ErrInvalidCatalogCar = errors.New("catalog_car_id does not reference a catalog car")

// CarInput carries the mutable fields of a car for create/update.
type CarInput struct {
	Name          string
	EVModelID     *uuid.UUID
	CatalogCarID  *uuid.UUID
	SpecOverrides map[string]any
	LicensePlate  *string
	OdometerKM    int
}

// CarRepository is the persistence boundary for user-owned cars. Every method
// is scoped by userID — there is no way to address another user's row.
type CarRepository interface {
	Create(ctx context.Context, userID uuid.UUID, in CarInput) (*domain.Car, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) (items []domain.Car, total int, err error)
	GetByID(ctx context.Context, userID, id uuid.UUID) (*domain.Car, error)
	Update(ctx context.Context, userID, id uuid.UUID, in CarInput) (*domain.Car, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
}

type pgxCarRepository struct {
	db *pgxpool.Pool
}

func NewCarRepository(db *pgxpool.Pool) CarRepository {
	return &pgxCarRepository{db: db}
}

const carCols = `id, user_id, ev_model_id, catalog_car_id, spec_overrides, name, license_plate, odometer_km, created_at, updated_at`

func (r *pgxCarRepository) Create(ctx context.Context, userID uuid.UUID, in CarInput) (*domain.Car, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO cars (user_id, ev_model_id, catalog_car_id, spec_overrides, name, license_plate, odometer_km)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+carCols,
		userID, evModelArg(in.EVModelID), evModelArg(in.CatalogCarID), overridesArg(in.SpecOverrides),
		in.Name, in.LicensePlate, in.OdometerKM,
	)
	return scanCar(row)
}

func (r *pgxCarRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Car, int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+carCols+`, COUNT(*) OVER() AS total
		 FROM cars WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]domain.Car, 0)
	total := 0
	for rows.Next() {
		c := domain.Car{}
		var id, uID, evID, catID pgtype.UUID
		if err := rows.Scan(&id, &uID, &evID, &catID, &c.SpecOverrides, &c.Name, &c.LicensePlate,
			&c.OdometerKM, &c.CreatedAt, &c.UpdatedAt, &total); err != nil {
			return nil, 0, err
		}
		c.ID = uuid.UUID(id.Bytes)
		c.UserID = uuid.UUID(uID.Bytes)
		if evID.Valid {
			m := uuid.UUID(evID.Bytes)
			c.EVModelID = &m
		}
		if catID.Valid {
			m := uuid.UUID(catID.Bytes)
			c.CatalogCarID = &m
		}
		items = append(items, c)
	}
	return items, total, rows.Err()
}

func (r *pgxCarRepository) GetByID(ctx context.Context, userID, id uuid.UUID) (*domain.Car, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+carCols+` FROM cars WHERE id = $1 AND user_id = $2`, id, userID,
	)
	return scanCar(row)
}

func (r *pgxCarRepository) Update(ctx context.Context, userID, id uuid.UUID, in CarInput) (*domain.Car, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE cars SET ev_model_id = $1, catalog_car_id = $2, spec_overrides = $3,
		        name = $4, license_plate = $5, odometer_km = $6
		 WHERE id = $7 AND user_id = $8 RETURNING `+carCols,
		evModelArg(in.EVModelID), evModelArg(in.CatalogCarID), overridesArg(in.SpecOverrides),
		in.Name, in.LicensePlate, in.OdometerKM, id, userID,
	)
	return scanCar(row)
}

func (r *pgxCarRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM cars WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// evModelArg converts an optional UUID into a driver argument (NULL when nil).
func evModelArg(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return *id
}

// overridesArg keeps the NOT NULL spec_overrides column happy when no
// overrides were supplied.
func overridesArg(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func scanCar(row pgx.Row) (*domain.Car, error) {
	c := &domain.Car{}
	var id, userID, evID, catID pgtype.UUID
	err := row.Scan(&id, &userID, &evID, &catID, &c.SpecOverrides, &c.Name, &c.LicensePlate,
		&c.OdometerKM, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation
			if pgErr.ConstraintName == "cars_catalog_car_id_fkey" {
				return nil, ErrInvalidCatalogCar
			}
			return nil, ErrInvalidEVModel
		}
		return nil, err
	}
	c.ID = uuid.UUID(id.Bytes)
	c.UserID = uuid.UUID(userID.Bytes)
	if evID.Valid {
		m := uuid.UUID(evID.Bytes)
		c.EVModelID = &m
	}
	if catID.Valid {
		m := uuid.UUID(catID.Bytes)
		c.CatalogCarID = &m
	}
	return c, nil
}
