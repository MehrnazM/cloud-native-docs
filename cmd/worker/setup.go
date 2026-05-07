package main

import (
	"context"
	"os"
	"time"

	"github.com/MehrnazM/cloud-native-docs/internal/messaging"
	"github.com/MehrnazM/cloud-native-docs/shared/util"
	"github.com/nats-io/nats.go/jetstream"
)

// getDocumentConsumer ensures the stream and consumer are set up correctly and returns the consumer for processing messages.
func getDocumentConsumer(conn *messaging.Connection) jetstream.Consumer {
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer setupCancel()

	replicas, err := util.GetIntEnv("NATS_REPLICA", 1)
	if err != nil {
		logger.Error("Invalid NATS_REPLICA value", "error", err)
		os.Exit(1)
	}

	stream, err := conn.JS.Stream(setupCtx, StreamName)
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
				logger.Error("Failed to create stream", "error", err)
				os.Exit(1)
			}
			logger.Info("Created stream", "stream", StreamName)
		} else {
			logger.Error("Failed to get stream", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("Stream is ready", "stream", StreamName)

	consumer, err := stream.Consumer(setupCtx, ConsumerName)
	if err != nil {
		if err == jetstream.ErrConsumerNotFound {
			consumer, err = stream.CreateConsumer(setupCtx, jetstream.ConsumerConfig{
				Durable:       ConsumerName,
				AckPolicy:     jetstream.AckExplicitPolicy,
				FilterSubject: SubjectDocumentCreated,
				MaxAckPending: 1000,
			})
			if err != nil {
				logger.Error("Failed to create consumer", "error", err)
				os.Exit(1)
			}
			logger.Info("Created consumer", "consumer", ConsumerName)
		} else {
			logger.Error("Failed to get consumer", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("Consumer is ready", "consumer", ConsumerName)

	return consumer
}

func setupDLQStream(conn *messaging.Connection) jetstream.Stream {
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer setupCancel()

	replicas, err := util.GetIntEnv("NATS_REPLICA", 1)
	if err != nil {
		logger.Error("Invalid NATS_REPLICA value", "error", err)
		os.Exit(1)
	}

	stream, err := conn.JS.Stream(setupCtx, DLQStreamName)
	if err != nil {
		if err == jetstream.ErrStreamNotFound {
			stream, err = conn.JS.CreateStream(setupCtx, jetstream.StreamConfig{
				Name:      DLQStreamName,
				Subjects:  []string{SubjectDLQ},
				Replicas:  replicas,
				Retention: jetstream.LimitsPolicy,
				MaxAge:    30 * 24 * time.Hour,
				MaxMsgs:   100000,
				Discard:   jetstream.DiscardOld,
				Storage:   jetstream.FileStorage,
			})
			if err != nil {
				logger.Error("Failed to create dlq stream", "error", err)
				os.Exit(1)
			}
			logger.Info("Created stream", "stream", DLQStreamName)
		} else {
			logger.Error("Failed to get stream", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("Stream is ready", "stream", DLQStreamName)

	return stream
}
