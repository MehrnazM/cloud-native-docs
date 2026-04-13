package events

import "time"

type EventMetadata struct {
	EventType     string    `json:"eventType"`
	Timestamp     time.Time `json:"timestamp"`
	CorrelationID string    `json:"correlationID"`
}

type Event struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DocumentCreatedEvent struct {
	Metadata EventMetadata `json:"metadata"`
	Data     Event         `json:"data"`
}
