package runtime

import (
	"time"

	"github.com/google/uuid"
)

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
