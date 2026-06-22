package contract

import "github.com/google/uuid"

type TriggerMessageRequest struct {
	TenantID    string
	MessageRef  string
	BusinessKey string
	Variables   map[string]any
}

type StartProcessRequest struct {
	TenantID        string
	ProcessKey      string
	BusinessKey     string
	Variables       map[string]any
	StartElementIDs []string
}

type RemoveUserRequest struct {
	TenantID    string
	UserID      string
	ProcessKeys []string
	ElementIDs  []string
	Reason      string
	Operator    string
}

type ReplaceAssigneesRequest struct {
	TenantID          string
	ProcessInstanceID uuid.UUID
	ActivityID        uuid.UUID
	Assignees         []string
	PendingAssignees  []string
	Operator          string
	Reason            string
}

type CompleteTaskRequest struct {
	ProcessInstanceID uuid.UUID
	ActivityID        uuid.UUID
	Assignee          string
	Action            string
	Comment           string
	Variables         map[string]any
	LockVersion       int
}

type CompleteActivityRequest struct {
	ProcessInstanceID uuid.UUID
	ActivityID        uuid.UUID
	Assignee          string
	Action            string
	Comment           string
	Variables         map[string]any
	LockVersion       int
	SelectedFlowID    string
}

type TriggerBoundaryRequest struct {
	TenantID          string
	ProcessInstanceID uuid.UUID
	HostElementID     string
	BoundaryElementID string
	Variables         map[string]any
}

type TerminateRequest struct {
	ProcessInstanceID uuid.UUID
	ScopeID           string
	Reason            string
	Operator          string
	LockVersion       int
}

// UserTask is a minimal inbox row for plugin task completion by business key.
type UserTask struct {
	ID                uuid.UUID `json:"id"`
	ProcessInstanceID uuid.UUID `json:"process_instance_id"`
	TenantID          string    `json:"tenant_id"`
	BusinessKey       string    `json:"business_key,omitempty"`
}

// ProcessInstance is a minimal instance view for plugin queries.
type ProcessInstance struct {
	ID         uuid.UUID `json:"id"`
	TenantID   string    `json:"tenant_id"`
	ProcessKey string    `json:"process_key"`
	Status     string    `json:"status"`
}

type ActivityInstance struct {
	ID        uuid.UUID `json:"id"`
	ElementID string    `json:"element_id"`
	Status    string    `json:"status"`
}

type DeployedProcess struct {
	TenantID string `json:"tenant_id"`
	Key      string `json:"key"`
	Version  int    `json:"version"`
	Name     string `json:"name"`
}
