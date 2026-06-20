package feishu_test

import (
	"context"
	"os"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/internal/plugin"
	"github.com/monoposer/lowcode-bpmn/plugins/go/feishu"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func TestFeishuAssigneeAdapterContract(t *testing.T) {
	payload, err := os.ReadFile("testdata/assignee_deleted.json")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	def := bpmn.ProcessDefinition{
		ID: "p",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"u1", "u2"}, ApprovalMode: "all"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t1", Key: "p", Definition: def})
	_, _ = eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t1", ProcessKey: "p"})

	ad := feishu.AssigneeAdapter{DefaultTenant: "t1"}
	host := plugin.NewHost(eng)
	if err := ad.Handle(ctx, event.InboundEvent{
		Stream: event.StreamAssignee, Source: "feishu", Payload: payload,
	}, host); err != nil {
		t.Fatal(err)
	}

	tasks, err := eng.ListUserTasks(ctx, "t1", "u2")
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task for u2, got %d", len(tasks))
	}
}
