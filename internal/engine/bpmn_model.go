package engine

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

var ErrVersionConflict = errors.New("process instance was modified concurrently")

// JobType identifies async work items.
type JobType string

const (
	JobTypeStart    JobType = "start"
	JobTypeContinue JobType = "continue"
)

// JobStatus tracks job lifecycle.
type JobStatus string

const (
	JobStatusPending JobStatus = "pending"
	JobStatusRunning JobStatus = "running"
	JobStatusDone    JobStatus = "done"
	JobStatusFailed  JobStatus = "failed"
)

// Job is an async continuation unit for the worker.
type Job struct {
	ID                uuid.UUID      `json:"id"`
	ProcessInstanceID uuid.UUID      `json:"process_instance_id"`
	Type              JobType        `json:"job_type"`
	Payload           map[string]any `json:"payload,omitempty"`
	Status            JobStatus      `json:"status"`
	Attempts          int            `json:"attempts"`
	ErrorMsg          string         `json:"error_message,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	LockedAt          *time.Time     `json:"locked_at,omitempty"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
}

// UserTask is an active userTask with process context for inbox queries.
type UserTask struct {
	ActivityInstance
	TenantID       string `json:"tenant_id"`
	ProcessKey     string `json:"process_key"`
	BusinessKey    string `json:"business_key,omitempty"`
	ProcessVersion int    `json:"process_version"`
}

// ApprovalRecord captures one assignee action on a multi-sign userTask.
type ApprovalRecord struct {
	Assignee  string         `json:"assignee"`
	Action    string         `json:"action"`
	Comment   string         `json:"comment,omitempty"`
	Variables map[string]any `json:"variables,omitempty"`
	At        time.Time      `json:"at"`
}

// DeployedProcess is a tenant-scoped BPMN process definition version.
type DeployedProcess struct {
	TenantID   string                 `json:"tenant_id"`
	Key        string                 `json:"key"`
	Version    int                    `json:"version"`
	Name       string                 `json:"name"`
	Definition bpmn.ProcessDefinition `json:"definition"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// ProcessInstanceStatus tracks BPMN process instance lifecycle.
type ProcessInstanceStatus string

const (
	ProcessStatusPending   ProcessInstanceStatus = "pending"
	ProcessStatusRunning   ProcessInstanceStatus = "running"
	ProcessStatusCompleted ProcessInstanceStatus = "completed"
	ProcessStatusFailed    ProcessInstanceStatus = "failed"
	ProcessStatusCancelled ProcessInstanceStatus = "cancelled"
)

// ActivityStatus tracks per-element execution state.
type ActivityStatus string

const (
	ActivityStatusActive    ActivityStatus = "active"
	ActivityStatusCompleted ActivityStatus = "completed"
	ActivityStatusFailed    ActivityStatus = "failed"
	ActivityStatusCancelled ActivityStatus = "cancelled"
)

// ProcessInstance is a running BPMN process.
type ProcessInstance struct {
	ID                 uuid.UUID             `json:"id"`
	TenantID           string                `json:"tenant_id"`
	ProcessKey         string                `json:"process_key"`
	ProcessVersion     int                   `json:"process_version"`
	BusinessKey        string                `json:"business_key,omitempty"`
	Status             ProcessInstanceStatus `json:"status"`
	Variables          map[string]any        `json:"variables"`
	ActiveElements     []string              `json:"active_elements,omitempty"`
	ErrorMsg           string                `json:"error_message,omitempty"`
	StartedAt          time.Time             `json:"started_at"`
	EndedAt            *time.Time            `json:"ended_at,omitempty"`
	UpdatedAt          time.Time             `json:"updated_at"`
	LockVersion        int                   `json:"lock_version"`

	DefinitionSnapshot bpmn.ProcessDefinition `json:"-" yaml:"definition_snapshot,omitempty"`
	InternalState      map[string]any         `json:"-" yaml:"internal_state,omitempty"`
}

// ActivityInstance records execution of a BPMN element within a process instance.
type ActivityInstance struct {
	ID                uuid.UUID        `json:"id"`
	ProcessInstanceID uuid.UUID        `json:"process_instance_id"`
	ElementID         string           `json:"element_id"`
	ElementKind       bpmn.ElementKind `json:"element_kind"`
	Status            ActivityStatus   `json:"status"`
	ScopeID           string           `json:"scope_id,omitempty"`
	BranchFlowID      string           `json:"branch_flow_id,omitempty"`
	Outcome           string           `json:"outcome,omitempty"` // approve | reject | cancelled
	Assignees         []string         `json:"assignees,omitempty"`
	ApprovalMode      string           `json:"approval_mode,omitempty"`
	RequiredApprovals int              `json:"required_approvals,omitempty"`
	PendingAssignees  []string         `json:"pending_assignees,omitempty"`
	ApprovalRecords   []ApprovalRecord `json:"approval_records,omitempty"`
	Input             map[string]any   `json:"input,omitempty"`
	Output            map[string]any   `json:"output,omitempty"`
	ErrorMsg          string           `json:"error_message,omitempty"`
	StartedAt         time.Time        `json:"started_at"`
	EndedAt           *time.Time       `json:"ended_at,omitempty"`
}

// CompleteTaskRequest completes a waiting UserTask.
type CompleteTaskRequest struct {
	ProcessInstanceID uuid.UUID
	ActivityID        uuid.UUID
	Assignee          string
	Action            string
	Comment           string
	Variables         map[string]any
	LockVersion       int
}

// TerminateRequest cancels a process instance or a sub-process scope.
type TerminateRequest struct {
	ProcessInstanceID uuid.UUID
	ScopeID           string
	Reason            string
	Operator          string
	LockVersion       int
}

// UpdateAssigneesRequest changes pending assignees on an active userTask.
type UpdateAssigneesRequest struct {
	ProcessInstanceID uuid.UUID
	ActivityID        uuid.UUID
	Assignees         []string
	PendingAssignees  []string
	Operator          string
	LockVersion       int
}
