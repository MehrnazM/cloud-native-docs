package events

import "github.com/google/uuid"

type Metadata struct {
	CorrelationID string `json:"correlationId"`
	TraceID       string `json:"traceId"`
}
type DocumentCreatedEvent struct {
	Metadata
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
