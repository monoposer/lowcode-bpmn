# AGENTS.md — Cursor / AI agent guide

Guide for agents working in **lowcode-bpmn**: a lightweight **BPMN 2.0** workflow engine in Go (`github.com/monoposer/lowcode-bpmn`).

Human docs: [README.md](./README.md), [ARCHITECTURE.md](./ARCHITECTURE.md). Prefer this file for day-to-day agent context; update ARCHITECTURE when making structural changes.

## What this repo is

- **In scope**: Deploy JSON process definitions, start instances, traverse gateways, wait on `userTask`, run `scriptTask`, task inbox API, async worker, persistence.
- **Out of scope**: User/role management, form designer, Chinese OA patterns (会签/或签/顺签), subprocesses, boundary events, BPMN XML import.
- **No legacy code**: Stage/DAG orchestrator was removed. Do not reintroduce it.

External systems own auth, forms, and approval UX. This service executes BPMN and exposes HTTP APIs.

## Repository layout

```
cmd/server/              HTTP entrypoint, CORS, worker startup
internal/api/            Chi routes, metrics, JSON error envelope
internal/bpmn/           JSON model, validate, registry, conditions
internal/engine/         Engine, worker, engine.Store interface, domain types
internal/script/         ScriptTask runner (goja JS + log; http.* in JS)
internal/store/
  open.go                STORE_BACKEND factory (db | file | memory)
  gormstore/             Postgres / MySQL / SQLite via GORM
  filestore/             YAML snapshot at {STORE_PATH}/state.yaml
  memory/                In-memory store for unit tests
internal/telemetry/      slog, OTel traces, HTTP middleware
deploy/docker/           Dockerfile, entrypoint (volume permissions), compose
```

Demo UI (separate path): `../examples/bpmn/client` — Vite + React playground; not in this repo.

## Environment

| Variable | Default | Notes |
|----------|---------|-------|
| `HTTP_ADDR` | `:8080` | Listen address |
| `STORE_BACKEND` | `db` | `db` \| `file` \| `memory` |
| `STORE_PATH` | `./data` | File store directory |
| `DB_DRIVER` | `postgres` | `postgres` \| `mysql` \| `sqlite` when `STORE_BACKEND=db` |
| `DATABASE_URL` | — | Required for `db` backend |
| `ASYNC_EXECUTION` | `false` | Background job worker |
| `WORKER_INTERVAL` | `500ms` | Job poll interval |
| `LOG_LEVEL`, `LOG_FORMAT` | — | Structured logging |
| `OTEL_*` | — | Optional OpenTelemetry |
| `API_KEY` / `API_KEYS` | — | API auth (`tenant:key` CSV); `/api/*` protected when set |
| `AUTH_REQUIRED` | `false` | Require keys even if list empty |
| `EVENT_CONSUMER` | `memory` | `redis` for HA multi-replica |
| `PLUGIN_*_ADAPTERS` | — | See [docs/plugins.md](./docs/plugins.md) |

Docker file store: image entrypoint `chown`s `STORE_PATH` before dropping to `appuser`.

## BPMN model (JSON)

Elements: `startEvent`, `endEvent`, `userTask`, `scriptTask`, `exclusiveGateway`, `parallelGateway`, `inclusiveGateway`, `subProcess`, `sequenceFlow`.

JSON Schema: [`schemas/process-definition.schema.json`](./schemas/process-definition.schema.json), [`schemas/trigger-message.schema.json`](./schemas/trigger-message.schema.json).

- **startEvent**: BPMN 2.0 `eventDefinition` (NOT sequenceFlow condition):
  - `none` — manual / `POST /process-instances`
  - `message` — `messageRef` + optional `condition`, `correlationKey` → triggered via `POST /api/v1/triggers/message`
  - `signal`, `timer`, `conditional` — metadata; timer/signal dispatch external
- **userTask**: Pauses until complete API. Supports **approval modes** on `assignees`:
  - `any` / `或签` — need `requiredApprovals` from pool (default **1**, GitHub review style)
  - `all` / `会签` — every assignee must approve
  - `sequential` / `顺签` — assignees act in list order
  - `returnTo` — element to re-activate on reject (default: upstream userTask)
  - `onReject`: `return` (default) | `terminateScope`
  - `scopeId` — sub-process scope for parallel regions
  - `assigneesVariable`: dot path in instance variables (e.g. `assignees.review`) — resolved at userTask activation; overrides static `assignees` when present
  - Complete with `{ "assignee": "...", "action": "approve|reject", "comment": "..." }`
- **Assignee sync (canonical)**: upper systems adapt their events → `POST /api/v1/assignee-sync/remove-user` or `/replace`, or plugin adapters via `POST /api/v1/events/assignee/{source}`.
- **Event plugins**: triple consumers (`assignee` + `trigger` + `task`), WASM + capabilities. See [docs/plugins.md](./docs/plugins.md).
- **subProcess**: marker with `scopeId`, `entryRef`, `exitRef` for scoped parallel work.
- **scriptTask**: `scriptLang: "javascript"` (default) or `"log"`; JS has `vars`, `http.get/post/request`
- **Conditions** (`internal/bpmn/expression.go`): `==`, `!=`, numeric compares, truthy field; dot paths supported (`item.kk >= 10`).

Each deploy creates a **new version**. Instances pin `definition_snapshot` at start.

## HTTP API (summary)

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/healthz` | Health |
| GET | `/metrics` | Prometheus |
| PUT | `/api/v1/tenants/{tenantId}/processes/{key}` | Deploy |
| GET | `/api/v1/tenants/{tenantId}/processes` | List latest per key |
| DELETE | `/api/v1/tenants/{tenantId}/processes/{key}` | Delete all versions |
| POST | `/api/v1/process-instances` | Start |
| POST | `/api/v1/triggers/message` | Message start (direct) |
| POST | `/api/v1/events/assignee/{source}` | Assignee stream plugin ingress |
| POST | `/api/v1/events/trigger/{source}` | Trigger stream plugin ingress |
| POST | `/api/v1/events/task/{source}` | Task stream plugin ingress |
| POST | `/api/v1/events/control/{source}` | Control stream (terminate) |
| GET | `/api/v1/process-instances/{id}` | Instance + variables + `lock_version` |
| GET | `/api/v1/process-instances/{id}/activities` | Activity trail |
| POST | `/api/v1/process-instances/{id}/tasks/{activityId}/complete` | Complete userTask |
| PATCH | `/api/v1/process-instances/{id}/tasks/{activityId}/assignees` | Update assignees |
| POST | `/api/v1/assignee-sync/remove-user` | Canonical: remove user from active tasks |
| POST | `/api/v1/assignee-sync/replace` | Canonical: replace task assignees |
| POST | `/api/v1/process-instances/{id}/terminate` | Cancel instance or scope (`scopeId` optional) |
| GET | `/api/v1/tasks?tenantId=&assignee=` | Active userTask inbox |

Errors: `{ "error": "...", "code": "..." }`. Stale `lockVersion` on complete → **409** `version_conflict`.

All handlers use `engine.Engine` via context — no global store singleton.

## Development commands

```bash
go test ./...
go run ./cmd/server

# Docker (from deploy/docker or examples/bpmn)
docker compose up -d --build
```

## Change guidelines for agents

1. **Minimal diffs** — Match existing style; extend `engine.Store` only when persistence needs it; implement in `memory`, `gormstore`, and `filestore` together.
2. **Tests** — Add/update tests in `internal/bpmn`, `internal/engine`, or store packages; use `memory` store for engine tests.
3. **Transactions** — Multi-step writes go through `Store.WithTx`.
4. **API** — Keep JSON-first; wire new routes in `internal/api/http.go` with consistent error codes.
5. **Do not** add auth middleware unless explicitly requested — use `API_KEY` / `AUTH_REQUIRED` env instead when enabling auth.

## Known gaps (do not assume implemented)

- Authentication / authorization (optional API keys — set `API_KEY` or `AUTH_REQUIRED`)
- Boundary events, BPMN XML
- Multi-instance loops
- Idempotent start by `businessKey` (message trigger — running instance dedupe)
- Webhooks / SSE for task events (inbound message trigger: `POST /api/v1/triggers/message`)

## When editing related projects

- **examples/bpmn**: Docker compose builds this repo; client calls HTTP API only.
- After engine API or model changes, check whether example client or README in examples needs updates (only if user asks or change is user-visible).
