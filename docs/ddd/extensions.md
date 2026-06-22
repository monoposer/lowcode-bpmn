# BPMN Extensions (DDD)

**lowcode-bpmn** targets full BPMN 2.0 *coverage* through a layered extension model. The engine core implements token flow for the most common constructs; everything else is **modeled in Process Design**, **carried at runtime**, and **executed via integration adapters** when an extension is registered.

Whether a feature is *active* in a deployment depends on which extensions are plugged in — not on hard engine exclusions.

## Three extension layers (DDD)

```
┌─────────────────────────────────────────────────────────────────┐
│  Integration — plugins, WASM, external schedulers, form/auth    │
│  internal/plugin · plugins/* · external HTTP services           │
└───────────────────────────────┬─────────────────────────────────┘
                                │ Host SDK / future execution port
┌───────────────────────────────▼─────────────────────────────────┐
│  Application — engine traversal, extension dispatch               │
│  internal/engine                                                  │
└───────────────┬─────────────────────────┬─────────────────────────┘
                │                         │
┌───────────────▼──────────────┐  ┌───────▼────────────────────────┐
│  Domain — Process Design     │  │  Domain — Process Runtime      │
│  IR, validation, XML codec   │  │  instance/activity extension   │
│  definition + bpmnxml        │  │  metadata on entities          │
└──────────────────────────────┘  └────────────────────────────────┘
```

| Layer | Package | Extension responsibility |
|-------|---------|--------------------------|
| **Process Design** | `internal/domain/definition`, `internal/bpmnxml` | Parse and validate BPMN constructs (including boundary events, pools/lanes, data refs, advanced gateways). Preserve `extensionElements` and custom namespace attributes. |
| **Process Runtime** | `internal/domain/runtime` | Attach extension context to activities (boundary subscriptions, lane hints, data object bindings). No vendor I/O. |
| **Application** | `internal/engine` | Native execution for core constructs; delegate to extension handlers when IR marks an element as extension-backed. |
| **Integration** | `internal/plugin`, `plugins/*` | User/role/form systems, timers, service tasks, boundary triggers, DMN, pool collaboration, call activity targets. |

### Dependency rule

Extension metadata flows **inward** (Integration → Application → Domain). Domain packages never import plugins. Plugins never import store implementations — only Host SDK and (future) execution ports.

## Support tiers

Used in [bpmn-compliance.md](../bpmn-compliance.md):

| Tier | Symbol | Meaning |
|------|--------|---------|
| **Core** | ✅ | Native engine execution without an adapter |
| **Extension** | 🔌 | Modeled in IR/XML; active when an extension adapter is registered |
| **Partial** | ⚠️ | Metadata + partial core behavior; external system completes semantics |
| **Not modeled** | ❌ | Not yet in IR — planned via definition extension work |

## Construct → extension map

### Events

| Construct | Tier | DDD touchpoint | Extension hook |
|-----------|------|----------------|----------------|
| None / message / conditional start | ✅ Core | `definition` start matching, `engine` start | — |
| Signal / timer start | ⚠️ Partial | `definition` metadata | **trigger** stream or external scheduler plugin |
| End event | ✅ Core | `engine` lifecycle | — |
| Intermediate catch/throw | 🔌 Extension | `definition` IR + `runtime` activity | **trigger** / **task** streams; message correlation plugin |
| Boundary (timer/message/error/…) | 🔌 Extension | `definition` attachment on activity | **trigger** stream + engine boundary subscription (extension dispatch) |
| Event sub-process | 🔌 Extension | `definition` scoped subgraph | **control** + **trigger** streams |

### Activities

| Construct | Tier | DDD touchpoint | Extension hook |
|-----------|------|----------------|----------------|
| User task | ✅ Core | `definition` approval rules, `engine` wait/complete | **task** / **assignee** streams; form UX external |
| Script task | ✅ Core | `internal/script` runner | Optional WASM **script-runner** plugin |
| Embedded sub-process | ✅ Core (scoped) | `definition` scope markers, `engine` scope join | Advanced call semantics → extension |
| Service / send / receive task | 🔌 Extension | `definition` + `bpmnxml` extensions | Service adapter plugin or `scriptTask` delegate |
| Business rule (DMN) | 🔌 Extension | `decisionRef` in IR | External DMN engine plugin |
| Call activity | 🔌 Extension | `calledElement` in IR | Sub-process invocation plugin via Host SDK |
| Multi-instance loop | 🔌 Extension | `loopCharacteristics` in IR | Loop coordinator plugin |
| Ad-hoc / transaction sub-process | 🔌 Extension | IR markers | **control** stream + scoped engine hooks |

### Gateways

| Construct | Tier | DDD touchpoint | Extension hook |
|-----------|------|----------------|----------------|
| Exclusive / parallel / inclusive | ✅ Core | `definition` conditions, `engine` fork/join | — |
| Event-based gateway | 🔌 Extension | IR + runtime wait set | **trigger** stream; first-arriving event wins |
| Complex gateway | 🔌 Extension | `activationCondition` per flow | Custom evaluator plugin (`complex_gateway` capability) |

### Data, pools, collaboration

| Construct | Tier | DDD touchpoint | Extension hook |
|-----------|------|----------------|----------------|
| Instance variables | ✅ Core | `runtime.ProcessInstance.Variables` | Canonical data carrier |
| Data object / data store | 🔌 Extension | IR refs + `extensionElements` | Maps to variables; external store via service plugin |
| Pool / lane | 🔌 Extension | IR collaboration section | Lane → assignee/role resolution via **assignee** plugin + external user/role system |
| Message flow (between pools) | 🔌 Extension | IR message flow edges | **trigger** stream + correlation plugin |

### Cross-cutting platform concerns

These are **never** in the engine core. They are integration extensions:

| Concern | Extension hook |
|---------|------------------|
| User management | **assignee** stream; `POST /assignee-sync/*`; plugin maps HR IDs |
| Role / group management | Lane `assigneesVariable` + external role resolver plugin |
| Form designer | `userTask` metadata (`formKey`, `formUrl` in `extensionElements`); rendering in business app |
| Authentication / authorization | `API_KEY` at HTTP edge; tenant scoping in engine |

## BPMN XML interchange

XML import/export is **in scope** for Process Design (`internal/bpmnxml`):

- Deploy accepts `Content-Type: application/xml` and `.bpmn20.xml` files.
- Standard BPMN elements map to JSON IR; unsupported runtime semantics are preserved as extension metadata.
- Low-code attributes use namespace `https://github.com/monoposer/lowcode-bpmn/extensions`.

Round-trip fidelity for extension-only constructs improves as IR fields are added — execution remains adapter-driven.

## Current plugin streams (integration extensions)

Today, quad event streams cover ingress extensions:

| Stream | Typical BPMN extension use |
|--------|----------------------------|
| **assignee** | Lane/role sync, HR user removal, assignee replacement |
| **trigger** | Message/signal/timer/boundary/event-gateway starts |
| **task** | External approve/reject (forms, IM cards) |
| **control** | Terminate, event sub-process cancel, compensation trigger |

WASM capabilities (`internal/plugin/wasm/capability.go`) gate Host SDK access. Future execution capabilities (e.g. `evaluate_complex_gateway`, `invoke_service_task`) follow the same pattern — declared in manifest, implemented in `plugins/wasm/*` or `plugins/go/*`.

## Adding a new extension (TDD workflow)

1. **Process Design** — Add IR fields + validation in `internal/domain/definition`; XML mapping in `internal/bpmnxml`; JSON Schema in `schemas/`. Write domain unit tests first.
2. **Process Runtime** — Add DTO fields if instances/activities carry extension state. Pure tests in `internal/domain/runtime`.
3. **Application** — Engine recognizes element type; if no native handler, dispatch to extension port or skip with explicit `extension_required` job/event.
4. **Integration** — Implement adapter in `plugins/go/*` or WASM; register in `plugins/registry/registry.go`; document env in [plugins.md](../plugins.md).
5. **Contract** — Port contract tests if a new persistence shape is needed; engine integration test with `memory` store.

Do **not** embed vendor logic in domain or engine packages.

## Roadmap (definition IR)

Implemented in `internal/domain/definition` (validated + indexed in `Registry`):

- `boundaryEvent` with `attachedToRef`, `eventDefinition`
- `intermediateCatchEvent` / `intermediateThrowEvent`
- `eventBasedGateway`, `complexGateway`
- `callActivity` with `calledElement`
- `multiInstance` loop characteristics on any element
- `laneSet`, `dataObjects`, `dataStores` on process
- `formKey`, `formUrl`, `extensionHandler` on elements

XML import/export: `internal/bpmnxml` round-trips the above. Engine pauses extension elements as `active` with `input.extensionRequired=true` until a plugin completes them.

Remaining (future):

- Async child instance tracking for callActivity (parent/child correlation)
- Non-interrupting boundary parallel branch join semantics
- Full multi-process collaboration orchestration UI

See [bpmn-compliance.md](../bpmn-compliance.md) for the live matrix and [integration.md](./integration.md) for delivery-layer wiring.
