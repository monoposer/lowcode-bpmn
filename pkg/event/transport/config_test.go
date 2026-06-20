package transport_test

import (
	"testing"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/event/transport"

	_ "github.com/monoposer/lowcode-bpmn/pkg/event/kafka"
	_ "github.com/monoposer/lowcode-bpmn/pkg/event/memory"
	_ "github.com/monoposer/lowcode-bpmn/pkg/event/nats"
	_ "github.com/monoposer/lowcode-bpmn/pkg/event/rabbitmq"
	_ "github.com/monoposer/lowcode-bpmn/pkg/event/redis"
)

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("EVENT_CONSUMER", "kafka")
	t.Setenv("EVENT_BROKER_URL", "kafka://localhost:9092")
	t.Setenv("EVENT_TRIGGER_DEST", "custom.trigger")

	cfg := transport.LoadConfigFromEnv()
	if cfg.Driver != "kafka" {
		t.Fatalf("driver: got %q", cfg.Driver)
	}
	if cfg.BrokerURL != "kafka://localhost:9092" {
		t.Fatalf("broker url: got %q", cfg.BrokerURL)
	}
	if cfg.Destination(event.StreamTrigger) != "custom.trigger" {
		t.Fatalf("trigger dest: got %q", cfg.Destination(event.StreamTrigger))
	}
	if cfg.Destination(event.StreamAssignee) != "bpmn.events.assignee" {
		t.Fatalf("assignee dest: got %q", cfg.Destination(event.StreamAssignee))
	}
}

func TestLoadConfigRedisLegacyKeys(t *testing.T) {
	t.Setenv("EVENT_CONSUMER", "redis")
	t.Setenv("EVENT_REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("EVENT_REDIS_TRIGGER_KEY", "legacy:trigger")

	cfg := transport.LoadConfigFromEnv()
	if cfg.BrokerURL != "redis://localhost:6379/0" {
		t.Fatalf("broker: %q", cfg.BrokerURL)
	}
	if cfg.Destination(event.StreamTrigger) != "legacy:trigger" {
		t.Fatalf("trigger dest: %q", cfg.Destination(event.StreamTrigger))
	}
	if cfg.Destination(event.StreamTask) != "bpmn:events:task" {
		t.Fatalf("task dest: %q", cfg.Destination(event.StreamTask))
	}
}

func TestNewStreamsMemoryDriver(t *testing.T) {
	cfg := transport.Config{Driver: transport.DriverMemory, BufferSize: 8}
	streams, err := transport.NewStreams(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer streams.Close()
	if streams.Trigger == nil {
		t.Fatal("expected trigger consumer")
	}
}

func TestUnknownDriver(t *testing.T) {
	_, err := transport.NewStreams(transport.Config{Driver: "pulsar"})
	if err == nil {
		t.Fatal("expected error for unknown driver")
	}
}
