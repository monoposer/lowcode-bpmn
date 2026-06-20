package script

import (
	"context"
	"fmt"
	"strings"
)

// LangRouter dispatches ScriptTask execution by scriptLang without changing the engine.
// Supports javascript/js (Default), wasm/webassembly (WASM), and log (Default).
type LangRouter struct {
	Default Runner
	WASM    Runner
}

func (r *LangRouter) Run(ctx context.Context, req RunRequest) (map[string]any, error) {
	if r == nil || r.Default == nil {
		return nil, errRunnerNotConfigured
	}
	switch normalizeLangKind(req.Lang) {
	case "wasm", "webassembly":
		if r.WASM == nil {
			return nil, fmt.Errorf("wasm script runner not configured (set SCRIPT_WASM_PATH)")
		}
		return r.WASM.Run(ctx, req)
	default:
		return r.Default.Run(ctx, req)
	}
}

func normalizeLangKind(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "", "javascript", "js":
		return "javascript"
	case "webassembly", "wasm":
		return "wasm"
	case "log":
		return "log"
	default:
		return strings.ToLower(strings.TrimSpace(lang))
	}
}
