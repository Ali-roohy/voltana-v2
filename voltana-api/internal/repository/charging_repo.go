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

// ErrInvalidCar is returned when a session references a car_id that does not
// exist (foreign-key violation). Ownership is enforced earlier in the service
// layer via CarRepository; this is defense-in-depth for non-existent cars.
var ErrInvalidCar = errors.New("car_id does not reference an existing car")

// ChargingRepository is the persistence boundary for user-owned charging
// sessions. Every method is scoped by userID — there is no way to address
// another user's row.
type ChargingRepository interface {
	Create(ctx context.Context, userID uuid.UUID, in domain.ChargingInput) (*domain.ChargingSession, error)
	ListByUser(ctx context.Context, userID uuid.UUID, f domain.ChargingFilter, limit, offset int) (items []domain.ChargingSession, total int, err error)
	GetByID(ctx context.Context, userID, id uuid.UUID) (*domain.ChargingSession, error)
	Update(ctx context.Context, userID, id uuid.UUID, in domain.ChargingInput) (*domain.ChargingSession, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	// AggregateByUser returns the user's lifetime totals (NULL sums → 0), computed
	// in SQL so it is not bounded by the list pagination cap.
	AggregateByUser(ctx context.Context, userID uuid.UUID) (totalKWh, totalCost float64, sessionCount int, err error)
	// EfficiencyAggregateByUser sums the energy and the odometer-derived distance
	// across consecutive sessions (per car, by time) that both carry an odometer
	// reading with a positive delta — the inputs for the fleet kWh/100km average.
	EfficiencyAggregateByUser(ctx context.Context, userID uuid.UUID) (sumKWh, sumKM float64, err error)
}

type pgxChargingRepository struct {
	db *pgxpool.Pool
}

func NewChargingRepository(db *pgxpool.Pool) ChargingRepository {
	return &pgxChargingRepository{db: db}
}

// Decimals are cast to float8 so they scan into *float64; nullable columns scan
// into pointer fields directly.
const chargingCols = `id, user_id, car_id, started_at, ended_at, location,
	kwh_charged::float8, energy_peak_kwh::float8, energy_mid_kwh::float8, energy_offpeak_kwh::float8,
	start_soc, end_soc, cost::float8, notes, odometer_km, created_at, updated_at`

func (r *pgxChargingRepository) AggregateByUser(ctx context.Context, userID uuid.UUID) (float64, float64, int, error) {
	var totalKWh, totalCost float64
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(kwh_charged), 0)::float8,
		        COALESCE(SUM(cost), 0)::float8,
		        COUNT(*)
		 FROM charging_sessions WHERE user_id = $1`,
		userID,
	).Scan(&totalKWh, &totalCost, &count)
	if err != nil {
		return 0, 0, 0, err
	}
	return totalKWh, totalCost, count, nil
}

func (r *pgxChargingRepository) EfficiencyAggregateByUser(ctx context.Context, userID uuid.UUID) (float64, float64, error) {
	// Per car, the distance for a session is its odometer minus the previous
	// session's odometer (by time). Sum energy + distance only over consecutive
	// pairs where both readings exist, the delta is positive, and energy is known.
	var sumKWh, sumKM float64
	err := r.db.QueryRow(ctx,
		`WITH deltas AS (
			SELECT kwh_charged,
			       odometer_km - LAG(odometer_km) OVER (PARTITION BY car_id ORDER BY started_at) AS km_driven
			FROM charging_sessions
			WHERE user_id = $1
		 )
		 SELECT COALESCE(SUM(kwh_charged) FILTER (WHERE km_driven > 0 AND kwh_charged IS NOT NULL), 0)::float8,
		        COALESCE(SUM(km_driven)   FILTER (WHERE km_driven > 0 AND kwh_charged IS NOT NULL), 0)::float8
		 FROM deltas`,
		userID,
	).Scan(&sumKWh, &sumKM)
	if err != nil {
		return 0, 0, err
	}
	return sumKWh, sumKM, nil
}

func (r *pgxChargingRepository) Create(ctx context.Context, userID uuid.UUID, in domain.ChargingInput) (*domain.ChargingSession, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO charging_sessions
			(user_id, car_id, started_at, ended_at, location, kwh_charged,
			 energy_peak_kwh, energy_mid_kwh, energy_offpeak_kwh, start_soc, end_soc, cost, notes, odometer_km)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 RETURNING `+chargingCols,
		userID, in.CarID, in.StartedAt, in.EndedAt, in.Location, in.KWhCharged,
		in.EnergyPeakKWh, in.EnergyMidKWh, in.EnergyOffpeakKWh, in.StartSOC, in.EndSOC, in.Cost, in.Notes, in.OdometerKM,
	)
	return scanChargingSession(row)
}

func (r *pgxChargingRepository) ListByUser(ctx context.Context, userID uuid.UUID, f domain.ChargingFilter, limit, offset int) ([]domain.ChargingSession, int, error) {
	rows, err := r.db.Query(ctx,
		// prev_odometer = the same car's immediately-earlier session odometer (window
		// runs over the WHERE-filtered set, before LIMIT/OFFSET, so it's correct
		// across pages). The service turns it into the per-session kWh/100km.
		`SELECT `+chargingCols+`,
		        LAG(odometer_km) OVER (PARTITION BY car_id ORDER BY started_at) AS prev_odometer,
		        COUNT(*) OVER() AS total
		 FROM charging_sessions
		 WHERE user_id = $1
		   AND ($2::uuid        IS NULL OR car_id     = $2)
		   AND ($3::timestamptz IS NULL OR started_at >= $3)
		   AND ($4::timestamptz IS NULL OR started_at <= $4)
		 ORDER BY started_at DESC
		 LIMIT $5 OFFSET $6`,
		userID, uuidArg(f.CarID), f.From, f.To, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]domain.ChargingSession, 0)
	total := 0
	for rows.Next() {
		s := domain.ChargingSession{}
		var id, uID, carID pgtype.UUID
		if err := rows.Scan(&id, &uID, &carID, &s.StartedAt, &s.EndedAt, &s.Location,
			&s.KWhCharged, &s.EnergyPeakKWh, &s.EnergyMidKWh, &s.EnergyOffpeakKWh,
			&s.StartSOC, &s.EndSOC, &s.Cost, &s.Notes, &s.OdometerKM, &s.CreatedAt, &s.UpdatedAt,
			&s.PrevOdometerKM, &total); err != nil {
			return nil, 0, err
		}
		s.ID = uuid.UUID(id.Bytes)
		s.UserID = uuid.UUID(uID.Bytes)
		s.CarID = uuid.UUID(carID.Bytes)
		items = append(items, s)
	}
	return items, total, rows.Err()
}

func (r *pgxChargingRepository) GetByID(ctx context.Context, userID, id uuid.UUID) (*domain.ChargingSession, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+chargingCols+` FROM charging_sessions WHERE id = $1 AND user_id = $2`, id, userID,
	)
	return scanChargingSession(row)
}

func (r *pgxChargingRepository) Update(ctx context.Context, userID, id uuid.UUID, in domain.ChargingInput) (*domain.ChargingSession, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE charging_sessions SET
			car_id = $1, started_at = $2, ended_at = $3, location = $4, kwh_charged = $5,
			energy_peak_kwh = $6, energy_mid_kwh = $7, energy_offpeak_kwh = $8,
			start_soc = $9, end_soc = $10, cost = $11, notes = $12, odometer_km = $13
		 WHERE id = $14 AND user_id = $15
		 RETURNING `+chargingCols,
		in.CarID, in.StartedAt, in.EndedAt, in.Location, in.KWhCharged,
		in.EnergyPeakKWh, in.EnergyMidKWh, in.EnergyOffpeakKWh, in.StartSOC, in.EndSOC, in.Cost, in.Notes, in.OdometerKM,
		id, userID,
	)
	return scanChargingSession(row)
}

func (r *pgxChargingRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM charging_sessions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// uuidArg converts an optional UUID into a driver argument (NULL when nil).
func uuidArg(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return *id
}

func scanChargingSession(row pgx.Row) (*domain.ChargingSession, error) {
	s := &domain.ChargingSession{}
	var id, userID, carID pgtype.UUID
	err := row.Scan(&id, &userID, &carID, &s.StartedAt, &s.EndedAt, &s.Location,
		&s.KWhCharged, &s.EnergyPeakKWh, &s.EnergyMidKWh, &s.EnergyOffpeakKWh,
		&s.StartSOC, &s.EndSOC, &s.Cost, &s.Notes, &s.OdometerKM, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation
			return nil, ErrInvalidCar
		}
		return nil, err
	}
	s.ID = uuid.UUID(id.Bytes)
	s.UserID = uuid.UUID(userID.Bytes)
	s.CarID = uuid.UUID(carID.Bytes)
	return s, nil
}
