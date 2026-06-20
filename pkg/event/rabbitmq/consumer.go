package rabbitmq

import (
	"context"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/event/codec"
	"github.com/monoposer/lowcode-bpmn/pkg/event/transport"
)

func init() {
	transport.Register(driver{})
}

type driver struct{}

func (driver) Name() string { return transport.DriverRabbitMQ }

func (driver) NewConsumer(stream event.Stream, cfg transport.Config) (event.Consumer, error) {
	url := cfg.BrokerURL
	if url == "" {
		return nil, fmt.Errorf("EVENT_BROKER_URL required when EVENT_CONSUMER=rabbitmq")
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	queue := cfg.Destination(stream)
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq queue declare: %w", err)
	}
	return &Consumer{
		stream: stream,
		conn:   conn,
		ch:     ch,
		queue:  queue,
	}, nil
}

// Consumer implements event.Consumer for RabbitMQ (direct queue publish/consume).
type Consumer struct {
	stream event.Stream
	conn   *amqp.Connection
	ch     *amqp.Channel
	queue  string
	mu     sync.Mutex
}

func (c *Consumer) Stream() event.Stream { return c.stream }

func (c *Consumer) Run(ctx context.Context, handler event.Handler) error {
	deliveries, err := c.ch.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			evt, err := codec.Unmarshal(d.Body)
			if err != nil {
				_ = d.Nack(false, false)
				continue
			}
			if evt.Stream == "" {
				evt.Stream = c.stream
			}
			ack := func() error {
				return d.Ack(false)
			}
			if err := handler(ctx, evt, ack); err != nil {
				_ = d.Nack(false, true)
				continue
			}
		}
	}
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
	return c.ch.PublishWithContext(ctx, "", c.queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         raw,
		DeliveryMode: amqp.Persistent,
	})
}

func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ch != nil {
		_ = c.ch.Close()
		c.ch = nil
	}
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
