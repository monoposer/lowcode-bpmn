package plugin

import (
	"context"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/contract"
)

// Host is the adapter-facing engine SDK (see pkg/plugin/contract).
type Host = contract.Host

// EngineHost implements Host by delegating to engine.Engine.
type EngineHost struct {
	Eng *engine.Engine
}

func NewHost(eng *engine.Engine) *EngineHost {
	return &EngineHost{Eng: eng}
}

func (h *EngineHost) TriggerMessage(ctx context.Context, req contract.TriggerMessageRequest) error {
	_, err := h.Eng.TriggerMessage(ctx, engine.TriggerMessageRequest{
		TenantID:    req.TenantID,
		MessageRef:  req.MessageRef,
		BusinessKey: req.BusinessKey,
		Variables:   req.Variables,
	})
	return err
}

func (h *EngineHost) StartProcess(ctx context.Context, req contract.StartProcessRequest) error {
	_, err := h.Eng.StartProcess(ctx, engine.StartProcessRequest{
		TenantID:        req.TenantID,
		ProcessKey:      req.ProcessKey,
		BusinessKey:     req.BusinessKey,
		Variables:       req.Variables,
		StartElementIDs: req.StartElementIDs,
	})
	return err
}

func (h *EngineHost) GetProcessInstance(ctx context.Context, id uuid.UUID) (*contract.ProcessInstance, error) {
	inst, err := h.Eng.GetProcessInstance(ctx, id)
	if err != nil || inst == nil {
		return nil, err
	}
	return &contract.ProcessInstance{
		ID:         inst.ID,
		TenantID:   inst.TenantID,
		ProcessKey: inst.ProcessKey,
		Status:     string(inst.Status),
	}, nil
}

func (h *EngineHost) ListActivities(ctx context.Context, processInstanceID uuid.UUID) ([]contract.ActivityInstance, error) {
	acts, err := h.Eng.ListActivities(ctx, processInstanceID)
	if err != nil {
		return nil, err
	}
	out := make([]contract.ActivityInstance, len(acts))
	for i, a := range acts {
		out[i] = contract.ActivityInstance{
			ID:        a.ID,
			ElementID: a.ElementID,
			Status:    string(a.Status),
		}
	}
	return out, nil
}

func (h *EngineHost) ListUserTasks(ctx context.Context, tenantID, assignee string) ([]contract.UserTask, error) {
	tasks, err := h.Eng.ListUserTasks(ctx, tenantID, assignee)
	if err != nil {
		return nil, err
	}
	out := make([]contract.UserTask, len(tasks))
	for i, t := range tasks {
		out[i] = contract.UserTask{
			ID:                t.ID,
			ProcessInstanceID: t.ProcessInstanceID,
			TenantID:          t.TenantID,
			BusinessKey:       t.BusinessKey,
		}
	}
	return out, nil
}

func (h *EngineHost) ListProcesses(ctx context.Context, tenantID string) ([]contract.DeployedProcess, error) {
	procs, err := h.Eng.ListProcesses(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]contract.DeployedProcess, len(procs))
	for i, p := range procs {
		out[i] = contract.DeployedProcess{
			TenantID: p.TenantID,
			Key:      p.Key,
			Version:  p.Version,
			Name:     p.Name,
		}
	}
	return out, nil
}

func (h *EngineHost) RemoveUserFromActiveTasks(ctx context.Context, req contract.RemoveUserRequest) error {
	_, err := h.Eng.RemoveUserFromActiveTasks(ctx, engine.RemoveUserSyncRequest{
		TenantID:    req.TenantID,
		UserID:      req.UserID,
		ProcessKeys: req.ProcessKeys,
		ElementIDs:  req.ElementIDs,
		Reason:      req.Reason,
		Operator:    req.Operator,
	})
	return err
}

func (h *EngineHost) ReplaceTaskAssignees(ctx context.Context, req contract.ReplaceAssigneesRequest) error {
	_, err := h.Eng.ReplaceTaskAssigneesSync(ctx, engine.ReplaceAssigneesSyncRequest{
		TenantID:          req.TenantID,
		ProcessInstanceID: req.ProcessInstanceID,
		ActivityID:        req.ActivityID,
		Assignees:         req.Assignees,
		PendingAssignees:  req.PendingAssignees,
		Operator:          req.Operator,
		Reason:            req.Reason,
	})
	return err
}

func (h *EngineHost) CompleteTask(ctx context.Context, req contract.CompleteTaskRequest) error {
	_, err := h.Eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: req.ProcessInstanceID,
		ActivityID:        req.ActivityID,
		Assignee:          req.Assignee,
		Action:            req.Action,
		Comment:           req.Comment,
		Variables:         req.Variables,
		LockVersion:       req.LockVersion,
	})
	return err
}

func (h *EngineHost) CompleteActivity(ctx context.Context, req contract.CompleteActivityRequest) error {
	_, err := h.Eng.CompleteActivity(ctx, engine.CompleteActivityRequest{
		ProcessInstanceID: req.ProcessInstanceID,
		ActivityID:        req.ActivityID,
		Assignee:          req.Assignee,
		Action:            req.Action,
		Comment:           req.Comment,
		Variables:         req.Variables,
		LockVersion:       req.LockVersion,
		SelectedFlowID:    req.SelectedFlowID,
	})
	return err
}

func (h *EngineHost) TriggerBoundary(ctx context.Context, req contract.TriggerBoundaryRequest) error {
	_, err := h.Eng.TriggerBoundary(ctx, engine.TriggerBoundaryRequest{
		TenantID:          req.TenantID,
		ProcessInstanceID: req.ProcessInstanceID,
		HostElementID:     req.HostElementID,
		BoundaryElementID: req.BoundaryElementID,
		Variables:         req.Variables,
	})
	return err
}

func (h *EngineHost) EvaluateComplexGateway(ctx context.Context, processInstanceID uuid.UUID, gatewayElementID string) ([]string, error) {
	return h.Eng.EvaluateComplexGateway(ctx, processInstanceID, gatewayElementID)
}

func (h *EngineHost) Terminate(ctx context.Context, req contract.TerminateRequest) error {
	_, err := h.Eng.Terminate(ctx, engine.TerminateRequest{
		ProcessInstanceID: req.ProcessInstanceID,
		ScopeID:           req.ScopeID,
		Reason:            req.Reason,
		Operator:          req.Operator,
		LockVersion:       req.LockVersion,
	})
	return err
}
