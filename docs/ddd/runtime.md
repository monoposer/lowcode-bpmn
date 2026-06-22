# Process Runtime (`internal/domain/runtime`)

The **Process Runtime** bounded context models live execution state. It references `definition.ProcessDefinition` via snapshot pinning but contains no traversal logic.

## Ubiquitous language

| Term | Meaning |
|------|---------|
| `DeployedProcess` | A versioned definition row `(tenant_id, process_key, version)` |
| `ProcessInstance` | Aggregate root for one running or completed execution |
| `ActivityInstance` | Per-element audit record (assignees, outcome, scope) |
| `Job` | Async continuation unit (`start` or `continue`) |
| `InboxTask` | Read model for active userTask inbox queries |

## ProcessInstance

Key fields:

- `DefinitionSnapshot` — pinned at start; unaffected by later deploys
- `Variables` — JSON-serializable process data
- `InternalState` — gateway join tokens and execution machinery (**not API-visible**)
- `LockVersion` — optimistic concurrency for complete/assignee updates
- `BusinessKey` — optional dedupe key for message triggers
- `ActiveElements` — denormalized list of waiting element ids

Status lifecycle: `pending` → `running` → `completed` | `failed` | `cancelled`.

## ActivityInstance

Tracks each visit to a BPMN element:

| Field | Purpose |
|-------|---------|
| `ElementID` | Which node in the definition |
| `Status` | `active`, `completed`, `failed`, `cancelled` |
| `ScopeID` | Sub-process scope membership |
| `BranchFlowID` | Parallel branch identity for scoped reject |
| `Outcome` | `approve`, `reject`, `cancelled` (userTask) |
| `Assignees` / `PendingAssignees` | Current and remaining approvers |
| `ApprovalRecords` | Partial approval history until node completes |
| `RequiredApprovals` | Resolved quota at activation |

## DeployedProcess

Created on each deploy. Version increments per `(tenant_id, process_key)`. Latest version is used for new starts; existing instances keep their snapshot version.

## Job

Async execution queue item:

- Types: `start`, `continue`
- Status: `pending`, `running`, `done`, `failed`
- Worker claims jobs with row locking (GORM store) for multi-replica safety

## Request DTOs

Inbound command structs shared by API and engine:

- `CompleteTaskRequest` — assignee, action, comment, variables, lockVersion
- `TerminateRequest` — optional `scopeId` for scope vs instance cancel
- `UpdateAssigneesRequest` — manual assignee override

## Errors

`ErrVersionConflict` — returned when `lockVersion` is stale (HTTP 409).

## Import guidance

```go
import "github.com/monoposer/lowcode-bpmn/internal/domain/runtime"
```

Engine re-exports these types as aliases for backward compatibility (`engine.ProcessInstance`, etc.).

## TDD

Runtime domain tests cover entity behavior and value-object rules without persistence or HTTP.

- Guide: [testing.md](./testing.md)
- Local example: [`internal/domain/runtime/lifecycle_test.go`](../../internal/domain/runtime/lifecycle_test.go) (table-driven; comments mark TDD domain templates)
