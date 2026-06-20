package gormstore

import (
	"context"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

func TestStoreSQLiteRoundTrip(t *testing.T) {
	db, err := Open(DriverSQLite, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	ctx := context.Background()
	store, err := NewStore(ctx, db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	proc := &engine.DeployedProcess{
		TenantID: "demo",
		Key:      "leave",
		Name:     "Leave Request",
		Definition: bpmn.ProcessDefinition{
			ID:   "leave",
			Name: "Leave Request",
			Elements: []bpmn.Element{
				{ID: "start", Kind: bpmn.KindStartEvent},
				{ID: "end", Kind: bpmn.KindEndEvent},
			},
			Flows: []bpmn.SequenceFlow{{ID: "f1", SourceRef: "start", TargetRef: "end"}},
		},
	}
	if err := store.InsertProcessVersion(ctx, proc); err != nil {
		t.Fatalf("insert process: %v", err)
	}

	got, err := store.GetProcess(ctx, "demo", "leave")
	if err != nil {
		t.Fatalf("get process: %v", err)
	}
	if got == nil || got.Key != "leave" || got.Version != 1 {
		t.Fatalf("unexpected process: %+v", got)
	}

	inst := &engine.ProcessInstance{
		TenantID:       "demo",
		ProcessKey:     "leave",
		ProcessVersion: 1,
		Status:         engine.ProcessStatusRunning,
		Variables:      map[string]any{"days": 3},
	}
	if err := store.CreateProcessInstance(ctx, inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	loaded, err := store.GetProcessInstance(ctx, inst.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if loaded == nil || loaded.Variables["days"] != float64(3) {
		t.Fatalf("unexpected instance: %+v", loaded)
	}
}
