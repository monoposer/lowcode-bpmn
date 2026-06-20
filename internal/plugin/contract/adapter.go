package contract

import (
	"context"

	"github.com/monoposer/lowcode-bpmn/internal/event"
)

// EventAdapter translates an external event format into Host API calls.
type EventAdapter interface {
	Name() string
	Stream() event.Stream
	Supports(evt event.InboundEvent) bool
	Handle(ctx context.Context, evt event.InboundEvent, host Host) error
}
