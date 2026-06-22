package runtime

import (
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/domain/definition"
)

// ActivityStatus tracks per-element execution state.
type ActivityStatus string

const (
	ActivityStatusActive    ActivityStatus = "active"
	ActivityStatusCompleted ActivityStatus = "completed"
	ActivityStatusFailed    ActivityStatus = "failed"
	ActivityStatusCancelled ActivityStatus = "cancelled"
)

// ApprovalRecord captures one assignee action on a multi-sign userTask element.
type ApprovalRecord struct {
	Assignee  string         `json:"assignee"`
	Action    string         `json:"action"`
	Comment   string         `json:"comment,omitempty"`
	Variables map[string]any `json:"variables,omitempty"`
	At        time.Time      `json:"at"`
}

// ActivityInstance records execution of a BPMN element within a process instance.
type ActivityInstance struct {
	ID                uuid.UUID           `json:"id"`
	ProcessInstanceID uuid.UUID           `json:"process_instance_id"`
	ElementID         string              `json:"element_id"`
	ElementKind       definition.ElementKind `json:"element_kind"`
	Status            ActivityStatus      `json:"status"`
	ScopeID           string              `json:"scope_id,omitempty"`
	BranchFlowID      string              `json:"branch_flow_id,omitempty"`
	Outcome           string              `json:"outcome,omitempty"` // approve | reject | cancelled
	Assignees         []string            `json:"assignees,omitempty"`
	ApprovalMode      string              `json:"approval_mode,omitempty"`
	RequiredApprovals int                 `json:"required_approvals,omitempty"`
	PendingAssignees  []string            `json:"pending_assignees,omitempty"`
	ApprovalRecords   []ApprovalRecord    `json:"approval_records,omitempty"`
	Input             map[string]any      `json:"input,omitempty"`
	Output            map[string]any      `json:"output,omitempty"`
	ErrorMsg          string              `json:"error_message,omitempty"`
	StartedAt         time.Time           `json:"started_at"`
	EndedAt           *time.Time          `json:"ended_at,omitempty"`
}
