package service

import (
	"context"
	"fmt"
	"time"

	"github.com/MehrnazM/cloud-native-docs/shared/events"
	"github.com/google/uuid"
)

const (
	SubjectDocumentCreated = "documents.created"
)

type DocumentService struct {
	Publisher
}

func NewDocumentService(publisher Publisher) *DocumentService {
	if publisher == nil {
		panic("publisher cannot be nil")
	}
	return &DocumentService{
		Publisher: publisher,
	}
}

func (s *DocumentService) CreateDocument(ctx context.Context, name, requestID string) (string, error) {
	id := uuid.New().String()

	event := events.DocumentCreatedEvent{}
	event.Data.ID = id
	event.Data.Name = name
	event.Metadata.EventType = "DocumentCreated"
	event.Metadata.Timestamp = time.Now().UTC()
	event.Metadata.CorrelationID = requestID

	if err := s.Publisher.Publish(ctx, SubjectDocumentCreated, event, 100*time.Millisecond); err != nil {
		return "", fmt.Errorf("publish failed: %w", err)
	}

	return id, nil
}
