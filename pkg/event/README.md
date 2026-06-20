# Event consumer — quad streams

Four independent consumers run in parallel:

| Stream | Purpose | Default destination |
|--------|---------|---------------------|
| `assignee` | HR / org change → remove or replace assignees | see driver below |
| `trigger` | Webhooks → message start / process trigger | |
| `task` | External approve / reject → CompleteTask | |
| `control` | Terminate instance / scope | |

## Driver registry (scheme A)

Set the transport driver with `EVENT_CONSUMER`. All drivers implement `event.Consumer` and share the same envelope (`InboundEvent` JSON).

| `EVENT_CONSUMER` | `EVENT_BROKER_URL` example | Default destination prefix |
|------------------|----------------------------|----------------------------|
| `memory` (default) | (ignored) | n/a |
| `redis` | `redis://localhost:6379/0` | `bpmn:events:{stream}` |
| `kafka` | `kafka://localhost:9092` | `bpmn.events.{stream}` (topic) |
| `nats` | `nats://localhost:4222` | `bpmn.events.{stream}` (subject) |
| `rabbitmq` | `amqp://guest:guest@localhost:5672/` | `bpmn.events.{stream}` (queue) |
| `none` | — | disables consumers |

### Unified env vars

```bash
EVENT_CONSUMER=kafka
EVENT_BROKER_URL=kafka://localhost:9092

# Optional per-stream override (topic / subject / queue / redis key)
EVENT_ASSIGNEE_DEST=bpmn.events.assignee
EVENT_TRIGGER_DEST=bpmn.events.trigger
EVENT_TASK_DEST=bpmn.events.task
EVENT_CONTROL_DEST=bpmn.events.control

# Driver-specific options (prefix EVENT_{DRIVER}_*)
EVENT_KAFKA_GROUP_ID=lowcode-bpmn
EVENT_KAFKA_BROKERS=broker1:9092,broker2:9092   # overrides single host from BROKER_URL
EVENT_NATS_QUEUE=lowcode-bpmn
```

Legacy Redis keys (`EVENT_REDIS_URL`, `EVENT_REDIS_*_KEY`) remain supported when `EVENT_CONSUMER=redis`.

## HTTP ingress

```http
POST /api/v1/events/assignee/feishu
POST /api/v1/events/trigger/airtable
POST /api/v1/events/task/feishu
POST /api/v1/events/control/generic
```

HTTP publishes into the configured driver via `RouterPublisher`.

## Adding a driver

1. Implement `event.Consumer` under `pkg/event/{name}/`
2. Register in `init()`: `transport.Register(driver{})`
3. Blank-import the package from `pkg/event/setup/setup.go`

Do not import MQ clients into `internal/engine` or plugin adapters.

## Extension points (adapters)

| Stream | Host SDK actions |
|--------|------------------|
| assignee | `remove_user`, `replace_assignees` |
| trigger | `trigger_message`, `start_process` |
| task | `complete_task` |
| control | `terminate` |

Canonical payload: `schemas/adapter-action.schema.json`.
