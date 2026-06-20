# Plugin system (Paca-style)

Triple event streams + WASM sandbox with capability permissions.

```
                    ┌─ assignee Consumer ─► AssigneeRuntime ─► feishu/wecom (HR)
Redis / HTTP / Kafka ┼─ trigger Consumer  ─► TriggerRuntime  ─► airtable/feishu (start)
                    └─ task Consumer     ─► TaskRuntime     ─► feishu/wecom (approve/reject)
                              │
                              ▼
                         Host SDK ─► Engine
```

## Three consumers

| Stream | Adapters (default) | Host calls |
|--------|-------------------|------------|
| **assignee** | feishu, wecom, generic, canonical | `RemoveUserFromActiveTasks`, `ReplaceTaskAssignees` |
| **trigger** | feishu, wecom, airtable, generic, canonical | `TriggerMessage`, `StartProcess` |
| **task** | feishu, wecom, generic, canonical | `CompleteTask`, `Terminate` |

Configure separately:

```bash
PLUGIN_ASSIGNEE_ADAPTERS=generic,feishu,wecom
PLUGIN_TRIGGER_ADAPTERS=generic,feishu,wecom,airtable
PLUGIN_TASK_ADAPTERS=generic,feishu,wecom
```

Feishu approval events (`approval.instance.status_changed_v1`) route to **task**, not trigger.

## Canonical actions

Adapters normalize inbound payloads to `CanonicalAction` before calling Host:

| `kind` | Stream | Host method |
|--------|--------|-------------|
| `remove_user` | assignee | `RemoveUserFromActiveTasks` |
| `replace_assignees` | assignee | `ReplaceTaskAssignees` |
| `trigger_message` | trigger | `TriggerMessage` |
| `start_process` | trigger | `StartProcess` |
| `complete_task` | task | `CompleteTask` |
| `terminate` | task | `Terminate` |

Schema: `schemas/adapter-action.schema.json`.

## WASM plugins (capability sandbox)

Like Paca: plugins declare **capabilities** in `plugin.json`; only permitted `bpmn_host` imports are registered.

```json
{
  "name": "my-adapter",
  "stream": "task",
  "sources": ["feishu"],
  "capabilities": ["complete_task", "read_tasks"],
  "wasm": "plugin.wasm"
}
```

```bash
PLUGIN_WASM_DIR=./plugins/wasm
```

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
| `read_tasks` | `bpmn_host.list_tasks` |
| `read_instances` | `bpmn_host.get_process_instance` |
| `read_activities` | `bpmn_host.list_activities` |

Undeclared capabilities → host import not exposed → guest cannot call.

## Event transport

| Env | Default |
|-----|---------|
| `EVENT_CONSUMER` | `memory` |
| `EVENT_REDIS_URL` | (required when `redis`) |
| `EVENT_REDIS_ASSIGNEE_KEY` | `bpmn:events:assignee` |
| `EVENT_REDIS_TRIGGER_KEY` | `bpmn:events:trigger` |
| `EVENT_REDIS_TASK_KEY` | `bpmn:events:task` |

## HTTP ingress

```http
POST /api/v1/events/assignee/feishu   # HR / org change
POST /api/v1/events/trigger/airtable  # process start
POST /api/v1/events/task/feishu       # approve / reject
```

## Go plugins

Implement `contract.EventAdapter` under `plugins/go/<name>/`, register in `plugins/registry/registry.go`.

Shared SDK: `plugins/sdk/` (Action schema, Apply* helpers). WASM: `plugins/wasm/*/plugin.json`.
