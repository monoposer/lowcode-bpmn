package api

import (
	"context"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/pkg/event"
)

// RouterDeps holds shared handlers for HTTP routes.
type RouterDeps struct {
	Engine *engine.Engine
	Events event.Publisher
}

type engineKey struct{}

func WithEngine(ctx context.Context, e *engine.Engine) context.Context {
	return context.WithValue(ctx, engineKey{}, e)
}

func getEngine(ctx context.Context, deps RouterDeps) *engine.Engine {
	if e, ok := ctx.Value(engineKey{}).(*engine.Engine); ok && e != nil {
		return e
	}
	return deps.Engine
}
