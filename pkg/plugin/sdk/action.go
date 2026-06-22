package sdk

import (
	"github.com/google/uuid"
)

// Action is the normalized intent adapters produce before calling Host.
// See schemas/adapter-action.schema.json.
type Action struct {
	Kind string `json:"kind"`

	TenantID    string `json:"tenant_id,omitempty"`
	ProcessKey  string `json:"process_key,omitempty"`
	MessageRef  string `json:"message_ref,omitempty"`
	BusinessKey string `json:"business_key,omitempty"`

	UserID    string   `json:"user_id,omitempty"`
	Reason    string   `json:"reason,omitempty"`
	Operator  string   `json:"operator,omitempty"`
	Assignees []string `json:"assignees,omitempty"`

	ProcessInstanceID string         `json:"process_instance_id,omitempty"`
	ActivityID        string         `json:"activity_id,omitempty"`
	Assignee          string         `json:"assignee,omitempty"`
	Action            string         `json:"action,omitempty"`
	Comment           string         `json:"comment,omitempty"`
	LockVersion       int            `json:"lock_version,omitempty"`
	ScopeID           string         `json:"scope_id,omitempty"`
	Variables         map[string]any `json:"variables,omitempty"`
	SelectedFlowID    string         `json:"selected_flow_id,omitempty"`
	BoundaryElementID string         `json:"boundary_element_id,omitempty"`
	HostElementID     string         `json:"host_element_id,omitempty"`
	GatewayElementID  string         `json:"gateway_element_id,omitempty"`
}

func (a Action) InstanceID() (uuid.UUID, error) {
	return uuid.Parse(a.ProcessInstanceID)
}

func (a Action) ActivityUUID() (uuid.UUID, error) {
	return uuid.Parse(a.ActivityID)
}
