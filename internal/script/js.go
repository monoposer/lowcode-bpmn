package script

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/dop251/goja"

	"github.com/monoposer/lowcode-bpmn/pkg/env"
)

// JSExecutor runs JavaScript ScriptTasks via goja.
type JSExecutor struct {
	client  *http.Client
	timeout time.Duration
	maxBody int64
}

// NewJSExecutor creates a JS runner. Pass nil client to use defaults.
func NewJSExecutor(client *http.Client) *JSExecutor {
	timeout := defaultHTTPTimeout
	if ms := env.Get("SCRIPT_TIMEOUT", ""); ms != "" {
		if d, err := time.ParseDuration(ms); err == nil && d > 0 {
			timeout = d
		}
	}
	maxBody := defaultHTTPMaxBody
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &JSExecutor{client: client, timeout: timeout, maxBody: int64(maxBody)}
}

func (e *JSExecutor) Run(ctx context.Context, req RunRequest) (map[string]any, error) {
	if req.Script == "" {
		return nil, fmt.Errorf("script is empty")
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if e.timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	vm := goja.New()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-runCtx.Done():
			vm.Interrupt(interruptReason(runCtx.Err()))
		case <-done:
		}
	}()

	varObj := vm.NewObject()
	for k, v := range req.Variables {
		_ = varObj.Set(k, v)
	}
	_ = vm.Set("vars", varObj)
	_ = vm.Set("variables", varObj)

	if err := bindHTTP(vm, runCtx, e.client, e.maxBody); err != nil {
		return nil, fmt.Errorf("javascript setup: %w", err)
	}

	val, err := vm.RunString(wrapScript(req.Script))
	if err != nil {
		attrs := append(slogAttrs(req), slog.String("error", err.Error()))
		slog.WarnContext(ctx, "javascript execution failed", attrs...)
		return nil, fmt.Errorf("javascript error: %w", err)
	}

	out := exportObjectMap(varObj)
	if val != nil && !goja.IsUndefined(val) {
		if exported := val.Export(); exported != nil {
			switch t := exported.(type) {
			case map[string]any:
				out = mergeMaps(out, t)
			default:
				out = mergeMaps(out, map[string]any{"result": exported})
			}
		}
	}
	return out, nil
}

func interruptReason(err error) string {
	if err == nil {
		return "script cancelled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "script timeout exceeded"
	}
	if errors.Is(err, context.Canceled) {
		return "script cancelled"
	}
	return err.Error()
}

func wrapScript(body string) string {
	return "(function() {\n" + body + "\n})();"
}

func exportObjectMap(o *goja.Object) map[string]any {
	if o == nil {
		return nil
	}
	exp := o.Export()
	m, ok := exp.(map[string]any)
	if !ok {
		return nil
	}
	return mergeMaps(nil, m)
}

func mergeMaps(base, extra map[string]any) map[string]any {
	if len(extra) == 0 {
		if base == nil {
			return map[string]any{}
		}
		return base
	}
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
