package contract

import (
	"context"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

// Host exposes engine read/write capabilities to event adapter plugins.
type Host interface {
	TriggerMessage(ctx context.Context, req engine.TriggerMessageRequest) (*engine.TriggerMessageResult, error)
	StartProcess(ctx context.Context, req engine.StartProcessRequest) (*engine.ProcessInstance, error)
	GetProcessInstance(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error)
	ListActivities(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error)
	ListUserTasks(ctx context.Context, tenantID, assignee string) ([]*engine.UserTask, error)
	ListProcesses(ctx context.Context, tenantID string) ([]*engine.DeployedProcess, error)
	RemoveUserFromActiveTasks(ctx context.Context, req engine.RemoveUserSyncRequest) (*engine.RemoveUserSyncResult, error)
	ReplaceTaskAssignees(ctx context.Context, req engine.ReplaceAssigneesSyncRequest) (*engine.ActivityInstance, error)
	CompleteTask(ctx context.Context, req engine.CompleteTaskRequest) (*engine.ProcessInstance, error)
	Terminate(ctx context.Context, req engine.TerminateRequest) (*engine.ProcessInstance, error)
}
