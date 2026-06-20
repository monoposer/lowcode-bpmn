package plugin

import (
	"context"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/internal/plugin/contract"
)

// Host is the adapter-facing engine SDK (see contract package).
type Host = contract.Host

// EngineHost implements Host by delegating to engine.Engine.
type EngineHost struct {
	Eng *engine.Engine
}

func NewHost(eng *engine.Engine) *EngineHost {
	return &EngineHost{Eng: eng}
}

func (h *EngineHost) TriggerMessage(ctx context.Context, req engine.TriggerMessageRequest) (*engine.TriggerMessageResult, error) {
	return h.Eng.TriggerMessage(ctx, req)
}

func (h *EngineHost) StartProcess(ctx context.Context, req engine.StartProcessRequest) (*engine.ProcessInstance, error) {
	return h.Eng.StartProcess(ctx, req)
}

func (h *EngineHost) GetProcessInstance(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error) {
	return h.Eng.GetProcessInstance(ctx, id)
}

func (h *EngineHost) ListActivities(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error) {
	return h.Eng.ListActivities(ctx, processInstanceID)
}

func (h *EngineHost) ListUserTasks(ctx context.Context, tenantID, assignee string) ([]*engine.UserTask, error) {
	return h.Eng.ListUserTasks(ctx, tenantID, assignee)
}

func (h *EngineHost) ListProcesses(ctx context.Context, tenantID string) ([]*engine.DeployedProcess, error) {
	return h.Eng.ListProcesses(ctx, tenantID)
}

func (h *EngineHost) RemoveUserFromActiveTasks(ctx context.Context, req engine.RemoveUserSyncRequest) (*engine.RemoveUserSyncResult, error) {
	return h.Eng.RemoveUserFromActiveTasks(ctx, req)
}

func (h *EngineHost) ReplaceTaskAssignees(ctx context.Context, req engine.ReplaceAssigneesSyncRequest) (*engine.ActivityInstance, error) {
	return h.Eng.ReplaceTaskAssigneesSync(ctx, req)
}

func (h *EngineHost) CompleteTask(ctx context.Context, req engine.CompleteTaskRequest) (*engine.ProcessInstance, error) {
	return h.Eng.CompleteTask(ctx, req)
}

func (h *EngineHost) Terminate(ctx context.Context, req engine.TerminateRequest) (*engine.ProcessInstance, error) {
	return h.Eng.Terminate(ctx, req)
}
