package contract

import (
	"context"

	"github.com/google/uuid"
)

// Host exposes engine capabilities to plugins without importing internal/engine.
type Host interface {
	TriggerMessage(ctx context.Context, req TriggerMessageRequest) error
	StartProcess(ctx context.Context, req StartProcessRequest) error
	GetProcessInstance(ctx context.Context, id uuid.UUID) (*ProcessInstance, error)
	ListActivities(ctx context.Context, processInstanceID uuid.UUID) ([]ActivityInstance, error)
	ListUserTasks(ctx context.Context, tenantID, assignee string) ([]UserTask, error)
	ListProcesses(ctx context.Context, tenantID string) ([]DeployedProcess, error)
	RemoveUserFromActiveTasks(ctx context.Context, req RemoveUserRequest) error
	ReplaceTaskAssignees(ctx context.Context, req ReplaceAssigneesRequest) error
	CompleteTask(ctx context.Context, req CompleteTaskRequest) error
	Terminate(ctx context.Context, req TerminateRequest) error
}
