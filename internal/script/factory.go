package script

import (
	"strings"
	"time"

	"github.com/monoposer/lowcode-bpmn/pkg/env"
)

// DefaultRunner returns the production script runner wired from environment variables.
// Engine callers inject this via NewEngine(store, script.DefaultRunner()).
//
// Layer order (outermost first): PolicyRunner → SandboxRunner → RemoteByTenant → LangRouter.
// See phases.go for short / medium / long-term roadmap.
func DefaultRunner() Runner {
	local := NewExecutor()
	wasmRunner := NewWASMRunner()

	var runner Runner = &LangRouter{
		Default: local,
		WASM:    wasmRunner,
	}

	var remote Runner
	if remoteURL := strings.TrimSpace(env.Get("SCRIPT_REMOTE_URL", "")); remoteURL != "" {
		remote = NewHTTPRemoteRunner(remoteURL, nil)
		runner = &RemoteByTenant{
			Local:         runner,
			Remote:        remote,
			RemoteTenants: TenantSet(env.CSV("SCRIPT_REMOTE_TENANTS")...),
			RemoteAll:     env.Bool("SCRIPT_REMOTE_ALL", false),
		}
	}

	if sandbox := loadSandboxConfig(); sandbox != nil {
		runner = &SandboxRunner{
			Base:           runner,
			DefaultTimeout: sandbox.defaultTimeout,
			TenantTimeout:  sandbox.tenantTimeouts,
		}
	}

	policy := loadPolicyStore()
	if policyStoreActive(policy) {
		runner = &PolicyRunner{
			Base:   runner,
			Remote: remote,
			Policy: policy,
		}
	}
	return runner
}

type sandboxConfig struct {
	defaultTimeout time.Duration
	tenantTimeouts map[string]time.Duration
}

func loadSandboxConfig() *sandboxConfig {
	defaultTimeout := parseDurationEnv("SCRIPT_SANDBOX_TIMEOUT")
	tenantTimeouts := parseTenantTimeouts(env.Get("SCRIPT_TENANT_TIMEOUTS", ""))
	if defaultTimeout <= 0 && len(tenantTimeouts) == 0 {
		return nil
	}
	return &sandboxConfig{
		defaultTimeout: defaultTimeout,
		tenantTimeouts: tenantTimeouts,
	}
}

func parseDurationEnv(key string) time.Duration {
	raw := env.Get(key, "")
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0
	}
	return d
}

// parseTenantTimeouts parses "tenant-a:5s,tenant-b:10s".
func parseTenantTimeouts(raw string) map[string]time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := make(map[string]time.Duration)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tenant, value, ok := strings.Cut(part, ":")
		tenant = strings.TrimSpace(tenant)
		value = strings.TrimSpace(value)
		if !ok || tenant == "" || value == "" {
			continue
		}
		d, err := time.ParseDuration(value)
		if err != nil || d <= 0 {
			continue
		}
		out[tenant] = d
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
