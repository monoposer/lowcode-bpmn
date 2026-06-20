package script

import (
	"context"
	"log/slog"
)

// RunRequest is the input for ScriptTask execution.
type RunRequest struct {
	Script     string
	Lang       string
	Variables  map[string]any
	InstanceID string // process instance UUID
	ElementID  string // BPMN element id
	TenantID   string
	ProcessKey string
}

// Runner executes ScriptTask scripts (javascript or log).
type Runner interface {
	Run(ctx context.Context, req RunRequest) (map[string]any, error)
}

// Ensure Executor implements Runner.
var _ Runner = (*Executor)(nil)

func slogAttrs(req RunRequest) []any {
	attrs := make([]any, 0, 4)
	if req.TenantID != "" {
		attrs = append(attrs, slog.String("tenantId", req.TenantID))
	}
	if req.InstanceID != "" {
		attrs = append(attrs, slog.String("processInstanceId", req.InstanceID))
	}
	if req.ElementID != "" {
		attrs = append(attrs, slog.String("elementId", req.ElementID))
	}
	if req.ProcessKey != "" {
		attrs = append(attrs, slog.String("processKey", req.ProcessKey))
	}
	return attrs
}
