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

type ProcessResult int

const (
	ProcessUnknown ProcessResult = iota
	ProcessFailed
	ProcessSuccess
)

type Processor struct {
	repo *repository.DocumentsRepository
}

func NewProcessor(repo *repository.DocumentsRepository) *Processor {
	return &Processor{repo: repo}
}

func (p *Processor) ReachedMaxRetries(ctx context.Context, id uuid.UUID) (maxedOut bool, retryCount int, err error) {
	doc, err := p.repo.GeDocumentByID(ctx, id)
	if err != nil {
		return false, 0, err
	}
	if doc == nil {
		slog.Warn("Document not found when checking retry", "id", id)
		// If the document doesn't exist, we can consider it as maxed out to prevent further processing attempts
		return true, 0, nil
	}
	if doc.Status == string(model.StatusFailed) && doc.RetryCount >= doc.MaxRetries {
		slog.Info("Document has reached max retry limit.", "id", id)
		return true, doc.RetryCount, nil
	}
	return false, doc.RetryCount, nil
}

func (p *Processor) MarkAsProcessing(ctx context.Context, id uuid.UUID) (locked bool, err error) {
	return p.repo.MarkAsProcessing(ctx, id)
}

func (p *Processor) IncrementRetryCount(ctx context.Context, id uuid.UUID) error {
	_, err := p.repo.IncrementRetryCount(ctx, id)
	return err
}

func (p *Processor) Process(ctx context.Context, doc events.DocumentCreatedEvent) (ProcessResult, error) {
	slog.Info("received document, start processing document", "id", doc.ID)

	time.Sleep(2 * time.Second)

	if rand.Intn(2) == 0 {
		_, err := p.repo.UpdateDocumentStatus(ctx, doc.ID, model.StatusFailed)
		if err != nil {
			slog.Error("failed to update document status", "id", doc.ID, "error", err)
			return ProcessUnknown, err
		}
		return ProcessFailed, nil
	} else {
		_, err := p.repo.UpdateDocumentStatus(ctx, doc.ID, model.StatusDone)
		if err != nil {
			slog.Error("failed to update document status", "id", doc.ID, "error", err)
			return -1, err
		}
		return ProcessSuccess, nil
	}

}
