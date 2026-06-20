package script_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/script"
)

func TestHostAllowlist(t *testing.T) {
	list := script.ParseHostAllowlist("api.example.com,*.internal.local")
	if !list.Allows("api.example.com") {
		t.Fatal("exact host")
	}
	if !list.Allows("svc.internal.local") {
		t.Fatal("wildcard suffix")
	}
	if list.Allows("evil.example.com") {
		t.Fatal("should not allow")
	}
}

func TestPolicyRunnerDisablesHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	runner := &script.PolicyRunner{
		Base: script.NewExecutor(),
		Policy: script.PolicyStore{
			Default: script.SecurityPolicy{HTTPEnabled: true},
			Tenants: map[string]script.SecurityPolicy{
				"no-http": {HTTPDisabled: true},
			},
		},
	}

	_, err := runner.Run(context.Background(), script.RunRequest{
		Script: `
			var resp = http.get("` + srv.URL + `");
			return { ok: resp.status === 200 };
		`,
		Lang:     "js",
		TenantID: "no-http",
	})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected http disabled error, got %v", err)
	}
}

func TestPolicyRunnerHTTPAllowlist(t *testing.T) {
	allowed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer allowed.Close()

	runner := &script.PolicyRunner{
		Base: script.NewExecutor(),
		Policy: script.PolicyStore{
			Default: script.SecurityPolicy{
				HTTPEnabled:    true,
				HTTPAllowHosts: script.ParseHostAllowlist("127.0.0.1,localhost"),
			},
		},
	}

	out, err := runner.Run(context.Background(), script.RunRequest{
		Script: `var resp = http.get("` + allowed.URL + `"); return { ok: resp.status === 200 };`,
		Lang:   "js",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["ok"] != true {
		t.Fatalf("got %v", out)
	}

	_, err = runner.Run(context.Background(), script.RunRequest{
		Script: `return http.get("http://example.com/");`,
		Lang:   "js",
	})
	if err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("expected allowlist error, got %v", err)
	}
}

func TestPolicyRunnerMaxScriptBytes(t *testing.T) {
	runner := &script.PolicyRunner{
		Base: script.NewExecutor(),
		Policy: script.PolicyStore{
			Default: script.SecurityPolicy{MaxScriptBytes: 8},
		},
	}
	_, err := runner.Run(context.Background(), script.RunRequest{
		Script: "return { ok: true }",
		Lang:   "js",
	})
	if err == nil || !strings.Contains(err.Error(), "max length") {
		t.Fatalf("expected max length error, got %v", err)
	}
}

func TestPolicyRunnerForceRemote(t *testing.T) {
	remoteCalled := false
	runner := &script.PolicyRunner{
		Base: script.NewExecutor(),
		Remote: script.FuncRunner(func(ctx context.Context, req script.RunRequest) (map[string]any, error) {
			remoteCalled = true
			return map[string]any{"via": "remote"}, nil
		}),
		Policy: script.PolicyStore{
			Tenants: map[string]script.SecurityPolicy{
				"t1": {ForceRemote: true},
			},
		},
	}
	out, err := runner.Run(context.Background(), script.RunRequest{
		Script: "return {}", Lang: "js", TenantID: "t1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !remoteCalled || out["via"] != "remote" {
		t.Fatalf("expected remote runner, got %v remoteCalled=%v", out, remoteCalled)
	}
}

func TestLangRouterWASMRequiresModule(t *testing.T) {
	router := &script.LangRouter{
		Default: script.NewExecutor(),
		WASM:    nil,
	}
	_, err := router.Run(context.Background(), script.RunRequest{
		Script: "export {}", Lang: "wasm",
	})
	if err == nil || !strings.Contains(err.Error(), "wasm script runner not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLangRouterWASMDelegation(t *testing.T) {
	router := &script.LangRouter{
		Default: script.NewExecutor(),
		WASM: script.FuncRunner(func(ctx context.Context, req script.RunRequest) (map[string]any, error) {
			return map[string]any{"lang": req.Lang}, nil
		}),
	}
	out, err := router.Run(context.Background(), script.RunRequest{
		Script: "ignored", Lang: "webassembly",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["lang"] != "webassembly" {
		t.Fatalf("got %v", out)
	}
}

func TestDefaultRunnerPolicyFromEnv(t *testing.T) {
	t.Setenv("SCRIPT_HTTP_ALLOW_HOSTS", "127.0.0.1,localhost")
	t.Setenv("SCRIPT_MAX_SCRIPT_BYTES", "65536")

	runner := script.DefaultRunner()
	out, err := runner.Run(context.Background(), script.RunRequest{
		Script: "return { ok: true }", Lang: "js", TenantID: "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["ok"] != true {
		t.Fatalf("got %v", out)
	}
}
