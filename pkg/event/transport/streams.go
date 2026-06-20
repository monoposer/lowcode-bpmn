package transport

import (
	"fmt"
	"strings"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
)

// Streams holds assignee, trigger, task, and control consumers.
type Streams struct {
	Assignee event.Consumer
	Trigger  event.Consumer
	Task     event.Consumer
	Control  event.Consumer
}

// NewStreams creates four stream consumers using the configured driver.
func NewStreams(cfg Config) (*Streams, error) {
	kind := strings.ToLower(strings.TrimSpace(cfg.Driver))
	switch kind {
	case "none", "off", "":
		return &Streams{}, nil
	}

	drv, err := Get(kind)
	if err != nil {
		return nil, err
	}

	a, err := drv.NewConsumer(event.StreamAssignee, cfg)
	if err != nil {
		return nil, fmt.Errorf("assignee stream: %w", err)
	}
	t, err := drv.NewConsumer(event.StreamTrigger, cfg)
	if err != nil {
		_ = a.Close()
		return nil, fmt.Errorf("trigger stream: %w", err)
	}
	k, err := drv.NewConsumer(event.StreamTask, cfg)
	if err != nil {
		_ = a.Close()
		_ = t.Close()
		return nil, fmt.Errorf("task stream: %w", err)
	}
	c, err := drv.NewConsumer(event.StreamControl, cfg)
	if err != nil {
		_ = a.Close()
		_ = t.Close()
		_ = k.Close()
		return nil, fmt.Errorf("control stream: %w", err)
	}
	return &Streams{Assignee: a, Trigger: t, Task: k, Control: c}, nil
}

// Close shuts down all consumers.
func (s *Streams) Close() {
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
