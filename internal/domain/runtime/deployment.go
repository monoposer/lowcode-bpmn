package runtime

import (
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/domain/definition"
)

// DeployedProcess is a tenant-scoped BPMN process definition version.
type DeployedProcess struct {
	TenantID   string                    `json:"tenant_id"`
	Key        string                    `json:"key"`
	Version    int                       `json:"version"`
	Name       string                    `json:"name"`
	Definition definition.ProcessDefinition `json:"definition"`
	CreatedAt  time.Time                 `json:"created_at"`
	UpdatedAt  time.Time                 `json:"updated_at"`
}
