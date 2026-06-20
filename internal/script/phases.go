package script

// Script execution roadmap — engine stays on script.Runner; layers compose here.
//
// Short term (done):
//   - Context timeout + goja Interrupt; slog with tenant/process/instance/element ids
//   - Runner injection: FuncRunner, TenantRouter, RemoteByTenant, HTTPRemoteRunner, SandboxRunner
//   - Engine tests for JS merge, JS error → activity failed, empty script validation
//
// Medium term (done):
//   - PolicyRunner: tenant HTTP off, host allowlist, max script bytes, force-remote tenants
//   - LangRouter: javascript | wasm | log without engine forks
//   - DefaultRunner env wiring; SCRIPT_PRODUCTION_HARDENING + SCRIPT_UNTRUSTED_TENANTS
//
// Long term (done / operational):
//   - WASM script modules: plugins/wasm/script-runner (TinyGo + script.json capabilities)
//   - Capability-gated script_host imports (log, http_fetch) aligned with plugin/wasm
//   - Untrusted tenants: SCRIPT_TENANT_REMOTE / force-remote + WASM or HTTP remote runners
