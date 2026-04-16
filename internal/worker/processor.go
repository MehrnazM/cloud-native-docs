package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/MehrnazM/cloud-native-docs/shared/events"
)

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) ProcessWithContext(ctx context.Context, doc events.DocumentCreatedEvent) error {
	fmt.Printf("received event, start processing the event with ID: %s\n", doc.ID)
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("finished processing the event with ID: %s\n", doc.ID)
	return nil
}
