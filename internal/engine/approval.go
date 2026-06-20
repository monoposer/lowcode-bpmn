package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

const (
	ApprovalActionApprove = "approve"
	ApprovalActionReject  = "reject"
)

func initUserTaskApproval(act *ActivityInstance, el bpmn.Element, assignees []string) {
	mode := bpmn.ParseApprovalMode(el.ApprovalMode)
	if len(assignees) == 0 {
		assignees = append([]string(nil), el.Assignees...)
	}
	act.ApprovalMode = string(mode)
	act.Assignees = assignees
	act.RequiredApprovals = bpmn.RequiredApprovals(el, mode)
	act.PendingAssignees = pendingForMode(mode, assignees)
}

func pendingForMode(mode bpmn.ApprovalMode, assignees []string) []string {
	switch mode {
	case bpmn.ApprovalSequential:
		if len(assignees) == 0 {
			return nil
		}
		return []string{assignees[0]}
	default:
		return append([]string(nil), assignees...)
	}
}

func taskPendingAssignees(act *ActivityInstance) []string {
	if act.PendingAssignees != nil {
		return act.PendingAssignees
	}
	return act.Assignees
}

func taskRequiredApprovals(act *ActivityInstance) int {
	if act.RequiredApprovals > 0 {
		return act.RequiredApprovals
	}
	return 1
}

func approvalCount(act *ActivityInstance) int {
	n := 0
	for _, r := range act.ApprovalRecords {
		if r.Action == ApprovalActionApprove {
			n++
		}
	}
	return n
}

func assigneeAlreadyActed(act *ActivityInstance, assignee string) bool {
	for _, r := range act.ApprovalRecords {
		if r.Assignee == assignee {
			return true
		}
	}
	return false
}

func assigneeCanAct(act *ActivityInstance, assignee string) bool {
	if assignee == "" {
		return len(taskPendingAssignees(act)) <= 1
	}
	for _, p := range taskPendingAssignees(act) {
		if p == assignee {
			return true
		}
	}
	return false
}

// TaskVisibleToAssignee reports whether a task should appear in an assignee's inbox.
func TaskVisibleToAssignee(act *ActivityInstance, assignee string) bool {
	if assignee == "" {
		return true
	}
	return assigneeCanAct(act, assignee) && !assigneeAlreadyActed(act, assignee)
}

type approvalStep struct {
	done     bool
	rejected bool
}

func applyApprovalStep(act *ActivityInstance, el bpmn.Element, req CompleteTaskRequest) (approvalStep, error) {
	mode := bpmn.ParseApprovalMode(el.ApprovalMode)
	if el.ApprovalMode != "" {
		act.ApprovalMode = string(mode)
	} else if act.ApprovalMode != "" {
		mode = bpmn.ParseApprovalMode(act.ApprovalMode)
	}
	if act.RequiredApprovals == 0 {
		act.RequiredApprovals = bpmn.RequiredApprovals(el, mode)
	}

	assignee := req.Assignee
	action := req.Action
	if action == "" {
		action = ApprovalActionApprove
	}
	if action != ApprovalActionApprove && action != ApprovalActionReject {
		return approvalStep{}, fmt.Errorf("invalid action %q (use approve or reject)", action)
	}

	if len(act.Assignees) > 1 && assignee == "" {
		return approvalStep{}, errors.New("assignee required for multi-assignee userTask")
	}
	if assignee != "" && !assigneeCanAct(act, assignee) {
		return approvalStep{}, fmt.Errorf("assignee %q is not pending on this task", assignee)
	}
	if assignee != "" && assigneeAlreadyActed(act, assignee) {
		return approvalStep{}, fmt.Errorf("assignee %q already acted on this task", assignee)
	}

	act.ApprovalRecords = append(act.ApprovalRecords, ApprovalRecord{
		Assignee:  assignee,
		Action:    action,
		Comment:   req.Comment,
		Variables: cloneVars(req.Variables),
		At:        time.Now().UTC(),
	})

	if action == ApprovalActionReject {
		return approvalStep{done: true, rejected: true}, nil
	}

	switch mode {
	case bpmn.ApprovalAny:
		act.PendingAssignees = removeString(act.PendingAssignees, assignee)
		required := taskRequiredApprovals(act)
		got := approvalCount(act)
		if got >= required {
			appendSkippedForQuota(act)
			return approvalStep{done: true}, nil
		}
		return approvalStep{done: false}, nil
	case bpmn.ApprovalAll:
		act.PendingAssignees = removeString(act.PendingAssignees, assignee)
		return approvalStep{done: len(act.PendingAssignees) == 0}, nil
	case bpmn.ApprovalSequential:
		idx := indexOfString(el.Assignees, assignee)
		if idx < 0 {
			idx = indexOfString(act.Assignees, assignee)
		}
		if idx+1 >= len(el.Assignees) {
			act.PendingAssignees = nil
			return approvalStep{done: true}, nil
		}
		next := el.Assignees[idx+1]
		act.PendingAssignees = []string{next}
		return approvalStep{done: false}, nil
	default:
		return approvalStep{done: true}, nil
	}
}

func indexOfString(list []string, target string) int {
	for i, v := range list {
		if v == target {
			return i
		}
	}
	return -1
}

func mergeApprovalOutput(act *ActivityInstance) map[string]any {
	out := make(map[string]any)
	for _, rec := range act.ApprovalRecords {
		for k, v := range rec.Variables {
			out[k] = v
		}
	}
	return out
}

func appendSkippedForQuota(act *ActivityInstance) {
	acted := make(map[string]struct{})
	for _, r := range act.ApprovalRecords {
		if r.Assignee != "" {
			acted[r.Assignee] = struct{}{}
		}
	}
	now := time.Now().UTC()
	for _, a := range act.Assignees {
		if _, ok := acted[a]; ok {
			continue
		}
		act.ApprovalRecords = append(act.ApprovalRecords, ApprovalRecord{
			Assignee: a,
			Action:   "skipped",
			Comment:  "approval quota already met",
			At:       now,
		})
	}
}
