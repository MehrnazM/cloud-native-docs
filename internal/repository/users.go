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

type UsersRepository struct {
	db     *sql.DB
	tracer trace.Tracer
}

func NewUsersRepository(db *sql.DB, tracername string) *UsersRepository {
	return &UsersRepository{
		db:     db,
		tracer: otel.Tracer(tracername),
	}
}

func (r *UsersRepository) RegisterUser(ctx context.Context, id uuid.UUID, email, passwordHash string) (err error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.RegisterUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", id.String()),
		attribute.String("user.email", email),
	)

	query := `INSERT INTO documents.users(id, email, password_hash)
			  VALUES($1, $2, $3)`

	_, err = r.db.ExecContext(ctx, query, id, email, passwordHash)
	if err != nil {
		return err
	}
	return nil
}

func (r *UsersRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.GetUser")
	defer span.End()

	query := `SELECT id, email, password_hash, created_at
			  FROM documents.users
			  WHERE email = $1`
	var user model.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil

}
