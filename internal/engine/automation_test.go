package engine_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/bpmnxml"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func TestPureAutomatedTaskFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	def := bpmn.ProcessDefinition{
		ID: "auto",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "fetch", Kind: bpmn.KindServiceTask, TaskType: "sync", ServiceURL: srv.URL, ServiceMethod: "GET"},
			{ID: "run", Kind: bpmn.KindScriptTask, ScriptLang: "javascript", Script: `return { step: "script" }`},
			{ID: "send", Kind: bpmn.KindSendTask, MessageRef: "done.notify", TaskType: "notify"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "fetch"},
			{ID: "f2", SourceRef: "fetch", TargetRef: "run"},
			{ID: "f3", SourceRef: "run", TargetRef: "send"},
			{ID: "f4", SourceRef: "send", TargetRef: "end"},
		},
	}

	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	_ = store.InsertProcessVersion(ctx, &engine.DeployedProcess{TenantID: "t", Key: "auto", Definition: def})

	inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{TenantID: "t", ProcessKey: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Status != engine.ProcessStatusCompleted {
		t.Fatalf("expected completed, got %s vars=%v", inst.Status, inst.Variables)
	}
	if inst.Variables["step"] != "script" {
		t.Fatalf("script output missing: %v", inst.Variables)
	}
	if inst.Variables["sent"] != true {
		t.Fatalf("send output missing: %v", inst.Variables)
	}
}

func TestDeployFromBPMNXML(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="sig" name="Signal" isExecutable="true">
    <startEvent id="start">
      <signalEventDefinition signalRef="inventory.low"/>
    </startEvent>
    <scriptTask id="run"><script scriptFormat="javascript">return { alert: true };</script></scriptTask>
    <endEvent id="end"/>
    <sequenceFlow id="f1" sourceRef="start" targetRef="run"/>
    <sequenceFlow id="f2" sourceRef="run" targetRef="end"/>
  </process>
</definitions>`)

	def, err := bpmnxml.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	if _, err := eng.DeployProcess(ctx, "t", "sig", def); err != nil {
		t.Fatal(err)
	}
	res, err := eng.TriggerSignal(ctx, engine.TriggerSignalRequest{
		TenantID: "t", SignalRef: "inventory.low",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Matches) != 1 || res.Matches[0].Error != "" && res.Matches[0].InstanceID == "" {
		t.Fatalf("matches %+v", res.Matches)
	}
}
