# Task Approval

UserTask approval spans **definition rules** (`domain/definition`) and **runtime enforcement** (`internal/engine`).

## Definition configuration

On `userTask` elements:

| Field | Values | Purpose |
|-------|--------|---------|
| `assignees` | string[] | Static approver list |
| `assigneesVariable` | dot path | Resolved from instance variables at activation; overrides static list when present |
| `approvalMode` | `any`, `all`, `sequential` | 或签 / 会签 / 顺签 |
| `requiredApprovals` | int | For `any` mode — how many approvals needed from pool (default **1**) |
| `returnTo` | element id | Re-activate target on reject |
| `onReject` | `return` (default), `terminateScope` | Reject behavior |
| `scopeId` | string | Sub-process scope for parallel regions |
| `autoComplete` | bool | Skip wait (tests only) |

## Approval modes (runtime)

### `any` (或签)

Any `requiredApprovals` assignees from the pool may approve. Default quota is **1** (GitHub-style review count).

Partial progress persists `pending_assignees` and `approval_records` on the activity until the node finishes.

### `all` (会签)

Every assignee must approve before the task completes.

### `sequential` (顺签)

Assignees act in list order. Only the current head of `pending_assignees` may complete.

## Complete API

```json
POST /api/v1/process-instances/{id}/tasks/{activityId}/complete
{
  "assignee": "u1",
  "action": "approve",
  "comment": "looks good",
  "lockVersion": 3,
  "variables": { "approvedAmount": 5000 }
}
```

Actions: `approve` | `reject`.

When multiple assignees exist, `assignee` is required. Stale `lockVersion` → **409** `version_conflict`.

## Reject handling

On `reject`:

1. Record outcome on activity
2. If `onReject: terminateScope` → cancel scope activities
3. If `onReject: return` (default) → resolve `returnTo` via `definition.ResolveReturnTarget`; re-activate upstream userTask; cancel same parallel branch only

## Assignee sync

Personnel changes from external HR systems:

| Endpoint | Purpose |
|----------|---------|
| `POST /api/v1/assignee-sync/remove-user` | Remove user from all active tasks |
| `POST /api/v1/assignee-sync/replace` | Replace assignees on one task |
| `PATCH .../tasks/{activityId}/assignees` | Manual override |

Plugin ingress: `POST /api/v1/events/assignee/{source}` → Host SDK → engine assignee sync.

## Inbox

`GET /api/v1/tasks?tenantId=&assignee=` returns active userTask projections (`runtime.InboxTask`) with process context.

## Separation of concerns

- **Definition** validates mode/quota configuration (`ValidateUserTaskApproval`, `RequiredApprovals`)
- **Engine** tracks partial state on `ActivityInstance` and decides when to continue traversal
- **Store** persists activity rows; no approval logic in infrastructure
