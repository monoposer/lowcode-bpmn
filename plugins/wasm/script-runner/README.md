# WASM ScriptTask runner

Build guest module (requires Docker):

```bash
./build.sh
```

## Guest exports

| Export | Signature | Purpose |
|--------|-----------|---------|
| `alloc` | `(size i32) -> i32` | Guest heap allocation |
| `run` | `(in_ptr, in_len, out_ptr, out_max i32) -> i32` | Run script; returns bytes written |

## Guest imports (`script_host`, capability-gated)

| Import | Capability | Args | Returns |
|--------|------------|------|---------|
| `log` | `log` | msg ptr, len | status |
| `http_fetch` | `http_fetch` | url ptr, len, out ptr, out max | bytes written or 0 |

Host enforces tenant `SecurityPolicy` (HTTP allowlist / disable) on `http_fetch`.

## Script body convention (example guest)

- JSON object string merged into variables: `{"approved":true}`
- `http:https://api.example.com/hook` — host fetch via `http_fetch`

## Deploy

```bash
export SCRIPT_WASM_PATH=./plugins/wasm/script-runner/script.wasm
# per-tenant: SCRIPT_WASM_DIR=./plugins/wasm/scripts  →  {tenant}/script.wasm + script.json
```

Use `scriptLang: "wasm"` on ScriptTask elements.

Capabilities not listed in `script.json` are not registered as host imports.
