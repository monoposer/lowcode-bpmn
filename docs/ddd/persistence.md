# Persistence

Persistence is defined by the outbound port `internal/domain/ports.ProcessRepository` and implemented by store backends.

## Port contract

`ProcessRepository` groups operations the engine needs:

| Group | Methods |
|-------|---------|
| Transactions | `WithTx(ctx, fn)` |
| Definitions | `InsertProcessVersion`, `DeleteProcess`, `GetProcess`, `GetProcessVersion`, `ListProcesses` |
| Instances | `CreateProcessInstance`, `UpdateProcessInstance`, `GetProcessInstance`, `GetProcessInstanceForUpdate`, `FindRunningInstanceByBusinessKey` |
| Activities | `CreateActivityInstance`, `UpdateActivityInstance`, `GetActivityInstance`, `ListActivitiesByProcess`, `ListActiveActivities`, `ListActiveUserTasks` |
| Jobs | `EnqueueJob`, `ClaimNextJob`, `CompleteJob`, `FailJob` |

Engine imports this as `engine.Store` (type alias).

## Backends

Factory: `internal/store/open.go` — `STORE_BACKEND`:

| Backend | Env | Use case |
|---------|-----|----------|
| `db` (default) | `DB_DRIVER`, `DATABASE_URL` | Production (Postgres, MySQL, SQLite via GORM) |
| `file` | `STORE_PATH` | Local dev — YAML snapshot |
| `memory` | — | Unit tests |

**Rule:** new store methods require implementation in **all three** backends.

## Schema (GORM / db)

| Table | Purpose |
|-------|---------|
| `bpmn_processes` | Versioned definitions |
| `bpmn_instances` | Variables, `internal_state`, `definition_snapshot`, `lock_version`, `business_key` |
| `bpmn_activities` | Audit trail, assignees, approval records, outcome |
| `bpmn_jobs` | Async queue |

GORM applies **AutoMigrate** on startup; Postgres also has embedded SQL baselines.

## Version pinning

Each deploy inserts a new version row. `ProcessInstance.ProcessVersion` and `DefinitionSnapshot` are set at start and never updated when a newer deploy occurs.

## Optimistic locking

`lock_version` on instances increments on each mutating engine operation. Clients pass `lockVersion` on complete; mismatch → `runtime.ErrVersionConflict` (HTTP 409).

## Internal state

`ProcessInstance.InternalState` (gateway join tokens) is persisted but excluded from public JSON responses. Store layers serialize it as JSON/YAML alongside variables.

## File store specifics

Definitions may be stored as BPMN XML (`.bpmn20.xml`) via `internal/store/filestore/process_xml.go` using `internal/bpmnxml` codec.

## Adding persistence features

1. Extend `ProcessRepository` in `internal/domain/ports/repository.go`
2. Implement in `memory`, `gormstore`, `filestore`
3. Use from engine inside `WithTx` when multiple rows must commit together
4. Do not add business logic to store implementations — keep them as adapters

## TDD

Port contract tests keep all store backends aligned. Extend `RunProcessRepositoryContract` when the port changes.

- Guide: [testing.md](./testing.md)
- Contract suite: [`internal/domain/ports/testing/contract.go`](../../internal/domain/ports/testing/contract.go)
- Local example: [`internal/store/memory/contract_test.go`](../../internal/store/memory/contract_test.go)
