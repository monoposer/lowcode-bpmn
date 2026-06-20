package setup

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/monoposer/lowcode-bpmn/pkg/env"
	"github.com/monoposer/lowcode-bpmn/pkg/event"
	memconsumer "github.com/monoposer/lowcode-bpmn/pkg/event/memory"
	redisconsumer "github.com/monoposer/lowcode-bpmn/pkg/event/redis"
)

// Config configures quad-stream consumers.
type Config struct {
	Kind              string
	RedisURL          string
	RedisAssigneeKey  string
	RedisTriggerKey   string
	RedisTaskKey      string
	RedisControlKey   string
	BufferSize        int
}

// Streams holds assignee, trigger, task, and control consumers.
type Streams struct {
	Assignee event.Consumer
	Trigger  event.Consumer
	Task     event.Consumer
	Control  event.Consumer
}

// NewStreams creates all stream consumers.
func NewStreams(cfg Config) (*Streams, error) {
	kind := strings.ToLower(strings.TrimSpace(cfg.Kind))
	switch kind {
	case "none", "off", "":
		return &Streams{}, nil
	case "memory":
		buf := cfg.BufferSize
		if buf <= 0 {
			buf = 512
		}
		return &Streams{
			Assignee: memconsumer.New(event.StreamAssignee, buf),
			Trigger:  memconsumer.New(event.StreamTrigger, buf),
			Task:     memconsumer.New(event.StreamTask, buf),
			Control:  memconsumer.New(event.StreamControl, buf),
		}, nil
	case "redis":
		if cfg.RedisURL == "" {
			return nil, fmt.Errorf("EVENT_REDIS_URL required when EVENT_CONSUMER=redis")
		}
		assigneeKey := cfg.RedisAssigneeKey
		if assigneeKey == "" {
			assigneeKey = "bpmn:events:assignee"
		}
		triggerKey := cfg.RedisTriggerKey
		if triggerKey == "" {
			triggerKey = "bpmn:events:trigger"
		}
		taskKey := cfg.RedisTaskKey
		if taskKey == "" {
			taskKey = "bpmn:events:task"
		}
		controlKey := cfg.RedisControlKey
		if controlKey == "" {
			controlKey = "bpmn:events:control"
		}
		a, err := redisconsumer.New(cfg.RedisURL, event.StreamAssignee, assigneeKey)
		if err != nil {
			return nil, err
		}
		t, err := redisconsumer.New(cfg.RedisURL, event.StreamTrigger, triggerKey)
		if err != nil {
			_ = a.Close()
			return nil, err
		}
		k, err := redisconsumer.New(cfg.RedisURL, event.StreamTask, taskKey)
		if err != nil {
			_ = a.Close()
			_ = t.Close()
			return nil, err
		}
		c, err := redisconsumer.New(cfg.RedisURL, event.StreamControl, controlKey)
		if err != nil {
			_ = a.Close()
			_ = t.Close()
			_ = k.Close()
			return nil, err
		}
		return &Streams{Assignee: a, Trigger: t, Task: k, Control: c}, nil
	default:
		return nil, fmt.Errorf("unsupported EVENT_CONSUMER %q", cfg.Kind)
	}
}

// LoadConfigFromEnv reads EVENT_* variables.
func LoadConfigFromEnv() Config {
	buf := 512
	if v := os.Getenv("EVENT_BUFFER_SIZE"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			buf = n
		}
	}
	return Config{
		Kind:             env.Get("EVENT_CONSUMER", "memory"),
		RedisURL:         os.Getenv("EVENT_REDIS_URL"),
		RedisAssigneeKey: os.Getenv("EVENT_REDIS_ASSIGNEE_KEY"),
		RedisTriggerKey:  os.Getenv("EVENT_REDIS_TRIGGER_KEY"),
		RedisTaskKey:     os.Getenv("EVENT_REDIS_TASK_KEY"),
		RedisControlKey:  os.Getenv("EVENT_REDIS_CONTROL_KEY"),
		BufferSize:       buf,
	}
}

// Close shuts down all consumers.
func (s *Streams) Close(context.Context) {
	if s == nil {
		return
	}
	if s.Assignee != nil {
		_ = s.Assignee.Close()
	}
	if s.Trigger != nil {
		_ = s.Trigger.Close()
	}
	if s.Task != nil {
		_ = s.Task.Close()
	}
	if s.Control != nil {
		_ = s.Control.Close()
	}
}
