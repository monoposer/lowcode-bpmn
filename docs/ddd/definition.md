# Process Design (`internal/domain/definition`)

The **Process Design** bounded context owns the static BPMN 2.0 process model — everything needed to deploy a definition without executing it.

## Ubiquitous language

| Term | Meaning |
|------|---------|
| `ProcessDefinition` | Deployable process graph: elements + sequence flows |
| `Element` | A BPMN node (task, gateway, event, subProcess marker) |
| `SequenceFlow` | Directed edge with optional condition |
| `Registry` | Indexed view of a definition for O(1) lookup during execution |
| `EventDefinition` | BPMN 2.0 start trigger metadata on `startEvent` |

## Core types

`ProcessDefinition` is the canonical interchange format:

```json
{
  "id": "order-approval",
  "name": "Order Approval",
  "elements": [ { "id": "start", "type": "startEvent", ... } ],
  "flows": [ { "id": "f1", "sourceRef": "start", "targetRef": "review" } ]
}
```

JSON is the primary API/designer format. BPMN XML (`.bpmn20.xml`) is supported via `internal/bpmnxml` and maps to the same IR.

## Validation principles

1. **Graph integrity** — every `sourceRef` / `targetRef` must resolve; at least one `startEvent` and one `endEvent`.
2. **Element-specific rules** — `userTask` assignees/approval, `scriptTask` script body, `subProcess` scope markers.
3. **Start events** — `message` requires `messageRef`; `conditional` requires `condition`.
4. **No runtime state** — validation never touches instances or variables beyond expression syntax.

Entry points: `Validate(def)`, `BuildRegistry(def)` (validate + index).

## Conditions (`expression.go`)

Sequence-flow conditions are evaluated at **gateways only**, not on linear edges from start events.

Supported operators: `==`, `!=`, `>=`, `>`, `<=`, `<`, bare truthy field name.

Dot paths resolve into instance variables: `order.amount >= 1000`, `event.fields.status == Pending`.

Missing path semantics:

- `==` / numeric compare → false
- `!=` → true

## Start-event matching (`event.go`)

`startEvent.eventDefinition` defines **when** a process may start. This is separate from gateway routing conditions.

| Type | Match function | Required fields |
|------|----------------|-----------------|
| `none` | Manual start only | — |
| `message` | `MessageStartMatch` | `messageRef`, optional `condition`, `correlationKey` |
| `signal` | `SignalStartMatch` | `signalRef` |
| `conditional` | `ConditionalStartMatch` | `condition` |
| `timer` | Metadata only | `timerCycle` — external scheduler dispatches |

`BusinessKeyFromCorrelation` extracts dedupe key from variables using `correlationKey` dot path.

## Approval definition rules (`approval.go`, `approval_quota.go`)

Definition-time constraints for `userTask`:

- `approvalMode`: `any` | `all` | `sequential` (aliases: 或签, 会签, 顺签)
- `requiredApprovals` — quota for `any` mode (default **1**)
- `returnTo`, `onReject` (`return` | `terminateScope`), `scopeId`

Runtime enforcement lives in `internal/engine`; definition package only validates configuration.

## SubProcess markers (`return.go`)

`subProcess` elements carry scope metadata:

- `scopeId` — groups elements belonging to the scope
- `entryRef` / `exitRef` — inner entry and exit element ids

`ScopeElementIDs`, `ScopeGatewayIDs`, `ResolveReturnTarget` support reject/terminate logic in the engine.

## Import guidance

```go
import "github.com/monoposer/lowcode-bpmn/internal/domain/definition"
```

Avoid new imports of `internal/bpmn` (deprecated compat shim).

## TDD

Process Design tests are pure unit tests — no I/O, no store. Use table-driven cases for validation and parsing rules.

- Guide: [testing.md](./testing.md)
- Local example: [`internal/domain/definition/approval_test.go`](../../internal/domain/definition/approval_test.go)
