package script

import "context"

// Runner executes ScriptTask scripts. Swap implementations for JS/Lua/HTTP delegates.
type Runner interface {
	Run(ctx context.Context, script, lang string, vars map[string]any) (map[string]any, error)
}

// Ensure Executor implements Runner.
var _ Runner = (*Executor)(nil)
