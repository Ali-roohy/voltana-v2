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

var (
	ErrNotFound   = errors.New("not found")
	ErrEmailTaken = errors.New("email already registered")
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	FindByPhone(ctx context.Context, phone string) (*domain.User, error)
	// UpdateBotLink writes the E.164 phone plus whichever chat_id is non-nil,
	// leaving the other chat_id column untouched (COALESCE semantics).
	UpdateBotLink(ctx context.Context, userID uuid.UUID, phone string, baleChatID, telegramChatID *string) error
}

type pgxUserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &pgxUserRepository{db: db}
}

const userCols = `id, email, password_hash, is_email_verified, is_admin, phone, bale_chat_id, telegram_chat_id, created_at, updated_at`

func (r *pgxUserRepository) Create(ctx context.Context, email, passwordHash string) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING `+userCols,
		email, passwordHash,
	)
	return scanUser(row)
}

func (r *pgxUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE email = $1`, email,
	)
	return scanUser(row)
}

func (r *pgxUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE id = $1`, id,
	)
	return scanUser(row)
}

func (r *pgxUserRepository) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE phone = $1`, phone,
	)
	return scanUser(row)
}

func (r *pgxUserRepository) UpdateBotLink(ctx context.Context, userID uuid.UUID, phone string, baleChatID, telegramChatID *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET
			phone            = $2,
			bale_chat_id     = COALESCE($3, bale_chat_id),
			telegram_chat_id = COALESCE($4, telegram_chat_id),
			updated_at       = now()
		WHERE id = $1`,
		userID, phone, baleChatID, telegramChatID,
	)
	return err
}

func scanUser(row pgx.Row) (*domain.User, error) {
	u := &domain.User{}
	var pgID pgtype.UUID
	err := row.Scan(
		&pgID, &u.Email, &u.PasswordHash, &u.IsEmailVerified, &u.IsAdmin,
		&u.Phone, &u.BaleChatID, &u.TelegramChatID,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	u.ID = uuid.UUID(pgID.Bytes)
	return u, nil
}
