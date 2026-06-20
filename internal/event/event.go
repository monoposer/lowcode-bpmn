package event

import (
	"context"
	"time"
)

// Stream separates event intents for independent consumers.
type Stream string

const (
	StreamAssignee Stream = "assignee" // HR / org change → assignee sync
	StreamTrigger  Stream = "trigger"  // webhooks → message start / process start
	StreamTask     Stream = "task"     // external approve / reject → CompleteTask
)

// InboundEvent is a transport-agnostic envelope delivered to adapter plugins.
type InboundEvent struct {
	ID         string            `json:"id"`
	Stream     Stream            `json:"stream"`
	Source     string            `json:"source"`
	Topic      string            `json:"topic,omitempty"`
	TenantID   string            `json:"tenant_id"`
	Headers    map[string]string `json:"headers,omitempty"`
	Payload    []byte            `json:"payload"`
	ReceivedAt time.Time         `json:"received_at"`
}

// AckFunc acknowledges successful processing.
type AckFunc func() error

// Handler processes one inbound event.
type Handler func(ctx context.Context, evt InboundEvent, ack AckFunc) error

// Consumer abstracts event ingress (Redis, Kafka, in-memory, …).
type Consumer interface {
	Stream() Stream
	Run(ctx context.Context, handler Handler) error
	Publish(ctx context.Context, evt InboundEvent) error
	Close() error
}

// Publisher publishes to a specific stream.
type Publisher interface {
	Publish(ctx context.Context, stream Stream, evt InboundEvent) error
}

// RouterPublisher routes HTTP / bridge events to stream consumers.
type RouterPublisher struct {
	assignee Consumer
	trigger  Consumer
	task     Consumer
}

func NewRouterPublisher(assignee, trigger, task Consumer) *RouterPublisher {
	return &RouterPublisher{assignee: assignee, trigger: trigger, task: task}
}

func (r *RouterPublisher) Publish(ctx context.Context, stream Stream, evt InboundEvent) error {
	if evt.Stream == "" {
		evt.Stream = stream
	}
	switch stream {
	case StreamAssignee:
		if r.assignee == nil {
			return ErrStreamDisabled
		}
		return r.assignee.Publish(ctx, evt)
	case StreamTrigger:
		if r.trigger == nil {
			return ErrStreamDisabled
		}
		return r.trigger.Publish(ctx, evt)
	case StreamTask:
		if r.task == nil {
			return ErrStreamDisabled
		}
		return r.task.Publish(ctx, evt)
	default:
		return ErrUnknownStream
	}
}
