package engine_test

import (
	"context"
	"testing"

	"lowcode-bpmn/internal/bpmn"
	"lowcode-bpmn/internal/engine"
	memstore "lowcode-bpmn/internal/store/memory"
)

func approvalProcess() bpmn.ProcessDefinition {
	return bpmn.ProcessDefinition{
		ID:   "approval",
		Name: "Approval Process",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent, Name: "Start"},
			{ID: "review", Kind: bpmn.KindUserTask, Name: "Review", Assignees: []string{"manager"}},
			{ID: "notify", Kind: bpmn.KindScriptTask, Name: "Notify", Script: "set:notified=true"},
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
	if inst.Variables["notified"] != "true" {
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
			{ID: "high", Kind: bpmn.KindScriptTask, Script: "set:path=high"},
			{ID: "low", Kind: bpmn.KindScriptTask, Script: "set:path=low"},
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

func TestParallelGateway(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "parallel",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "fork", Kind: bpmn.KindParallelGateway},
			{ID: "a", Kind: bpmn.KindScriptTask, Script: "set:a=1"},
			{ID: "b", Kind: bpmn.KindScriptTask, Script: "set:b=2"},
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
	if inst.Variables["a"] != "1" || inst.Variables["b"] != "2" {
		t.Fatalf("expected parallel branches to run, vars=%v", inst.Variables)
	}
}
