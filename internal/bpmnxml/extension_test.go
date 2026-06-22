package bpmnxml_test

import (
	"strings"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/bpmnxml"
)

const extensionSampleXML = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"
  xmlns:lc="https://github.com/monoposer/lowcode-bpmn/extensions"
  targetNamespace="http://definition.io/schema/bpmn">
  <process id="ext-proc" name="Extension Sample" isExecutable="true">
    <laneSet id="LaneSet_1">
      <lane id="Lane_Manager" name="Manager">
        <flowNodeRef>review</flowNodeRef>
      </lane>
    </laneSet>
    <dataObjectReference id="DataObject_1" name="Invoice"/>
    <startEvent id="start"/>
    <userTask id="review" name="Review">
      <extensionElements>
        <formKey>review-form</formKey>
      </extensionElements>
    </userTask>
    <boundaryEvent id="timer1" attachedToRef="review" cancelActivity="true">
      <timerEventDefinition><timeCycle>PT1H</timeCycle></timerEventDefinition>
    </boundaryEvent>
    <callActivity id="call1" calledElement="sub-key"/>
    <eventBasedGateway id="egw1"/>
    <complexGateway id="cgw1"/>
    <intermediateCatchEvent id="catch1">
      <messageEventDefinition messageRef="order.paid"/>
    </intermediateCatchEvent>
    <endEvent id="end"/>
    <sequenceFlow id="f1" sourceRef="start" targetRef="review"/>
    <sequenceFlow id="f2" sourceRef="review" targetRef="end"/>
  </process>
</definitions>`

func TestParseExtensionElements(t *testing.T) {
	def, err := bpmnxml.Parse([]byte(extensionSampleXML))
	if err != nil {
		t.Fatal(err)
	}
	if err := bpmn.Validate(def); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(def.LaneSet) != 1 || def.LaneSet[0].ID != "Lane_Manager" {
		t.Fatalf("laneSet: %+v", def.LaneSet)
	}
	if len(def.DataObjects) != 1 || def.DataObjects[0].Name != "Invoice" {
		t.Fatalf("dataObjects: %+v", def.DataObjects)
	}
	var boundary, call *bpmn.Element
	for i := range def.Elements {
		switch def.Elements[i].ID {
		case "timer1":
			boundary = &def.Elements[i]
		case "call1":
			call = &def.Elements[i]
		}
	}
	if boundary == nil || boundary.Kind != bpmn.KindBoundaryEvent {
		t.Fatalf("boundary: %+v", boundary)
	}
	if boundary.AttachedToRef != "review" || boundary.EventDefinition == nil {
		t.Fatalf("boundary meta: %+v", boundary)
	}
	if call == nil || call.CalledElement != "sub-key" {
		t.Fatalf("callActivity: %+v", call)
	}
}

func TestMarshalExtensionRoundTrip(t *testing.T) {
	def, err := bpmnxml.Parse([]byte(extensionSampleXML))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := bpmnxml.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "boundaryEvent") {
		t.Fatal("missing boundaryEvent in marshaled xml")
	}
	back, err := bpmnxml.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Elements) != len(def.Elements) {
		t.Fatalf("element count %d vs %d", len(back.Elements), len(def.Elements))
	}
}
