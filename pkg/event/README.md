# Event consumer — quad streams

Four independent consumers run in parallel:

| Stream | Purpose | Default Redis key |
|--------|---------|-------------------|
| `assignee` | HR / org change → remove or replace assignees | `bpmn:events:assignee` |
| `trigger` | Webhooks → message start / process trigger | `bpmn:events:trigger` |
| `task` | External approve / reject → CompleteTask | `bpmn:events:task` |
| `control` | Terminate instance / scope | `bpmn:events:control` |

## Backends

| `EVENT_CONSUMER` | Description |
|------------------|-------------|
| `memory` (default) | In-process queues |
| `redis` | Redis lists (`LPUSH` / `BRPOP`) — use for multi-replica deployments |
| `none` | Disable consumers; use direct HTTP APIs only |

## Redis

```bash
EVENT_CONSUMER=redis
EVENT_REDIS_URL=redis://localhost:6379/0
EVENT_REDIS_ASSIGNEE_KEY=bpmn:events:assignee
EVENT_REDIS_TRIGGER_KEY=bpmn:events:trigger
EVENT_REDIS_TASK_KEY=bpmn:events:task
EVENT_REDIS_CONTROL_KEY=bpmn:events:control
```

External producers `LPUSH` JSON-serialized `InboundEvent` (must include `"stream"`).

## HTTP ingress

```http
POST /api/v1/events/assignee/feishu
POST /api/v1/events/trigger/airtable
POST /api/v1/events/task/feishu
POST /api/v1/events/control/generic
```

## Extension points

| Stream | Host SDK actions |
|--------|------------------|
| assignee | `remove_user`, `replace_assignees` |
| trigger | `trigger_message`, `start_process` |
| task | `complete_task` |
| control | `terminate` |

Canonical payload shape: `schemas/adapter-action.schema.json`.

## Adding Kafka

Implement `event.Consumer` per stream — see interface in `event.go`. Do not import Kafka into `internal/engine`.
