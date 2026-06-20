# WASM plugin example

Build with TinyGo (guest must export `alloc` and `handle`):

```bash
tinygo build -o plugin.wasm -target=wasi main.go
```

Guest imports module **`bpmn_host`**:

| Import | Capability required | Args (i32) | Returns |
|--------|---------------------|------------|---------|
| `trigger_message` | `trigger_message` | ptr, len (JSON) | status code |
| `start_process` | `start_process` | ptr, len | status code |
| `remove_user` | `remove_user` | ptr, len | status code |
| `replace_assignees` | `replace_assignees` | ptr, len | status code |
| `complete_task` | `complete_task` | ptr, len | status code |
| `terminate` | `terminate` | ptr, len | status code |
| `list_tasks` | `read_tasks` | tenant ptr/len, assignee ptr/len, out ptr | status code |
| `get_process_instance` | `read_instances` | id ptr/len, out ptr | status code |
| `list_activities` | `read_activities` | id ptr/len, out ptr | status code |

## Guest exports

- `alloc(size i32) -> i32` — allocate buffer in guest memory
- `handle(ptr i32, len i32) -> i32` — receives full `InboundEvent` JSON; return 0 on success

## Deploy

```bash
export PLUGIN_WASM_DIR=/path/to/lowcode-bpmn/plugins/wasm
# restart server — loads */plugin.json + plugin.wasm
```

Capabilities not declared in `plugin.json` are **not** registered as host imports (Paca-style sandbox).
