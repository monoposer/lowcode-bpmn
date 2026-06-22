package bpmnxml_test

import (
	"strings"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/bpmnxml"
)

const collaborationXML = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <collaboration id="Collab_1">
    <participant id="Pool_A" name="Main" processRef="proc1"/>
    <participant id="Pool_B" name="Partner" processRef="proc2"/>
    <messageFlow id="mf1" sourceRef="Pool_A" targetRef="Pool_B" messageRef="handoff"/>
  </collaboration>
  <process id="proc1" isExecutable="true">
    <startEvent id="start"/>
    <endEvent id="end"/>
    <sequenceFlow id="f1" sourceRef="start" targetRef="end"/>
  </process>
</definitions>`

func TestParseCollaboration(t *testing.T) {
	def, err := bpmnxml.Parse([]byte(collaborationXML))
	if err != nil {
		t.Fatal(err)
	}
	if def.Collaboration == nil || len(def.Collaboration.Pools) != 2 {
		t.Fatalf("collaboration: %+v", def.Collaboration)
	}
	if len(def.Collaboration.MessageFlows) != 1 {
		t.Fatalf("message flows: %+v", def.Collaboration.MessageFlows)
	}
	if err := bpmn.Validate(def); err != nil {
		t.Fatal(err)
	}
	raw, err := bpmnxml.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "messageFlow") {
		t.Fatal("missing messageFlow in output")
	}
}
