package telemetry

import (
	"context"
	"fmt"
	"log/slog"
)

// Init configures structured logging and OpenTelemetry.
func Init(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	logger := InitLogger(cfg)

	var traceShutdown func(context.Context) error
	traceShutdown, err = InitTracer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("telemetry initialized",
		slog.String("service", cfg.ServiceName),
		slog.String("version", cfg.ServiceVersion),
		slog.String("log_level", cfg.LogLevel),
		slog.String("log_format", cfg.LogFormat),
		slog.Bool("otel_enabled", cfg.OTelEnabled),
	)

	return func(ctx context.Context) error {
		if traceShutdown == nil {
			return nil
		}
		if err := traceShutdown(ctx); err != nil {
			return fmt.Errorf("shutdown tracer: %w", err)
		}
		logger.Info("telemetry shutdown complete")
		return nil
	}, nil
}
