package script_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/internal/script"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func scriptRunnerWasmPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..")
	path := filepath.Join(root, "plugins", "wasm", "script-runner", "script.wasm")
	if _, err := os.Stat(path); err != nil {
		t.Skip("script.wasm not built; run plugins/wasm/script-runner/build.sh")
	}
	return path
}

func TestWASMRunnerJSONMerge(t *testing.T) {
	t.Setenv("SCRIPT_WASM_PATH", scriptRunnerWasmPath(t))

	runner := script.NewWASMRunner()
	if runner == nil {
		t.Fatal("expected wasm runner")
	}

	out, err := runner.Run(context.Background(), script.RunRequest{
		Script:    `{"flag":true,"count":1}`,
		Lang:      "wasm",
		Variables: map[string]any{"seed": "x"},
		TenantID:  "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["via"] != "wasm" {
		t.Fatalf("expected wasm marker, got %v", out)
	}
	if out["flag"] != true {
		t.Fatalf("expected merge, got %v", out)
	}
	if out["seed"] != "x" {
		t.Fatalf("expected preserved vars, got %v", out)
	}
}

func TestWASMRunnerHTTPRespectsPolicy(t *testing.T) {
	t.Setenv("SCRIPT_WASM_PATH", scriptRunnerWasmPath(t))

	runner := &script.PolicyRunner{
		Base: script.NewWASMRunner(),
		Policy: script.PolicyStore{
			Default: script.SecurityPolicy{HTTPEnabled: false},
		},
	}
	_, err := runner.Run(context.Background(), script.RunRequest{
		Script: "http:http://127.0.0.1:1/",
		Lang:   "wasm",
	})
	if err == nil {
		t.Fatal("expected http policy error")
	}
}

func TestEngineScriptTaskWASM(t *testing.T) {
	t.Setenv("SCRIPT_WASM_PATH", scriptRunnerWasmPath(t))

	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, &script.LangRouter{
		Default: script.NewExecutor(),
		WASM:    script.NewWASMRunner(),
	})

	def := bpmn.ProcessDefinition{
		ID: "wasm-flow",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "run", Kind: bpmn.KindScriptTask, ScriptLang: "wasm", Script: `{"done":true}`},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "run"},
			{ID: "f2", SourceRef: "run", TargetRef: "end"},
		},
	}

	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: "wasm-flow", Definition: def})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t", ProcessKey: "wasm-flow"})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Variables["done"] != true {
		t.Fatalf("expected wasm output merged, vars=%v", inst.Variables)
	}
	if inst.Variables["via"] != "wasm" {
		t.Fatalf("expected via=wasm, vars=%v", inst.Variables)
	}
}

func TestProductionHardeningFromEnv(t *testing.T) {
	t.Setenv("SCRIPT_PRODUCTION_HARDENING", "true")
	t.Setenv("SCRIPT_UNTRUSTED_TENANTS", "guest")

	runner := script.DefaultRunner()
	_, err := runner.Run(context.Background(), script.RunRequest{
		Script: "return {}", Lang: "js", TenantID: "guest",
	})
	// guest tenant force-remote without SCRIPT_REMOTE_URL configured → error from PolicyRunner path
	if err == nil {
		t.Fatal("expected force-remote error without remote runner configured")
	}
}
