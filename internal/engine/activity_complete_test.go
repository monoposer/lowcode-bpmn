package engine_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func TestCompleteActivityEventBasedGateway(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	parentDef := bpmn.ProcessDefinition{
		ID: "parent",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "egw", Kind: bpmn.KindEventBasedGateway},
			{ID: "path-a", Kind: bpmn.KindScriptTask, Script: "a", ScriptLang: "log"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "egw"},
			{ID: "f2", SourceRef: "egw", TargetRef: "path-a", Condition: "route == a"},
			{ID: "f3", SourceRef: "path-a", TargetRef: "end"},
		},
	}
	if _, err := eng.DeployProcess(ctx, "t", "parent", parentDef); err != nil {
		t.Fatal(err)
	}
	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{
		TenantID: "t", ProcessKey: "parent", Variables: map[string]any{"route": "a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	acts, err := eng.ListActivities(ctx, inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	var egwActID uuid.UUID
	for _, a := range acts {
		if a.ElementID == "egw" && a.Status == engine.ActivityStatusActive {
			egwActID = a.ID
		}
	}
	if egwActID == uuid.Nil {
		t.Fatal("eventBasedGateway should be active waiting extension")
	}
	_, err = eng.CompleteActivity(ctx, engine.CompleteActivityRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        egwActID,
		SelectedFlowID:    "f2",
	})
	if err != nil {
		t.Fatal(err)
	}
	inst, _ = eng.GetProcessInstance(ctx, inst.ID)
	if inst.Status != engine.ProcessStatusCompleted {
		t.Fatalf("expected completed, got %s", inst.Status)
	}
}

func TestCompleteActivityCallActivity(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	childDef := bpmn.ProcessDefinition{
		ID: "child-proc",
		Elements: []bpmn.Element{
			{ID: "s", Kind: bpmn.KindStartEvent},
			{ID: "e", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{{ID: "f", SourceRef: "s", TargetRef: "e"}},
	}
	parentDef := bpmn.ProcessDefinition{
		ID: "parent-proc",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "call", Kind: bpmn.KindCallActivity, CalledElement: "child-proc"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "call"},
			{ID: "f2", SourceRef: "call", TargetRef: "end"},
		},
	}
	if _, err := eng.DeployProcess(ctx, "t", "child-proc", childDef); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.DeployProcess(ctx, "t", "parent-proc", parentDef); err != nil {
		t.Fatal(err)
	}
	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t", ProcessKey: "parent-proc"})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Status != engine.ProcessStatusRunning {
		t.Fatalf("parent should wait on call activity, got %s", inst.Status)
	}
	acts, _ := eng.ListActivities(ctx, inst.ID)
	var callActID uuid.UUID
	for _, a := range acts {
		if a.ElementID == "call" && a.Status == engine.ActivityStatusActive {
			callActID = a.ID
		}
	}
	if callActID == uuid.Nil {
		t.Fatal("call activity should be active")
	}
	_, err = eng.CompleteActivity(ctx, engine.CompleteActivityRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        callActID,
	})
	if err != nil {
		t.Fatal(err)
	}
	inst, _ = eng.GetProcessInstance(ctx, inst.ID)
	if inst.Status != engine.ProcessStatusCompleted {
		t.Fatalf("expected parent completed, got %s", inst.Status)
	}
}

func TestEvaluateComplexGateway(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	def := bpmn.ProcessDefinition{
		ID: "cx",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "cx-gw", Kind: bpmn.KindComplexGateway},
			{ID: "a", Kind: bpmn.KindScriptTask, Script: "1", ScriptLang: "log"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "cx-gw"},
			{ID: "fa", SourceRef: "cx-gw", TargetRef: "a", Condition: "x >= 10"},
			{ID: "f2", SourceRef: "a", TargetRef: "end"},
		},
	}
	if _, err := eng.DeployProcess(ctx, "t", "cx", def); err != nil {
		t.Fatal(err)
	}
	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{
		TenantID: "t", ProcessKey: "cx", Variables: map[string]any{"x": 15},
	})
	if err != nil {
		t.Fatal(err)
	}
	matched, err := eng.EvaluateComplexGateway(ctx, inst.ID, "cx-gw")
	if err != nil {
		t.Fatal(err)
	}
	if len(matched) != 1 || matched[0] != "fa" {
		t.Fatalf("matched flows: %v", matched)
	}
}
