package repository

import (
	"context"
	"database/sql"

	"github.com/MehrnazM/cloud-native-docs/internal/model"
	"github.com/google/uuid"
)

type DocumentsRepository struct {
	db *sql.DB
}

func NewDocumentsRepository(db *sql.DB) *DocumentsRepository {
	return &DocumentsRepository{db: db}
}

func (r *DocumentsRepository) CreateDocument(ctx context.Context, id uuid.UUID, name string) error {
	query := `INSERT INTO documents.documents(id, name, status) VALUES ($1, $2, $3)`

	_, err := r.db.ExecContext(ctx, query, id, name, model.StatusPending)
	return err

}

func (r *DocumentsRepository) UpdateDocumentStatus(ctx context.Context, id uuid.UUID, status model.Status) (int64, error) {
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

func (r *DocumentsRepository) GeDocumentByID(ctx context.Context, id uuid.UUID) (*model.Document, error) {
	query := `SELECT id, name, status, created_at, updated_at FROM documents.documents WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	var doc model.Document
	err := row.Scan(&doc.ID, &doc.Name, &doc.Status, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &doc, nil
}

func (r *DocumentsRepository) MarkAsProcessing(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `UPDATE documents.documents
	          SET status = $1, updated_at = NOW()
			  WHERE id = $2 AND status = $3`

	res, err := r.db.ExecContext(ctx, query, model.StatusProcessing, id, model.StatusPending)
	if err != nil {
		return false, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected == 1, nil
}
