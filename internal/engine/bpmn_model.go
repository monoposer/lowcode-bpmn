package engine

import (
	"github.com/monoposer/lowcode-bpmn/internal/domain/ports"
	"github.com/monoposer/lowcode-bpmn/internal/domain/runtime"
)

// Runtime entity aliases (domain/runtime bounded context).
type (
	Job                    = runtime.Job
	JobType                = runtime.JobType
	JobStatus              = runtime.JobStatus
	InboxTask              = runtime.InboxTask
	UserTask               = runtime.UserTask // deprecated: use InboxTask
	ApprovalRecord         = runtime.ApprovalRecord
	DeployedProcess        = runtime.DeployedProcess
	ProcessInstanceStatus  = runtime.ProcessInstanceStatus
	ActivityStatus         = runtime.ActivityStatus
	ProcessInstance        = runtime.ProcessInstance
	ActivityInstance       = runtime.ActivityInstance
	CompleteTaskRequest    = runtime.CompleteTaskRequest
	TerminateRequest       = runtime.TerminateRequest
	UpdateAssigneesRequest = runtime.UpdateAssigneesRequest
)

var ErrVersionConflict = runtime.ErrVersionConflict

const (
	JobTypeStart    = runtime.JobTypeStart
	JobTypeContinue = runtime.JobTypeContinue
	JobStatusPending = runtime.JobStatusPending
	JobStatusRunning = runtime.JobStatusRunning
	JobStatusDone    = runtime.JobStatusDone
	JobStatusFailed  = runtime.JobStatusFailed
	ProcessStatusPending   = runtime.ProcessStatusPending
	ProcessStatusRunning   = runtime.ProcessStatusRunning
	ProcessStatusCompleted = runtime.ProcessStatusCompleted
	ProcessStatusFailed    = runtime.ProcessStatusFailed
	ProcessStatusCancelled = runtime.ProcessStatusCancelled
	ActivityStatusActive    = runtime.ActivityStatusActive
	ActivityStatusCompleted = runtime.ActivityStatusCompleted
	ActivityStatusFailed    = runtime.ActivityStatusFailed
	ActivityStatusCancelled = runtime.ActivityStatusCancelled
)

// Store is the persistence port (domain/ports.ProcessRepository).
type Store = ports.ProcessRepository
