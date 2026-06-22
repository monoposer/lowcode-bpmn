# Integration Layer

Delivery and adapter code sits outside the domain core. It translates external protocols into engine use cases.

## HTTP API (`internal/api`)

Chi router exposes REST endpoints. Handlers resolve `engine.Engine` from request context — no global store singleton.

| Area | Handlers | Domain touchpoints |
|------|----------|-------------------|
| Processes | deploy, list, delete, get | `definition.ProcessDefinition`, `bpmnxml` for XML accept/content-type |
| Instances | start, get, activities, terminate | `runtime.ProcessInstance` |
| Tasks | complete, assignees, inbox | `CompleteTaskRequest`, `InboxTask` |
| Triggers | message | `TriggerMessageRequest` |
| Assignee sync | remove-user, replace | engine assignee sync |
| Events | `/events/{stream}/{source}` | plugin runtime → Host SDK |

Error envelope: `{ "error", "code" }`.

## BPMN XML (`internal/bpmnxml`)

Codec maps BPMN 2.0 XML ↔ `definition.ProcessDefinition`:

- `Parse` / `ParseReader` — import `.bpmn` / `.bpmn20.xml`
- `Marshal` — export for file store and `Accept: application/xml` deploy responses

Low-code extensions use `https://github.com/monoposer/lowcode-bpmn/extensions` namespace for assignees, approval mode, scope markers, etc.

Full BPMN 2.0 coverage (boundary events, advanced gateways, pools/lanes, data objects, user/role/form) is **extension-backed** — see [extensions.md](./extensions.md).

**Import:** `internal/domain/definition` (not deprecated `internal/bpmn`).

## Script execution (`internal/script`)

`scriptTask` runs via injectable `script.Runner`:

- `javascript` (default) — goja with `vars`, `http.*`
- `log` — structured log only

Engine calls runner during traversal; output merges into instance variables.

## Event plugins (`internal/plugin`, `plugins/`)

Quad streams decouple vendor concerns:

| Stream | Ingress | Host SDK |
|--------|---------|----------|
| assignee | `/events/assignee/{source}` | Remove/replace assignees |
| trigger | `/events/trigger/{source}` | TriggerMessage, StartProcess |
| task | `/events/task/{source}` | CompleteTask |
| control | `/events/control/{source}` | Terminate |

Transport: `pkg/event` (memory, redis, kafka, nats, rabbitmq). Vendor mapping lives in `plugins/go/*` and `plugins/wasm/*` — never in domain packages.

## Telemetry (`internal/telemetry`)

Structured logging (slog) and optional OpenTelemetry middleware for HTTP and engine spans.

## Boundary rules

```
External world → api / plugin → engine (application) → ports → store
                                      ↓
                              definition + runtime (domain)
```

- Domain packages never import `api`, `plugin`, or store implementations.
- Plugins never import store directly — always through Host SDK → engine.
- JSON is the primary designer interchange; XML is supported for deploy and file persistence.

See [plugins.md](../plugins.md) for adapter configuration and WASM capabilities.
