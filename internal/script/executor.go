package script

import (
	"context"
	"fmt"
	"log"
)

// Executor runs ScriptTask scripts with set/log stubs and delegates JS to goja.
type Executor struct {
	js Runner
}

// NewExecutor returns the default script executor (JS via goja).
func NewExecutor() *Executor {
	js := NewJSExecutor()
	return &Executor{js: js}
}

func (e *Executor) Run(ctx context.Context, scriptBody, lang string, vars map[string]any) (map[string]any, error) {
	if scriptBody == "" {
		return nil, fmt.Errorf("script is empty")
	}

	if len(scriptBody) > 4 && scriptBody[:4] == "set:" {
		return e.runSet(scriptBody, vars)
	}

	lang = normalizeLang(lang)
	switch lang {
	case "javascript", "js":
		if e.js != nil {
			return e.js.Run(ctx, scriptBody, lang, vars)
		}
		return nil, fmt.Errorf("javascript runtime not configured")
	case "log", "noop", "":
		log.Printf("[script] %s vars=%d", scriptBody, len(vars))
		return map[string]any{"executed": true, "script": scriptBody}, nil
	case "set":
		return e.runSet(scriptBody, vars)
	default:
		log.Printf("[script] unsupported lang=%q, running as log stub", lang)
		return map[string]any{"executed": true}, nil
	}
}

func (e *Executor) runSet(script string, vars map[string]any) (map[string]any, error) {
	rest := script[4:]
	for i, c := range rest {
		if c == '=' {
			key := rest[:i]
			val := rest[i+1:]
			if vars == nil {
				vars = make(map[string]any)
			}
			vars[key] = val
			return map[string]any{key: val}, nil
		}
	}
	return nil, fmt.Errorf("invalid set script: %s", script)
}

func normalizeLang(lang string) string {
	switch lang {
	case "javascript", "js":
		return "javascript"
	case "groovy":
		return "groovy"
	default:
		return lang
	}
}
