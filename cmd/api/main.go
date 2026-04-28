package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MehrnazM/cloud-native-docs/internal/http"
	"github.com/MehrnazM/cloud-native-docs/internal/messaging"
	"github.com/MehrnazM/cloud-native-docs/internal/repository"
	"github.com/MehrnazM/cloud-native-docs/internal/service"
	"github.com/MehrnazM/cloud-native-docs/shared/util"
)

const (
	addr            = ":8080"
	shutdownTimeout = 20 * time.Second
)

var logger *slog.Logger

func init() {
	var level slog.Leveler
	levelInt, err := util.GetIntEnv("SLOG_LEVEL", int(slog.LevelDebug))
	if err != nil {
		level = slog.LevelDebug
	} else {
		level = slog.Level(levelInt)
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger = slog.New(handler)
	logger = logger.With("service", "api")
	slog.SetDefault(logger)
}

func main() {
	ctx := context.Background()

	jsConn, err := messaging.NewConnection(logger)
	if err != nil {
		logger.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer jsConn.NC.Close()

	db, err := repository.NewPostgresDB()
	if err != nil {
		logger.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := repository.NewDocumentsRepository(db)
	svc := service.NewDocumentService(jsConn, repo)

	router := http.NewRouter(svc, jsConn.NC.IsConnected, logger)
	server := http.NewServer(addr, router)

	// Start HTTP server
	serverErrors := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			serverErrors <- err
		}
	}()

	// Wait for shutdown signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case err := <-serverErrors:
		logger.Error("server failed", "error", err)
		os.Exit(1)

	case sig := <-shutdown:
		logger.Info("shutdown initiated", "signal", sig)

		// Graceful HTTP shutdown
		shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}

		logger.Info("shutdown complete")
	}

}
