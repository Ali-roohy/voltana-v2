package repository

import (
	"context"

	"voltana-api/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PushSubscriptionRepository persists web-push subscriptions (TASK-0039).
// One row per browser/device endpoint; rows cascade with the user.
type PushSubscriptionRepository interface {
	// Create upserts on endpoint (re-subscribing the same browser refreshes keys
	// and ownership rather than erroring).
	Create(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string) error
	DeleteByEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.PushSubscription, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
}

type pgxPushRepository struct {
	db *pgxpool.Pool
}

func NewPushRepository(db *pgxpool.Pool) PushSubscriptionRepository {
	return &pgxPushRepository{db: db}
}

func (r *pgxPushRepository) Create(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (endpoint) DO UPDATE SET
		   user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth`,
		userID, endpoint, p256dh, auth,
	)
	return err
}

func (r *pgxPushRepository) DeleteByEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`, userID, endpoint)
	return err
}

func (r *pgxPushRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.PushSubscription, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []domain.PushSubscription
	for rows.Next() {
		var s domain.PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (r *pgxPushRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM push_subscriptions WHERE id = $1`, id)
	return err
}
