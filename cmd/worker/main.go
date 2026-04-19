package main

import (
	"context"
	"encoding/json"
	"log/slog"
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
)

const (
	SubjectDocumentCreated = "documents.created"
	StreamName             = "DOCUMENT_EVENTS"
	ConsumerName           = "WORKER_CONSUMER"
)

func main() {

	conn, err := messaging.NewConnection()
	if err != nil {
		slog.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer conn.NC.Close()

	slog.Info("Connected to NATS", "status", conn.NC.Status())

	db, err := repository.NewPostgresDB()
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := repository.NewDocumentsRepository(db)
	w := worker.NewProcessor(repo)

	var stream jetstream.Stream
	replicas, err := util.GetIntEnv("NATS_REPLICA", 1)
	if err != nil {
		slog.Error("Invalid NATS_REPLICA value", "error", err)
		os.Exit(1)
	}
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer setupCancel()

	consumeCtx, consumeCancel := context.WithCancel(context.Background())
	defer consumeCancel()

	stream, err = conn.JS.Stream(setupCtx, StreamName)
	if err != nil {
		if err == jetstream.ErrStreamNotFound {
			stream, err = conn.JS.CreateStream(setupCtx, jetstream.StreamConfig{
				Name:      StreamName,
				Subjects:  []string{SubjectDocumentCreated},
				Replicas:  replicas,
				Retention: jetstream.WorkQueuePolicy,
				MaxAge:    7 * 24 * time.Hour,
				MaxMsgs:   100000,
				Discard:   jetstream.DiscardOld,
				Storage:   jetstream.FileStorage,
			})
			if err != nil {
				slog.Error("Failed to create stream", "error", err)
				os.Exit(1)
			}
			slog.Info("Created stream", "stream", StreamName)
		} else {
			slog.Error("Failed to get stream", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("Stream is ready", "stream", StreamName)

	var consumer jetstream.Consumer
	consumer, err = stream.Consumer(setupCtx, ConsumerName)
	if err != nil {
		if err == jetstream.ErrConsumerNotFound {
			consumer, err = stream.CreateConsumer(setupCtx, jetstream.ConsumerConfig{
				Durable:       ConsumerName,
				AckPolicy:     jetstream.AckExplicitPolicy,
				FilterSubject: SubjectDocumentCreated,
				MaxAckPending: 1000,
			})
			if err != nil {
				slog.Error("Failed to create consumer", "error", err)
				os.Exit(1)
			}
			slog.Info("Created consumer", "consumer", ConsumerName)
		} else {
			slog.Error("Failed to get consumer", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("Consumer is ready", "consumer", ConsumerName)

	consumeHandle, err := consumer.Consume(func(msg jetstream.Msg) {
		var event events.DocumentCreatedEvent
		err := json.Unmarshal(msg.Data(), &event)
		if err != nil {
			slog.Error("Failed to unmarshal message", "error", err)
			msg.Nak()
			return
		}

		skip, retryCount, err := w.ReachedMaxRetries(consumeCtx, event.ID)
		if err != nil {
			slog.Error("Failed to check retry count", "id", event.ID, "error", err)
			msg.Nak()
			return
		}
		if skip {
			msg.Ack()
			return
		}

		locked, err := w.MarkAsProcessing(consumeCtx, event.ID)
		if err != nil {
			slog.Error("Failed to mark document as processing", "id", event.ID, "error", err)
			msg.Nak()
			return
		}

		if !locked {
			slog.Info("Document is already being processed by another worker", "id", event.ID)
			msg.Ack()
			return
		}
		processResult, err := w.Process(consumeCtx, event)
		if err != nil {
			msg.Nak()
			slog.Error("Failed to process message", "error", err)
			return
		}
		if processResult == worker.ProcessFailed {
			slog.Error("Document processing failed, will retry if max retries not reached", "id", event.ID)

			err = w.IncrementRetryCount(consumeCtx, event.ID)
			if err != nil {
				slog.Error("Failed to increment retry count", "id", event.ID, "error", err)
			}
			msg.NakWithDelay(time.Duration(retryCount+2) * time.Second)
			return
		} else {
			slog.Info("Document processed successfully", "id", event.ID)
			msg.Ack()
		}

	})
	if err != nil {
		slog.Error("Failed to start consuming", "error", err)
		os.Exit(1)
	}
	defer consumeHandle.Stop()

	slog.Info("Worker started successfully", "consumer", ConsumerName)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-shutdown
	slog.Info("Shutdown signal received", "signal", sig)

	consumeHandle.Drain()
	consumeCancel()
	slog.Info("Worker shutdown complete")
}
