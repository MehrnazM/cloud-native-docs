package repository

import (
	"context"
	"database/sql"

	"github.com/MehrnazM/cloud-native-docs/internal/model"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type DocumentsRepository struct {
	db     *sql.DB
	tracer trace.Tracer
}

func NewDocumentsRepository(db *sql.DB, tracername string) *DocumentsRepository {
	return &DocumentsRepository{
		db:     db,
		tracer: otel.Tracer(tracername),
	}
}

func (r *DocumentsRepository) CreateDocument(ctx context.Context, id uuid.UUID, name string) error {
	ctx, span := r.tracer.Start(ctx, "DocumentsRepository.CreateDocument")
	defer span.End()

	span.SetAttributes(
		attribute.String("document.id", id.String()),
		attribute.String("document.name", name),
	)

	query := `INSERT INTO documents.documents(id, name, status) VALUES ($1, $2, $3)`

	_, err := r.db.ExecContext(ctx, query, id, name, model.StatusPending)
	return err

}

func (r *DocumentsRepository) UpdateDocumentStatus(ctx context.Context, id uuid.UUID, status model.Status) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "DocumentsRepository.UpdateDocumentStatus")
	defer span.End()

	span.SetAttributes(
		attribute.String("document.id", id.String()),
		attribute.String("document.status", string(status)),
	)

	query := `UPDATE documents.documents SET status = $1, updated_at = NOW() WHERE id = $2`

	res, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rowsAffected == 0 {
		return 0, nil
	}
	return rowsAffected, nil
}

func (r *DocumentsRepository) GetDocumentByID(ctx context.Context, id uuid.UUID) (*model.Document, error) {
	ctx, span := r.tracer.Start(ctx, "DocumentsRepository.GetDocumentByID")
	defer span.End()

	span.SetAttributes(attribute.String("document.id", id.String()))

	query := `SELECT 
					id, 
					name, 
					status, 
					retry_count, 
					max_retries, 
					created_at, 
					updated_at 
				FROM documents.documents WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	var doc model.Document
	err := row.Scan(&doc.ID, &doc.Name, &doc.Status, &doc.RetryCount, &doc.MaxRetries, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &doc, nil
}

func (r *DocumentsRepository) MarkAsProcessing(ctx context.Context, id uuid.UUID) (bool, error) {
	ctx, span := r.tracer.Start(ctx, "DocumentsRepository.MarkAsProcessing")
	defer span.End()

	span.SetAttributes(attribute.String("document.id", id.String()))

	query := `UPDATE documents.documents
	          SET status = $1, 
			  	  updated_at = NOW(),
				  locked_at = NOW()
			  WHERE id = $2 AND (
			  status = $3
			  OR (status = $4 AND retry_count < max_retries))
			  OR (status = $5 AND locked_at < NOW() - INTERVAL '5 minute')`

	res, err := r.db.ExecContext(ctx, query,
		model.StatusProcessing,
		id,
		model.StatusPending,
		model.StatusFailed,
		model.StatusProcessing)
	if err != nil {
		return false, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected == 1, nil
}

func (r *DocumentsRepository) IncrementRetryCount(ctx context.Context, id uuid.UUID) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "DocumentsRepository.IncrementRetryCount")
	defer span.End()

	span.SetAttributes(attribute.String("document.id", id.String()))

	query := `UPDATE documents.documents
	          SET retry_count = retry_count + 1, updated_at = NOW()
			  WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

func (r *DocumentsRepository) DeleteDocument(ctx context.Context, id uuid.UUID) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "DocumentsRepository.DeleteDocument")
	defer span.End()

	span.SetAttributes(attribute.String("document.id", id.String()))

	query := `DELETE FROM documents.documents WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}
