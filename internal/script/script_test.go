package script_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/script"
)

func jsReq(body string, vars map[string]any) script.RunRequest {
	return script.RunRequest{Script: body, Lang: "javascript", Variables: vars}
}

func TestExecutorLog(t *testing.T) {
	ex := script.NewExecutor()
	out, err := ex.Run(context.Background(), script.RunRequest{
		Script: "hello", Lang: "log", Variables: map[string]any{"a": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("log should not mutate vars, got %v", out)
	}
}

func TestSetDSLRemoved(t *testing.T) {
	ex := script.NewExecutor()
	out, err := ex.Run(context.Background(), jsReq("set:notified=true", nil))
	if err != nil {
		t.Fatal(err)
	}
	if out["notified"] != nil {
		t.Fatalf("set: prefix is not a DSL anymore, got %v", out)
	}
}

func TestJSReturnAndVarsMerge(t *testing.T) {
	ex := script.NewExecutor()
	out, err := ex.Run(context.Background(), jsReq(`
		vars.notified = true;
		return { extra: 1 };
	`, nil))
	if err != nil {
		t.Fatal(err)
	}
	if out["notified"] != true {
		t.Fatalf("expected vars merge, got %v", out)
	}
	if n, ok := out["extra"].(int64); !ok || n != 1 {
		t.Fatalf("expected return merge, got %v", out)
	}
}

func TestEmptyScript(t *testing.T) {
	ex := script.NewExecutor()
	_, err := ex.Run(context.Background(), script.RunRequest{Script: "", Lang: "js"})
	if err == nil {
		t.Fatal("expected error for empty script")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSContextCancel(t *testing.T) {
	ex := script.NewExecutor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ex.Run(ctx, jsReq("while (true) {}", nil))
	if err == nil {
		t.Fatal("expected cancel error")
	}
}

func TestJSTimeout(t *testing.T) {
	t.Setenv("SCRIPT_TIMEOUT", "50ms")
	ex := script.NewExecutor()
	_, err := ex.Run(context.Background(), jsReq(`
		var deadline = Date.now() + 500;
		while (Date.now() < deadline) {}
	`, nil))
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("expected timeout-related error, got %v", err)
	}
}

func TestUnsupportedScriptLang(t *testing.T) {
	ex := script.NewExecutor()
	_, err := ex.Run(context.Background(), script.RunRequest{
		Script: "x", Lang: "python",
		InstanceID: "inst-1", ElementID: "task-1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported scriptLang") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSHTTPGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hook" {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	ex := script.NewExecutor()
	out, err := ex.Run(context.Background(), jsReq(`
		var resp = http.get("`+srv.URL+`/hook");
		if (resp.status !== 200) throw new Error("bad status");
		return { hook: JSON.parse(resp.body).ok };
	`, nil))
	if err != nil {
		t.Fatal(err)
	}
	if out["hook"] != true {
		t.Fatalf("got %v", out)
	}
}

func TestJSHTTPPost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	ex := script.NewExecutor()
	_, err := ex.Run(context.Background(), jsReq(`
		var resp = http.post("`+srv.URL+`", JSON.stringify({ id: vars.id }));
		if (resp.status !== 201) throw new Error("expected 201");
	`, map[string]any{"id": "42"}))
	if err != nil {
		t.Fatal(err)
	}
}

func TestJSTimeoutRespectsParentCancel(t *testing.T) {
	// Ensure parent cancellation works even without SCRIPT_TIMEOUT.
	_ = os.Unsetenv("SCRIPT_TIMEOUT")
	ex := script.NewExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := ex.Run(ctx, jsReq(`
		var deadline = Date.now() + 500;
		while (Date.now() < deadline) {}
	`, nil))
	if err == nil {
		t.Fatal("expected error from cancelled/timed-out context")
	}
}
