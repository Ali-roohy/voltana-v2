package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VerificationTokenRepository manages email-verification tokens. Only the
// SHA-256 hash of a token is ever stored (raw token lives in the emailed link).
type VerificationTokenRepository interface {
	// ReplaceVerificationToken invalidates any existing tokens for the user and
	// stores a fresh hash + expiry (single outstanding token per user).
	ReplaceVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	// ConsumeVerificationToken looks up an unexpired token by hash and, in ONE
	// transaction, flips users.is_email_verified and deletes the user's tokens.
	// Returns the owning user and whether the account was already verified.
	// ErrNotFound when the hash is unknown/expired (caller maps to 400).
	ConsumeVerificationToken(ctx context.Context, tokenHash string) (userID uuid.UUID, alreadyVerified bool, err error)
}

type pgxVerificationRepository struct {
	db *pgxpool.Pool
}

func NewVerificationTokenRepository(db *pgxpool.Pool) VerificationTokenRepository {
	return &pgxVerificationRepository{db: db}
}

func (r *pgxVerificationRepository) ReplaceVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM email_verification_tokens WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO email_verification_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *pgxVerificationRepository) ConsumeVerificationToken(ctx context.Context, tokenHash string) (uuid.UUID, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, false, err
	}
	defer tx.Rollback(ctx)

	// Single indexed lookup by hash (no "user exists" branch). Lock the row so a
	// concurrent verify of the same token can't double-process.
	var pgUID pgtype.UUID
	var alreadyVerified bool
	err = tx.QueryRow(ctx, `
		SELECT evt.user_id, u.is_email_verified
		FROM email_verification_tokens evt
		JOIN users u ON u.id = evt.user_id
		WHERE evt.token_hash = $1 AND evt.expires_at > now()
		FOR UPDATE OF evt`, tokenHash).Scan(&pgUID, &alreadyVerified)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, false, ErrNotFound
		}
		return uuid.Nil, false, err
	}
	userID := uuid.UUID(pgUID.Bytes)

	// Single-use: drop all of the user's tokens in the same transaction.
	if _, err := tx.Exec(ctx, `DELETE FROM email_verification_tokens WHERE user_id = $1`, userID); err != nil {
		return uuid.Nil, false, err
	}
	if !alreadyVerified {
		if _, err := tx.Exec(ctx,
			`UPDATE users SET is_email_verified = true, updated_at = now() WHERE id = $1`, userID,
		); err != nil {
			return uuid.Nil, false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, false, err
	}
	return userID, alreadyVerified, nil
}
