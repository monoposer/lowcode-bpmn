package transport

import (
	"fmt"
	"os"
	"strings"

	"github.com/monoposer/lowcode-bpmn/pkg/env"
	"github.com/monoposer/lowcode-bpmn/pkg/event"
)

const (
	DriverMemory   = "memory"
	DriverRedis    = "redis"
	DriverKafka    = "kafka"
	DriverNATS     = "nats"
	DriverRabbitMQ = "rabbitmq"
)

// Config is the unified event transport configuration (scheme A).
type Config struct {
	Driver     string
	BrokerURL  string
	Dest       map[event.Stream]string
	BufferSize int
	Extra      map[string]string
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

	driver := strings.ToLower(strings.TrimSpace(env.Get("EVENT_CONSUMER", DriverMemory)))
	brokerURL := strings.TrimSpace(os.Getenv("EVENT_BROKER_URL"))
	if brokerURL == "" {
		brokerURL = strings.TrimSpace(os.Getenv("EVENT_REDIS_URL"))
	}

	cfg := Config{
		Driver:     driver,
		BrokerURL:  brokerURL,
		BufferSize: buf,
		Dest:       loadDestinations(driver),
		Extra:      loadExtra(driver),
	}
	return cfg
}

func loadDestinations(driver string) map[event.Stream]string {
	defaultFor := func(stream event.Stream) string {
		switch driver {
		case DriverRedis:
			return "bpmn:events:" + string(stream)
		default:
			return "bpmn.events." + string(stream)
		}
	}

	dest := map[event.Stream]string{
		event.StreamAssignee: env.Get("EVENT_ASSIGNEE_DEST", defaultFor(event.StreamAssignee)),
		event.StreamTrigger:  env.Get("EVENT_TRIGGER_DEST", defaultFor(event.StreamTrigger)),
		event.StreamTask:     env.Get("EVENT_TASK_DEST", defaultFor(event.StreamTask)),
		event.StreamControl:  env.Get("EVENT_CONTROL_DEST", defaultFor(event.StreamControl)),
	}

	// Backward compatibility: legacy Redis key env vars override dest when driver=redis.
	if driver == DriverRedis {
		if v := os.Getenv("EVENT_REDIS_ASSIGNEE_KEY"); v != "" {
			dest[event.StreamAssignee] = v
		}
		if v := os.Getenv("EVENT_REDIS_TRIGGER_KEY"); v != "" {
			dest[event.StreamTrigger] = v
		}
		if v := os.Getenv("EVENT_REDIS_TASK_KEY"); v != "" {
			dest[event.StreamTask] = v
		}
		if v := os.Getenv("EVENT_REDIS_CONTROL_KEY"); v != "" {
			dest[event.StreamControl] = v
		}
	}
	return dest
}

func loadExtra(driver string) map[string]string {
	prefix := "EVENT_" + strings.ToUpper(driver) + "_"
	out := make(map[string]string)
	for _, kv := range os.Environ() {
		key, val, ok := strings.Cut(kv, "=")
		if !ok || !strings.HasPrefix(key, prefix) {
			continue
		}
		out[strings.ToLower(strings.TrimPrefix(key, prefix))] = val
	}
	return out
}

func (c Config) Destination(stream event.Stream) string {
	if c.Dest != nil {
		if d, ok := c.Dest[stream]; ok && d != "" {
			return d
		}
	}
	return "bpmn.events." + string(stream)
}

func (c Config) ExtraGet(key, def string) string {
	if c.Extra != nil {
		if v, ok := c.Extra[strings.ToLower(key)]; ok && v != "" {
			return v
		}
	}
	return def
}
