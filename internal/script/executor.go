package script

import (
	"context"
	"fmt"
	"log/slog"
)

// Executor runs ScriptTask scripts (javascript or log only).
type Executor struct {
	js *JSExecutor
}

// NewExecutor returns the default script executor (JS via goja + log mode).
func NewExecutor() *Executor {
	return &Executor{js: NewJSExecutor(nil)}
}

func (e *Executor) Run(ctx context.Context, req RunRequest) (map[string]any, error) {
	if req.Script == "" {
		return nil, fmt.Errorf("script is empty")
	}

	switch normalizeLang(req.Lang) {
	case "javascript", "js":
		if e.js == nil {
			return nil, fmt.Errorf("javascript runtime not configured")
		}
		return e.js.Run(ctx, req)
	case "log":
		attrs := append(slogAttrs(req),
			slog.String("script", req.Script),
			slog.Int("vars", len(req.Variables)),
		)
		slog.InfoContext(ctx, "scriptTask log", attrs...)
		return map[string]any{}, nil
	default:
		attrs := append(slogAttrs(req), slog.String("scriptLang", req.Lang))
		slog.WarnContext(ctx, "unsupported scriptLang", attrs...)
		return nil, fmt.Errorf("unsupported scriptLang %q (use js or log)", req.Lang)
	}
}

func normalizeLang(lang string) string {
	switch lang {
	case "", "javascript", "js":
		return "javascript"
	case "log":
		return "log"
	default:
		return lang
	}
}
