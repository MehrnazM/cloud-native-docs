package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Connection struct {
	JS jetstream.JetStream
	NC *nats.Conn
}

func NewConnection() (conn *Connection, err error) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		return nil, nats.ErrNoServers
	}
	var nc *nats.Conn
	retry := os.Getenv("NATS_CONN_RETRY")
	if retry == "" {
		retry = "10"
	}
	// Connect to NATS
	retryCount, err := strconv.Atoi(retry)
	if err != nil {
		return nil, fmt.Errorf("invalid NATS_CONN_RETRY value: %w", err)
	}
	for i := 0; i < retryCount; i++ {
		nc, err = nats.Connect(url)
		if err == nil {
			break
		}
		slog.Warn("NATS connection failed, retrying", "attempt", i+1, "error", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, err
	}

	// Connect to JetStream
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}
	return &Connection{JS: js, NC: nc}, nil
}

func (c *Connection) Publish(ctx context.Context, subject string, message any, retryWait time.Duration) (err error) {
	if retryWait == 0 {
		retryWait = 100 * time.Millisecond
	}

	var data []byte
	switch v := message.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(message)
		if err != nil {
			return err
		}
	}

	_, err = c.JS.Publish(ctx, subject, data, jetstream.WithRetryWait(retryWait))
	if err != nil {
		return err
	}

	return nil
}
