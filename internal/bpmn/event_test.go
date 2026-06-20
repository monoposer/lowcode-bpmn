package bpmn_test

import (
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

func TestValidateMessageStartRequiresMessageRef(t *testing.T) {
	def := bpmn.ProcessDefinition{
		ID: "p",
		Elements: []bpmn.Element{
			{
				ID:   "start",
				Kind: bpmn.KindStartEvent,
				EventDefinition: &bpmn.EventDefinition{
					Type: bpmn.EventTypeMessage,
				},
			},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "end"},
		},
	}
	if err := bpmn.Validate(def); err == nil {
		t.Fatal("expected validation error for missing messageRef")
	}
}

func TestMessageStartMatchCondition(t *testing.T) {
	el := bpmn.Element{
		Kind: bpmn.KindStartEvent,
		EventDefinition: &bpmn.EventDefinition{
			Type:       bpmn.EventTypeMessage,
			MessageRef: "airtable.orders.updated",
			Condition:  "event.fields.status == Pending",
		},
	}
	vars := map[string]any{
		"event": map[string]any{"fields": map[string]any{"status": "Pending"}},
	}
	ok, err := bpmn.MessageStartMatch(el, "airtable.orders.updated", vars)
	if err != nil || !ok {
		t.Fatalf("expected match, ok=%v err=%v", ok, err)
	}
	ok, err = bpmn.MessageStartMatch(el, "other.message", vars)
	if err != nil || ok {
		t.Fatalf("expected no match on wrong messageRef")
	}
}
