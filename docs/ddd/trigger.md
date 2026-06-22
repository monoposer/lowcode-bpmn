# Triggers and Start Events

Process starts are driven by BPMN 2.0 `startEvent.eventDefinition` — **not** by sequence-flow conditions.

## Start event types

| BPMN 2.0 | JSON `type` | Entry point |
|----------|-------------|-------------|
| None start | `none` / omitted | `POST /api/v1/process-instances` |
| Message start | `message` | `POST /api/v1/triggers/message` or plugin |
| Conditional start | `conditional` | Evaluated on trigger payload |
| Signal start | `signal` | `Engine.TriggerSignal` (metadata + engine scan) |
| Timer start | `timer` | External scheduler; definition stores `timerCycle` |

## Message trigger flow

```
Webhook (Airtable, Feishu, …)
  → POST /api/v1/events/trigger/{source}
  → plugin adapter normalizes payload
  → Host.TriggerMessage
  → Engine scans deployed processes for matching messageRef
  → Starts instance per matching startEvent
```

Direct canonical API (bypass adapter):

```json
POST /api/v1/triggers/message
{
  "tenantId": "demo",
  "messageRef": "airtable.orders.updated",
  "variables": { "event": { "fields": { "status": "Pending" } } },
  "businessKey": "rec123"
}
```

## Matching rules (`definition.MessageStartMatch`)

1. `messageRef` must equal start event's `messageRef`
2. Optional `condition` evaluated against trigger variables
3. Optional `correlationKey` dot path → `businessKey` for dedupe

Running instance dedupe: `(tenant, processKey, businessKey)` — existing running instance may be skipped.

## Example definition

```json
{
  "id": "start",
  "type": "startEvent",
  "eventDefinition": {
    "type": "message",
    "messageRef": "airtable.orders.updated",
    "correlationKey": "event.recordId",
    "condition": "event.fields.status == Pending"
  }
}
```

## Signal and conditional starts

- **Signal** — `SignalStartMatch(el, signalRef, vars)`; engine exposes trigger API for signal dispatch.
- **Conditional** — `ConditionalStartMatch(el, vars)` evaluates `eventDefinition.condition` without messageRef.

## Timer metadata

Timer starts are **not** scheduled by the engine. Definitions store cycle/date expressions; an external cron or scheduler must call start/trigger APIs.

## Plugin integration

Trigger stream adapters (`PLUGIN_TRIGGER_ADAPTERS`) map vendor webhooks to Host SDK `TriggerMessage` / `StartProcess`. See [plugins.md](../plugins.md).

## Common mistake

Do **not** put webhook filter logic on the first sequence flow from a start event. Use `eventDefinition.condition` on the `startEvent` instead. Sequence-flow conditions are for gateway routing only.
