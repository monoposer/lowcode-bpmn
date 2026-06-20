# Plugin system (Paca-style)

Quad event streams + WASM sandbox with capability permissions.

```
                    ┌─ assignee Consumer ─► HR sync
Redis / HTTP / Kafka ┼─ trigger Consumer  ─► process start
                    ┼─ task Consumer     ─► approve / reject
                    └─ control Consumer  ─► terminate
                              │
                              ▼
                         Host SDK ─► Engine
```

## Four consumers

| Stream | Adapters (default) | Host calls |
|--------|-------------------|------------|
| **assignee** | feishu, wecom, generic, canonical | `RemoveUserFromActiveTasks`, `ReplaceTaskAssignees` |
| **trigger** | feishu, wecom, airtable, generic, canonical | `TriggerMessage`, `StartProcess` |
| **task** | feishu, wecom, generic, canonical | `CompleteTask` |
| **control** | generic, canonical | `Terminate` |

Configure separately:

```bash
PLUGIN_ASSIGNEE_ADAPTERS=generic,feishu,wecom
PLUGIN_TRIGGER_ADAPTERS=generic,feishu,wecom,airtable
PLUGIN_TASK_ADAPTERS=generic,feishu,wecom
PLUGIN_CONTROL_ADAPTERS=generic,canonical
```

Feishu approval events route to **task**, not trigger. Terminate routes to **control**.

## Canonical actions

| `kind` | Stream | Host method |
|--------|--------|-------------|
| `remove_user` | assignee | `RemoveUserFromActiveTasks` |
| `replace_assignees` | assignee | `ReplaceTaskAssignees` |
| `trigger_message` | trigger | `TriggerMessage` |
| `start_process` | trigger | `StartProcess` |
| `complete_task` | task | `CompleteTask` |
| `terminate` | control | `Terminate` |

Schema: `schemas/adapter-action.schema.json`.

## WASM plugins (capability sandbox)

See [plugins/wasm/example-echo/README.md](../plugins/wasm/example-echo/README.md).

### Capabilities

| Capability | Host function |
|------------|---------------|
| `trigger_message` | `bpmn_host.trigger_message` |
| `start_process` | `bpmn_host.start_process` |
| `remove_user` | `bpmn_host.remove_user` |
| `replace_assignees` | `bpmn_host.replace_assignees` |
| `complete_task` | `bpmn_host.complete_task` |
| `terminate` | `bpmn_host.terminate` |
| `read_tasks` | `bpmn_host.list_tasks` (+ `outMaxLen`) |
| `read_instances` | `bpmn_host.get_process_instance` (+ `outMaxLen`) |
| `read_activities` | `bpmn_host.list_activities` (+ `outMaxLen`) |

| `413` | WASM host write exceeded guest buffer (`outMaxLen`) |

## Event transport

| Env | Default |
|-----|---------|
| `EVENT_CONSUMER` | `memory` — also `redis`, `kafka`, `nats`, `rabbitmq`, `none` |
| `EVENT_BROKER_URL` | Driver connection URL |
| `EVENT_{STREAM}_DEST` | Per-stream topic / subject / queue / redis key |
| `EVENT_{DRIVER}_*` | Driver options (e.g. `EVENT_KAFKA_GROUP_ID`) |

Legacy: `EVENT_REDIS_URL`, `EVENT_REDIS_*_KEY` when `EVENT_CONSUMER=redis`.

Production HA: `docker compose --profile db --profile redis up -d` with `EVENT_CONSUMER=redis`.

## HTTP ingress

```http
POST /api/v1/events/assignee/feishu
POST /api/v1/events/trigger/airtable
POST /api/v1/events/task/feishu
POST /api/v1/events/control/generic
```

Protected by `API_KEY` / `API_KEYS` when configured (`AUTH_REQUIRED=true` recommended).

## Go plugins

Native adapters: `plugins/go/*` (feishu/wecom/airtable are separate Go modules — see `go.work`).

Register new adapters in `plugins/registry/registry.go`.

Contract tests: `go test ./plugins/go/feishu/ ./plugins/go/airtable/`
