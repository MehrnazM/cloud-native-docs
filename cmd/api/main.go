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
	"github.com/MehrnazM/cloud-native-docs/internal/service"
)

const (
	addr            = ":8080"
	shutdownTimeout = 20 * time.Second
)

func main() {
	ctx := context.Background()

	jsConn, err := messaging.NewConnection()
	if err != nil {
		slog.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}

	svc := service.NewDocumentService(jsConn)

	router := http.NewRouter(svc, jsConn.NC.IsConnected)
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
		slog.Error("server failed", "error", err)
		os.Exit(1)

	case sig := <-shutdown:
		slog.Info("shutdown initiated", "signal", sig)

		// Graceful HTTP shutdown
		shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()
		jsConn.NC.Close()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}

		slog.Info("shutdown complete")
	}

}
