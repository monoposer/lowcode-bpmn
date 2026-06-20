package setup

import (
	"context"

	"github.com/monoposer/lowcode-bpmn/pkg/event/transport"

	// Register event transport drivers (memory, redis, kafka, nats, rabbitmq).
	_ "github.com/monoposer/lowcode-bpmn/pkg/event/kafka"
	_ "github.com/monoposer/lowcode-bpmn/pkg/event/memory"
	_ "github.com/monoposer/lowcode-bpmn/pkg/event/nats"
	_ "github.com/monoposer/lowcode-bpmn/pkg/event/rabbitmq"
	_ "github.com/monoposer/lowcode-bpmn/pkg/event/redis"
)

// Config configures quad-stream consumers.
type Config = transport.Config

// Streams holds assignee, trigger, task, and control consumers.
type Streams transport.Streams

// NewStreams creates all stream consumers via the configured driver registry.
func NewStreams(cfg Config) (*Streams, error) {
	s, err := transport.NewStreams(cfg)
	if err != nil {
		return nil, err
	}
	return (*Streams)(s), nil
}

// LoadConfigFromEnv reads EVENT_* variables.
func LoadConfigFromEnv() Config {
	return transport.LoadConfigFromEnv()
}

// Close shuts down all consumers.
func (s *Streams) Close(_ context.Context) {
	if s == nil {
		return
	}
	(*transport.Streams)(s).Close()
}
