package script

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
)

// JSExecutor runs JavaScript ScriptTasks via goja.
type JSExecutor struct{}

func NewJSExecutor() *JSExecutor {
	return &JSExecutor{}
}

func (e *JSExecutor) Run(ctx context.Context, scriptBody, lang string, vars map[string]any) (map[string]any, error) {
	if scriptBody == "" {
		return nil, fmt.Errorf("script is empty")
	}
	if len(scriptBody) > 4 && scriptBody[:4] == "set:" {
		return NewExecutor().runSet(scriptBody, vars)
	}

	vm := goja.New()
	varObj := vm.NewObject()
	for k, v := range vars {
		_ = varObj.Set(k, v)
	}
	_ = vm.Set("vars", varObj)
	_ = vm.Set("variables", varObj)

	val, err := vm.RunString(scriptBody)
	if err != nil {
		return nil, fmt.Errorf("javascript error: %w", err)
	}
	if val == nil || goja.IsUndefined(val) {
		return map[string]any{}, nil
	}
	if exported := val.Export(); exported != nil {
		if m, ok := exported.(map[string]any); ok {
			return m, nil
		}
	}
	return map[string]any{"result": val.Export()}, nil
}
