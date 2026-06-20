package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

func (e *Engine) UpdateActivityAssignees(ctx context.Context, req UpdateAssigneesRequest) (*ActivityInstance, error) {
	inst, err := e.store.GetProcessInstance(ctx, req.ProcessInstanceID)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, errors.New("process instance not found")
	}
	if inst.Status != ProcessStatusRunning {
		return nil, fmt.Errorf("process not running: %s", inst.Status)
	}
	if req.LockVersion > 0 && req.LockVersion != inst.LockVersion {
		return nil, ErrVersionConflict
	}

	act, err := e.store.GetActivityInstance(ctx, req.ActivityID)
	if err != nil {
		return nil, err
	}
	if act == nil {
		return nil, errors.New("activity instance not found")
	}
	if act.Status != ActivityStatusActive {
		return nil, fmt.Errorf("activity not active: %s", act.Status)
	}
	if act.ElementKind != bpmn.KindUserTask {
		return nil, fmt.Errorf("element is not a userTask: %s", act.ElementKind)
	}

	if len(req.Assignees) == 0 {
		return nil, errors.New("assignees must not be empty")
	}

	acted := make(map[string]struct{})
	for _, r := range act.ApprovalRecords {
		if r.Action != "skipped" && r.Assignee != "" {
			acted[r.Assignee] = struct{}{}
		}
	}
	for _, removed := range act.Assignees {
		found := false
		for _, n := range req.Assignees {
			if n == removed {
				found = true
				break
			}
		}
		if !found {
			if _, was := acted[removed]; was {
				return nil, fmt.Errorf("cannot remove assignee %q who already acted", removed)
			}
		}
	}

	pending := req.PendingAssignees
	if pending == nil {
		mode := bpmn.ParseApprovalMode(act.ApprovalMode)
		pending = pendingForMode(mode, req.Assignees)
		// preserve sequential position when possible
		if mode == bpmn.ApprovalSequential {
			for _, p := range taskPendingAssignees(act) {
				for _, n := range req.Assignees {
					if p == n {
						pending = []string{p}
						break
					}
				}
				break
			}
		}
	}

	act.Assignees = append([]string(nil), req.Assignees...)
	act.PendingAssignees = append([]string(nil), pending...)
	if bpmn.ParseApprovalMode(act.ApprovalMode) == bpmn.ApprovalAny && act.RequiredApprovals > len(req.Assignees) {
		return nil, fmt.Errorf("requiredApprovals (%d) exceeds assignee count (%d)", act.RequiredApprovals, len(req.Assignees))
	}
	inst.UpdatedAt = time.Now().UTC()

	err = e.store.WithTx(ctx, func(tx Store) error {
		if err := tx.UpdateActivityInstance(ctx, act); err != nil {
			return err
		}
		return tx.UpdateProcessInstance(ctx, inst)
	})
	if err != nil {
		return nil, err
	}
	return e.store.GetActivityInstance(ctx, act.ID)
}
