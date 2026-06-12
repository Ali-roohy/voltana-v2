package repository

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SystemSettingsRepository persists key-value system settings.
type SystemSettingsRepository interface {
	// GetOTPDeliveryMethod returns "deeplink" or "contact_share".
	// Returns "deeplink" as a safe default if the row is missing.
	GetOTPDeliveryMethod(ctx context.Context) (string, error)
	SetOTPDeliveryMethod(ctx context.Context, method string) error
	// GetDefaultRates returns the admin default electricity rates copied into
	// each NEW user's settings (TASK-0037 FEAT-6). Missing keys fall back to
	// the historical 2000/1000/500.
	GetDefaultRates(ctx context.Context) (Rates, error)
	SetDefaultRates(ctx context.Context, r Rates) error
}

type pgxSystemSettingsRepo struct {
	db *pgxpool.Pool
}

func NewSystemSettingsRepository(db *pgxpool.Pool) SystemSettingsRepository {
	return &pgxSystemSettingsRepo{db: db}
}

func (r *pgxSystemSettingsRepo) GetOTPDeliveryMethod(ctx context.Context) (string, error) {
	var val string
	err := r.db.QueryRow(ctx,
		`SELECT value FROM system_settings WHERE key = 'otp_delivery_method'`,
	).Scan(&val)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "deeplink", nil
		}
		return "", err
	}
	return val, nil
}

func (r *pgxSystemSettingsRepo) GetDefaultRates(ctx context.Context) (Rates, error) {
	rt := Rates{Peak: 2000, Mid: 1000, Offpeak: 500}
	rows, err := r.db.Query(ctx,
		`SELECT key, value::float8 FROM system_settings
		 WHERE key IN ('default_peak_rate', 'default_mid_rate', 'default_offpeak_rate')`)
	if err != nil {
		return rt, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var val float64
		if err := rows.Scan(&key, &val); err != nil {
			return rt, err
		}
		switch key {
		case "default_peak_rate":
			rt.Peak = val
		case "default_mid_rate":
			rt.Mid = val
		case "default_offpeak_rate":
			rt.Offpeak = val
		}
	}
	return rt, rows.Err()
}

func (r *pgxSystemSettingsRepo) SetDefaultRates(ctx context.Context, rt Rates) error {
	for key, val := range map[string]float64{
		"default_peak_rate":    rt.Peak,
		"default_mid_rate":     rt.Mid,
		"default_offpeak_rate": rt.Offpeak,
	} {
		// value column is TEXT — format in Go (a $n::text cast on a float64
		// parameter trips pgx's type inference).
		str := strconv.FormatFloat(val, 'f', -1, 64)
		if _, err := r.db.Exec(ctx,
			`INSERT INTO system_settings (key, value) VALUES ($1, $2)
			 ON CONFLICT (key) DO UPDATE SET value = $2`, key, str,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *pgxSystemSettingsRepo) SetOTPDeliveryMethod(ctx context.Context, method string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO system_settings (key, value) VALUES ('otp_delivery_method', $1)
		 ON CONFLICT (key) DO UPDATE SET value = $1`,
		method,
	)
	return err
}
