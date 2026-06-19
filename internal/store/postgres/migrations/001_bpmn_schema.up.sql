CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Remove legacy tables from earlier project iterations (no-op if absent).
DROP TABLE IF EXISTS stage_tasks;
DROP TABLE IF EXISTS flow_instances;
DROP TABLE IF EXISTS flow_specs;
DROP TABLE IF EXISTS node_executions;
DROP TABLE IF EXISTS runs;
DROP TABLE IF EXISTS flows;

CREATE TABLE IF NOT EXISTS bpmn_processes (
    tenant_id TEXT NOT NULL,
    process_key TEXT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    name TEXT NOT NULL,
    definition JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, process_key, version)
);

CREATE TABLE IF NOT EXISTS bpmn_instances (
    id UUID PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    process_key TEXT NOT NULL,
    process_version INT NOT NULL DEFAULT 1,
    business_key TEXT,
    status TEXT NOT NULL,
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

CREATE TABLE IF NOT EXISTS bpmn_activities (
    id UUID PRIMARY KEY,
    process_instance_id UUID NOT NULL REFERENCES bpmn_instances(id),
    element_id TEXT NOT NULL,
    element_kind TEXT NOT NULL,
    status TEXT NOT NULL,
    assignees JSONB,
    input JSONB,
    output JSONB,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS bpmn_jobs (
    id UUID PRIMARY KEY,
    process_instance_id UUID NOT NULL REFERENCES bpmn_instances(id),
    job_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_bpmn_instances_tenant ON bpmn_instances (tenant_id, process_key);
CREATE INDEX IF NOT EXISTS idx_bpmn_activities_instance ON bpmn_activities (process_instance_id);
CREATE INDEX IF NOT EXISTS idx_bpmn_jobs_pending ON bpmn_jobs (status, created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_bpmn_activities_user_task ON bpmn_activities (process_instance_id, status) WHERE element_kind = 'userTask' AND status = 'active';
