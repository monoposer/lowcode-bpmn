package engine

import (
	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	pkgvars "github.com/monoposer/lowcode-bpmn/pkg/vars"
)

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
	v, ok := pkgvars.ResolvePath(vars, path)
	if !ok {
		return nil
	}
	return pkgvars.ToStringSlice(v)
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
