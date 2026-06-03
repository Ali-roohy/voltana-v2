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

// BatteryRepository persists and reads battery-health snapshots. Reads are scoped
// by userID so one user can never address another's car — there is no unscoped
// accessor.
type BatteryRepository interface {
	// Save inserts a new snapshot (the table is append-only history).
	Save(ctx context.Context, snap *domain.BatteryHealthSnapshot) error
	// GetLatest returns the most recent snapshot for a user's car, or ErrNotFound
	// when none has been computed yet.
	GetLatest(ctx context.Context, userID, carID uuid.UUID) (*domain.BatteryHealthSnapshot, error)
	// ListByCar returns up to limit snapshots for a user's car, ordered oldest→newest
	// (so a trend chart's x-axis is chronological). Empty slice when none exist.
	ListByCar(ctx context.Context, userID, carID uuid.UUID, limit int) ([]domain.BatteryHealthSnapshot, error)
}

type pgxBatteryRepository struct {
	db *pgxpool.Pool
}

func NewBatteryRepository(db *pgxpool.Pool) BatteryRepository {
	return &pgxBatteryRepository{db: db}
}

const bhsCols = `id, car_id, user_id, soh_pct, estimated_capacity_kwh, nominal_capacity_kwh,
	sample_session_count, confidence, method, computed_at`

func (r *pgxBatteryRepository) Save(ctx context.Context, snap *domain.BatteryHealthSnapshot) error {
	row := r.db.QueryRow(ctx,
		`INSERT INTO battery_health_snapshots
		 (car_id, user_id, soh_pct, estimated_capacity_kwh, nominal_capacity_kwh,
		  sample_session_count, confidence, method)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING `+bhsCols,
		snap.CarID, snap.UserID, snap.SOHPct, snap.EstimatedCapacityKWh, snap.NominalCapacityKWh,
		snap.SampleSessionCount, snap.Confidence, snap.Method,
	)
	return scanSnapshot(row, snap)
}

func (r *pgxBatteryRepository) GetLatest(ctx context.Context, userID, carID uuid.UUID) (*domain.BatteryHealthSnapshot, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+bhsCols+`
		 FROM battery_health_snapshots
		 WHERE car_id = $1 AND user_id = $2
		 ORDER BY computed_at DESC
		 LIMIT 1`,
		carID, userID,
	)
	snap := &domain.BatteryHealthSnapshot{}
	if err := scanSnapshot(row, snap); err != nil {
		return nil, err
	}
	return snap, nil
}

func (r *pgxBatteryRepository) ListByCar(ctx context.Context, userID, carID uuid.UUID, limit int) ([]domain.BatteryHealthSnapshot, error) {
	// Take the most recent `limit` snapshots (DESC, index-aligned with
	// bhs_car_computed_idx), then reverse to chronological (ASC) so the trend
	// chart's x-axis reads oldest→newest without ever dropping recent points.
	rows, err := r.db.Query(ctx,
		`SELECT `+bhsCols+`
		 FROM battery_health_snapshots
		 WHERE car_id = $1 AND user_id = $2
		 ORDER BY computed_at DESC
		 LIMIT $3`,
		carID, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.BatteryHealthSnapshot, 0)
	for rows.Next() {
		snap := domain.BatteryHealthSnapshot{}
		if err := scanSnapshot(rows, &snap); err != nil {
			return nil, err
		}
		items = append(items, snap)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Reverse the newest-first rows into chronological order.
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items, nil
}

func scanSnapshot(row pgx.Row, snap *domain.BatteryHealthSnapshot) error {
	var id, carID, userID pgtype.UUID
	err := row.Scan(&id, &carID, &userID, &snap.SOHPct, &snap.EstimatedCapacityKWh,
		&snap.NominalCapacityKWh, &snap.SampleSessionCount, &snap.Confidence, &snap.Method, &snap.ComputedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	snap.ID = uuid.UUID(id.Bytes)
	snap.CarID = uuid.UUID(carID.Bytes)
	snap.UserID = uuid.UUID(userID.Bytes)
	return nil
}
