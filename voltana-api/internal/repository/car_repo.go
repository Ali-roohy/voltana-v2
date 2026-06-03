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

// CarInput carries the mutable fields of a car for create/update.
type CarInput struct {
	Name         string
	EVModelID    *uuid.UUID
	LicensePlate *string
	OdometerKM   int
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

const carCols = `id, user_id, ev_model_id, name, license_plate, odometer_km, created_at, updated_at`

func (r *pgxCarRepository) Create(ctx context.Context, userID uuid.UUID, in CarInput) (*domain.Car, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO cars (user_id, ev_model_id, name, license_plate, odometer_km)
		 VALUES ($1, $2, $3, $4, $5) RETURNING `+carCols,
		userID, evModelArg(in.EVModelID), in.Name, in.LicensePlate, in.OdometerKM,
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
		var id, uID, evID pgtype.UUID
		if err := rows.Scan(&id, &uID, &evID, &c.Name, &c.LicensePlate, &c.OdometerKM,
			&c.CreatedAt, &c.UpdatedAt, &total); err != nil {
			return nil, 0, err
		}
		c.ID = uuid.UUID(id.Bytes)
		c.UserID = uuid.UUID(uID.Bytes)
		if evID.Valid {
			m := uuid.UUID(evID.Bytes)
			c.EVModelID = &m
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
		`UPDATE cars SET ev_model_id = $1, name = $2, license_plate = $3, odometer_km = $4
		 WHERE id = $5 AND user_id = $6 RETURNING `+carCols,
		evModelArg(in.EVModelID), in.Name, in.LicensePlate, in.OdometerKM, id, userID,
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

func scanCar(row pgx.Row) (*domain.Car, error) {
	c := &domain.Car{}
	var id, userID, evID pgtype.UUID
	err := row.Scan(&id, &userID, &evID, &c.Name, &c.LicensePlate, &c.OdometerKM, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation
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
	return c, nil
}
