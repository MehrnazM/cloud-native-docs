package service

import (
	"context"
	"fmt"
	"time"

	"github.com/MehrnazM/cloud-native-docs/internal/model"
	"github.com/MehrnazM/cloud-native-docs/internal/repository"
	"github.com/MehrnazM/cloud-native-docs/shared/events"
	"github.com/google/uuid"
)

const (
	SubjectDocumentCreated = "documents.created"
)

type DocumentService struct {
	repo *repository.DocumentsRepository
	Publisher
}

func NewDocumentService(publisher Publisher, repo *repository.DocumentsRepository) *DocumentService {
	if publisher == nil {
		panic("publisher cannot be nil")
	}
	return &DocumentService{
		repo:      repo,
		Publisher: publisher,
	}
}

func (s *DocumentService) CreateDocument(ctx context.Context, name string) (string, error) {
	id := uuid.New()

	event := events.DocumentCreatedEvent{}
	event.ID = id
	event.Name = name

	if err := s.repo.CreateDocument(ctx, id, name); err != nil {
		return "", fmt.Errorf("failed to create document: %w", err)
	}

	if err := s.Publisher.Publish(ctx, SubjectDocumentCreated, event, 100*time.Millisecond); err != nil {
		return "", fmt.Errorf("publish failed: %w", err)
	}

	return id.String(), nil
}

func (s *DocumentService) GetDocumentByID(ctx context.Context, id uuid.UUID) (*model.Document, error) {

	doc, err := s.repo.GeDocumentByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return doc, nil
}
