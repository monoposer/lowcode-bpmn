package api

import (
	"net/http"
	"strings"

	"github.com/monoposer/lowcode-bpmn/pkg/env"
)

// AuthConfig controls API key authentication for protected routes.
type AuthConfig struct {
	// Disabled when empty and RequireAuth is false.
	Keys map[string]string // apiKey -> tenantID ("" = any tenant)
	// RequireAuth forces 401 when no keys configured (production mode).
	RequireAuth bool
}

// LoadAuthConfigFromEnv reads API_KEY, API_KEYS, AUTH_REQUIRED.
//
// API_KEY=secret — single global key (any tenant).
// API_KEYS=tenant1:key1,tenant2:key2 — tenant-scoped keys.
// AUTH_REQUIRED=true — reject all /api/* when no keys configured.
func LoadAuthConfigFromEnv() AuthConfig {
	require := env.Bool("AUTH_REQUIRED", false)
	keys := parseAPIKeys(env.Get("API_KEYS", ""))
	if k := strings.TrimSpace(env.Get("API_KEY", "")); k != "" {
		if keys == nil {
			keys = make(map[string]string)
		}
		keys[k] = ""
	}
	return AuthConfig{Keys: keys, RequireAuth: require}
}

func parseAPIKeys(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	out := make(map[string]string)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i := strings.Index(part, ":"); i > 0 {
			tenant := strings.TrimSpace(part[:i])
			key := strings.TrimSpace(part[i+1:])
			if key != "" {
				out[key] = tenant
			}
		} else {
			out[part] = ""
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func extractAPIKey(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if k := r.Header.Get("X-API-Key"); k != "" {
		return k
	}
	return ""
}

// AuthMiddleware protects /api/* routes when keys are configured or AUTH_REQUIRED=true.
func AuthMiddleware(cfg AuthConfig) func(http.Handler) http.Handler {
	enabled := len(cfg.Keys) > 0 || cfg.RequireAuth
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled || !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}
			key := extractAPIKey(r)
			if key == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "missing API key (Authorization: Bearer … or X-API-Key)",
					"code":  "unauthorized",
				})
				return
			}
			tenant, ok := cfg.Keys[key]
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "invalid API key",
					"code":  "unauthorized",
				})
				return
			}
			if tenant != "" {
				r.Header.Set("X-Auth-Tenant", tenant)
			}
			next.ServeHTTP(w, r)
		})
	}
}
