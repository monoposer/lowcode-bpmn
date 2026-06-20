package script_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/internal/script"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func TestTenantRouter(t *testing.T) {
	localCalled := false
	remoteCalled := false

	router := &script.TenantRouter{
		Default: script.FuncRunner(func(ctx context.Context, req script.RunRequest) (map[string]any, error) {
			localCalled = true
			return map[string]any{"runner": "local"}, nil
		}),
		Tenants: map[string]script.Runner{
			"remote-tenant": script.FuncRunner(func(ctx context.Context, req script.RunRequest) (map[string]any, error) {
				remoteCalled = true
				return map[string]any{"runner": "remote"}, nil
			}),
		},
	}

	out, err := router.Run(context.Background(), script.RunRequest{
		Script: "x", Lang: "js", TenantID: "remote-tenant",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !remoteCalled || localCalled {
		t.Fatalf("expected remote runner only, local=%v remote=%v", localCalled, remoteCalled)
	}
	if out["runner"] != "remote" {
		t.Fatalf("got %v", out)
	}

	localCalled = false
	remoteCalled = false
	out, err = router.Run(context.Background(), script.RunRequest{
		Script: "x", Lang: "js", TenantID: "other",
	})
	if err != nil {
		t.Fatal(err)
	}
	if localCalled != true || remoteCalled {
		t.Fatalf("expected local runner, local=%v remote=%v", localCalled, remoteCalled)
	}
	if out["runner"] != "local" {
		t.Fatalf("got %v", out)
	}
}

func TestRemoteByTenant(t *testing.T) {
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/run" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Tenant-Id"); got != "t-remote" {
			t.Fatalf("tenant header %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"variables": map[string]any{"via": "remote"}})
	}))
	defer remote.Close()

	runner := &script.RemoteByTenant{
		Local:  script.NewExecutor(),
		Remote: script.NewHTTPRemoteRunner(remote.URL, nil),
		RemoteTenants: script.TenantSet("t-remote"),
	}

	out, err := runner.Run(context.Background(), script.RunRequest{
		Script: "return { ok: true }", Lang: "js", TenantID: "t-remote",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["via"] != "remote" {
		t.Fatalf("expected remote runner, got %v", out)
	}

	out, err = runner.Run(context.Background(), script.RunRequest{
		Script: "return { ok: true }", Lang: "js", TenantID: "t-local",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["ok"] != true {
		t.Fatalf("expected local runner, got %v", out)
	}
}

func TestHTTPRemoteRunnerError(t *testing.T) {
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":"syntax error"}`))
	}))
	defer remote.Close()

	runner := script.NewHTTPRemoteRunner(remote.URL, nil)
	_, err := runner.Run(context.Background(), script.RunRequest{
		Script: "bad()", Lang: "js", TenantID: "t1", InstanceID: "i1", ElementID: "e1",
	})
	if err == nil || !strings.Contains(err.Error(), "syntax error") {
		t.Fatalf("expected remote error, got %v", err)
	}
}

func TestSandboxRunnerTenantTimeout(t *testing.T) {
	start := time.Now()
	runner := &script.SandboxRunner{
		Base: script.FuncRunner(func(ctx context.Context, req script.RunRequest) (map[string]any, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return map[string]any{"done": true}, nil
			}
		}),
		DefaultTimeout: 500 * time.Millisecond,
		TenantTimeout: map[string]time.Duration{
			"strict": 30 * time.Millisecond,
		},
	}

	_, err := runner.Run(context.Background(), script.RunRequest{
		Script: "x", TenantID: "strict",
	})
	if err == nil {
		t.Fatal("expected sandbox timeout")
	}
	if elapsed := time.Since(start); elapsed > 150*time.Millisecond {
		t.Fatalf("expected fast tenant timeout, took %v", elapsed)
	}
}

func TestDefaultRunnerRemoteEnv(t *testing.T) {
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"variables": map[string]any{"source": "env-remote"}})
	}))
	defer remote.Close()

	t.Setenv("SCRIPT_REMOTE_URL", remote.URL)
	t.Setenv("SCRIPT_REMOTE_ALL", "true")

	runner := script.DefaultRunner()
	out, err := runner.Run(context.Background(), script.RunRequest{
		Script: "ignored", Lang: "js", TenantID: "any",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["source"] != "env-remote" {
		t.Fatalf("got %v", out)
	}
}

func TestEngineCustomRunnerInjection(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()

	custom := script.FuncRunner(func(ctx context.Context, req script.RunRequest) (map[string]any, error) {
		if req.TenantID != "tenant-x" {
			t.Fatalf("tenantId %q", req.TenantID)
		}
		if req.ProcessKey != "inject" {
			t.Fatalf("processKey %q", req.ProcessKey)
		}
		return map[string]any{"injected": req.ElementID}, nil
	})

	eng := engine.NewEngine(store, custom)
	def := bpmn.ProcessDefinition{
		ID: "inject",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "run", Kind: bpmn.KindScriptTask, ScriptLang: "javascript", Script: `return { ignored: true }`},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "run"},
			{ID: "f2", SourceRef: "run", TargetRef: "end"},
		},
	}
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "tenant-x", Key: "inject", Definition: def})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "tenant-x", ProcessKey: "inject"})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Variables["injected"] != "run" {
		t.Fatalf("custom runner output not merged, vars=%v", inst.Variables)
	}
}
