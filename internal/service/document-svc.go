package service

import (
	"context"
	"fmt"
	"time"

	"github.com/MehrnazM/cloud-native-docs/internal/model"
	"github.com/MehrnazM/cloud-native-docs/internal/repository"
	"github.com/MehrnazM/cloud-native-docs/shared/events"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	SubjectDocumentCreated = "documents.created"
)

type DocumentService struct {
	repo   *repository.DocumentsRepository
	tracer trace.Tracer
	Publisher
}

func NewDocumentService(publisher Publisher, repo *repository.DocumentsRepository, tracerName string) *DocumentService {
	if publisher == nil {
		panic("publisher cannot be nil")
	}
	return &DocumentService{
		repo:      repo,
		tracer:    otel.Tracer(tracerName),
		Publisher: publisher,
	}
}

func (s *DocumentService) CreateDocument(ctx context.Context, name string, correlationID string) (string, error) {
	ctx, span := s.tracer.Start(ctx, "DocumentService.CreateDocument")
	defer span.End()

	span.SetAttributes(
		attribute.String("document.name", name),
		attribute.String("correlation.id", correlationID),
	)
	id := uuid.New()

	event := events.DocumentCreatedEvent{}
	event.ID = id
	event.Name = name
	event.Metadata.CorrelationID = correlationID
	event.Metadata.TraceID = span.SpanContext().TraceID().String()

	if err := s.repo.CreateDocument(ctx, id, name); err != nil {
		return "", fmt.Errorf("failed to create document: %w", err)
	}

	if err := s.Publisher.Publish(ctx, SubjectDocumentCreated, event, 100*time.Millisecond); err != nil {
		s.repo.DeleteDocument(ctx, id)
		return "", fmt.Errorf("publish failed: %w", err)
	}

	return id.String(), nil
}

func (s *DocumentService) GetDocumentByID(ctx context.Context, id uuid.UUID) (*model.Document, error) {

	doc, err := s.repo.GetDocumentByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return doc, nil
}
