# JSON Schema — process definition protocol

Canonical JSON schemas for the lowcode-bpmn engine. Use these in designers, adapters, and CI validation.

| File | Purpose |
|------|---------|
| [process-definition.schema.json](./process-definition.schema.json) | Deployable BPMN process (`PUT /api/v1/tenants/{tenantId}/processes/{key}`) |
| [trigger-message.schema.json](./trigger-message.schema.json) | Direct message trigger (`POST /api/v1/triggers/message`) |
| [inbound-event.schema.json](./inbound-event.schema.json) | Plugin envelope (`POST /api/v1/events/{assignee|trigger|task}/{source}`) |
| [adapter-action.schema.json](./adapter-action.schema.json) | Canonical adapter intent before Host SDK calls |

Plugin architecture: [docs/plugins.md](../docs/plugins.md).

## BPMN 2.0: startEvent vs sequenceFlow condition

| Concept | BPMN 2.0 | JSON field | When evaluated |
|---------|----------|------------|----------------|
| **Start trigger** | `messageEventDefinition`, `conditionalEventDefinition`, etc. | `startEvent.eventDefinition` | Before instance is created (webhook / adapter) |
| **Routing** | Sequence flow condition on gateway outgoing | `flows[].condition` | When token leaves exclusive/inclusive gateway |

**Do not** put Airtable filter logic on the first sequence flow after `startEvent`. Model it as:

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

External flow:

```
Airtable webhook → POST /api/v1/events/trigger/airtable → AirtableAdapter → Host.TriggerMessage
Feishu 离职 → POST /api/v1/events/assignee/feishu → FeishuAssigneeAdapter → Host.RemoveUserFromActiveTasks
Feishu 审批通过 → POST /api/v1/events/task/feishu → FeishuTaskAdapter → Host.CompleteTask
```

Or pre-normalized: `POST /api/v1/triggers/message`.

Manual / API start (`POST /api/v1/process-instances`) ignores `eventDefinition` and still works for `type: none` or testing.

## Validate locally

```bash
# with ajv-cli (npm i -g ajv-cli)
ajv validate -s schemas/process-definition.schema.json -d your-process.json
```
