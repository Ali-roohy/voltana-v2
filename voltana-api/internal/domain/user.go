package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID
	Email           string
	PasswordHash    string
	IsEmailVerified bool
	IsAdmin         bool
	FullName        *string // optional display name, nil until user sets it
	Phone           *string // E.164, nil until bot is linked
	BaleChatID      *string // nil until linked via Bale
	TelegramChatID  *string // nil until linked via Telegram
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TokenClaims is the parsed payload of a validated access token,
// set on the Gin context by the Auth middleware.
type TokenClaims struct {
	UserID uuid.UUID
	JTI    string
}
