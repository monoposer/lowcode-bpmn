package bpmn_test

import (
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

func TestRequiredApprovalsDefault(t *testing.T) {
	el := bpmn.Element{Assignees: []string{"a", "b", "c"}, ApprovalMode: "any"}
	if got := bpmn.RequiredApprovals(el, bpmn.ApprovalAny); got != 1 {
		t.Fatalf("default any = 1, got %d", got)
	}
}

func TestRequiredApprovalsExplicit(t *testing.T) {
	el := bpmn.Element{Assignees: []string{"a", "b", "c"}, ApprovalMode: "any", RequiredApprovals: 2}
	if got := bpmn.RequiredApprovals(el, bpmn.ApprovalAny); got != 2 {
		t.Fatalf("got %d", got)
	}
}

func TestValidateUserTaskQuota(t *testing.T) {
	el := bpmn.Element{
		ID: "r", Kind: bpmn.KindUserTask,
		Assignees: []string{"a", "b"}, ApprovalMode: "any", RequiredApprovals: 3,
	}
	if err := bpmn.ValidateUserTaskApproval(el); err == nil {
		t.Fatal("expected validation error")
	}
}
