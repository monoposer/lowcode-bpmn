package transport

import (
	"fmt"
	"strings"
	"sync"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
)

// Driver constructs one stream-bound event.Consumer from unified transport config.
type Driver interface {
	Name() string
	NewConsumer(stream event.Stream, cfg Config) (event.Consumer, error)
}

var (
	mu       sync.RWMutex
	registry = map[string]Driver{}
)

// Register adds a transport driver. Called from driver package init().
func Register(d Driver) {
	if d == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	registry[strings.ToLower(d.Name())] = d
}

// Get returns a registered driver by name.
func Get(name string) (Driver, error) {
	mu.RLock()
	defer mu.RUnlock()
	d, ok := registry[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, fmt.Errorf("event transport: unknown driver %q (registered: %s)", name, RegisteredNames())
	}
	return d, nil
}

// RegisteredNames returns sorted driver names for error messages.
func RegisteredNames() string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}
