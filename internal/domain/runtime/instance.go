package runtime

import (
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/domain/definition"
)

// ProcessInstanceStatus tracks BPMN process instance lifecycle.
type ProcessInstanceStatus string

const (
	ProcessStatusPending   ProcessInstanceStatus = "pending"
	ProcessStatusRunning   ProcessInstanceStatus = "running"
	ProcessStatusCompleted ProcessInstanceStatus = "completed"
	ProcessStatusFailed    ProcessInstanceStatus = "failed"
	ProcessStatusCancelled ProcessInstanceStatus = "cancelled"
)

// ProcessInstance is the aggregate root for a running BPMN process.
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

	DefinitionSnapshot definition.ProcessDefinition `json:"-" yaml:"definition_snapshot,omitempty"`
	// InternalState holds gateway join tokens and other execution machinery (not API-visible).
	InternalState map[string]any `json:"-" yaml:"internal_state,omitempty"`
}
