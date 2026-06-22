package definition

import (
	"testing"
)

func TestValidateMessageStartRequiresMessageRef(t *testing.T) {
	def := ProcessDefinition{
		ID: "p",
		Elements: []Element{
			{
				ID:   "start",
				Kind: KindStartEvent,
				EventDefinition: &EventDefinition{
					Type: EventTypeMessage,
				},
			},
			{ID: "end", Kind: KindEndEvent},
		},
		Flows: []SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "end"},
		},
	}
	if err := Validate(def); err == nil {
		t.Fatal("expected validation error for missing messageRef")
	}
}

func TestMessageStartMatchCondition(t *testing.T) {
	el := Element{
		Kind: KindStartEvent,
		EventDefinition: &EventDefinition{
			Type:       EventTypeMessage,
			MessageRef: "airtable.orders.updated",
			Condition:  "event.fields.status == Pending",
		},
	}
	vars := map[string]any{
		"event": map[string]any{"fields": map[string]any{"status": "Pending"}},
	}
	ok, err := MessageStartMatch(el, "airtable.orders.updated", vars)
	if err != nil || !ok {
		t.Fatalf("expected match, ok=%v err=%v", ok, err)
	}
	ok, err = MessageStartMatch(el, "other.message", vars)
	if err != nil || ok {
		t.Fatalf("expected no match on wrong messageRef")
	}
}
