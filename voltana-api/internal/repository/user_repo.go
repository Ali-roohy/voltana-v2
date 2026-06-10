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
	ErrPhoneTaken = errors.New("phone already registered")
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*domain.User, error)
	// CreateWithPhone creates a passwordless user identified by phone + bot chat_id.
	// email is optional (nil → NULL). Returns ErrPhoneTaken or ErrEmailTaken on conflict.
	CreateWithPhone(ctx context.Context, phone string, email *string, baleChatID, telegramChatID *string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	FindByPhone(ctx context.Context, phone string) (*domain.User, error)
	// UpdateBotLink writes the E.164 phone plus whichever chat_id is non-nil,
	// leaving the other chat_id column untouched (COALESCE semantics).
	UpdateBotLink(ctx context.Context, userID uuid.UUID, phone string, baleChatID, telegramChatID *string) error

	// Admin user management
	ListAll(ctx context.Context, limit, offset int) ([]*domain.User, int, error)
	AdminUpdate(ctx context.Context, id uuid.UUID, isAdmin *bool, isEmailVerified *bool) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountAdmins(ctx context.Context) (int, error)
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
		`INSERT INTO users (email, password_hash, is_admin)
		 VALUES ($1, $2, NOT EXISTS (SELECT 1 FROM users))
		 RETURNING `+userCols,
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

func (r *pgxUserRepository) CreateWithPhone(ctx context.Context, phone string, email *string, baleChatID, telegramChatID *string) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO users (phone, email, password_hash, bale_chat_id, telegram_chat_id, is_admin)
		 VALUES ($1, $2, '', $3, $4, NOT EXISTS (SELECT 1 FROM users))
		 RETURNING `+userCols,
		phone, email, baleChatID, telegramChatID,
	)
	u := &domain.User{}
	var pgID pgtype.UUID
	var pgEmail pgtype.Text
	err := row.Scan(
		&pgID, &pgEmail, &u.PasswordHash, &u.IsEmailVerified, &u.IsAdmin,
		&u.Phone, &u.BaleChatID, &u.TelegramChatID,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "uq_users_phone" {
				return nil, ErrPhoneTaken
			}
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	u.ID = uuid.UUID(pgID.Bytes)
	if pgEmail.Valid {
		u.Email = pgEmail.String
	}
	return u, nil
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

func (r *pgxUserRepository) ListAll(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+userCols+` FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		var pgID pgtype.UUID
		var pgEmail pgtype.Text
		if err := rows.Scan(
			&pgID, &pgEmail, &u.PasswordHash, &u.IsEmailVerified, &u.IsAdmin,
			&u.Phone, &u.BaleChatID, &u.TelegramChatID,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		u.ID = uuid.UUID(pgID.Bytes)
		if pgEmail.Valid {
			u.Email = pgEmail.String
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *pgxUserRepository) AdminUpdate(ctx context.Context, id uuid.UUID, isAdmin *bool, isEmailVerified *bool) (*domain.User, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE users SET
			is_admin          = COALESCE($2, is_admin),
			is_email_verified = COALESCE($3, is_email_verified),
			updated_at        = now()
		 WHERE id = $1
		 RETURNING `+userCols,
		id, isAdmin, isEmailVerified,
	)
	return scanUser(row)
}

func (r *pgxUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgxUserRepository) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE is_admin = true`).Scan(&n)
	return n, err
}

func scanUser(row pgx.Row) (*domain.User, error) {
	u := &domain.User{}
	var pgID pgtype.UUID
	var pgEmail pgtype.Text // nullable after migration 000012
	err := row.Scan(
		&pgID, &pgEmail, &u.PasswordHash, &u.IsEmailVerified, &u.IsAdmin,
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
	if pgEmail.Valid {
		u.Email = pgEmail.String
	}
	return u, nil
}
