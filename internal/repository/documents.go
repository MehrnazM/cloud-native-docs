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
