package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"

	"github.com/google/uuid"
)

// Import sanity caps — generous for a personal EV log, tight enough to stop
// abuse (the HTTP layer additionally caps the body at 5 MB).
const (
	importMaxCars      = 100
	importMaxSessions  = 20000
	importMaxSnapshots = 10000
)

var ErrInvalidBackup = errors.New("invalid backup payload")

// BackupValidationError carries a user-facing reason for a 400.
type BackupValidationError struct{ Reason string }

func (e *BackupValidationError) Error() string { return e.Reason }
func (e *BackupValidationError) Unwrap() error { return ErrInvalidBackup }

// BackupService exports/imports the authenticated user's own data
// (TASK-0037 FEAT-4). Import is strictly replace-own-data: the repository
// wipes the user's rows and re-inserts with fresh ids, so payload ids can
// never reach another user's data.
type BackupService struct {
	repo  repository.BackupRepository
	cache TokenStore // dashboard cache bust after import
}

func NewBackupService(repo repository.BackupRepository, cache TokenStore) *BackupService {
	return &BackupService{repo: repo, cache: cache}
}

func (s *BackupService) Export(ctx context.Context, userID uuid.UUID) (*domain.UserBackup, error) {
	b, err := s.repo.ExportUserData(ctx, userID)
	if err != nil {
		return nil, err
	}
	b.ExportedAt = time.Now().UTC()
	return b, nil
}

func (s *BackupService) Import(ctx context.Context, userID uuid.UUID, b *domain.UserBackup) (*domain.ImportStats, error) {
	if err := validateBackup(b); err != nil {
		return nil, err
	}
	stats, err := s.repo.ImportUserData(ctx, userID, b)
	if err != nil {
		return nil, err
	}
	// The import bypasses the charging-service write hooks — bust the cached
	// dashboard aggregate so it recomputes from the imported rows.
	_ = s.cache.CacheDel(ctx, "analytics:dashboard:"+userID.String())
	return stats, nil
}

func validateBackup(b *domain.UserBackup) error {
	if b == nil {
		return &BackupValidationError{Reason: "empty payload"}
	}
	if b.SchemaVersion != domain.BackupSchemaVersion {
		return &BackupValidationError{Reason: fmt.Sprintf("unsupported schema_version %d (expected %d)", b.SchemaVersion, domain.BackupSchemaVersion)}
	}
	if len(b.Cars) > importMaxCars || len(b.Sessions) > importMaxSessions || len(b.Snapshots) > importMaxSnapshots {
		return &BackupValidationError{Reason: "backup exceeds size limits"}
	}

	carIDs := make(map[string]bool, len(b.Cars))
	for i, c := range b.Cars {
		if c.ID == "" || c.Name == "" {
			return &BackupValidationError{Reason: fmt.Sprintf("car %d: id and name are required", i)}
		}
		if carIDs[c.ID] {
			return &BackupValidationError{Reason: fmt.Sprintf("car %d: duplicate id %q", i, c.ID)}
		}
		carIDs[c.ID] = true
		if c.OdometerKM < 0 {
			return &BackupValidationError{Reason: fmt.Sprintf("car %d: negative odometer", i)}
		}
	}
	for i, sess := range b.Sessions {
		if !carIDs[sess.CarID] {
			return &BackupValidationError{Reason: fmt.Sprintf("session %d: car_id %q not in backup", i, sess.CarID)}
		}
		if sess.StartedAt.IsZero() {
			return &BackupValidationError{Reason: fmt.Sprintf("session %d: started_at required", i)}
		}
	}
	for i, sn := range b.Snapshots {
		if !carIDs[sn.CarID] {
			return &BackupValidationError{Reason: fmt.Sprintf("snapshot %d: car_id %q not in backup", i, sn.CarID)}
		}
		if sn.SohPct <= 0 || sn.SohPct > 100 {
			return &BackupValidationError{Reason: fmt.Sprintf("snapshot %d: soh_pct out of range", i)}
		}
	}
	if b.Settings != nil {
		switch b.Settings.Currency {
		case "toman", "rial", "usd":
		default:
			return &BackupValidationError{Reason: "settings: invalid currency"}
		}
		if b.Settings.PeakRate < 0 || b.Settings.MidRate < 0 || b.Settings.OffpeakRate < 0 {
			return &BackupValidationError{Reason: "settings: negative rate"}
		}
	}
	return nil
}
