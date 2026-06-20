package engine_test

import (
	"context"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func TestTriggerMessageStartsMatchingProcess(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID:   "airtable-order",
		Name: "Airtable order",
		Elements: []bpmn.Element{
			{
				ID:   "start",
				Kind: bpmn.KindStartEvent,
				EventDefinition: &bpmn.EventDefinition{
					Type:           bpmn.EventTypeMessage,
					MessageRef:     "airtable.orders.updated",
					CorrelationKey: "event.recordId",
					Condition:      "event.fields.status == Pending",
				},
			},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"ops-lead"}, ApprovalMode: "any"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}
	if _, err := eng.DeployProcess(ctx, "t", "airtable-order", def); err != nil {
		t.Fatal(err)
	}

	vars := map[string]any{
		"event": map[string]any{
			"recordId": "recABC",
			"fields":   map[string]any{"status": "Pending"},
		},
	}
	result, err := eng.TriggerMessage(ctx, engine.TriggerMessageRequest{
		TenantID:   "t",
		MessageRef: "airtable.orders.updated",
		Variables:  vars,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result.Matches))
	}
	if result.Matches[0].Error != "" {
		t.Fatalf("start error: %s", result.Matches[0].Error)
	}
	if result.Matches[0].InstanceID == "" {
		t.Fatal("expected instance id")
	}

	result2, err := eng.TriggerMessage(ctx, engine.TriggerMessageRequest{
		TenantID:   "t",
		MessageRef: "airtable.orders.updated",
		Variables: map[string]any{
			"event": map[string]any{
				"recordId": "recXYZ",
				"fields":   map[string]any{"status": "Done"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result2.Matches) != 0 {
		t.Fatalf("expected 0 matches on condition fail, got %d", len(result2.Matches))
	}
}
