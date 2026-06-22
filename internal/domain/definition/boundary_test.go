package definition

import "testing"

func TestBoundaryMessageMatch(t *testing.T) {
	el := Element{
		ID: "b1", Kind: KindBoundaryEvent, AttachedToRef: "review",
		EventDefinition: &EventDefinition{Type: EventTypeMessage, MessageRef: "escalate"},
	}
	ok, err := BoundaryMessageMatch(el, "escalate", nil)
	if err != nil || !ok {
		t.Fatalf("match: ok=%v err=%v", ok, err)
	}
	ok, _ = BoundaryMessageMatch(el, "other", nil)
	if ok {
		t.Fatal("expected no match for other messageRef")
	}
}

func TestBoundaryCancelsActivityDefault(t *testing.T) {
	el := Element{Kind: KindBoundaryEvent}
	if !BoundaryCancelsActivity(el) {
		t.Fatal("default should cancel host")
	}
	v := false
	el.CancelActivity = &v
	if BoundaryCancelsActivity(el) {
		t.Fatal("explicit false should not cancel")
	}
}
