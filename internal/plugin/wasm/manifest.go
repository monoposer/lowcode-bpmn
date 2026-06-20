package wasm

import (
	"encoding/json"
	"fmt"

	"github.com/monoposer/lowcode-bpmn/internal/event"
)

// Manifest describes a WASM plugin (plugin.json beside plugin.wasm).
type Manifest struct {
	Name         string   `json:"name"`
	Stream       string   `json:"stream"` // assignee | trigger | task
	Sources      []string `json:"sources"`
	Capabilities []string `json:"capabilities"`
	WASM         string   `json:"wasm"`
}

func (m Manifest) EventStream() (event.Stream, error) {
	switch event.Stream(m.Stream) {
	case event.StreamAssignee:
		return event.StreamAssignee, nil
	case event.StreamTrigger:
		return event.StreamTrigger, nil
	case event.StreamTask:
		return event.StreamTask, nil
	default:
		return "", fmt.Errorf("wasm manifest %q: invalid stream %q", m.Name, m.Stream)
	}
}

func (m Manifest) CapabilitySet() Set {
	if len(m.Capabilities) == 0 {
		return nil
	}
	return ParseCapabilities(m.Capabilities)
}

func ParseManifest(raw []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return m, err
	}
	if m.Name == "" {
		return m, fmt.Errorf("wasm manifest: name required")
	}
	if m.WASM == "" {
		m.WASM = "plugin.wasm"
	}
	return m, nil
}
