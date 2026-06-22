package runtime

// InboxTask is an active userTask element with process context for inbox queries.
// Distinct from the BPMN element kind "userTask" (definition.Element).
type InboxTask struct {
	ActivityInstance
	TenantID       string `json:"tenant_id"`
	ProcessKey     string `json:"process_key"`
	BusinessKey    string `json:"business_key,omitempty"`
	ProcessVersion int    `json:"process_version"`
}

// UserTask is a deprecated alias for InboxTask (inbox projection, not BPMN element).
//
// Deprecated: use InboxTask.
type UserTask = InboxTask
