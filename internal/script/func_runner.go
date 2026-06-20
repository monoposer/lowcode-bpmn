package script

import "context"

// FuncRunner adapts a function to the Runner interface.
type FuncRunner func(ctx context.Context, req RunRequest) (map[string]any, error)

func (f FuncRunner) Run(ctx context.Context, req RunRequest) (map[string]any, error) {
	if f == nil {
		return nil, errRunnerNotConfigured
	}
	return f(ctx, req)
}
