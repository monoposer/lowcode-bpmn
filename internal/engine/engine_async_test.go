package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func TestAsyncStartProcess(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	eng.SetAsync(true)

	def := approvalProcess()
	def.Elements[1].AutoComplete = true
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{
		TenantID: "t", Key: "approval", Version: 1, Definition: def,
	})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{
		TenantID: "t", ProcessKey: "approval",
	})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Status != engine.ProcessStatusPending {
		t.Fatalf("expected pending, got %s", inst.Status)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := eng.ProcessNextJob(ctx); err != nil {
			t.Fatal(err)
		}
		inst, err = eng.GetProcessInstance(ctx, inst.ID)
		if err != nil {
			t.Fatal(err)
		}
		if inst.Status == engine.ProcessStatusCompleted {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("instance not completed after worker run: %s", inst.Status)
}

func TestListUserTasks(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	def := approvalProcess()
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{
		TenantID: "t1", Key: "approval", Version: 1, Definition: def,
	})

	_, err := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t1", ProcessKey: "approval"})
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := eng.ListUserTasks(ctx, "t1", "manager")
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task for manager, got %d", len(tasks))
	}
}

func TestProcessVersionPinning(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	v1 := approvalProcess()
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{
		TenantID: "t", Key: "p", Version: 1, Definition: v1,
	})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t", ProcessKey: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if inst.ProcessVersion != 1 {
		t.Fatalf("expected pinned version 1, got %d", inst.ProcessVersion)
	}

	v2 := approvalProcess()
	v2.Elements = append(v2.Elements, bpmn.Element{
		ID: "extra", Kind: bpmn.KindScriptTask, Script: "set:extra=1",
	})
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{
		TenantID: "t", Key: "p", Version: 2, Definition: v2,
	})

	loaded, _ := eng.GetProcessInstance(ctx, inst.ID)
	if loaded.ProcessVersion != 1 {
		t.Fatalf("running instance should stay on v1, got %d", loaded.ProcessVersion)
	}
}
