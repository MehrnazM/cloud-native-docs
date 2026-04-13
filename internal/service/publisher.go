package service

import (
	"context"
	"time"
)

type Publisher interface {
	Publish(ctx context.Context, subject string, message any, retryWait time.Duration) error
}
