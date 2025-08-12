package models

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeCreate EventType = "create"
	EventTypeUpdate EventType = "update"
	EventTypeDelete EventType = "delete"
)

type SyncEvent struct {
	ID       int64     `json:"id" db:"id"`
	UserID   uuid.UUID `json:"user_id" db:"user_id"`
	DataID   uuid.UUID `json:"data_id" db:"data_id"`
	Action   EventType `json:"action" db:"event_type"` // create, update, delete
	Version  int64     `json:"version" db:"data_version"`
	ClientID string    `json:"client_id" db:"client_id"`
	Created  time.Time `json:"created" db:"timestamp"`
}
