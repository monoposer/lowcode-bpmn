package engine_test

import (
	"context"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func TestAssigneeSyncRemoveUser(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "purchase",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"finance-a", "finance-b", "finance-c"}, ApprovalMode: "all"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}
	inst := deployAndStart(t, eng, store, def, "purchase", nil)

	result, err := eng.RemoveUserFromActiveTasks(ctx, engine.RemoveUserSyncRequest{
		TenantID: "t",
		UserID:   "finance-a",
		Reason:   "resigned",
		Operator: "org-service",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 1 {
		t.Fatalf("expected 1 updated, got %d (%+v)", result.Updated, result)
	}

	tasks, _ := eng.ListUserTasks(ctx, "t", "finance-a")
	if len(tasks) != 0 {
		t.Fatalf("finance-a should have no inbox tasks")
	}
	tasks, _ = eng.ListUserTasks(ctx, "t", "finance-b")
	if len(tasks) != 1 {
		t.Fatalf("finance-b should still have task, got %d", len(tasks))
	}
	_ = inst
}

func TestAssigneeSyncFromVariable(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "dynamic",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{
				ID: "review", Kind: bpmn.KindUserTask,
				AssigneesVariable: "assignees.review",
				ApprovalMode:      "all",
			},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: "dynamic", Definition: def})
	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{
		TenantID: "t", ProcessKey: "dynamic",
		Variables: map[string]any{
			"assignees": map[string]any{
				"review": []any{"picker-a", "picker-b"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	acts, err := eng.ListActivities(ctx, inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	var review *engine.ActivityInstance
	for _, a := range acts {
		if a.ElementID == "review" && a.Status == engine.ActivityStatusActive {
			review = a
			break
		}
	}
	if review == nil {
		t.Fatal("review task not found")
	}
	if len(review.Assignees) != 2 || review.Assignees[0] != "picker-a" {
		t.Fatalf("expected variable assignees, got %v", review.Assignees)
	}
}
