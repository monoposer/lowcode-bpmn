package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/domain/runtime"
)

// ProcessRepository persists process definitions and runtime state (outbound port).
type ProcessRepository interface {
	WithTx(ctx context.Context, fn func(ProcessRepository) error) error

	InsertProcessVersion(ctx context.Context, p *runtime.DeployedProcess) error
	DeleteProcess(ctx context.Context, tenantID, key string) error
	GetProcess(ctx context.Context, tenantID, key string) (*runtime.DeployedProcess, error)
	GetProcessVersion(ctx context.Context, tenantID, key string, version int) (*runtime.DeployedProcess, error)
	ListProcesses(ctx context.Context, tenantID string) ([]*runtime.DeployedProcess, error)

	CreateProcessInstance(ctx context.Context, inst *runtime.ProcessInstance) error
	UpdateProcessInstance(ctx context.Context, inst *runtime.ProcessInstance) error
	GetProcessInstance(ctx context.Context, id uuid.UUID) (*runtime.ProcessInstance, error)
	GetProcessInstanceForUpdate(ctx context.Context, id uuid.UUID) (*runtime.ProcessInstance, error)
	FindRunningInstanceByBusinessKey(ctx context.Context, tenantID, processKey, businessKey string) (*runtime.ProcessInstance, error)
	ListRunningInstances(ctx context.Context, tenantID string) ([]*runtime.ProcessInstance, error)

	CreateActivityInstance(ctx context.Context, act *runtime.ActivityInstance) error
	UpdateActivityInstance(ctx context.Context, act *runtime.ActivityInstance) error
	GetActivityInstance(ctx context.Context, id uuid.UUID) (*runtime.ActivityInstance, error)
	ListActivitiesByProcess(ctx context.Context, processInstanceID uuid.UUID) ([]*runtime.ActivityInstance, error)
	ListActiveActivities(ctx context.Context, processInstanceID uuid.UUID) ([]*runtime.ActivityInstance, error)
	ListActiveUserTasks(ctx context.Context, tenantID, assignee string) ([]*runtime.InboxTask, error)

	EnqueueJob(ctx context.Context, job *runtime.Job) error
	ClaimNextJob(ctx context.Context) (*runtime.Job, error)
	CompleteJob(ctx context.Context, jobID uuid.UUID) error
	FailJob(ctx context.Context, jobID uuid.UUID, errMsg string) error
}
