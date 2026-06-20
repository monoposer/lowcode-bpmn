package contract

import (
	"context"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
)

// EventAdapter maps vendor payloads to Host SDK calls.
type EventAdapter interface {
	Name() string
	Stream() event.Stream
	Supports(evt event.InboundEvent) bool
	Handle(ctx context.Context, evt event.InboundEvent, host Host) error
}
