# BPMN 2.0 Compliance Matrix

This document maps OMG BPMN 2.0 constructs to **lowcode-bpmn** support status. The engine provides **core execution** for common workflows; full spec coverage is achieved through **extensions** (see [docs/ddd/extensions.md](./ddd/extensions.md)).

Legend:

| Symbol | Tier | Meaning |
|--------|------|---------|
| ✅ | **Core** | Native engine execution |
| 🔌 | **Extension** | Modeled in IR/XML; active when an adapter is registered |
| ⚠️ | **Partial** | Metadata + partial core; external system completes semantics |
| ❌ | **Not modeled** | Not yet in IR (planned via definition extension work) |

## Events

| Construct | Status | Notes |
|-----------|--------|-------|
| None start event | ✅ | `POST /process-instances` |
| Message start event | ✅ | `messageRef`, `condition`, `correlationKey`; `POST /triggers/message` |
| Conditional start event | ✅ | Condition evaluated on trigger variables |
| Signal start event | ⚠️ | Definition + engine match; dispatch via **trigger** plugin/API |
| Timer start event | ⚠️ | `timerCycle` stored; external scheduler or **trigger** plugin |
| End event | ✅ | Terminates instance when no active activities |
| Intermediate catch/throw events | 🔌 | IR + **trigger** / **task** plugin streams |
| Boundary events | 🔌 | Attached in IR; **trigger** + engine boundary subscription (extension dispatch) |
| Event sub-process | 🔌 | Scoped subgraph; **trigger** + **control** streams |

## Activities

| Construct | Status | Notes |
|-----------|--------|-------|
| User task | ✅ | Multi-sign modes, reject return, scope terminate |
| Script task | ✅ | JavaScript (goja) or log |
| Service task | 🔌 | Modeled in IR/XML; service adapter or script delegate plugin |
| Send task | 🔌 | Modeled in IR/XML; outbound adapter plugin |
| Receive task | 🔌 | Modeled in IR/XML; **trigger** correlation plugin |
| Business rule task | 🔌 | `decisionRef` in IR; external DMN plugin |
| Sub-process (embedded) | ✅ | Scope marker: `scopeId`, `entryRef`, `exitRef` |
| Call activity | 🔌 | `calledElement`; subprocess invocation plugin |
| Multi-instance loop | 🔌 | `loopCharacteristics` in IR; loop coordinator plugin |
| Ad-hoc sub-process | 🔌 | IR markers; **control** stream |
| Transaction sub-process | 🔌 | IR markers; compensation via **control** plugin |

## Gateways

| Construct | Status | Notes |
|-----------|--------|-------|
| Exclusive (XOR) | ✅ | First match + optional default flow |
| Parallel (AND) | ✅ | Fork all / join all |
| Inclusive (OR) | ✅ | Fork matching / join all |
| Event-based gateway | 🔌 | IR + runtime wait set; **trigger** stream (first event wins) |
| Complex gateway | 🔌 | Per-flow `activationCondition`; custom evaluator plugin |

## Flows and data

| Construct | Status | Notes |
|-----------|--------|-------|
| Sequence flow | ✅ | Optional condition at gateways |
| Message flow (pool) | 🔌 | IR collaboration; **trigger** + correlation plugin |
| Data object / store | 🔌 | IR refs; maps to instance `variables` + optional store plugin |
| Pool / lane | 🔌 | IR collaboration; lane → assignee via **assignee** + external user/role system |
| Text annotation | ⚠️ | Preserved or ignored in XML import; no runtime effect |

## Expressions

| Feature | Status | Notes |
|---------|--------|-------|
| Gateway conditions | ✅ | `==`, `!=`, compares, truthy, dot paths |
| Start event conditions | ✅ | On `eventDefinition`, not sequence flow |
| FEEL / XPath | 🔌 | Simple subset in core; FEEL/XPath via expression plugin |
| Formal language URI | 🔌 | `language` URI → plugin evaluator |

## Interchange

| Format | Status | Notes |
|--------|--------|-------|
| JSON process definition | ✅ | Primary API and schema |
| BPMN 2.0 XML import | ✅ | `internal/bpmnxml.Parse`; deploy `Content-Type: application/xml` |
| BPMN 2.0 XML export | ✅ | `internal/bpmnxml.Marshal`, file store |
| BPMN DI (diagram) | ⚠️ | Namespace declared; layout not preserved |
| Full spec XML round-trip | ⚠️ | Extension constructs preserved via custom namespace + planned IR fields |

## Runtime semantics

| Feature | Status | Notes |
|---------|--------|-------|
| Process versioning | ✅ | Deploy bumps version; instances pin snapshot |
| Optimistic locking | ✅ | `lock_version` on complete |
| Async execution | ✅ | Job worker optional |
| Business key dedupe | ✅ | Message trigger running-instance skip |
| Compensation | 🔌 | **control** stream + scoped engine hooks |
| Error handling events | 🔌 | Boundary error events via extension dispatch |

## Platform extensions (integration layer)

These are **not OMG BPMN** but required for enterprise workflows — always via integration adapters, never in domain core:

| Extension | Status | Notes |
|-----------|--------|-------|
| User / role management | 🔌 | External auth; **assignee** stream; lane assignee resolution |
| Form designer / rendering | 🔌 | `formKey` / `formUrl` on userTask; business app owns UX |
| Approval modes (或签/会签/顺签) | ✅ | Engine extension on userTask — `approvalMode`, `requiredApprovals` |
| Reject return / scope terminate | ✅ | `returnTo`, `onReject` |
| Dynamic assignees | ✅ | `assigneesVariable` dot path |
| Plugin event streams | ✅ | assignee / trigger / task / control |

## Summary

**Core engine:** JSON and XML process definitions, token flow, XOR/AND/OR gateways, userTask with rich approval, scriptTask, embedded subProcess scopes, message/conditional starts, persistence, plugin ingress.

**Extension-backed (plug in adapters):** Boundary and intermediate events, event-based/complex gateways, call activity, multi-instance, pools/lanes, data objects/stores, service/send/receive/businessRule tasks, user/role/form platforms, compensation, DMN, advanced expression languages.

For DDD layout and extension design see [docs/ddd/extensions.md](./ddd/extensions.md), [docs/ddd/README.md](./ddd/README.md), and [ARCHITECTURE.md](../ARCHITECTURE.md).
