package telemetry

import (
	"strings"

	"github.com/monoposer/lowcode-bpmn/pkg/env"
)

// Config holds logging and OpenTelemetry settings.
type Config struct {
	ServiceName    string
	ServiceVersion string
	LogLevel       string
	LogFormat      string
	OTelEnabled    bool
}

// LoadConfig reads telemetry settings from environment variables.
func LoadConfig() Config {
	return Config{
		ServiceName:    env.Get("OTEL_SERVICE_NAME", "lowcode-bpmn"),
		ServiceVersion: env.Get("SERVICE_VERSION", "dev"),
		LogLevel:       strings.ToLower(env.Get("LOG_LEVEL", "info")),
		LogFormat:      strings.ToLower(env.Get("LOG_FORMAT", "json")),
		OTelEnabled:    env.Bool("OTEL_ENABLED", false),
	}
}
