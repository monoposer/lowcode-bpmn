# Architecture

## Overview

**lowcode-bpmn** is a lightweight BPMN 2.0 workflow engine microservice in Go, inspired by [tumbleweed](https://github.com/lzw5399/tumbleweed). It focuses on process execution — user, role, and form concerns stay in external systems.

| Component | Role |
|-----------|------|
| **Process designer** (external) | Produces BPMN JSON definitions |
| **lowcode-bpmn** (this service) | Deploys definitions, runs instances, waits on UserTask, executes ScriptTask |
| **Business apps / IM platforms** | Complete tasks, query inbox, push webhooks via plugin adapters |
| **Plugin adapters** (`plugins/go`, `plugins/wasm`) | Map Feishu / WeCom / Airtable / custom payloads → Host SDK |

The codebase is a **pure BPMN 2.0 engine** (legacy stage/DAG orchestration has been removed).

## System diagram

```mermaid
flowchart TB
    subgraph External["External systems"]
        Designer["Process designer"]
        BizApp["Business applications"]
        Webhooks["Feishu / WeCom / Airtable"]
    end

    subgraph Service["lowcode-bpmn"]
        API["internal/api"]
        Engine["internal/engine"]
        Worker["Job worker"]
        BPMN["internal/bpmn"]
        Script["internal/script"]
        Plugin["internal/plugin"]
        Event["pkg/event"]
        Store["engine.Store"]
        GORM["gormstore"]
        File["filestore"]
        Mem["memory.Store"]
    end

    subgraph Plugins["plugins/ (external)"]
        GoPlug["go/feishu, wecom, airtable, …"]
        WasmPlug["wasm/* + capability manifest"]
        SDK["pkg/plugin/sdk + registry"]
    end

    DB[(PostgreSQL / MySQL / SQLite)]

    Designer -->|BPMN JSON| API
    BizApp -->|Complete task / inbox| API
    Webhooks -->|POST /events/{stream}/{source}| API
    API --> Event
    Event --> Plugin
    Plugin --> GoPlug
    Plugin --> WasmPlug
    GoPlug --> SDK
    Plugin -->|Host SDK| Engine
    API --> Engine
    Worker --> Engine
    Engine --> BPMN
    Engine --> Script
    Engine --> Store
    Store --> GORM
    Store --> File
    Store --> Mem
    GORM --> DB
```

### Startup (`cmd/server/main.go`)

1. Initialize **telemetry** (structured logging + optional OpenTelemetry via `OTEL_ENABLED`)
2. Open **store** via `store.Open` (`STORE_BACKEND=db|file|memory`; DB drivers: postgres, mysql, sqlite)
3. Create `engine.Engine`; optionally enable async execution (`ASYNC_EXECUTION=true`)
4. **Plugin bootstrap** — triple event consumers + Go/WASM adapters (`plugin.BootstrapFromEngine`)
5. Start **four stream consumers** (assignee / trigger / task / control) and background **job worker**
6. Mount Chi HTTP routes, CORS, metrics, graceful shutdown

## BPMN 2.0 model

Supported elements:

| Category | Elements |
|----------|----------|
| **Event** | `startEvent` (+ `eventDefinition`), `endEvent` |
| **Activity** | `userTask`, `scriptTask`, `subProcess` (scoped parallel regions) |
| **Gateway** | `exclusiveGateway`, `parallelGateway`, `inclusiveGateway` |
| **Flow** | `sequenceFlow` (optional condition expression) |

Process definitions are JSON documents (`internal/bpmn`):

```
ProcessDefinition
├── id, name
├── elements[]     — BPMN nodes (type, assignees, script, eventDefinition, …)
└── flows[]        — sequenceFlow (sourceRef, targetRef, condition)
```

JSON protocol and examples: [`schemas/`](./schemas/README.md).

### Task types

- **userTask** — waits for external completion via API or plugin (`assignees` / `assigneesVariable`)
  - **Approval modes**: `any` (或签), `all` (会签), `sequential` (顺序签)
  - **requiredApprovals** — quota within assignee pool (GitHub-style review count for `any`)
  - **Complete** with `{ "assignee", "action": "approve|reject", "comment", "lockVersion" }`
  - **onReject**: `return` (default, rewind to upstream) | `terminateScope`
- **scriptTask** — runs via `internal/script`:
  - `scriptLang: "javascript"` (default) — goja with `vars` / `variables`, `http.get|post|request`, merge `return` + `vars` mutations
  - `scriptLang: "log"` — structured log only
- **subProcess** — marker with `scopeId`, `entryRef`, `exitRef` for scoped parallel work and reject/terminate boundaries

### Gateway behavior

| Type | Fork | Join |
|------|------|------|
| exclusiveGateway | First matching condition; optional `isDefault` flow | N/A |
| parallelGateway | All outgoing flows | Wait for all incoming flows |
| inclusiveGateway | All matching conditions | Same as parallel |

Conditions support simple expressions: `field == value`, `amount >= 1000`, dot paths (`event.fields.status == Pending`), truthy field names (`internal/bpmn/expression.go`).

Join state is stored in `ProcessInstance.InternalState` (persisted in DB, **not exposed via API**).

## Runtime objects

| Object | Description |
|--------|-------------|
| `DeployedProcess` | `(tenant_id, process_key, version)` → `ProcessDefinition` |
| `ProcessInstance` | Running/completed instance with pinned `definition_snapshot`, `variables`, `lock_version`, `business_key` |
| `ActivityInstance` | Per-element audit trail; `scope_id`, `branch_flow_id`, `outcome` (approve/reject/cancelled) |
| `Job` | Async continuation unit (`start` or `continue`) |
| `UserTask` | Inbox query DTO (active userTask + process context) |

Each deploy creates a **new process version**. Instances pin the definition snapshot at start so in-flight runs are unaffected by later deploys.

## Execution flow

1. **Deploy** — validate graph, insert new version row
2. **Start** — manual API, message trigger, or plugin adapter → create instance with pinned snapshot
3. **Traverse** — follow `sequenceFlow`; gateways branch/join per BPMN semantics
4. **UserTask** — instance stays `running`, activity stays `active` until complete or reject handling
5. **ScriptTask** — execute script, merge output into variables, continue
6. **EndEvent** — when no active activities remain, instance → `completed`

### Execution modes

| Mode | Config | Behavior |
|------|--------|----------|
| **Sync** (default) | — | `StartProcess` / `CompleteTask` advance the token to the next wait point inside the HTTP request |
| **Async** | `ASYNC_EXECUTION=true` | HTTP returns quickly; background worker drains `bpmn_jobs` |

Worker poll interval: `WORKER_INTERVAL` (default `500ms`). Job claiming uses row locking for safe multi-replica polling (GORM store).

## Persistence

Store abstraction: `internal/store` → `engine.Store`.

| Backend | Env | Use case |
|---------|-----|----------|
| **db** (default) | `STORE_BACKEND=db`, `DB_DRIVER=postgres\|mysql\|sqlite`, `DATABASE_URL` | Production |
| **file** | `STORE_BACKEND=file`, `STORE_PATH=./data` | Local dev / single-node YAML |
| **memory** | `STORE_BACKEND=memory` | Unit tests |

GORM store (`internal/store/gormstore`) applies schema via **AutoMigrate** on startup.

| Table | Purpose |
|-------|---------|
| `bpmn_processes` | Versioned definitions `(tenant_id, process_key, version)` |
| `bpmn_instances` | Instances: `variables`, `internal_state`, `definition_snapshot`, `lock_version`, `active_elements`, `business_key` |
| `bpmn_activities` | Element-level execution audit (assignees, approval records, outcome) |
| `bpmn_jobs` | Async job queue |

`StartProcess`, `CompleteTask`, and assignee sync paths run inside `Store.WithTx` for atomic writes.

## API

| Route | Purpose |
|-------|---------|
| `GET /healthz` | Health check |
| `GET /metrics` | Prometheus metrics |
| `PUT /api/v1/tenants/{tenantId}/processes/{key}` | Deploy new process version |
| `GET /api/v1/tenants/{tenantId}/processes` | List latest version per process key |
| `DELETE /api/v1/tenants/{tenantId}/processes/{key}` | Delete all versions for a key |
| `POST /api/v1/process-instances` | Start instance (manual / none start) |
| `POST /api/v1/triggers/message` | Message start (direct or after adapter normalization) |
| `GET /api/v1/process-instances/{id}` | Instance status, variables, `lock_version` |
| `GET /api/v1/process-instances/{id}/activities` | Activity audit trail |
| `POST /api/v1/process-instances/{id}/tasks/{activityId}/complete` | Complete UserTask (approve/reject) |
| `POST /api/v1/process-instances/{id}/terminate` | Terminate instance or sub-process scope |
| `PATCH /api/v1/process-instances/{id}/tasks/{activityId}/assignees` | Manual assignee override |
| `GET /api/v1/tasks?tenantId=&assignee=` | UserTask inbox |
| `POST /api/v1/assignee-sync/remove-user` | Remove user from active tasks (HR offboarding) |
| `POST /api/v1/assignee-sync/replace` | Replace assignees on one task |
| `POST /api/v1/events/assignee/{source}` | Plugin ingress — HR / org change stream |
| `POST /api/v1/events/trigger/{source}` | Plugin ingress — process start stream |
| `POST /api/v1/events/task/{source}` | Plugin ingress — approve / reject stream |
| `POST /api/v1/events/control/{source}` | Plugin ingress — terminate / admin control |

All handlers go through `engine.Engine` (no global store singleton).

### Error envelope

```json
{ "error": "human readable message", "code": "machine_code" }
```

Concurrent task completion with a stale `lockVersion` returns **409** with code `version_conflict`.

## Event-driven start (BPMN 2.0)

**Important:** `startEvent.eventDefinition` defines *when* a process may start. **Sequence flow `condition`** is only for routing at gateways — not for webhook filters.

| BPMN 2.0 | JSON `eventDefinition.type` | Entry |
|----------|----------------------------|-------|
| None start | `none` / omitted | `POST /api/v1/process-instances` |
| Message start | `message` + `messageRef` | `POST /api/v1/triggers/message` or plugin → Host |
| Conditional start | `conditional` + `condition` | Evaluated on trigger payload |
| Signal / Timer | `signal` / `timer` | External dispatcher (metadata in definition) |

Example (Airtable row update):

```json
{
  "id": "start",
  "type": "startEvent",
  "eventDefinition": {
    "type": "message",
    "messageRef": "airtable.orders.updated",
    "correlationKey": "event.recordId",
    "condition": "event.fields.status == Pending"
  }
}
```

End-to-end (recommended path):

```
Airtable webhook
  → POST /api/v1/events/trigger/airtable
  → plugins/go/airtable
  → Host.TriggerMessage
  → Engine starts matching message startEvent
```

Direct canonical trigger (bypass adapter):

```
POST /api/v1/triggers/message
{ "tenantId", "messageRef", "variables" }
```

## Event plugins (Paca-style)

Full reference: [docs/plugins.md](./docs/plugins.md), [plugins/README.md](./plugins/README.md).

### Quad streams

Four independent consumers decouple extension concerns:

| Stream | Purpose | Example adapters | Host SDK |
|--------|---------|------------------|----------|
| **assignee** | HR / org change | feishu, wecom, generic | `RemoveUserFromActiveTasks`, `ReplaceTaskAssignees` |
| **trigger** | Process start | airtable, feishu, generic | `TriggerMessage`, `StartProcess` |
| **task** | External approve/reject | feishu, wecom, generic | `CompleteTask` |
| **control** | Terminate / admin | generic, canonical | `Terminate` |

```
External event
  → event.Consumer (memory | redis)
  → plugin.Runtime (per stream)
  → EventAdapter (plugins/go/* or plugins/wasm/*)
  → Host SDK (internal/plugin/host.go)
  → Engine
```

| Layer | Package | Role |
|-------|---------|------|
| Consumer | `pkg/event`, `pkg/event/setup` | Transport abstraction (memory, redis) |
| Host contract | `pkg/plugin/contract` | Plugin-facing Host interface + DTOs |
| Host adapter | `internal/plugin/host.go` | Maps contract DTOs → engine |
| Go plugins | `plugins/go/*`, `plugins/registry` | Native vendor adapters |
| WASM plugins | `plugins/wasm/*` | Sandboxed adapters with `plugin.json` capabilities |
| Shared SDK | `pkg/plugin/sdk` | `Action` schema + `Apply*` helpers |
| Ingress | `POST /api/v1/events/{assignee\|trigger\|task}/{source}` | HTTP → stream publisher |

### Configuration

| Env | Default | Notes |
|-----|---------|-------|
| `EVENT_CONSUMER` | `memory` | `redis` for multi-process ingress |
| `EVENT_REDIS_*_KEY` | `bpmn:events:{stream}` | Per-stream Redis lists |
| `PLUGIN_ASSIGNEE_ADAPTERS` | generic, canonical, feishu, wecom | Registry keys |
| `PLUGIN_TRIGGER_ADAPTERS` | … + airtable | |
| `PLUGIN_TASK_ADAPTERS` | generic, canonical, feishu, wecom | |
| `PLUGIN_CONTROL_ADAPTERS` | generic, canonical | terminate stream |
| `PLUGIN_WASM_DIR` | (optional) | Loads `*/plugin.json` + `plugin.wasm` |
| `PLUGIN_DEFAULT_TENANT` | `demo` | Adapter fallback tenant |

Canonical adapter payload: `schemas/adapter-action.schema.json`.

Assignee resolution at userTask activation: `assigneesVariable` (dot path) → static `assignees`. No HTTP resolver in engine.

### Optimistic locking

Instances carry `lock_version`. Clients may pass `lockVersion` when completing a UserTask:

```json
{ "assignee": "u1", "action": "approve", "comment": "ok", "lockVersion": 3 }
```

## Package layout

```
cmd/server/                 HTTP entrypoint + worker + plugin consumers
internal/api/               Chi routes (router, processes, instances, tasks), metrics, auth
internal/bpmn/              Model, validation, expressions, approval
internal/engine/            Engine, worker, Store interface, trigger/reject
internal/plugin/            Runtime, bootstrap, WASM loader (no vendor code)
internal/script/            Runner (goja JS + set DSL)
internal/store/             Backend factory (db / file / memory)
internal/store/gormstore/   GORM persistence + migrations
internal/store/filestore/   YAML file persistence
internal/store/memory/      In-memory store (tests)
internal/telemetry/         Logging + OpenTelemetry
pkg/env/                    Environment variable helpers
pkg/event/                  Stream model, memory/redis consumers
pkg/plugin/contract/        Host interface + plugin DTOs
pkg/plugin/sdk/             Action + Apply* for Go plugins
pkg/vars/                   Variable path resolution helpers
plugins/registry/           Name → adapter lookup
plugins/go/                 feishu, wecom, airtable, generic, canonical
plugins/wasm/               WASM sandbox plugins
schemas/                    JSON Schema for definitions and adapter actions
```

**Boundary rule:** `internal/` holds engine core only; vendor-specific mapping lives under `plugins/`.

## Design strengths

| Area | Notes |
|------|-------|
| **Single engine** | One BPMN mental model; no legacy orchestration paths |
| **Pluggable ingress** | Quad streams + Go/WASM adapters without engine forks |
| **Capability sandbox** | WASM host imports gated by manifest (Paca-style) |
| **Multi-store** | Postgres / MySQL / SQLite / file / memory via one interface |
| **JSON-first** | Designer-friendly; no XML BPMN required |
| **Rich userTask** | 或签/会签/顺序签, reject return, scope terminate |
| **Versioning** | Deploy increments version; instances pin snapshots |
| **Async option** | Job table + worker for non-blocking start/continue |
| **Observability** | Prometheus `/metrics`; optional OTLP traces |
| **Transactions** | Multi-step engine writes wrapped in `WithTx` |

## Known limitations

| Item | Status |
|------|--------|
| Authentication / authorization | Optional API keys via `API_KEY` / `API_KEYS`; set `AUTH_REQUIRED=true` in production |
| Boundary events / call activity | Not supported |
| Timer start | Metadata only; external scheduler required |
| Script sandbox | goja with basic isolation; harden for untrusted scripts |
| Event consumer | In-memory default is single-process; Redis supported (`EVENT_CONSUMER=redis`) |
| Business-key deduplication | Message trigger dedupes running instances by `(tenant, processKey, businessKey)` |
| WASM host I/O | Read exports enforce `outMaxLen`; returns 413 when buffer too small |
| GORM schema | Versioned SQL baseline on Postgres + AutoMigrate for model drift |

## Production checklist

| Item | Configuration |
|------|----------------|
| API auth | `AUTH_REQUIRED=true`, `API_KEY` or `API_KEYS=tenant:key,...` |
| HA event ingress | `EVENT_CONSUMER=redis`, `EVENT_REDIS_URL`, compose `--profile redis` |
| Idempotent webhooks | Automatic when `businessKey` / `correlationKey` resolves |
| Tracing | `OTEL_ENABLED=true`, compose `--profile otel` |
| Independent plugins | `go.work` + `plugins/go/{feishu,wecom,airtable}/go.mod` |
| DB migrations | Postgres: embedded `gormstore/migrations/*.sql` then AutoMigrate |
| Plugin regression | `plugins/go/*/testdata` contract tests |
| Control plane | `PLUGIN_CONTROL_ADAPTERS`, `POST /events/control/{source}` |

## Future enhancements

- JWT authentication alongside API keys
- Webhooks or SSE for task created / process completed
- Stronger script + WASM sandbox (memory limits, no network)
- Kafka/NATS `event.Consumer` implementation
- Versioned migrations for MySQL / SQLite

## Maturity summary

| Dimension | Assessment |
|-----------|------------|
| **Purpose** | Clear, embeddable BPMN microservice for low-code / IM-integrated platforms |
| **Code quality** | Focused layout; engine + plugin paths covered by unit tests |
| **Production readiness** | Beta — enable `AUTH_REQUIRED`, Redis events, and OTEL for multi-tenant production |
| **Highlights** | Version pinning, quad plugin streams, idempotent triggers, API keys, multi-store, metrics + traces |
