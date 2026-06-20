package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/monoposer/lowcode-bpmn/internal/event"
)

// Consumer reads events from a Redis list (LPUSH / BRPOP).
type Consumer struct {
	client *goredis.Client
	stream event.Stream
	key    string
	own    bool
}

// New connects to Redis and binds to a list key for one stream.
func New(url string, stream event.Stream, key string) (*Consumer, error) {
	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis parse url: %w", err)
	}
	client := goredis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Consumer{client: client, stream: stream, key: key, own: true}, nil
}

func (c *Consumer) Stream() event.Stream { return c.stream }

func (c *Consumer) Run(ctx context.Context, handler event.Handler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		res, err := c.client.BRPop(ctx, 2*time.Second, c.key).Result()
		if err == goredis.Nil {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if len(res) < 2 {
			continue
		}
		var evt event.InboundEvent
		if err := json.Unmarshal([]byte(res[1]), &evt); err != nil {
			continue
		}
		if evt.Stream == "" {
			evt.Stream = c.stream
		}
		ack := func() error { return nil }
		if err := handler(ctx, evt, ack); err != nil {
			// Re-queue on failure for at-least-once (best effort).
			_ = c.client.LPush(ctx, c.key, res[1]).Err()
		}
	}
}

func (c *Consumer) Publish(ctx context.Context, evt event.InboundEvent) error {
	if evt.ID == "" {
		evt.ID = uuid.NewString()
	}
	if evt.Stream == "" {
		evt.Stream = c.stream
	}
	if evt.ReceivedAt.IsZero() {
		evt.ReceivedAt = time.Now().UTC()
	}
	raw, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	return c.client.LPush(ctx, c.key, raw).Err()
}

func (c *Consumer) Close() error {
	if c.own && c.client != nil {
		return c.client.Close()
	}
	return nil
}
