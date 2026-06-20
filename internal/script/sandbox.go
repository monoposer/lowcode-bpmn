package script

import (
	"context"
	"time"
)

// SandboxRunner applies per-tenant execution timeouts on top of another Runner.
// Tenant timeouts override DefaultTimeout when present.
type SandboxRunner struct {
	Base           Runner
	DefaultTimeout time.Duration
	TenantTimeout  map[string]time.Duration
}

func (s *SandboxRunner) Run(ctx context.Context, req RunRequest) (map[string]any, error) {
	if s == nil || s.Base == nil {
		return nil, errRunnerNotConfigured
	}
	timeout := s.DefaultTimeout
	if req.TenantID != "" {
		if t, ok := s.TenantTimeout[req.TenantID]; ok && t > 0 {
			timeout = t
		}
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return s.Base.Run(ctx, req)
}
