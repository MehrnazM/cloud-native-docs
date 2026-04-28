package messaging

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/MehrnazM/cloud-native-docs/shared/util"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Connection struct {
	JS jetstream.JetStream
	NC *nats.Conn
}

func NewConnection(logger *slog.Logger) (conn *Connection, err error) {
	url, err := util.MustGetString("NATS_URL")
	if err != nil || url == "" {
		return nil, nats.ErrNoServers
	}
	var nc *nats.Conn
	retryCount, err := util.GetIntEnv("NATS_RETRY_COUNT", 10)
	if err != nil {
		logger.Error("Invalid NATS_RETRY_COUNT value", "error", err)
		return nil, err
	}
	for i := 0; i < retryCount; i++ {
		nc, err = nats.Connect(url)
		if err == nil {
			break
		}
		logger.Warn("NATS connection failed, retrying", "attempt", i+1, "error", err)
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
