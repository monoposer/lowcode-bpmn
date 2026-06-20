package engine_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func deployAndStart(t *testing.T, eng *engine.Engine, store *memstore.Store, def bpmn.ProcessDefinition, key string, vars map[string]any) *engine.ProcessInstance {
	t.Helper()
	ctx := context.Background()
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: key, Definition: def})
	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{
		TenantID: "t", ProcessKey: key, Variables: vars,
	})
	if err != nil {
		t.Fatal(err)
	}
	return inst
}

func activeUserTask(t *testing.T, eng *engine.Engine, instID uuid.UUID) *engine.ActivityInstance {
	t.Helper()
	ctx := context.Background()
	acts, err := eng.ListActivities(ctx, instID)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range acts {
		if a.ElementKind == bpmn.KindUserTask && a.Status == engine.ActivityStatusActive {
			return a
		}
	}
	t.Fatal("no active user task")
	return nil
}

func TestApprovalAny(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "any",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"a", "b"}, ApprovalMode: "any"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}
	inst := deployAndStart(t, eng, store, def, "any", nil)
	act := activeUserTask(t, eng, inst.ID)

	_, err := eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        act.ID,
		Assignee:          "b",
		Variables:         map[string]any{"approved": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	inst, _ = eng.GetProcessInstance(ctx, inst.ID)
	if inst.Status != engine.ProcessStatusCompleted {
		t.Fatalf("expected completed, got %s", inst.Status)
	}
}

func TestApprovalAnyRequiredTwo(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "any2",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{
				ID: "review", Kind: bpmn.KindUserTask,
				Assignees: []string{"a", "b", "c"}, ApprovalMode: "any", RequiredApprovals: 2,
			},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}
	inst := deployAndStart(t, eng, store, def, "any2", nil)
	act := activeUserTask(t, eng, inst.ID)

	inst, err := eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        act.ID,
		Assignee:          "a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Status == engine.ProcessStatusCompleted {
		t.Fatal("expected still running after 1/2 approvals")
	}

	tasks, _ := eng.ListUserTasks(ctx, "t", "b")
	if len(tasks) != 1 {
		t.Fatalf("b should still see task, got %d", len(tasks))
	}
	tasks, _ = eng.ListUserTasks(ctx, "t", "a")
	if len(tasks) != 0 {
		t.Fatalf("a already approved, inbox should be empty")
	}

	act2 := activeUserTask(t, eng, inst.ID)
	_, err = eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        act2.ID,
		Assignee:          "b",
	})
	if err != nil {
		t.Fatal(err)
	}
	inst, _ = eng.GetProcessInstance(ctx, inst.ID)
	if inst.Status != engine.ProcessStatusCompleted {
		t.Fatalf("expected completed after 2/2, got %s", inst.Status)
	}
}

func TestApprovalAll(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "all",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"a", "b"}, ApprovalMode: "all"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}
	inst := deployAndStart(t, eng, store, def, "all", nil)
	act := activeUserTask(t, eng, inst.ID)

	inst, err := eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        act.ID,
		Assignee:          "a",
		Variables:         map[string]any{"approved": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Status == engine.ProcessStatusCompleted {
		t.Fatal("expected still running after first countersign")
	}

	act2 := activeUserTask(t, eng, inst.ID)
	_, err = eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        act2.ID,
		Assignee:          "b",
		Variables:         map[string]any{"approved": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	inst, _ = eng.GetProcessInstance(ctx, inst.ID)
	if inst.Status != engine.ProcessStatusCompleted {
		t.Fatalf("expected completed after all countersign, got %s", inst.Status)
	}
}

func TestApprovalSequential(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "seq",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"a", "b", "c"}, ApprovalMode: "sequential"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}
	inst := deployAndStart(t, eng, store, def, "seq", nil)

	tasks, err := eng.ListUserTasks(ctx, "t", "b")
	if err != nil || len(tasks) != 0 {
		t.Fatalf("b should not see task yet, got %d tasks err=%v", len(tasks), err)
	}
	tasks, err = eng.ListUserTasks(ctx, "t", "a")
	if err != nil || len(tasks) != 1 {
		t.Fatalf("a should see task, got %d", len(tasks))
	}

	act := activeUserTask(t, eng, inst.ID)
	_, err = eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        act.ID,
		Assignee:          "a",
		Variables:         map[string]any{"approved": true},
	})
	if err != nil {
		t.Fatal(err)
	}

	tasks, _ = eng.ListUserTasks(ctx, "t", "b")
	if len(tasks) != 1 {
		t.Fatalf("b should see task after a, got %d", len(tasks))
	}

	act = activeUserTask(t, eng, inst.ID)
	_, err = eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        act.ID,
		Assignee:          "b",
	})
	if err != nil {
		t.Fatal(err)
	}

	act = activeUserTask(t, eng, inst.ID)
	_, err = eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        act.ID,
		Assignee:          "c",
	})
	if err != nil {
		t.Fatal(err)
	}
	inst, _ = eng.GetProcessInstance(ctx, inst.ID)
	if inst.Status != engine.ProcessStatusCompleted {
		t.Fatalf("expected completed, got %s", inst.Status)
	}
}
