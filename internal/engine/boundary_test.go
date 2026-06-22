package engine_test

import (
	"context"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func boundaryProcess() bpmn.ProcessDefinition {
	return bpmn.ProcessDefinition{
		ID: "boundary-flow",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"alice"}},
			{
				ID: "msg-boundary", Kind: bpmn.KindBoundaryEvent, AttachedToRef: "review",
				EventDefinition: &bpmn.EventDefinition{Type: bpmn.EventTypeMessage, MessageRef: "escalate"},
			},
			{ID: "escalate-task", Kind: bpmn.KindScriptTask, Script: "return { escalated: true }", ScriptLang: "log"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
			{ID: "bf1", SourceRef: "msg-boundary", TargetRef: "escalate-task"},
			{ID: "bf2", SourceRef: "escalate-task", TargetRef: "end"},
		},
	}
}

func TestTriggerMessageFiresBoundaryAndCancelsHost(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	def := boundaryProcess()
	if _, err := eng.DeployProcess(ctx, "t", "boundary-flow", def); err != nil {
		t.Fatal(err)
	}
	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{
		TenantID: "t", ProcessKey: "boundary-flow",
	})
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := eng.ListUserTasks(ctx, "t", "alice")
	if err != nil || len(tasks) == 0 {
		t.Fatalf("expected inbox task, tasks=%v err=%v", tasks, err)
	}

	result, err := eng.TriggerMessage(ctx, engine.TriggerMessageRequest{
		TenantID: "t", MessageRef: "escalate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.BoundaryMatches) != 1 {
		t.Fatalf("boundary matches: %+v", result.BoundaryMatches)
	}
	if result.BoundaryMatches[0].Error != "" {
		t.Fatalf("boundary error: %s", result.BoundaryMatches[0].Error)
	}
	if !result.BoundaryMatches[0].CancelledHost {
		t.Fatal("expected host cancelled")
	}

	inst, _ = eng.GetProcessInstance(ctx, inst.ID)
	if contains(inst.ActiveElements, "review") {
		t.Fatalf("review should not be active: %v", inst.ActiveElements)
	}
	tasks, _ = eng.ListUserTasks(ctx, "t", "alice")
	if len(tasks) != 0 {
		t.Fatalf("inbox should be empty after cancel, got %d", len(tasks))
	}
}

func TestTriggerBoundaryTimerExplicit(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	def := bpmn.ProcessDefinition{
		ID: "timer-boundary",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "wait", Kind: bpmn.KindUserTask, Assignees: []string{"bob"}, AutoComplete: false},
			{
				ID: "timer1", Kind: bpmn.KindBoundaryEvent, AttachedToRef: "wait",
				EventDefinition: &bpmn.EventDefinition{Type: bpmn.EventTypeTimer, TimerCycle: "PT1H"},
			},
			{ID: "timeout", Kind: bpmn.KindScriptTask, Script: "x", ScriptLang: "log"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "wait"},
			{ID: "f2", SourceRef: "wait", TargetRef: "end"},
			{ID: "tf1", SourceRef: "timer1", TargetRef: "timeout"},
			{ID: "tf2", SourceRef: "timeout", TargetRef: "end"},
		},
	}
	if _, err := eng.DeployProcess(ctx, "t", "timer-boundary", def); err != nil {
		t.Fatal(err)
	}
	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t", ProcessKey: "timer-boundary"})
	if err != nil {
		t.Fatal(err)
	}
	match, err := eng.TriggerBoundary(ctx, engine.TriggerBoundaryRequest{
		TenantID: "t", ProcessInstanceID: inst.ID, BoundaryElementID: "timer1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !match.CancelledHost {
		t.Fatal("timer boundary should cancel host")
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
