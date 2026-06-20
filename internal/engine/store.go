package engine

import (
	"context"

	"github.com/google/uuid"
)

// Store persists BPMN process definitions and runtime state.
type Store interface {
	WithTx(ctx context.Context, fn func(Store) error) error

	InsertProcessVersion(ctx context.Context, p *DeployedProcess) error
	DeleteProcess(ctx context.Context, tenantID, key string) error
	GetProcess(ctx context.Context, tenantID, key string) (*DeployedProcess, error)
	GetProcessVersion(ctx context.Context, tenantID, key string, version int) (*DeployedProcess, error)
	ListProcesses(ctx context.Context, tenantID string) ([]*DeployedProcess, error)

	CreateProcessInstance(ctx context.Context, inst *ProcessInstance) error
	UpdateProcessInstance(ctx context.Context, inst *ProcessInstance) error
	GetProcessInstance(ctx context.Context, id uuid.UUID) (*ProcessInstance, error)
	GetProcessInstanceForUpdate(ctx context.Context, id uuid.UUID) (*ProcessInstance, error)
	FindRunningInstanceByBusinessKey(ctx context.Context, tenantID, processKey, businessKey string) (*ProcessInstance, error)

	CreateActivityInstance(ctx context.Context, act *ActivityInstance) error
	UpdateActivityInstance(ctx context.Context, act *ActivityInstance) error
	GetActivityInstance(ctx context.Context, id uuid.UUID) (*ActivityInstance, error)
	ListActivitiesByProcess(ctx context.Context, processInstanceID uuid.UUID) ([]*ActivityInstance, error)
	ListActiveActivities(ctx context.Context, processInstanceID uuid.UUID) ([]*ActivityInstance, error)
	ListActiveUserTasks(ctx context.Context, tenantID, assignee string) ([]*UserTask, error)

	EnqueueJob(ctx context.Context, job *Job) error
	ClaimNextJob(ctx context.Context) (*Job, error)
	CompleteJob(ctx context.Context, jobID uuid.UUID) error
	FailJob(ctx context.Context, jobID uuid.UUID, errMsg string) error
}
