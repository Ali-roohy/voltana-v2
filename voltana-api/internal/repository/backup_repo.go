package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"voltana-api/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BackupRepository exports and imports a single user's data (TASK-0037
// FEAT-4). All queries are scoped to the given userID; import runs in one
// transaction, replaces the user's existing data, and mints fresh ids —
// ids embedded in the payload are never written.
type BackupRepository interface {
	ExportUserData(ctx context.Context, userID uuid.UUID) (*domain.UserBackup, error)
	ImportUserData(ctx context.Context, userID uuid.UUID, b *domain.UserBackup) (*domain.ImportStats, error)
}

type pgxBackupRepository struct {
	db *pgxpool.Pool
}

func NewBackupRepository(db *pgxpool.Pool) BackupRepository {
	return &pgxBackupRepository{db: db}
}

func (r *pgxBackupRepository) ExportUserData(ctx context.Context, userID uuid.UUID) (*domain.UserBackup, error) {
	b := &domain.UserBackup{
		SchemaVersion: domain.BackupSchemaVersion,
		Cars:          []domain.BackupCar{},
		Sessions:      []domain.BackupSession{},
		Snapshots:     []domain.BackupSnapshot{},
	}

	// Settings (may not exist yet — GET /v1/settings auto-creates, export tolerates absence).
	var s domain.BackupSettings
	var defaultCarID *uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT default_car_id, peak_rate, mid_rate, offpeak_rate, currency
		   FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&defaultCarID, &s.PeakRate, &s.MidRate, &s.OffpeakRate, &s.Currency)
	if err == nil {
		if defaultCarID != nil {
			str := defaultCarID.String()
			s.DefaultCarID = &str
		}
		b.Settings = &s
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, ev_model_id, catalog_car_id, name, license_plate, odometer_km, spec_overrides
		   FROM cars WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("export cars: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var c domain.BackupCar
		var id uuid.UUID
		var evID, catID *uuid.UUID
		var overrides []byte
		if err := rows.Scan(&id, &evID, &catID, &c.Name, &c.LicensePlate, &c.OdometerKM, &overrides); err != nil {
			return nil, err
		}
		c.ID = id.String()
		if evID != nil {
			str := evID.String()
			c.EVModelID = &str
		}
		if catID != nil {
			str := catID.String()
			c.CatalogCarID = &str
		}
		c.SpecOverrides = json.RawMessage(overrides)
		b.Cars = append(b.Cars, c)
	}
	rows.Close()

	rows, err = r.db.Query(ctx,
		`SELECT car_id, started_at, ended_at, location, kwh_charged, start_soc, end_soc,
		        cost, notes, energy_peak_kwh, energy_mid_kwh, energy_offpeak_kwh, odometer_km,
		        rate_peak_at_time, rate_mid_at_time, rate_offpeak_at_time
		   FROM charging_sessions WHERE user_id = $1 ORDER BY started_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("export sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s domain.BackupSession
		var carID uuid.UUID
		if err := rows.Scan(&carID, &s.StartedAt, &s.EndedAt, &s.Location, &s.KwhCharged,
			&s.StartSoc, &s.EndSoc, &s.Cost, &s.Notes,
			&s.EnergyPeakKwh, &s.EnergyMidKwh, &s.EnergyOffpeakKwh, &s.OdometerKM,
			&s.RatePeakAtTime, &s.RateMidAtTime, &s.RateOffpeakAtTime); err != nil {
			return nil, err
		}
		s.CarID = carID.String()
		b.Sessions = append(b.Sessions, s)
	}
	rows.Close()

	rows, err = r.db.Query(ctx,
		`SELECT car_id, soh_pct, estimated_capacity_kwh, nominal_capacity_kwh,
		        sample_session_count, confidence, method, computed_at
		   FROM battery_health_snapshots WHERE user_id = $1 ORDER BY computed_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("export snapshots: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sn domain.BackupSnapshot
		var carID uuid.UUID
		if err := rows.Scan(&carID, &sn.SohPct, &sn.EstimatedCapacityKwh, &sn.NominalCapacityKwh,
			&sn.SampleSessionCount, &sn.Confidence, &sn.Method, &sn.ComputedAt); err != nil {
			return nil, err
		}
		sn.CarID = carID.String()
		b.Snapshots = append(b.Snapshots, sn)
	}
	return b, rows.Err()
}

func (r *pgxBackupRepository) ImportUserData(ctx context.Context, userID uuid.UUID, b *domain.UserBackup) (*domain.ImportStats, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck — no-op after commit

	// Replace strategy: wipe the user's own rows first (FK order: snapshots
	// and sessions cascade from cars, but delete explicitly for clarity).
	for _, q := range []string{
		`DELETE FROM battery_health_snapshots WHERE user_id = $1`,
		`DELETE FROM charging_sessions WHERE user_id = $1`,
		`DELETE FROM cars WHERE user_id = $1`,
	} {
		if _, err := tx.Exec(ctx, q, userID); err != nil {
			return nil, fmt.Errorf("import wipe: %w", err)
		}
	}

	stats := &domain.ImportStats{}

	// Cars — fresh ids; old id → new id map drives session/snapshot re-linking.
	// ev_model_id / catalog_car_id are global catalogs: unknown ids degrade to
	// NULL via the scalar subquery instead of failing the FK.
	carIDMap := make(map[string]uuid.UUID, len(b.Cars))
	for _, c := range b.Cars {
		overrides := c.SpecOverrides
		if len(overrides) == 0 {
			overrides = json.RawMessage(`{}`)
		}
		var newID uuid.UUID
		err := tx.QueryRow(ctx,
			`INSERT INTO cars (user_id, ev_model_id, catalog_car_id, name, license_plate, odometer_km, spec_overrides)
			 VALUES ($1,
			         (SELECT id FROM ev_models  WHERE id = $2),
			         (SELECT id FROM ev_catalog WHERE id = $3),
			         $4, $5, $6, $7)
			 RETURNING id`,
			userID, c.EVModelID, c.CatalogCarID, c.Name, c.LicensePlate, c.OdometerKM, overrides,
		).Scan(&newID)
		if err != nil {
			return nil, fmt.Errorf("import car %q: %w", c.Name, err)
		}
		carIDMap[c.ID] = newID
		stats.Cars++
	}

	for i, s := range b.Sessions {
		newCarID, ok := carIDMap[s.CarID]
		if !ok {
			return nil, fmt.Errorf("import session %d: unknown car_id %q", i, s.CarID)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO charging_sessions
			   (user_id, car_id, started_at, ended_at, location, kwh_charged, start_soc, end_soc,
			    cost, notes, energy_peak_kwh, energy_mid_kwh, energy_offpeak_kwh, odometer_km,
			    rate_peak_at_time, rate_mid_at_time, rate_offpeak_at_time)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
			userID, newCarID, s.StartedAt, s.EndedAt, s.Location, s.KwhCharged, s.StartSoc, s.EndSoc,
			s.Cost, s.Notes, s.EnergyPeakKwh, s.EnergyMidKwh, s.EnergyOffpeakKwh, s.OdometerKM,
			s.RatePeakAtTime, s.RateMidAtTime, s.RateOffpeakAtTime,
		); err != nil {
			return nil, fmt.Errorf("import session %d: %w", i, err)
		}
		stats.Sessions++
	}

	for i, sn := range b.Snapshots {
		newCarID, ok := carIDMap[sn.CarID]
		if !ok {
			return nil, fmt.Errorf("import snapshot %d: unknown car_id %q", i, sn.CarID)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO battery_health_snapshots
			   (user_id, car_id, soh_pct, estimated_capacity_kwh, nominal_capacity_kwh,
			    sample_session_count, confidence, method, computed_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			userID, newCarID, sn.SohPct, sn.EstimatedCapacityKwh, sn.NominalCapacityKwh,
			sn.SampleSessionCount, sn.Confidence, sn.Method, sn.ComputedAt,
		); err != nil {
			return nil, fmt.Errorf("import snapshot %d: %w", i, err)
		}
		stats.Snapshots++
	}

	// Settings: upsert rates/currency; default_car_id re-mapped (NULL if the
	// referenced car wasn't part of the backup).
	if b.Settings != nil {
		var defaultCar *uuid.UUID
		if b.Settings.DefaultCarID != nil {
			if mapped, ok := carIDMap[*b.Settings.DefaultCarID]; ok {
				defaultCar = &mapped
			}
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_settings (user_id, default_car_id, peak_rate, mid_rate, offpeak_rate, currency)
			 VALUES ($1,$2,$3,$4,$5,$6)
			 ON CONFLICT (user_id) DO UPDATE SET
			   default_car_id = EXCLUDED.default_car_id,
			   peak_rate      = EXCLUDED.peak_rate,
			   mid_rate       = EXCLUDED.mid_rate,
			   offpeak_rate   = EXCLUDED.offpeak_rate,
			   currency       = EXCLUDED.currency,
			   updated_at     = now()`,
			userID, defaultCar, b.Settings.PeakRate, b.Settings.MidRate, b.Settings.OffpeakRate, b.Settings.Currency,
		); err != nil {
			return nil, fmt.Errorf("import settings: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return stats, nil
}
