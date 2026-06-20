# External plugins

Built-in vendor adapters live here, outside `internal/`. The core engine only provides Host SDK, runtime, and WASM loader.

```
plugins/
  sdk/              # Shared helpers for Go plugins (Action, Apply*)
  registry/         # Name → adapter lookup (used by bootstrap)
  go/
    generic/        # Canonical JSON on each stream (in root module)
    canonical/
    feishu/         # separate go.mod — independent release
    wecom/          # separate go.mod
    airtable/       # separate go.mod
  wasm/
    */plugin.json   # WASM sandbox plugins
```

Root `go.work` includes `./plugins/go/feishu`, `wecom`, `airtable` for multi-module development.

## Go plugin

Implement `contract.EventAdapter` in `plugins/go/<name>/`, then register in `plugins/registry/registry.go`.

Hot vendors (feishu, wecom, airtable) ship as **separate Go modules** with `replace github.com/monoposer/lowcode-bpmn => ../../../` for local dev.

## WASM plugin

See [wasm/example-echo/README.md](./wasm/example-echo/README.md).

## Configuration

```bash
PLUGIN_ASSIGNEE_ADAPTERS=generic,feishu,wecom
PLUGIN_TRIGGER_ADAPTERS=generic,feishu,wecom,airtable
PLUGIN_TASK_ADAPTERS=generic,feishu,wecom
PLUGIN_CONTROL_ADAPTERS=generic,canonical
PLUGIN_WASM_DIR=./plugins/wasm
```

Contract tests: `go test ./plugins/go/feishu/ ./plugins/go/airtable/`
