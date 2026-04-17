package events

import "github.com/google/uuid"

type DocumentCreatedEvent struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
