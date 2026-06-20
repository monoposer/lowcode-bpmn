# External plugins

Built-in vendor adapters live here, outside `internal/`. The core engine only provides Host SDK, runtime, and WASM loader.

```
plugins/
  sdk/              # Shared helpers for Go plugins (Action, Apply*)
  registry/         # Name → adapter lookup (used by bootstrap)
  go/
    generic/        # Canonical JSON on each stream
    canonical/      # Topic-based canonical actions
    feishu/         # assignee + trigger + task
    wecom/
    airtable/
  wasm/
    */plugin.json   # WASM sandbox plugins
```

## Go plugin

Implement `contract.EventAdapter` in `plugins/go/<name>/`, then register in `plugins/registry/registry.go`.

Use `plugins/sdk` to map vendor payloads to Host calls:

```go
return sdk.ApplyTriggerAction(ctx, host, sdk.Action{
    Kind: "trigger_message", TenantID: tenant, MessageRef: "...", Variables: vars,
})
```

## WASM plugin

See [wasm/example-echo/README.md](./wasm/example-echo/README.md).

## Configuration

Adapter names in env vars match registry keys (`feishu`, `wecom`, `airtable`, `generic`, `canonical`):

```bash
PLUGIN_ASSIGNEE_ADAPTERS=generic,feishu,wecom
PLUGIN_TRIGGER_ADAPTERS=generic,feishu,wecom,airtable
PLUGIN_TASK_ADAPTERS=generic,feishu,wecom
PLUGIN_WASM_DIR=./plugins/wasm
```
