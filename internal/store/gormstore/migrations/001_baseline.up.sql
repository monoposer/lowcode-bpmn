-- Baseline BPMN schema (idempotent). Complemented by GORM AutoMigrate for drift.

CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(64) PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bpmn_processes (
    tenant_id VARCHAR(128) NOT NULL,
    process_key VARCHAR(128) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    name VARCHAR(255) NOT NULL,
    definition JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, process_key, version)
);

CREATE TABLE IF NOT EXISTS bpmn_instances (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL,
    process_key VARCHAR(128) NOT NULL,
    process_version INT NOT NULL DEFAULT 1,
    business_key VARCHAR(255),
    status VARCHAR(32) NOT NULL,
    variables JSONB NOT NULL DEFAULT '{}',
    internal_state JSONB NOT NULL DEFAULT '{}',
    active_elements JSONB NOT NULL DEFAULT '[]',
    definition_snapshot JSONB,
    lock_version INT NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bpmn_instances_tenant ON bpmn_instances (tenant_id, process_key);
CREATE INDEX IF NOT EXISTS idx_bpmn_instances_running_bk ON bpmn_instances (tenant_id, process_key, business_key)
    WHERE status = 'running' AND business_key IS NOT NULL AND business_key <> '';

CREATE TABLE IF NOT EXISTS bpmn_activities (
    id UUID PRIMARY KEY,
    process_instance_id UUID NOT NULL,
    element_id VARCHAR(128) NOT NULL,
    element_kind VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    scope_id VARCHAR(128),
    branch_flow_id VARCHAR(128),
    outcome VARCHAR(32),
    assignees JSONB,
    approval_mode VARCHAR(32),
    required_approvals INT,
    pending_assignees JSONB,
    approval_records JSONB,
    input JSONB,
    output JSONB,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_bpmn_activities_instance ON bpmn_activities (process_instance_id);

CREATE TABLE IF NOT EXISTS bpmn_jobs (
    id UUID PRIMARY KEY,
    process_instance_id UUID NOT NULL,
    job_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    locked_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_bpmn_jobs_status_created ON bpmn_jobs (status, created_at);
