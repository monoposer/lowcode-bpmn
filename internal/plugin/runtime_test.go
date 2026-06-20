package plugin_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/pkg/event"
	memconsumer "github.com/monoposer/lowcode-bpmn/pkg/event/memory"
	"github.com/monoposer/lowcode-bpmn/internal/plugin"
	"github.com/monoposer/lowcode-bpmn/plugins/go/airtable"
	"github.com/monoposer/lowcode-bpmn/plugins/go/feishu"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func TestFeishuAssigneeAdapterRemoveUser(t *testing.T) {
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
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: "p", Definition: def})
	_, _ = eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t", ProcessKey: "p"})

	host := plugin.NewHost(eng)
	ad := feishu.AssigneeAdapter{DefaultTenant: "t"}
	payload, _ := json.Marshal(map[string]any{
		"header": map[string]any{"event_type": "contact.user.deleted_v3"},
		"event":  map[string]any{"object": map[string]any{"user_id": "u1"}},
	})
	err := ad.Handle(ctx, event.InboundEvent{Stream: event.StreamAssignee, Source: "feishu", Payload: payload}, host)
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := eng.ListUserTasks(ctx, "t", "u2")
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task for u2, got %d", len(tasks))
	}
}

func TestTriggerStreamConsumer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	cons := memconsumer.New(event.StreamTrigger, 8)
	host := plugin.NewHost(eng)
	rt := plugin.NewRuntime(host, airtable.Adapter{DefaultTenant: "demo"})

	done := make(chan error, 1)
	go func() { done <- cons.Run(ctx, rt.Handler()) }()

	payload := []byte(`{"payload":{"recordId":"rec1","fields":{"status":"Pending"}}}`)
	if err := cons.Publish(ctx, event.InboundEvent{Stream: event.StreamTrigger, Source: "airtable", TenantID: "demo", Payload: payload}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)

	tasks, err := eng.ListUserTasks(ctx, "demo", "ops")
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected task after airtable event, got %d", len(tasks))
	}
	cancel()
	<-done
}

func TestQuadStreamRouter(t *testing.T) {
	assignee := memconsumer.New(event.StreamAssignee, 4)
	trigger := memconsumer.New(event.StreamTrigger, 4)
	task := memconsumer.New(event.StreamTask, 4)
	control := memconsumer.New(event.StreamControl, 4)
	router := event.NewRouterPublisher(assignee, trigger, task, control)
	ctx := context.Background()
	if err := router.Publish(ctx, event.StreamAssignee, event.InboundEvent{Source: "feishu", Payload: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if err := router.Publish(ctx, event.StreamTrigger, event.InboundEvent{Source: "airtable", Payload: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if err := router.Publish(ctx, event.StreamTask, event.InboundEvent{Source: "feishu", Payload: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if err := router.Publish(ctx, event.StreamControl, event.InboundEvent{Source: "generic", Payload: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
}

func TestFeishuTaskAdapterComplete(t *testing.T) {
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)

	def := bpmn.ProcessDefinition{
		ID: "p",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"u1"}, ApprovalMode: "any"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: "p", Definition: def})
	inst, _ := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t", ProcessKey: "p", BusinessKey: "inst-001"})

	host := plugin.NewHost(eng)
	ad := feishu.TaskAdapter{DefaultTenant: "t"}
	payload, _ := json.Marshal(map[string]any{
		"header": map[string]any{"event_type": "approval.instance.status_changed_v1"},
		"event": map[string]any{
			"instance_code": "inst-001",
			"status":        "APPROVED",
			"user_id":       "u1",
		},
	})
	err := ad.Handle(ctx, event.InboundEvent{Stream: event.StreamTask, Source: "feishu", Payload: payload}, host)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := eng.GetProcessInstance(ctx, inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != engine.ProcessStatusCompleted {
		t.Fatalf("expected completed instance, got %s", updated.Status)
	}
}
