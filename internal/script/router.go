package script

import "context"

// TenantRouter delegates ScriptTask execution to a tenant-specific Runner.
// Unlisted tenants use Default; when Default is nil, errRunnerNotConfigured is returned.
type TenantRouter struct {
	Default Runner
	Tenants map[string]Runner
}

func (r *TenantRouter) Run(ctx context.Context, req RunRequest) (map[string]any, error) {
	if r == nil {
		return nil, errRunnerNotConfigured
	}
	if req.TenantID != "" {
		if tenantRunner, ok := r.Tenants[req.TenantID]; ok && tenantRunner != nil {
			return tenantRunner.Run(ctx, req)
		}
	}
	if r.Default == nil {
		return nil, errRunnerNotConfigured
	}
	return r.Default.Run(ctx, req)
}

// RemoteByTenant routes listed tenants (or all when RemoteAll) to a remote Runner.
type RemoteByTenant struct {
	Local         Runner
	Remote        Runner
	RemoteTenants map[string]struct{}
	RemoteAll     bool
}

func (r *RemoteByTenant) Run(ctx context.Context, req RunRequest) (map[string]any, error) {
	if r == nil || r.Remote == nil {
		return nil, errRunnerNotConfigured
	}
	if r.shouldUseRemote(req.TenantID) {
		return r.Remote.Run(ctx, req)
	}
	if r.Local == nil {
		return nil, errRunnerNotConfigured
	}
	return r.Local.Run(ctx, req)
}

func (r *RemoteByTenant) shouldUseRemote(tenantID string) bool {
	if r.RemoteAll {
		return true
	}
	if tenantID == "" || len(r.RemoteTenants) == 0 {
		return false
	}
	_, ok := r.RemoteTenants[tenantID]
	return ok
}

// TenantSet builds a lookup set from tenant ids.
func TenantSet(tenantIDs ...string) map[string]struct{} {
	if len(tenantIDs) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(tenantIDs))
	for _, id := range tenantIDs {
		if id != "" {
			out[id] = struct{}{}
		}
	}
	return out
}
