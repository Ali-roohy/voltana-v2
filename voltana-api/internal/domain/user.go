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
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TokenClaims is the parsed payload of a validated access token,
// set on the Gin context by the Auth middleware.
type TokenClaims struct {
	UserID uuid.UUID
	JTI    string
}
