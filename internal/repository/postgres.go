package repository

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/MehrnazM/cloud-native-docs/shared/util"
	_ "github.com/lib/pq"
)

func NewPostgresDB() (dbConn *sql.DB, err error) {
	db, err := util.MustGetString("POSTGRES_DB")
	if err != nil {
		return nil, err
	}
	user, err := util.MustGetString("POSTGRES_USER")
	if err != nil {
		return nil, err
	}
	password, err := util.MustGetString("POSTGRES_PASSWORD")
	if err != nil {
		return nil, err
	}
	host, err := util.MustGetString("POSTGRES_HOST")
	if err != nil {
		return nil, err
	}
	port, err := util.MustGetString("POSTGRES_PORT")
	if err != nil {
		return nil, err
	}

	connStr := "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + db + "?sslmode=disable"

	dbConn, err = sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	dbConn.SetMaxOpenConns(25)
	dbConn.SetMaxIdleConns(5)
	dbConn.SetConnMaxLifetime(5 * time.Minute)
	if err := dbConn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	slog.Info("Connected to POSTGRES")

	return dbConn, nil
}
