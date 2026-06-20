package script

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
)

// SecurityPolicy controls ScriptTask execution constraints for a tenant.
type SecurityPolicy struct {
	HTTPEnabled    bool
	HTTPDisabled   bool // explicit tenant override (SCRIPT_TENANT_NO_HTTP)
	HTTPAllowHosts HostAllowlist
	MaxScriptBytes int
	ForceRemote    bool
}

// PolicyStore holds default and per-tenant security policies.
type PolicyStore struct {
	Default SecurityPolicy
	Tenants map[string]SecurityPolicy
}

func (s PolicyStore) ForTenant(tenantID string) SecurityPolicy {
	if tenantID != "" {
		if p, ok := s.Tenants[tenantID]; ok {
			return mergePolicy(s.Default, p)
		}
	}
	return s.Default
}

func mergePolicy(base, override SecurityPolicy) SecurityPolicy {
	out := base
	if override.HTTPDisabled {
		out.HTTPEnabled = false
	}
	if override.HTTPAllowHosts.Defined() {
		out.HTTPAllowHosts = override.HTTPAllowHosts
	}
	if override.MaxScriptBytes > 0 {
		out.MaxScriptBytes = override.MaxScriptBytes
	}
	if override.ForceRemote {
		out.ForceRemote = true
	}
	return out
}

type policyContextKey struct{}

// WithSecurityPolicy attaches a security policy to the script execution context.
func WithSecurityPolicy(ctx context.Context, policy SecurityPolicy) context.Context {
	return context.WithValue(ctx, policyContextKey{}, policy)
}

// SecurityPolicyFrom returns the policy attached to ctx, if any.
func SecurityPolicyFrom(ctx context.Context) (SecurityPolicy, bool) {
	v, ok := ctx.Value(policyContextKey{}).(SecurityPolicy)
	return v, ok
}

// PolicyRunner enforces tenant security before delegating to Base or Remote.
type PolicyRunner struct {
	Base   Runner
	Remote Runner
	Policy PolicyStore
}

func (p *PolicyRunner) Run(ctx context.Context, req RunRequest) (map[string]any, error) {
	if p == nil || p.Base == nil {
		return nil, errRunnerNotConfigured
	}

	pol := p.Policy.ForTenant(req.TenantID)
	if pol.MaxScriptBytes > 0 && len(req.Script) > pol.MaxScriptBytes {
		attrs := append(slogAttrs(req), slog.Int("maxScriptBytes", pol.MaxScriptBytes))
		slog.WarnContext(ctx, "script rejected: exceeds max size", attrs...)
		return nil, fmt.Errorf("script exceeds max length %d bytes", pol.MaxScriptBytes)
	}

	if pol.ForceRemote {
		if p.Remote == nil {
			return nil, fmt.Errorf("tenant %q requires remote script runner", req.TenantID)
		}
		return p.Remote.Run(ctx, req)
	}

	ctx = WithSecurityPolicy(ctx, pol)
	return p.Base.Run(ctx, req)
}

func httpAllowed(ctx context.Context, rawURL string) error {
	pol, ok := SecurityPolicyFrom(ctx)
	if !ok {
		return nil
	}
	if !pol.HTTPEnabled {
		return fmt.Errorf("http: disabled by tenant security policy")
	}
	if !pol.HTTPAllowHosts.Enforced() {
		return nil
	}
	host, err := hostFromURL(rawURL)
	if err != nil {
		return err
	}
	if !pol.HTTPAllowHosts.Allows(host) {
		return fmt.Errorf("http: host %q not in allowlist", host)
	}
	return nil
}

func hostFromURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("http: invalid url: %w", err)
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		return "", fmt.Errorf("http: missing host")
	}
	return host, nil
}
