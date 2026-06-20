package plugin

import (
	"context"
	"fmt"

	"github.com/monoposer/lowcode-bpmn/internal/event"
	"github.com/monoposer/lowcode-bpmn/internal/plugin/contract"
)

// EventAdapter translates an external event format into Host API calls.
type EventAdapter = contract.EventAdapter

// Runtime dispatches inbound events to the first matching adapter.
type Runtime struct {
	host      Host
	adapters  []EventAdapter
	onNoMatch func(event.InboundEvent)
}

func NewRuntime(host Host, adapters ...EventAdapter) *Runtime {
	return &Runtime{host: host, adapters: adapters}
}

func (r *Runtime) SetNoMatchHandler(fn func(event.InboundEvent)) {
	r.onNoMatch = fn
}

func (r *Runtime) Handler() event.Handler {
	return func(ctx context.Context, evt event.InboundEvent, ack event.AckFunc) error {
		if err := r.Handle(ctx, evt); err != nil {
			return err
		}
		if ack != nil {
			return ack()
		}
		return nil
	}
}

func (r *Runtime) Handle(ctx context.Context, evt event.InboundEvent) error {
	for _, ad := range r.adapters {
		if evt.Stream != "" && ad.Stream() != evt.Stream {
			continue
		}
		if ad.Supports(evt) {
			return ad.Handle(ctx, evt, r.host)
		}
	}
	if r.onNoMatch != nil {
		r.onNoMatch(evt)
		return nil
	}
	return fmt.Errorf("plugin: no adapter for source %q topic %q stream %q", evt.Source, evt.Topic, evt.Stream)
}

func (r *Runtime) AdapterNames() []string {
	names := make([]string, len(r.adapters))
	for i, ad := range r.adapters {
		names[i] = ad.Name()
	}
	return names
}
