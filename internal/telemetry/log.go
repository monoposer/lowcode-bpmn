package telemetry

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// InitLogger configures the global slog logger.
func InitLogger(cfg Config) *slog.Logger {
	level := parseLevel(cfg.LogLevel)
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch cfg.LogFormat {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(&contextHandler{Handler: handler})
	slog.SetDefault(logger)
	return logger
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(raw) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type contextHandler struct {
	slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	r = appendContextAttrs(ctx, r)
	return h.Handler.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithGroup(name)}
}

func appendContextAttrs(ctx context.Context, r slog.Record) slog.Record {
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			r.AddAttrs(
				slog.String("trace_id", sc.TraceID().String()),
				slog.String("span_id", sc.SpanID().String()),
			)
		}
	}
	if reqID, ok := ctx.Value(requestIDKey{}).(string); ok && reqID != "" {
		r.AddAttrs(slog.String("request_id", reqID))
	}
	return r
}

type requestIDKey struct{}

// WithRequestID stores a request identifier on the context for structured logs.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// Ctx returns a logger enriched with trace and request context.
func Ctx(ctx context.Context) *slog.Logger {
	return slog.Default().With(extractContextAttrs(ctx)...)
}

func extractContextAttrs(ctx context.Context) []any {
	var attrs []any
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			attrs = append(attrs,
				slog.String("trace_id", sc.TraceID().String()),
				slog.String("span_id", sc.SpanID().String()),
			)
		}
	}
	if reqID, ok := ctx.Value(requestIDKey{}).(string); ok && reqID != "" {
		attrs = append(attrs, slog.String("request_id", reqID))
	}
	return attrs
}

// DiscardWriter satisfies io.Writer for tests.
var DiscardWriter io.Writer = io.Discard
