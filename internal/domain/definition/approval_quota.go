package definition

import "fmt"

// RequiredApprovals resolves how many approve actions are needed to complete a userTask.
// GitHub-style: pool = assignees, quota = requiredApprovals (default 1 for any/or-sign).
func RequiredApprovals(el Element, mode ApprovalMode) int {
	if el.RequiredApprovals > 0 {
		return el.RequiredApprovals
	}
	switch mode {
	case ApprovalAll:
		if n := len(el.Assignees); n > 0 {
			return n
		}
		return 1
	case ApprovalAny:
		return 1
	default:
		return 1
}
}

// ValidateUserTaskApproval checks assignee / quota configuration.
func ValidateUserTaskApproval(el Element) error {
	if el.Kind != KindUserTask || el.AutoComplete {
		return nil
	}
	mode := ParseApprovalMode(el.ApprovalMode)
	n := len(el.Assignees)
	if n == 0 {
		return nil
	}
	req := RequiredApprovals(el, mode)
	if req < 1 {
		return fmt.Errorf("userTask %s: requiredApprovals must be at least 1", el.ID)
	}
	if req > n {
		return fmt.Errorf("userTask %s: requiredApprovals (%d) exceeds assignee count (%d)", el.ID, req, n)
	}
	if mode == ApprovalSequential && req != 1 {
		return fmt.Errorf("userTask %s: sequential mode does not support requiredApprovals > 1", el.ID)
	}
	return nil
}
