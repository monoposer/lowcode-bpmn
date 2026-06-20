package engine_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func activeUserTaskFor(t *testing.T, eng *engine.Engine, instID uuid.UUID, assignee string) *engine.ActivityInstance {
	t.Helper()
	ctx := context.Background()
	acts, err := eng.ListActivities(ctx, instID)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range acts {
		if a.ElementKind != bpmn.KindUserTask || a.Status != engine.ActivityStatusActive {
			continue
		}
		if assignee == "" {
			return a
		}
		if engine.TaskVisibleToAssignee(a, assignee) {
			return a
		}
	}
	t.Fatalf("no active user task for %q", assignee)
	return nil
}

func TestRejectReturnParallelBranch(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "parallel-reject",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "sub1", Kind: bpmn.KindSubProcess, ScopeID: "sub1", EntryRef: "submit", ExitRef: "join"},
			{ID: "submit", Kind: bpmn.KindUserTask, Name: "Submit", ScopeID: "sub1", Assignees: []string{"applicant"}},
			{ID: "fork", Kind: bpmn.KindParallelGateway, ScopeID: "sub1"},
			{ID: "review-a", Kind: bpmn.KindUserTask, Name: "Review A", ScopeID: "sub1", Assignees: []string{"leader-a"}, ReturnTo: "submit"},
			{ID: "review-b", Kind: bpmn.KindUserTask, Name: "Review B", ScopeID: "sub1", Assignees: []string{"leader-b"}},
			{ID: "join", Kind: bpmn.KindParallelGateway, ScopeID: "sub1"},
			{ID: "notify", Kind: bpmn.KindScriptTask, ScopeID: "sub1", ScriptLang: "javascript", Script: `return { done: true }`},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "submit"},
			{ID: "f2", SourceRef: "submit", TargetRef: "fork"},
			{ID: "fa", SourceRef: "fork", TargetRef: "review-a"},
			{ID: "fb", SourceRef: "fork", TargetRef: "review-b"},
			{ID: "fja", SourceRef: "review-a", TargetRef: "join"},
			{ID: "fjb", SourceRef: "review-b", TargetRef: "join"},
			{ID: "fj", SourceRef: "join", TargetRef: "notify"},
			{ID: "fe", SourceRef: "notify", TargetRef: "end"},
		},
	}

	inst := deployAndStart(t, eng, store, def, "parallel-reject", map[string]any{"title": "demo"})

	act := activeUserTaskFor(t, eng, inst.ID, "applicant")
	if _, err := eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        act.ID,
		Assignee:          "applicant",
		Action:            engine.ApprovalActionApprove,
	}); err != nil {
		t.Fatal(err)
	}

	actA := activeUserTaskFor(t, eng, inst.ID, "leader-a")
	if _, err := eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        actA.ID,
		Assignee:          "leader-a",
		Action:            engine.ApprovalActionReject,
		Comment:           "need more info",
	}); err != nil {
		t.Fatal(err)
	}

	tasks, _ := eng.ListUserTasks(ctx, "t", "applicant")
	if len(tasks) != 1 || tasks[0].ElementID != "submit" {
		t.Fatalf("expected resubmit for applicant, got %+v", tasks)
	}

	tasksB, _ := eng.ListUserTasks(ctx, "t", "leader-b")
	if len(tasksB) != 1 {
		t.Fatalf("leader-b should still have review-b, got %d", len(tasksB))
	}

	inst, _ = eng.GetProcessInstance(ctx, inst.ID)
	if inst.Status != engine.ProcessStatusRunning {
		t.Fatalf("instance should still run, got %s", inst.Status)
	}
	if inst.Variables["approved"] != false {
		t.Fatalf("expected approved=false, got %v", inst.Variables["approved"])
	}
}

func TestTerminateScope(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "terminate-scope",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "sub1", Kind: bpmn.KindSubProcess, ScopeID: "sub1", EntryRef: "submit", ExitRef: "join"},
			{ID: "submit", Kind: bpmn.KindUserTask, ScopeID: "sub1", Assignees: []string{"applicant"}},
			{ID: "fork", Kind: bpmn.KindParallelGateway, ScopeID: "sub1"},
			{ID: "review-a", Kind: bpmn.KindUserTask, ScopeID: "sub1", Assignees: []string{"leader-a"}},
			{ID: "review-b", Kind: bpmn.KindUserTask, ScopeID: "sub1", Assignees: []string{"leader-b"}},
			{ID: "join", Kind: bpmn.KindParallelGateway, ScopeID: "sub1"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "submit"},
			{ID: "f2", SourceRef: "submit", TargetRef: "fork"},
			{ID: "fa", SourceRef: "fork", TargetRef: "review-a"},
			{ID: "fb", SourceRef: "fork", TargetRef: "review-b"},
			{ID: "fja", SourceRef: "review-a", TargetRef: "join"},
			{ID: "fjb", SourceRef: "review-b", TargetRef: "join"},
			{ID: "fj", SourceRef: "join", TargetRef: "end"},
		},
	}

	inst := deployAndStart(t, eng, store, def, "terminate-scope", nil)
	act := activeUserTaskFor(t, eng, inst.ID, "applicant")
	if _, err := eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        act.ID,
		Assignee:          "applicant",
	}); err != nil {
		t.Fatal(err)
	}

	inst, err := eng.Terminate(ctx, engine.TerminateRequest{
		ProcessInstanceID: inst.ID,
		ScopeID:           "sub1",
		Reason:            "admin cancelled parallel review",
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Variables["scopeTerminated"] != "sub1" {
		t.Fatalf("expected scopeTerminated=sub1, got %v", inst.Variables["scopeTerminated"])
	}

	tasks, _ := eng.ListUserTasks(ctx, "t", "leader-a")
	if len(tasks) != 0 {
		t.Fatalf("leader-a inbox should be empty after scope terminate, got %d", len(tasks))
	}
}

func TestTerminateInstanceCancelled(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "simple",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"manager"}},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}
	inst := deployAndStart(t, eng, store, def, "simple", nil)

	inst, err := eng.Terminate(ctx, engine.TerminateRequest{
		ProcessInstanceID: inst.ID,
		Reason:            "withdrawn",
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Status != engine.ProcessStatusCancelled {
		t.Fatalf("expected cancelled, got %s", inst.Status)
	}
}

func TestRejectDefaultReturnUpstream(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "reject-return",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "submit", Kind: bpmn.KindUserTask, Assignees: []string{"applicant"}},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"manager"}},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "submit"},
			{ID: "f2", SourceRef: "submit", TargetRef: "review"},
			{ID: "f3", SourceRef: "review", TargetRef: "end"},
		},
	}
	inst := deployAndStart(t, eng, store, def, "reject-return", nil)

	submit := activeUserTaskFor(t, eng, inst.ID, "applicant")
	if _, err := eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        submit.ID,
		Assignee:          "applicant",
	}); err != nil {
		t.Fatal(err)
	}

	review := activeUserTaskFor(t, eng, inst.ID, "manager")
	if _, err := eng.CompleteTask(ctx, engine.CompleteTaskRequest{
		ProcessInstanceID: inst.ID,
		ActivityID:        review.ID,
		Assignee:          "manager",
		Action:            engine.ApprovalActionReject,
	}); err != nil {
		t.Fatal(err)
	}

	tasks, _ := eng.ListUserTasks(ctx, "t", "applicant")
	if len(tasks) != 1 || tasks[0].ElementID != "submit" {
		t.Fatalf("expected return to submit, got %+v", tasks)
	}
}
