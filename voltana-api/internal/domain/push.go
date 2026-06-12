package domain

import "github.com/google/uuid"

// PushSubscription is one browser/device web-push registration (TASK-0039).
type PushSubscription struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Endpoint string
	P256dh   string
	Auth     string
}
