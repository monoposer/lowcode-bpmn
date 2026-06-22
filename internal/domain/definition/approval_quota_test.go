package definition

import (
	"testing"
)

func TestRequiredApprovalsDefault(t *testing.T) {
	el := Element{Assignees: []string{"a", "b", "c"}, ApprovalMode: "any"}
	if got := RequiredApprovals(el, ApprovalAny); got != 1 {
		t.Fatalf("default any = 1, got %d", got)
	}
}

func TestRequiredApprovalsExplicit(t *testing.T) {
	el := Element{Assignees: []string{"a", "b", "c"}, ApprovalMode: "any", RequiredApprovals: 2}
	if got := RequiredApprovals(el, ApprovalAny); got != 2 {
		t.Fatalf("got %d", got)
	}
}

func TestValidateUserTaskQuota(t *testing.T) {
	el := Element{
		ID: "r", Kind: KindUserTask,
		Assignees: []string{"a", "b"}, ApprovalMode: "any", RequiredApprovals: 3,
	}
	if err := ValidateUserTaskApproval(el); err == nil {
		t.Fatal("expected validation error")
	}
}
