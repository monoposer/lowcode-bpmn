package gormstore

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type BpmnProcess struct {
	TenantID   string         `gorm:"column:tenant_id;primaryKey"`
	ProcessKey string         `gorm:"column:process_key;primaryKey"`
	Version    int            `gorm:"primaryKey;default:1"`
	Name       string         `gorm:"not null"`
	Definition datatypes.JSON `gorm:"not null"`
	CreatedAt  time.Time      `gorm:"not null"`
	UpdatedAt  time.Time      `gorm:"not null"`
}

func (BpmnProcess) TableName() string { return "bpmn_processes" }

type BpmnInstance struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TenantID           string         `gorm:"not null;index:idx_bpmn_instances_tenant,priority:1"`
	ProcessKey         string         `gorm:"not null;index:idx_bpmn_instances_tenant,priority:2"`
	ProcessVersion     int            `gorm:"not null;default:1"`
	BusinessKey        string
	Status             string         `gorm:"not null"`
	Variables          datatypes.JSON `gorm:"not null;default:'{}'"`
	InternalState      datatypes.JSON `gorm:"not null;default:'{}'"`
	ActiveElements     datatypes.JSON `gorm:"not null;default:'[]'"`
	DefinitionSnapshot datatypes.JSON
	LockVersion        int            `gorm:"not null;default:0"`
	ErrorMessage       string         `gorm:"column:error_message"`
	StartedAt          time.Time      `gorm:"not null"`
	EndedAt            *time.Time
	UpdatedAt          time.Time      `gorm:"not null"`
}

func (BpmnInstance) TableName() string { return "bpmn_instances" }

type BpmnActivity struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey"`
	ProcessInstanceID uuid.UUID      `gorm:"type:uuid;not null;index:idx_bpmn_activities_instance"`
	ElementID         string         `gorm:"not null"`
	ElementKind       string         `gorm:"not null"`
	Status            string         `gorm:"not null"`
	ScopeID           string         `gorm:"column:scope_id"`
	BranchFlowID      string         `gorm:"column:branch_flow_id"`
	Outcome           string         `gorm:"column:outcome"`
	Assignees         datatypes.JSON
	ApprovalMode      string         `gorm:"column:approval_mode"`
	RequiredApprovals int            `gorm:"column:required_approvals"`
	PendingAssignees  datatypes.JSON `gorm:"column:pending_assignees"`
	ApprovalRecords   datatypes.JSON `gorm:"column:approval_records"`
	Input             datatypes.JSON
	Output            datatypes.JSON
	ErrorMessage      string         `gorm:"column:error_message"`
	StartedAt         time.Time      `gorm:"not null"`
	EndedAt           *time.Time
}

func (BpmnActivity) TableName() string { return "bpmn_activities" }

type BpmnJob struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey"`
	ProcessInstanceID uuid.UUID      `gorm:"type:uuid;not null"`
	JobType           string         `gorm:"column:job_type;not null"`
	Payload           datatypes.JSON `gorm:"not null;default:'{}'"`
	Status            string         `gorm:"not null;default:pending;index:idx_bpmn_jobs_status_created,priority:1"`
	Attempts          int            `gorm:"not null;default:0"`
	ErrorMessage      string         `gorm:"column:error_message"`
	CreatedAt         time.Time      `gorm:"not null;index:idx_bpmn_jobs_status_created,priority:2"`
	LockedAt          *time.Time
	CompletedAt       *time.Time
}

func (BpmnJob) TableName() string { return "bpmn_jobs" }
