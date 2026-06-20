package nats

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/event/codec"
	"github.com/monoposer/lowcode-bpmn/pkg/event/transport"
)

func init() {
	transport.Register(driver{})
}

type driver struct{}

func (driver) Name() string { return transport.DriverNATS }

func (driver) NewConsumer(stream event.Stream, cfg transport.Config) (event.Consumer, error) {
	url := normalizeURL(cfg.BrokerURL)
	if url == "" {
		return nil, fmt.Errorf("EVENT_BROKER_URL required when EVENT_CONSUMER=nats")
	}
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	queue := cfg.ExtraGet("queue", "lowcode-bpmn")
	return &Consumer{
		stream: stream,
		nc:     nc,
		subject: cfg.Destination(stream),
		queue:  queue,
		own:    true,
	}, nil
}

// Consumer implements event.Consumer for NATS (core pub/sub with queue groups).
type Consumer struct {
	stream  event.Stream
	nc      *nats.Conn
	subject string
	queue   string
	sub     *nats.Subscription
	mu      sync.Mutex
	own     bool
}

func (c *Consumer) Stream() event.Stream { return c.stream }

func (c *Consumer) Run(ctx context.Context, handler event.Handler) error {
	sub, err := c.nc.QueueSubscribe(c.subject, c.queue, func(msg *nats.Msg) {
		evt, err := codec.Unmarshal(msg.Data)
		if err != nil {
			return
		}
		if evt.Stream == "" {
			evt.Stream = c.stream
		}
		ack := func() error {
			return nil
		}
		_ = handler(ctx, evt, ack)
	})
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.sub = sub
	c.mu.Unlock()

	<-ctx.Done()
	return ctx.Err()
}

func (c *Consumer) Publish(ctx context.Context, evt event.InboundEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	codec.Normalize(&evt, c.stream)
	raw, err := codec.Marshal(evt)
	if err != nil {
		return err
	}
	return c.nc.Publish(c.subject, raw)
}

func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sub != nil {
		_ = c.sub.Unsubscribe()
		c.sub = nil
	}
	if c.own && c.nc != nil {
		c.nc.Close()
		c.nc = nil
	}
	return nil
}

// BrokerURL helpers accept nats://host:4222 or plain host:port.
func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if strings.Contains(raw, "://") {
		return raw
	}
	return "nats://" + raw
}
