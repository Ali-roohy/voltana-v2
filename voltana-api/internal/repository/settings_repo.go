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

// Rates holds a user's electricity tariff (Toman/kWh) for cost calculation.
type Rates struct {
	Peak    float64
	Mid     float64
	Offpeak float64
}

// SettingsRepository is access to the one-per-user user_settings row.
type SettingsRepository interface {
	// GetRates returns just the tariff (used by the charging cost calculation).
	GetRates(ctx context.Context, userID uuid.UUID) (Rates, error)
	// GetOrCreate returns the user's settings, creating a default row on first call.
	GetOrCreate(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error)
	// Update upserts the user's settings (full replace of rates + default_car_id).
	Update(ctx context.Context, userID uuid.UUID, in domain.SettingsInput) (*domain.UserSettings, error)
}

type pgxSettingsRepository struct {
	db *pgxpool.Pool
}

func NewSettingsRepository(db *pgxpool.Pool) SettingsRepository {
	return &pgxSettingsRepository{db: db}
}

// DECIMAL rates are cast to float8 so they scan into float64.
const settingsCols = `id, user_id, default_car_id,
	peak_rate::float8, mid_rate::float8, offpeak_rate::float8, currency, created_at, updated_at`

// GetRates returns the user's rates, or all-zero rates when no settings row
// exists yet. (GetOrCreate is the write path; this read stays side-effect-free.)
func (r *pgxSettingsRepository) GetRates(ctx context.Context, userID uuid.UUID) (Rates, error) {
	row := r.db.QueryRow(ctx,
		`SELECT peak_rate::float8, mid_rate::float8, offpeak_rate::float8
		 FROM user_settings WHERE user_id = $1`, userID,
	)
	var rt Rates
	if err := row.Scan(&rt.Peak, &rt.Mid, &rt.Offpeak); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Rates{}, nil
		}
		return Rates{}, err
	}
	return rt, nil
}

// GetOrCreate inserts a default row if none exists (ON CONFLICT DO NOTHING, so a
// plain read does not bump updated_at), then returns the row.
func (r *pgxSettingsRepository) GetOrCreate(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error) {
	if _, err := r.db.Exec(ctx,
		`INSERT INTO user_settings (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID,
	); err != nil {
		return nil, err
	}
	row := r.db.QueryRow(ctx, `SELECT `+settingsCols+` FROM user_settings WHERE user_id = $1`, userID)
	return scanUserSettings(row)
}

// Update upserts the user's settings so PUT works whether or not a row exists yet.
func (r *pgxSettingsRepository) Update(ctx context.Context, userID uuid.UUID, in domain.SettingsInput) (*domain.UserSettings, error) {
	currency := in.Currency
	if currency == "" {
		currency = "toman"
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO user_settings (user_id, default_car_id, peak_rate, mid_rate, offpeak_rate, currency)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id) DO UPDATE SET
		   default_car_id = EXCLUDED.default_car_id, peak_rate = EXCLUDED.peak_rate,
		   mid_rate = EXCLUDED.mid_rate, offpeak_rate = EXCLUDED.offpeak_rate,
		   currency = EXCLUDED.currency
		 RETURNING `+settingsCols,
		userID, uuidArg(in.DefaultCarID), in.PeakRate, in.MidRate, in.OffpeakRate, currency,
	)
	return scanUserSettings(row)
}

func scanUserSettings(row pgx.Row) (*domain.UserSettings, error) {
	s := &domain.UserSettings{}
	var id, userID, defCar pgtype.UUID
	err := row.Scan(&id, &userID, &defCar, &s.PeakRate, &s.MidRate, &s.OffpeakRate, &s.Currency, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation (bad default_car_id)
			return nil, ErrInvalidCar
		}
		return nil, err
	}
	s.ID = uuid.UUID(id.Bytes)
	s.UserID = uuid.UUID(userID.Bytes)
	if defCar.Valid {
		c := uuid.UUID(defCar.Bytes)
		s.DefaultCarID = &c
	}
	return s, nil
}
