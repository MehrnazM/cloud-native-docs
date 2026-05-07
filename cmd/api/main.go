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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	propagation "go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

const (
	addr            = ":8080"
	tracerName      = "docs-api"
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

func initTracer() func() {
	ctx := context.Background()
	jaeger := util.GetStringEnv("JAEGER_COLLECTOR", "localhost:4318")
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(jaeger),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		logger.Error("failed to create OTLP trace exporter", "error", err)
		os.Exit(1)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(tracerName),
		)))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown tracer provider", "error", err)
		}
	}
}

func main() {
	ctx := context.Background()

	tracerShutdown := initTracer()
	defer tracerShutdown()

	jsConn, err := messaging.NewConnection(logger, tracerName)
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

	docRepo := repository.NewDocumentsRepository(db, tracerName)
	docSvc := service.NewDocumentService(jsConn, docRepo, tracerName)

	jwtSecret, err := util.MustGetString("JWT_SECRET")
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	userRepo := repository.NewUsersRepository(db, tracerName)
	authSvc := service.NewAuthService(userRepo, jwtSecret)

	router := http.NewRouter(docSvc, authSvc, jsConn.NC.IsConnected, logger, tracerName, jwtSecret, jsConn)
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
