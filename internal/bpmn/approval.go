package bpmn

import "strings"

// ApprovalMode controls multi-assignee userTask completion semantics.
//
//   - any (或签): need requiredApprovals from assignee pool (default 1, GitHub-style review quota)
//   - all (会签): every assignee must approve
//   - sequential (顺签): assignees act in list order
type ApprovalMode string

const (
	ApprovalAny        ApprovalMode = "any"
	ApprovalAll        ApprovalMode = "all"
	ApprovalSequential ApprovalMode = "sequential"
)

// ParseApprovalMode normalizes designer input (English or Chinese aliases).
func ParseApprovalMode(raw string) ApprovalMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "all", "counter", "countersign", "会签":
		return ApprovalAll
	case "sequential", "sequence", "ordered", "顺签":
		return ApprovalSequential
	case "any", "or", "one", "或签", "":
		return ApprovalAny
	default:
		return ApprovalAny
	}
}

func (m ApprovalMode) Valid() bool {
	switch m {
	case ApprovalAny, ApprovalAll, ApprovalSequential:
		return true
	default:
		return false
	}
}
