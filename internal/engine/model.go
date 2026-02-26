package engine

import (
	"time"

	"github.com/google/uuid"
)

// Workspace represents a tenant.
type Workspace struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type FlowStatus string

const (
	FlowStatusDraft     FlowStatus = "draft"
	FlowStatusPublished FlowStatus = "published"
	FlowStatusArchived  FlowStatus = "archived"
)

type Flow struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Status      FlowStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Definition is the current workflow graph definition attached to this flow.
	// Versioning can be added later via FlowVersion if needed.
	Definition *FlowDefinition `json:"definition,omitempty"`
}

// TriggerType lists supported trigger types.
type TriggerType string

const (
	TriggerTypeTimer   TriggerType = "timer"
	TriggerTypeEvent   TriggerType = "event"
	TriggerTypeWebhook TriggerType = "webhook"
	TriggerTypeManual  TriggerType = "manual"
)

// TimerTriggerConfig describes a simple interval-based timer trigger.
type TimerTriggerConfig struct {
	// IntervalSeconds defines how often the trigger should fire.
	IntervalSeconds int `json:"interval_seconds"`
}

// EventTriggerConfig describes an event-based trigger.
type EventTriggerConfig struct {
	// EventType is the routing key, e.g. "user.created".
	EventType string `json:"event_type"`
}

// FlowVersion stores the concrete JSON/YAML-like definition for a flow.
type FlowVersion struct {
	ID          uuid.UUID       `json:"id"`
	FlowID      uuid.UUID       `json:"flow_id"`
	Version     int             `json:"version"`
	Definition  FlowDefinition  `json:"definition"`
	Triggers    []Trigger       `json:"triggers"`
	CreatedAt   time.Time       `json:"created_at"`
	CreatedBy   uuid.UUID       `json:"created_by"`
	PublishedAt *time.Time      `json:"published_at,omitempty"`
	Meta        map[string]any  `json:"meta,omitempty"`
}

// FlowDefinition is the structured in-memory form of flow JSON.
type FlowDefinition struct {
	Nodes      []Node      `json:"nodes"`
	Edges      []Edge      `json:"edges"`
	EntryNodes []string    `json:"entry_nodes"` // usually single trigger node
	Settings   FlowRuntime `json:"settings"`
}

// FlowRuntime holds execution-related defaults.
type FlowRuntime struct {
	// Reserved for global runtime options (timeouts, concurrency limits, etc.).
	MaxConcurrentRuns int `json:"max_concurrent_runs,omitempty"`
}

// Trigger describes one trigger bound to a flow version.
type Trigger struct {
	ID          uuid.UUID           `json:"id"`
	WorkspaceID uuid.UUID           `json:"workspace_id"`
	FlowID      uuid.UUID           `json:"flow_id"`
	Type        TriggerType         `json:"type"`
	Timer       *TimerTriggerConfig `json:"timer,omitempty"`
	Event       *EventTriggerConfig `json:"event,omitempty"`
	Enabled     bool                `json:"enabled"`
	LastFiredAt *time.Time          `json:"last_fired_at,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
}

// NodeType describes what kind of node this is.
type NodeType string

const (
	NodeTypeTrigger   NodeType = "trigger"
	NodeTypeAction    NodeType = "action"
	NodeTypeCondition NodeType = "condition"
	NodeTypeTransform NodeType = "transform"
	NodeTypeLoop      NodeType = "loop"
	NodeTypeSubflow   NodeType = "subflow"
)

// Node is a single step in the flow.
type Node struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        NodeType          `json:"type"`
	Adapter     string            `json:"adapter"` // e.g. "http", "log", "email"
	Config      map[string]any    `json:"config"`  // adapter-specific configuration
	Conditions  []BranchCondition `json:"conditions,omitempty"`
	RetryPolicy *RetryPolicy      `json:"retry_policy,omitempty"`
	Meta        map[string]any    `json:"meta,omitempty"`
}

// Edge connects two nodes by ID.
type Edge struct {
	FromNodeID string `json:"from_node_id"`
	ToNodeID   string `json:"to_node_id"`
	// Optional label, e.g. branch name from a condition node.
	Label string `json:"label,omitempty"`
}

// BranchCondition is used by condition nodes.
type BranchCondition struct {
	// Name is an optional branch label, like "true", "false", or business label.
	Name string `json:"name,omitempty"`
	// Expression is evaluated against flow context, returns bool.
	Expression string `json:"expression"`
	// TargetNodeID is the ID of the node that will be activated when expression is true.
	TargetNodeID string `json:"target_node_id"`
}

// RetryPolicy defines retries for a node execution.
type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	Backoff     time.Duration `json:"backoff"` // simple fixed backoff for now
}

// RunStatus captures the lifecycle of a flow run.
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusSucceeded RunStatus = "succeeded"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
	RunStatusTimeout   RunStatus = "timeout"
)

// Run represents one execution instance of a flow version.
type Run struct {
	ID            uuid.UUID `json:"id"`
	FlowID        uuid.UUID `json:"flow_id"`
	FlowVersionID uuid.UUID `json:"flow_version_id"`
	WorkspaceID   uuid.UUID `json:"workspace_id"`

	Status RunStatus `json:"status"`

	TriggerType TriggerType  `json:"trigger_type"`
	TriggerID   *uuid.UUID   `json:"trigger_id,omitempty"`
	Input       map[string]any `json:"input"`

	ErrorMessage string     `json:"error_message,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// NodeExecution is the per-node execution record.
type NodeExecution struct {
	ID     uuid.UUID `json:"id"`
	RunID  uuid.UUID `json:"run_id"`
	NodeID string    `json:"node_id"`

	Status       RunStatus        `json:"status"`
	Input        map[string]any   `json:"input"`
	Output       map[string]any   `json:"output"`
	ErrorMessage string           `json:"error_message,omitempty"`
	RetryCount   int              `json:"retry_count"`
	StartedAt    *time.Time       `json:"started_at,omitempty"`
	FinishedAt   *time.Time       `json:"finished_at,omitempty"`
	Meta         map[string]any   `json:"meta,omitempty"`
}

