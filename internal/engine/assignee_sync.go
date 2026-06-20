package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

// RemoveUserSyncRequest removes a user from all matching active userTasks.
type RemoveUserSyncRequest struct {
	TenantID     string
	UserID       string
	ProcessKeys  []string
	ElementIDs   []string
	Reason       string
	Operator     string
}

// SyncTaskResult describes one activity touched by sync.
type SyncTaskResult struct {
	ActivityID        uuid.UUID `json:"activity_id"`
	ProcessInstanceID uuid.UUID `json:"process_instance_id"`
	ElementID         string    `json:"element_id"`
	ProcessKey        string    `json:"process_key,omitempty"`
	Status            string    `json:"status"`
	Message           string    `json:"message,omitempty"`
	Assignees         []string  `json:"assignees,omitempty"`
}

// RemoveUserSyncResult aggregates a remove-user sync run.
type RemoveUserSyncResult struct {
	TenantID    string           `json:"tenant_id"`
	UserID      string           `json:"user_id"`
	Updated     int              `json:"updated"`
	Skipped     int              `json:"skipped"`
	Failed      int              `json:"failed"`
	Results     []SyncTaskResult `json:"results"`
}

const (
	syncStatusUpdated          = "updated"
	syncStatusSkippedNotMember = "skipped_not_member"
	syncStatusSkippedActed     = "skipped_already_acted"
	syncStatusFailed           = "failed"
)

// ReplaceAssigneesSyncRequest replaces assignees on one active userTask.
type ReplaceAssigneesSyncRequest struct {
	TenantID          string
	ProcessInstanceID uuid.UUID
	ActivityID        uuid.UUID
	Assignees         []string
	PendingAssignees  []string
	Operator          string
	Reason            string
}

func (e *Engine) RemoveUserFromActiveTasks(ctx context.Context, req RemoveUserSyncRequest) (*RemoveUserSyncResult, error) {
	if e.store == nil {
		return nil, fmt.Errorf("engine: store not configured")
	}
	if req.TenantID == "" || req.UserID == "" {
		return nil, fmt.Errorf("tenantId and userId are required")
	}

	tasks, err := e.store.ListActiveUserTasks(ctx, req.TenantID, "")
	if err != nil {
		return nil, err
	}

	result := &RemoveUserSyncResult{
		TenantID: req.TenantID,
		UserID:   req.UserID,
	}

	for _, task := range tasks {
		if !matchesSyncScope(task, req.ProcessKeys, req.ElementIDs) {
			continue
		}
		if !containsString(task.Assignees, req.UserID) && !containsString(taskPendingAssignees(&task.ActivityInstance), req.UserID) {
			result.Skipped++
			result.Results = append(result.Results, SyncTaskResult{
				ActivityID:        task.ID,
				ProcessInstanceID: task.ProcessInstanceID,
				ElementID:         task.ElementID,
				ProcessKey:        task.ProcessKey,
				Status:            syncStatusSkippedNotMember,
			})
			continue
		}
		if assigneeAlreadyActedNonSkipped(&task.ActivityInstance, req.UserID) {
			result.Skipped++
			result.Results = append(result.Results, SyncTaskResult{
				ActivityID:        task.ID,
				ProcessInstanceID: task.ProcessInstanceID,
				ElementID:         task.ElementID,
				ProcessKey:        task.ProcessKey,
				Status:            syncStatusSkippedActed,
				Message:           "user already acted on this task",
			})
			continue
		}

		newAssignees := removeString(task.Assignees, req.UserID)
		newPending := removeString(taskPendingAssignees(&task.ActivityInstance), req.UserID)
		if len(newAssignees) == 0 {
			result.Failed++
			result.Results = append(result.Results, SyncTaskResult{
				ActivityID:        task.ID,
				ProcessInstanceID: task.ProcessInstanceID,
				ElementID:         task.ElementID,
				ProcessKey:        task.ProcessKey,
				Status:            syncStatusFailed,
				Message:           "no assignees left after removal",
			})
			continue
		}

		mode := bpmn.ParseApprovalMode(task.ApprovalMode)
		if mode == bpmn.ApprovalAny && task.RequiredApprovals > len(newAssignees) {
			result.Failed++
			result.Results = append(result.Results, SyncTaskResult{
				ActivityID:        task.ID,
				ProcessInstanceID: task.ProcessInstanceID,
				ElementID:         task.ElementID,
				ProcessKey:        task.ProcessKey,
				Status:            syncStatusFailed,
				Message:           fmt.Sprintf("requiredApprovals (%d) exceeds remaining assignees (%d)", task.RequiredApprovals, len(newAssignees)),
			})
			continue
		}

		act, err := e.applyAssigneeList(ctx, task.ProcessInstanceID, task.ID, newAssignees, newPending)
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, SyncTaskResult{
				ActivityID:        task.ID,
				ProcessInstanceID: task.ProcessInstanceID,
				ElementID:         task.ElementID,
				ProcessKey:        task.ProcessKey,
				Status:            syncStatusFailed,
				Message:           err.Error(),
			})
			continue
		}

		result.Updated++
		result.Results = append(result.Results, SyncTaskResult{
			ActivityID:        act.ID,
			ProcessInstanceID: act.ProcessInstanceID,
			ElementID:         act.ElementID,
			ProcessKey:        task.ProcessKey,
			Status:            syncStatusUpdated,
			Assignees:         act.Assignees,
		})
	}

	return result, nil
}

func (e *Engine) ReplaceTaskAssigneesSync(ctx context.Context, req ReplaceAssigneesSyncRequest) (*ActivityInstance, error) {
	if req.TenantID == "" {
		return nil, fmt.Errorf("tenantId is required")
	}
	inst, err := e.store.GetProcessInstance(ctx, req.ProcessInstanceID)
	if err != nil {
		return nil, err
	}
	if inst == nil || inst.TenantID != req.TenantID {
		return nil, fmt.Errorf("process instance not found")
	}
	return e.applyAssigneeList(ctx, req.ProcessInstanceID, req.ActivityID, req.Assignees, req.PendingAssignees)
}

func (e *Engine) applyAssigneeList(ctx context.Context, instanceID, activityID uuid.UUID, assignees, pending []string) (*ActivityInstance, error) {
	return e.UpdateActivityAssignees(ctx, UpdateAssigneesRequest{
		ProcessInstanceID: instanceID,
		ActivityID:        activityID,
		Assignees:         assignees,
		PendingAssignees:  pending,
	})
}

func matchesSyncScope(task *UserTask, processKeys, elementIDs []string) bool {
	if len(processKeys) > 0 && !containsString(processKeys, task.ProcessKey) {
		return false
	}
	if len(elementIDs) > 0 && !containsString(elementIDs, task.ElementID) {
		return false
	}
	return true
}

func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

func assigneeAlreadyActedNonSkipped(act *ActivityInstance, assignee string) bool {
	for _, r := range act.ApprovalRecords {
		if r.Assignee == assignee && r.Action != "skipped" {
			return true
		}
	}
	return false
}

// AssigneeSource records how assignees were resolved at activation.
type AssigneeSource string

const (
	AssigneeSourceDefinition AssigneeSource = "definition"
	AssigneeSourceVariable   AssigneeSource = "variable"
)

func resolvedAssignees(state *execState, elementID string, el bpmn.Element) []string {
	list, _ := resolveTaskAssignees(ResolveAssigneesRequest{
		TenantID:            state.inst.TenantID,
		ProcessKey:          state.inst.ProcessKey,
		ProcessInstanceID:   state.inst.ID.String(),
		ElementID:           elementID,
		Element:             el,
		Variables:           state.inst.Variables,
		DefinitionAssignees: el.Assignees,
	})
	return list
}

func resolvedAssigneeSource(state *execState, elementID string, el bpmn.Element) AssigneeSource {
	_, source := resolveTaskAssignees(ResolveAssigneesRequest{
		TenantID:            state.inst.TenantID,
		ProcessKey:          state.inst.ProcessKey,
		ProcessInstanceID:   state.inst.ID.String(),
		ElementID:           elementID,
		Element:             el,
		Variables:           state.inst.Variables,
		DefinitionAssignees: el.Assignees,
	})
	return source
}

func stampAssigneeSyncMeta(act *ActivityInstance, source AssigneeSource, operator string) {
	if act.Input == nil {
		act.Input = make(map[string]any)
	}
	act.Input["assignee_source"] = string(source)
	if operator != "" {
		act.Input["assignee_sync_at"] = time.Now().UTC().Format(time.RFC3339)
	}
}
