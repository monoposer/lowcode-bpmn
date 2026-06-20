package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
)

// Consumer is an in-process queue for dev, tests, and HTTP webhook bridging.
type Consumer struct {
	stream event.Stream
	ch     chan event.InboundEvent
	closed chan struct{}
	once   sync.Once
}

// New creates a buffered in-memory consumer for one stream.
func New(stream event.Stream, buffer int) *Consumer {
	if buffer <= 0 {
		buffer = 256
	}
	return &Consumer{
		stream: stream,
		ch:     make(chan event.InboundEvent, buffer),
		closed: make(chan struct{}),
	}
}

func (c *Consumer) Stream() event.Stream { return c.stream }

func (c *Consumer) Run(ctx context.Context, handler event.Handler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closed:
			return nil
		case evt, ok := <-c.ch:
			if !ok {
				return nil
			}
			if evt.Stream == "" {
				evt.Stream = c.stream
			}
			ack := func() error { return nil }
			if err := handler(ctx, evt, ack); err != nil {
				continue
			}
		}
	}
}

func (c *Consumer) Publish(ctx context.Context, evt event.InboundEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closed:
		return context.Canceled
	default:
	}
	if evt.ID == "" {
		evt.ID = uuid.NewString()
	}
	if evt.Stream == "" {
		evt.Stream = c.stream
	}
	if evt.ReceivedAt.IsZero() {
		evt.ReceivedAt = time.Now().UTC()
	}
	select {
	case c.ch <- evt:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Consumer) Close() error {
	c.once.Do(func() { close(c.closed); close(c.ch) })
	return nil
}
