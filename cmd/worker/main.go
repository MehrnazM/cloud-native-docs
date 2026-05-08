package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MehrnazM/cloud-native-docs/internal/messaging"
	"github.com/MehrnazM/cloud-native-docs/internal/repository"
	"github.com/MehrnazM/cloud-native-docs/internal/worker"
	"github.com/MehrnazM/cloud-native-docs/shared/events"
	"github.com/MehrnazM/cloud-native-docs/shared/util"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

const (
	SubjectDocumentCreated = "documents.created"
	StreamName             = "DOCUMENT_EVENTS"
	ConsumerName           = "WORKER_CONSUMER"
	tracerName             = "docs-worker"
	SubjectDLQ             = "documents.dlq"
	DLQConsumerName        = "WORKER_DLQ_CONSUMER"
	DLQStreamName          = "DOCUMENT_DLQ"
)

var (
	processedCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "documents_processed_total",
			Help: "Total processed documents",
		},
	)

	failedCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "documents_failed_total",
			Help: "Total failed documents",
		},
	)
	inProgressCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "documents_in_processing",
			Help: "Current documents being processed",
		},
	)
	processingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "document_processing_duration_seconds",
			Help:    "Time spent processing a document",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)
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
	logger = logger.With("service", "worker", "operation", "document_processing")
	slog.SetDefault(logger)

	prometheus.MustRegister(processedCounter, failedCounter, inProgressCounter, processingDuration)
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

	tracerShutdown := initTracer()
	defer tracerShutdown()

	conn, err := messaging.NewConnection(logger, tracerName)
	if err != nil {
		logger.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer conn.NC.Close()

	logger.Info("Connected to NATS", "status", conn.NC.Status())

	db, err := repository.NewPostgresDB()
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := repository.NewDocumentsRepository(db, tracerName)
	w := worker.NewProcessor(repo, logger, tracerName)

	setupDLQStream(conn)

	consumer := getDocumentConsumer(conn)
	consumeHandle, err := consumer.Consume(func(msg jetstream.Msg) {
		process(msg, w, conn)
	})
	if err != nil {
		logger.Error("Failed to start consuming", "error", err)
		os.Exit(1)
	}
	defer consumeHandle.Stop()

	logger.Info("Worker started successfully", "consumer", ConsumerName)

	// Start HTTP server for Prometheus metrics
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9090", nil)
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-shutdown
	logger.Info("Shutdown signal received", "signal", sig)

	consumeHandle.Drain()
	logger.Info("Worker shutdown complete")
}

// process handles a single document event message: it unmarshals the event,
// checks retry limits, marks the document as processing, invokes the worker logic, manages metrics,
// and acknowledges or negatively acknowledges the message based on processing outcome.
func process(msg jetstream.Msg, w *worker.Processor, conn *messaging.Connection) {
	var event events.DocumentCreatedEvent
	err := json.Unmarshal(msg.Data(), &event)
	if err != nil {
		logger.Error("Failed to unmarshal message", "error", err)
		failedCounter.Inc()
		msg.Nak()
		return
	}

	consumerLogger := logger.With("correlation_id", event.CorrelationID, "id", event.ID)
	carrier := propagation.MapCarrier(event.Metadata.TraceContext)
	consumeCtx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)

	// check if document has reached max document retry number
	skip, retryCount, err := w.ReachedMaxRetries(consumeCtx, event.ID)
	if err != nil {
		consumerLogger.Error("Failed to check retry count", "id", event.ID, "error", err)
		failedCounter.Inc()
		msg.Nak()
		return
	}

	// send a message to dead document queue to log the failure
	if skip {
		failedRecord := events.DLQEvent{
			OriginalEvent: event,
			FailureReason: "reached max retry",
			RetryCount:    retryCount,
			FailedAt:      time.Now().UTC(),
		}
		if err := conn.Publish(consumeCtx, SubjectDLQ, failedRecord, 0); err != nil {
			consumerLogger.Error("Failed to publish to DLQ", "id", event.ID, "error", err)
		}
		consumerLogger.Warn("Document moved to DLQ", "id", event.ID, "retryCount", retryCount)
		failedCounter.Inc()
		msg.Ack()
		return
	}

	// check if another worker has picked the document otherwise marked it as locked
	locked, err := w.MarkAsProcessing(consumeCtx, event.ID)
	if err != nil {
		consumerLogger.Error("Failed to mark document as processing", "id", event.ID, "error", err)
		failedCounter.Inc()
		msg.Nak()
		return
	}

	if !locked {
		consumerLogger.Info("Document is already being processed by another worker", "id", event.ID)
		msg.Ack()
		return
	}

	inProgressCounter.Inc()
	defer inProgressCounter.Dec()

	start := time.Now()
	processResult, err := w.Process(consumeCtx, event)
	duration := time.Since(start)
	if err != nil {
		msg.Nak()
		consumerLogger.Error("Failed to process message", "error", err)
		failedCounter.Inc()
		return
	}
	if processResult == worker.ProcessFailed {
		processingDuration.WithLabelValues("failed").Observe(duration.Seconds())
		consumerLogger.Error("Document processing failed, will retry if max retries not reached", "id", event.ID)

		err = w.IncrementRetryCount(consumeCtx, event.ID)
		if err != nil {
			consumerLogger.Error("Failed to increment retry count", "id", event.ID, "error", err)
		}
		msg.NakWithDelay(time.Duration(retryCount+2) * time.Second)
		failedCounter.Inc()
		return
	} else {
		processingDuration.WithLabelValues("done").Observe(duration.Seconds())
		consumerLogger.Info("Document processed successfully", "id", event.ID)
		processedCounter.Inc()
		msg.Ack()
	}
}
