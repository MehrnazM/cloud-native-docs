package events

import "github.com/google/uuid"

type Metadata struct {
	CorrelationID string `json:"correlation_id"`
}
type DocumentCreatedEvent struct {
	Metadata
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
