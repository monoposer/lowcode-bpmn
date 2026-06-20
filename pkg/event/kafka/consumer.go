package kafka

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/event/codec"
	"github.com/monoposer/lowcode-bpmn/pkg/event/transport"
)

func init() {
	transport.Register(driver{})
}

type driver struct{}

func (driver) Name() string { return transport.DriverKafka }

func (driver) NewConsumer(stream event.Stream, cfg transport.Config) (event.Consumer, error) {
	brokers, err := parseBrokers(cfg)
	if err != nil {
		return nil, err
	}
	topic := cfg.Destination(stream)
	groupID := cfg.ExtraGet("group_id", "lowcode-bpmn")
	return &Consumer{
		stream:  stream,
		brokers: brokers,
		topic:   topic,
		groupID: groupID,
	}, nil
}

// Consumer implements event.Consumer for Apache Kafka.
type Consumer struct {
	stream  event.Stream
	brokers []string
	topic   string
	groupID string

	writer *kafka.Writer
	reader *kafka.Reader
	mu     sync.Mutex
	closed bool
}

func parseBrokers(cfg transport.Config) ([]string, error) {
	if raw := cfg.ExtraGet("brokers", ""); raw != "" {
		parts := strings.Split(raw, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	if cfg.BrokerURL == "" {
		return nil, fmt.Errorf("EVENT_BROKER_URL or EVENT_KAFKA_BROKERS required when EVENT_CONSUMER=kafka")
	}
	u, err := url.Parse(cfg.BrokerURL)
	if err != nil {
		return nil, fmt.Errorf("kafka broker url: %w", err)
	}
	host := u.Host
	if host == "" {
		host = strings.TrimPrefix(cfg.BrokerURL, "kafka://")
	}
	if host == "" {
		return nil, fmt.Errorf("kafka broker url: empty host")
	}
	return []string{host}, nil
}

func (c *Consumer) Stream() event.Stream { return c.stream }

func (c *Consumer) ensureWriter() *kafka.Writer {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.writer == nil {
		c.writer = &kafka.Writer{
			Addr:     kafka.TCP(c.brokers...),
			Topic:    c.topic,
			Balancer: &kafka.LeastBytes{},
		}
	}
	return c.writer
}

func (c *Consumer) Run(ctx context.Context, handler event.Handler) error {
	c.mu.Lock()
	if c.reader == nil {
		c.reader = kafka.NewReader(kafka.ReaderConfig{
			Brokers: c.brokers,
			Topic:   c.topic,
			GroupID: c.groupID,
		})
	}
	reader := c.reader
	c.mu.Unlock()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		evt, err := codec.Unmarshal(msg.Value)
		if err != nil {
			_ = reader.CommitMessages(ctx, msg)
			continue
		}
		if evt.Stream == "" {
			evt.Stream = c.stream
		}
		ack := func() error {
			return reader.CommitMessages(ctx, msg)
		}
		if err := handler(ctx, evt, ack); err != nil {
			continue
		}
	}
}

func (c *Consumer) Publish(ctx context.Context, evt event.InboundEvent) error {
	codec.Normalize(&evt, c.stream)
	raw, err := codec.Marshal(evt)
	if err != nil {
		return err
	}
	return c.ensureWriter().WriteMessages(ctx, kafka.Message{
		Key:   []byte(evt.ID),
		Value: raw,
		Time:  time.Now().UTC(),
	})
}

func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	var err error
	if c.writer != nil {
		if e := c.writer.Close(); e != nil {
			err = e
		}
		c.writer = nil
	}
	if c.reader != nil {
		if e := c.reader.Close(); e != nil && err == nil {
			err = e
		}
		c.reader = nil
	}
	return err
}
