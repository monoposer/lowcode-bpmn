package engine_test

import (
	"context"
	"strings"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func approvalProcess() bpmn.ProcessDefinition {
	return bpmn.ProcessDefinition{
		ID:   "approval",
		Name: "Approval Process",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent, Name: "Start"},
			{ID: "review", Kind: bpmn.KindUserTask, Name: "Review", Assignees: []string{"manager"}},
			{ID: "notify", Kind: bpmn.KindScriptTask, Name: "Notify", ScriptLang: "javascript", Script: "return { notified: true }"},
			{ID: "end", Kind: bpmn.KindEndEvent, Name: "Done"},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "notify"},
			{ID: "f3", SourceRef: "notify", TargetRef: "end"},
		},
	}
}

func TestAutoCompleteUserTask(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := approvalProcess()
	def.Elements[1].AutoComplete = true

	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{
		TenantID: "tenant-1", Key: "approval", Version: 1, Definition: def,
	})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{
		TenantID: "tenant-1", ProcessKey: "approval", BusinessKey: "PO-1",
		Variables: map[string]any{"amount": 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Status != engine.ProcessStatusCompleted {
		t.Fatalf("expected completed, got %s active=%v", inst.Status, inst.ActiveElements)
	}
	if inst.Variables["notified"] != true {
		t.Fatalf("expected script to set notified=true, got %v", inst.Variables["notified"])
	}
}

func TestHumanTaskCompletion(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	def := approvalProcess()

	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{
		TenantID: "t1", Key: "approval", Definition: def,
	})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t1", ProcessKey: "approval"})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Status != engine.ProcessStatusRunning {
		t.Fatalf("expected running, got %s", inst.Status)
	}

	active, err := store.ListActiveActivities(ctx, inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].ElementID != "review" {
		t.Fatalf("expected active review task, got %+v", active)
	}

	inst, err = eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        active[0].ID,
		Variables:         map[string]any{"approved": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Status != engine.ProcessStatusCompleted {
		t.Fatalf("expected completed after task, got %s", inst.Status)
	}
}

func TestExclusiveGateway(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "branch",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "gw", Kind: bpmn.KindExclusiveGateway},
			{ID: "high", Kind: bpmn.KindScriptTask, ScriptLang: "javascript", Script: `return { path: "high" }`},
			{ID: "low", Kind: bpmn.KindScriptTask, ScriptLang: "javascript", Script: `return { path: "low" }`},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "gw"},
			{ID: "f2", SourceRef: "gw", TargetRef: "high", Condition: "amount >= 1000"},
			{ID: "f3", SourceRef: "gw", TargetRef: "low", IsDefault: true},
			{ID: "f4", SourceRef: "high", TargetRef: "end"},
			{ID: "f5", SourceRef: "low", TargetRef: "end"},
		},
	}
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: "branch", Definition: def})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{
		TenantID: "t", ProcessKey: "branch", Variables: map[string]any{"amount": 1500},
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Variables["path"] != "high" {
		t.Fatalf("expected high path, got %v", inst.Variables["path"])
	}
}

func TestExclusiveGateway_nestedCondition(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "branch-nested",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "gw", Kind: bpmn.KindExclusiveGateway},
			{ID: "high", Kind: bpmn.KindScriptTask, ScriptLang: "javascript", Script: `return { path: "high" }`},
			{ID: "low", Kind: bpmn.KindScriptTask, ScriptLang: "javascript", Script: `return { path: "low" }`},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "gw"},
			{ID: "f2", SourceRef: "gw", TargetRef: "high", Condition: "item.kk >= 10"},
			{ID: "f3", SourceRef: "gw", TargetRef: "low", IsDefault: true},
			{ID: "f4", SourceRef: "high", TargetRef: "end"},
			{ID: "f5", SourceRef: "low", TargetRef: "end"},
		},
	}
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: "branch-nested", Definition: def})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{
		TenantID: "t", ProcessKey: "branch-nested", Variables: map[string]any{
			"item": map[string]any{"kk": 11},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Variables["path"] != "high" {
		t.Fatalf("expected high path via item.kk, got %v", inst.Variables["path"])
	}
}

func TestParallelGateway(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "parallel",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "fork", Kind: bpmn.KindParallelGateway},
			{ID: "a", Kind: bpmn.KindScriptTask, ScriptLang: "javascript", Script: `return { a: 1 }`},
			{ID: "b", Kind: bpmn.KindScriptTask, ScriptLang: "javascript", Script: `return { b: 2 }`},
			{ID: "join", Kind: bpmn.KindParallelGateway},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "fork"},
			{ID: "f2", SourceRef: "fork", TargetRef: "a"},
			{ID: "f3", SourceRef: "fork", TargetRef: "b"},
			{ID: "f4", SourceRef: "a", TargetRef: "join"},
			{ID: "f5", SourceRef: "b", TargetRef: "join"},
			{ID: "f6", SourceRef: "join", TargetRef: "end"},
		},
	}
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: "parallel", Definition: def})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t", ProcessKey: "parallel"})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Status != engine.ProcessStatusCompleted {
		t.Fatalf("expected completed, got %s", inst.Status)
	}
	if inst.Variables["a"] != int64(1) || inst.Variables["b"] != int64(2) {
		t.Fatalf("expected parallel branches to run, vars=%v", inst.Variables)
	}
}

func TestScriptTaskJSReturnMergeVariables(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "script-merge",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "run", Kind: bpmn.KindScriptTask, ScriptLang: "javascript", Script: `
				vars.seed = vars.input * 2;
				return { output: vars.seed + 1 };
			`},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "run"},
			{ID: "f2", SourceRef: "run", TargetRef: "end"},
		},
	}
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: "script-merge", Definition: def})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{
		TenantID: "t", ProcessKey: "script-merge", Variables: map[string]any{"input": 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Variables["seed"] != int64(10) {
		t.Fatalf("expected vars mutation merge, got seed=%v", inst.Variables["seed"])
	}
	if inst.Variables["output"] != int64(11) {
		t.Fatalf("expected return merge, got output=%v", inst.Variables["output"])
	}
}

func TestScriptTaskJSErrorMarksActivityFailed(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "script-error",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "run", Kind: bpmn.KindScriptTask, ScriptLang: "javascript", Script: `throw new Error("boom");`},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "run"},
			{ID: "f2", SourceRef: "run", TargetRef: "end"},
		},
	}
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: "script-error", Definition: def})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t", ProcessKey: "script-error"})
	if err == nil {
		t.Fatal("expected start to fail on script error")
	}
	if inst.Status != engine.ProcessStatusFailed {
		t.Fatalf("expected instance failed, got %s", inst.Status)
	}

	acts, err := eng.ListActivities(ctx, inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	var scriptAct *engine.ActivityInstance
	for _, a := range acts {
		if a.ElementID == "run" {
			scriptAct = a
			break
		}
	}
	if scriptAct == nil {
		t.Fatal("script activity not found")
	}
	if scriptAct.Status != engine.ActivityStatusFailed {
		t.Fatalf("expected activity failed, got %s", scriptAct.Status)
	}
	if scriptAct.ErrorMsg == "" {
		t.Fatal("expected activity error message")
	}
}

func TestScriptTaskEmptyScriptFails(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "script-empty",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "run", Kind: bpmn.KindScriptTask, ScriptLang: "javascript", Script: ""},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "run"},
			{ID: "f2", SourceRef: "run", TargetRef: "end"},
		},
	}
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: "script-empty", Definition: def})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t", ProcessKey: "script-empty"})
	if err == nil {
		t.Fatal("expected start to fail on empty script")
	}
	if inst != nil {
		t.Fatalf("expected nil instance on definition validation error, got %v", inst)
	}
	if !strings.Contains(err.Error(), "requires script") {
		t.Fatalf("unexpected error: %v", err)
	}
}
