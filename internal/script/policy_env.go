package script

import (
	"strings"

	"github.com/monoposer/lowcode-bpmn/pkg/env"
)

// HostAllowlist restricts outbound HTTP hosts from ScriptTask JavaScript.
// Empty patterns with Enforced()==false means no restriction (legacy default).
type HostAllowlist struct {
	patterns []string
}

func ParseHostAllowlist(raw string) HostAllowlist {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return HostAllowlist{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return HostAllowlist{patterns: out}
}

func (h HostAllowlist) Defined() bool {
	return len(h.patterns) > 0
}

func (h HostAllowlist) Enforced() bool {
	return len(h.patterns) > 0
}

func (h HostAllowlist) Allows(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	for _, pattern := range h.patterns {
		if hostMatches(pattern, host) {
			return true
		}
	}
	return false
}

func hostMatches(pattern, host string) bool {
	if pattern == host {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(host, suffix) && host != strings.TrimPrefix(suffix, ".")
	}
	return false
}

func loadPolicyStore() PolicyStore {
	defaultPolicy := SecurityPolicy{
		HTTPEnabled:    env.Bool("SCRIPT_HTTP_ENABLED", true),
		HTTPAllowHosts: ParseHostAllowlist(env.Get("SCRIPT_HTTP_ALLOW_HOSTS", "")),
		MaxScriptBytes: envInt("SCRIPT_MAX_SCRIPT_BYTES", 0),
	}

	tenants := make(map[string]SecurityPolicy)
	for _, tenantID := range env.CSV("SCRIPT_TENANT_NO_HTTP") {
		tenants[tenantID] = SecurityPolicy{HTTPDisabled: true}
	}
	for _, tenantID := range env.CSV("SCRIPT_TENANT_REMOTE") {
		p := tenants[tenantID]
		p.ForceRemote = true
		tenants[tenantID] = p
	}
	for _, part := range env.CSV("SCRIPT_TENANT_ALLOW_HOSTS") {
		tenantID, hosts, ok := strings.Cut(part, ":")
		tenantID = strings.TrimSpace(tenantID)
		hosts = strings.TrimSpace(hosts)
		if !ok || tenantID == "" || hosts == "" {
			continue
		}
		p := tenants[tenantID]
		p.HTTPAllowHosts = ParseHostAllowlist(hosts)
		tenants[tenantID] = p
	}

	store := PolicyStore{Default: defaultPolicy, Tenants: tenants}
	applyProductionHardening(&store)
	return store
}

// applyProductionHardening tightens defaults when SCRIPT_PRODUCTION_HARDENING=true:
//   - HTTP disabled unless SCRIPT_HTTP_ALLOW_HOSTS is set
//   - SCRIPT_UNTRUSTED_TENANTS force remote + no in-process HTTP
func applyProductionHardening(store *PolicyStore) {
	if !env.Bool("SCRIPT_PRODUCTION_HARDENING", false) {
		return
	}
	if !store.Default.HTTPAllowHosts.Enforced() {
		store.Default.HTTPEnabled = false
	}
	for _, tenantID := range env.CSV("SCRIPT_UNTRUSTED_TENANTS") {
		p := store.Tenants[tenantID]
		p.ForceRemote = true
		p.HTTPDisabled = true
		store.Tenants[tenantID] = p
	}
}

func policyStoreActive(store PolicyStore) bool {
	if store.Default.MaxScriptBytes > 0 || !store.Default.HTTPEnabled || store.Default.HTTPAllowHosts.Enforced() {
		return true
	}
	return len(store.Tenants) > 0
}

func envInt(key string, def int) int {
	raw := env.Get(key, "")
	if raw == "" {
		return def
	}
	var n int
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	if n <= 0 {
		return def
	}
	return n
}
