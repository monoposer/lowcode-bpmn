# DDD Layer Overview

**lowcode-bpmn** is organized as a lightweight Domain-Driven Design layout. The engine microservice separates **process design** (static definitions) from **process runtime** (live instances), with **application orchestration** and **infrastructure adapters** at the edges.

## Layer map

```
┌─────────────────────────────────────────────────────────────┐
│  Delivery (HTTP, plugins, worker)                           │
│  internal/api · internal/plugin · cmd/server                │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│  Application (use cases)                                    │
│  internal/engine · internal/application                     │
└───────────────┬─────────────────────────┬───────────────────┘
                │                         │
┌───────────────▼──────────────┐  ┌───────▼───────────────────┐
│  Domain — Process Design     │  │  Domain — Process Runtime │
│  internal/domain/definition  │  │  internal/domain/runtime  │
└───────────────┬──────────────┘  └───────┬───────────────────┘
                │                         │
                └────────────┬────────────┘
                             │
┌────────────────────────────▼────────────────────────────────┐
│  Ports (outbound)                                           │
│  internal/domain/ports.ProcessRepository                    │
└────────────────────────────┬────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────┐
│  Infrastructure                                             │
│  internal/store/{memory,gormstore,filestore}                │
│  internal/bpmnxml · internal/script · pkg/event             │
└─────────────────────────────────────────────────────────────┘
```

## Bounded contexts

| Context | Package | Responsibility |
|---------|---------|----------------|
| **Process Design** | `internal/domain/definition` | BPMN 2.0 JSON IR, graph validation, conditions, start-event matching, approval rules on definitions |
| **Process Runtime** | `internal/domain/runtime` | Instances, activities, jobs, inbox DTOs, request structs |
| **Persistence port** | `internal/domain/ports` | `ProcessRepository` — store contract without implementation |
| **Application** | `internal/engine` | Token traversal, userTask wait/complete, triggers, reject handling, async jobs |
| **Integration** | `internal/bpmnxml`, `internal/plugin`, `internal/api` | XML codec, vendor adapters, HTTP surface |

## Compatibility shims

During migration, stable import paths are preserved:

| Legacy import | Canonical import | Notes |
|---------------|------------------|-------|
| `internal/bpmn` | `internal/domain/definition` | `compat.go` re-exports types and functions |
| `engine.Store` | `ports.ProcessRepository` | Alias in `engine/bpmn_model.go` |
| `engine.ProcessInstance`, etc. | `runtime.*` | Type aliases in `engine/bpmn_model.go` |

New code should import domain packages directly. `internal/bpmn` is deprecated.

## Domain documents

| Document | Topic |
|----------|-------|
| [definition.md](./definition.md) | Process Design context — model, validation, expressions |
| [runtime.md](./runtime.md) | Process Runtime entities and lifecycle |
| [execution.md](./execution.md) | Engine token model, gateways, subprocess scopes |
| [task-approval.md](./task-approval.md) | UserTask approval modes, reject, assignee sync |
| [trigger.md](./trigger.md) | Message/signal/conditional start events |
| [persistence.md](./persistence.md) | `ProcessRepository` and store backends |
| [integration.md](./integration.md) | API, plugins, XML interchange |
| [extensions.md](./extensions.md) | BPMN extension model — core vs adapter-backed constructs |
| [testing.md](./testing.md) | TDD workflow mapped to each DDD layer |

See also [BPMN 2.0 compliance matrix](../bpmn-compliance.md).

## Dependency rule

- **Domain** packages (`definition`, `runtime`, `ports`) must not import `engine`, `api`, or store implementations.
- **Application** (`engine`) imports domain and ports; it does not embed persistence logic beyond calling the port.
- **Infrastructure** implements `ProcessRepository` and calls into `engine` only from the outside (HTTP handlers, worker, plugins).

## Tests

See **[testing.md](./testing.md)** for the full TDD guide (Red-Green-Refactor, layer map, copy-paste examples).

Quick reference:

- `internal/domain/definition/*_test.go` — validation, conditions, approval rules
- `internal/domain/runtime/*_test.go` — pure lifecycle and entity rules
- `internal/domain/ports/testing/contract.go` — shared `ProcessRepository` contract
- `internal/store/memory/contract_test.go` — contract against memory adapter
- `internal/engine/*_test.go` — end-to-end execution with `memory` store
- `internal/bpmnxml/*_test.go` — XML round-trip

Run: `go test ./...`
