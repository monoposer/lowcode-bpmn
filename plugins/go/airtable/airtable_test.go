package airtable_test

import (
	"context"
	"os"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/internal/plugin"
	"github.com/monoposer/lowcode-bpmn/plugins/go/airtable"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func TestAirtableAdapterContract(t *testing.T) {
	payload, err := os.ReadFile("testdata/webhook.json")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	def := bpmn.ProcessDefinition{
		ID: "air",
		Elements: []bpmn.Element{
			{
				ID: "start", Kind: bpmn.KindStartEvent,
				EventDefinition: &bpmn.EventDefinition{Type: bpmn.EventTypeMessage, MessageRef: "airtable.orders.updated"},
			},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"ops"}, ApprovalMode: "any"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}
	_, _ = eng.DeployProcess(ctx, "demo", "air", def)

	ad := airtable.Adapter{DefaultTenant: "demo"}
	host := plugin.NewHost(eng)
	if err := ad.Handle(ctx, event.InboundEvent{
		Stream: event.StreamTrigger, Source: "airtable", TenantID: "demo", Payload: payload,
	}, host); err != nil {
		t.Fatal(err)
	}

	tasks, err := eng.ListUserTasks(ctx, "demo", "ops")
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 active task, got %d", len(tasks))
	}
}
