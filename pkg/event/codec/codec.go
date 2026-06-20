package codec

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
)

// Normalize fills missing fields on an inbound event before publish.
func Normalize(evt *event.InboundEvent, stream event.Stream) {
	if evt.ID == "" {
		evt.ID = uuid.NewString()
	}
	if evt.Stream == "" {
		evt.Stream = stream
	}
	if evt.ReceivedAt.IsZero() {
		evt.ReceivedAt = time.Now().UTC()
	}
}

func Marshal(evt event.InboundEvent) ([]byte, error) {
	return json.Marshal(evt)
}

func Unmarshal(b []byte) (event.InboundEvent, error) {
	var evt event.InboundEvent
	if err := json.Unmarshal(b, &evt); err != nil {
		return evt, err
	}
	return evt, nil
}
