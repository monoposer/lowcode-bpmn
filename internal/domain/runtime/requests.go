package runtime

import "github.com/google/uuid"

// CompleteTaskRequest completes a waiting userTask activity.
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
