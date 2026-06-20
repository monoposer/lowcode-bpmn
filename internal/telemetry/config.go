package telemetry

import (
	"os"
	"strconv"
	"strings"
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
		ServiceName:    envOr("OTEL_SERVICE_NAME", "lowcode-bpmn"),
		ServiceVersion: envOr("SERVICE_VERSION", "dev"),
		LogLevel:       strings.ToLower(envOr("LOG_LEVEL", "info")),
		LogFormat:      strings.ToLower(envOr("LOG_FORMAT", "json")),
		OTelEnabled:    envBool("OTEL_ENABLED", false),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
