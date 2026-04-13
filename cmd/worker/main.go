package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/MehrnazM/cloud-native-docs/internal/messaging"
	"github.com/MehrnazM/cloud-native-docs/internal/worker"
	"github.com/MehrnazM/cloud-native-docs/shared/events"
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

	worker := worker.NewProcessor()

	var stream jetstream.Stream
	replica := os.Getenv("NATS_STREAM_REPLICAS")
	if replica == "" {
		replica = "1"
	}
	replicas, err := strconv.Atoi(replica)
	if err != nil {
		slog.Error("Invalid NATS_STREAM_REPLICAS value", "error", err)
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

		// Pass context for cancellation support
		err = worker.ProcessWithContext(consumeCtx, event)
		if err != nil {
			slog.Error("Failed to process message", "error", err)
			msg.Nak()
			return
		}

		msg.Ack()
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

	consumeCancel()
	slog.Info("Worker shutdown complete")
}
