# Execution (`internal/engine`)

The **engine** is the application layer: it orchestrates domain definition rules and runtime entities through the `ProcessRepository` port.

## Responsibilities

| Concern | Location |
|---------|----------|
| Deploy / validate | `DeployProcess` → `definition.BuildRegistry` |
| Start instance | `StartProcess`, message trigger paths |
| Token traversal | `advance`, gateway fork/join |
| UserTask wait | Pause with `ActivityStatusActive` |
| ScriptTask | `internal/script` runner, merge variables |
| SubProcess scope | Scope-aware reject and terminate |
| Async continuation | Enqueue `Job`, worker drains queue |

## Token model

Execution follows BPMN sequence flows from the pinned `DefinitionSnapshot`:

1. Activate element → create `ActivityInstance`
2. If automated (`scriptTask`, gateways) → complete immediately and follow outgoing flows
3. If `userTask` → stop traversal; instance stays `running`
4. If `endEvent` → mark activity complete; when no active activities remain → instance `completed`

Join tokens for parallel/inclusive gateways are stored in `ProcessInstance.InternalState` (opaque to API clients).

## Gateway behavior

| Gateway | Fork | Join |
|---------|------|------|
| `exclusiveGateway` | First matching condition; optional `isDefault` flow | N/A |
| `parallelGateway` | All outgoing flows | Wait for all incoming |
| `inclusiveGateway` | All matching conditions | Same as parallel on join |

Conditions use `definition.EvalCondition(vars, expr)`.

## SubProcess scopes

`subProcess` is a **marker** element with `scopeId`, `entryRef`, `exitRef`. Elements sharing `scopeId` belong to the scope.

- **Reject with `onReject: return`** — re-activate `returnTo` (default: upstream userTask); cancels same parallel branch only; clears join tokens in scope.
- **Terminate scope** — `TerminateRequest.ScopeID` cancels active activities in scope; instance may stay `running` if work remains outside.
- **Terminate instance** — no `scopeId` → `status: cancelled`.

## Sync vs async

| Mode | Config | Behavior |
|------|--------|----------|
| Sync (default) | — | HTTP request runs traversal to next wait point |
| Async | `ASYNC_EXECUTION=true` | Enqueue job; worker continues in background |

Worker poll interval: `WORKER_INTERVAL` (default `500ms`).

## Transactions

Multi-step writes (`StartProcess`, `CompleteTask`, assignee sync) run inside `Store.WithTx` so instance, activities, and jobs commit atomically.

## Testing strategy

Engine tests use `internal/store/memory` implementing `ProcessRepository`. Domain validation tests stay in `definition`; execution integration tests stay in `engine`.

## TDD

Application-layer tests integrate domain + port via the in-memory store — not HTTP.

- Guide: [testing.md](./testing.md)
- Local example: [`internal/engine/engine_test.go`](../../internal/engine/engine_test.go)
