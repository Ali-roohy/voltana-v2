package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SystemSettingsRepository persists key-value system settings.
type SystemSettingsRepository interface {
	// GetOTPDeliveryMethod returns "deeplink" or "contact_share".
	// Returns "deeplink" as a safe default if the row is missing.
	GetOTPDeliveryMethod(ctx context.Context) (string, error)
	SetOTPDeliveryMethod(ctx context.Context, method string) error
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

func (r *pgxSystemSettingsRepo) SetOTPDeliveryMethod(ctx context.Context, method string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO system_settings (key, value) VALUES ('otp_delivery_method', $1)
		 ON CONFLICT (key) DO UPDATE SET value = $1`,
		method,
	)
	return err
}
