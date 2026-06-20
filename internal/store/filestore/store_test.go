package filestore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

func TestStoreYAMLRoundTrip(t *testing.T) {
	dir := t.TempDir()

	store, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	ctx := context.Background()
	if err := store.InsertProcessVersion(ctx, &engine.DeployedProcess{
		TenantID: "demo",
		Key:      "leave",
		Name:     "Leave",
		Definition: bpmn.ProcessDefinition{
			ID: "leave",
			Elements: []bpmn.Element{
				{ID: "start", Kind: bpmn.KindStartEvent},
				{ID: "end", Kind: bpmn.KindEndEvent},
			},
			Flows: []bpmn.SequenceFlow{{ID: "f1", SourceRef: "start", TargetRef: "end"}},
		},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	stateFile := filepath.Join(dir, defaultStateFile)
	if _, err := os.Stat(stateFile); err != nil {
		t.Fatalf("state file missing: %v", err)
	}

	reloaded, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	got, err := reloaded.GetProcess(ctx, "demo", "leave")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil || got.Key != "leave" {
		t.Fatalf("unexpected process: %+v", got)
	}
}

func TestStoreWithTxPersistsOnce(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = store.WithTx(ctx, func(tx engine.Store) error {
		return tx.InsertProcessVersion(ctx, &engine.DeployedProcess{
			TenantID: "demo",
			Key:      "tx",
			Definition: bpmn.ProcessDefinition{
				ID: "tx",
				Elements: []bpmn.Element{
					{ID: "start", Kind: bpmn.KindStartEvent},
				},
			},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	reloaded, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reloaded.GetProcess(ctx, "demo", "tx")
	if err != nil || got == nil {
		t.Fatalf("expected process after tx, got %+v err=%v", got, err)
	}
}
