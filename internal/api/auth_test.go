package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/api"
)

func TestAuthMiddleware(t *testing.T) {
	cfg := api.AuthConfig{Keys: map[string]string{"secret": "t1"}}
	h := api.AuthMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("skip health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("healthz: %d", rec.Code)
		}
	})

	t.Run("reject missing key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/process-instances", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("accept bearer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/process-instances", nil)
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if req.Header.Get("X-Auth-Tenant") != "t1" {
			t.Fatalf("expected tenant t1")
		}
	})
}

func TestParseAPIKeysEnv(t *testing.T) {
	t.Setenv("API_KEY", "")
	t.Setenv("API_KEYS", "demo:abc123,global-key")
	cfg := api.LoadAuthConfigFromEnv()
	if cfg.Keys["abc123"] != "demo" || cfg.Keys["global-key"] != "" {
		t.Fatalf("unexpected keys: %v", cfg.Keys)
	}
}
