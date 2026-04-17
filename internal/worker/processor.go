package worker

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/MehrnazM/cloud-native-docs/internal/model"
	"github.com/MehrnazM/cloud-native-docs/internal/repository"
	"github.com/MehrnazM/cloud-native-docs/shared/events"
	"github.com/google/uuid"
)

type Processor struct {
	repo *repository.DocumentsRepository
}

func NewProcessor(repo *repository.DocumentsRepository) *Processor {
	return &Processor{repo: repo}
}

func (p *Processor) MarkAsProcessing(ctx context.Context, id uuid.UUID) (locked bool, err error) {
	return p.repo.MarkAsProcessing(ctx, id)
}

func (p *Processor) ProcessWithContext(ctx context.Context, doc events.DocumentCreatedEvent) error {
	slog.Info("received document, start processing document", "id", doc.ID)

	time.Sleep(2 * time.Second)

	if rand.Intn(2) == 0 {
		rowsAffected, err := p.repo.UpdateDocumentStatus(ctx, doc.ID, model.StatusFailed)
		if err != nil {
			slog.Error("failed to update document status", "id", doc.ID, "error", err)
			return err
		}
		if rowsAffected == 0 {
			slog.Warn("Document not found when updating status to failed", "id", doc.ID)
			return nil
		}
		slog.Error("Document processing failed", "id", doc.ID)
	} else {
		rowsAffected, err := p.repo.UpdateDocumentStatus(ctx, doc.ID, model.StatusDone)
		if err != nil {
			slog.Error("failed to update document status", "id", doc.ID, "error", err)
			return err
		}
		if rowsAffected == 0 {
			slog.Warn("Document not found when updating status to done", "id", doc.ID)
			return nil
		}
		slog.Info("Document processed successfully", "id", doc.ID)
	}

	return nil
}
