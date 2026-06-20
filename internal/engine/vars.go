package engine

import "github.com/monoposer/lowcode-bpmn/internal/bpmn"

// assignee resolution helpers (variable paths, string slices).

// ResolveAssigneesRequest carries context for assignee resolution at userTask activation.
type ResolveAssigneesRequest struct {
	TenantID            string
	ProcessKey          string
	ProcessInstanceID   string
	ElementID           string
	Element             bpmn.Element
	Variables           map[string]any
	DefinitionAssignees []string
}

func assigneesFromVariable(vars map[string]any, path string) []string {
	v, ok := resolveVarPath(vars, path)
	if !ok {
		return nil
	}
	return toStringSlice(v)
}

func toStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return append([]string(nil), t...)
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	default:
		return nil
	}
}

func resolveVarPath(vars map[string]any, path string) (any, bool) {
	if path == "" {
		return nil, false
	}
	parts := splitDotPath(path)
	var cur any = vars
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[p]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func splitDotPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

func resolveTaskAssignees(req ResolveAssigneesRequest) (assignees []string, source AssigneeSource) {
	if req.Element.AssigneesVariable != "" {
		if list := assigneesFromVariable(req.Variables, req.Element.AssigneesVariable); len(list) > 0 {
			return list, AssigneeSourceVariable
		}
	}
	if len(req.DefinitionAssignees) > 0 {
		return append([]string(nil), req.DefinitionAssignees...), AssigneeSourceDefinition
	}
	return nil, AssigneeSourceDefinition
}
