package definition

import "testing"

func extensionProcess() ProcessDefinition {
	return ProcessDefinition{
		ID: "ext-demo",
		Elements: []Element{
			{ID: "start", Kind: KindStartEvent},
			{ID: "review", Kind: KindUserTask, Assignees: []string{"alice"}},
			{
				ID: "timer-boundary", Kind: KindBoundaryEvent,
				AttachedToRef: "review",
				EventDefinition: &EventDefinition{Type: EventTypeTimer, TimerCycle: "PT1H"},
			},
			{ID: "call", Kind: KindCallActivity, CalledElement: "sub-process-key"},
			{ID: "evt-gw", Kind: KindEventBasedGateway},
			{ID: "cx-gw", Kind: KindComplexGateway},
			{
				ID: "catch-msg", Kind: KindIntermediateCatchEvent,
				EventDefinition: &EventDefinition{Type: EventTypeMessage, MessageRef: "order.paid"},
			},
			{ID: "end", Kind: KindEndEvent},
		},
		Flows: []SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "call"},
			{ID: "f3", SourceRef: "call", TargetRef: "end"},
		},
		LaneSet: []Lane{
			{ID: "lane-mgr", Name: "Manager", FlowNodeRefs: []string{"review"}},
		},
		DataObjects: []DataObject{{ID: "do-invoice", Name: "Invoice"}},
	}
}

func TestValidateExtensionElements(t *testing.T) {
	def := extensionProcess()
	if err := Validate(def); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateBoundaryMissingAttach(t *testing.T) {
	def := extensionProcess()
	def.Elements[2].AttachedToRef = ""
	if err := Validate(def); err == nil {
		t.Fatal("expected error for boundary without attachedToRef")
	}
}

func TestValidateCallActivityMissingCalledElement(t *testing.T) {
	def := extensionProcess()
	for i := range def.Elements {
		if def.Elements[i].ID == "call" {
			def.Elements[i].CalledElement = ""
		}
	}
	if err := Validate(def); err == nil {
		t.Fatal("expected error for callActivity without calledElement")
	}
}

func TestRegistryBoundaryIndex(t *testing.T) {
	reg, err := BuildRegistry(extensionProcess())
	if err != nil {
		t.Fatal(err)
	}
	boundary := reg.BoundaryEvents("review")
	if len(boundary) != 1 || boundary[0].ID != "timer-boundary" {
		t.Fatalf("boundary index: %+v", boundary)
	}
}

func TestIsExtensionKind(t *testing.T) {
	if !IsExtensionKind(KindCallActivity) {
		t.Fatal("callActivity should be extension kind")
	}
	if IsExtensionKind(KindUserTask) {
		t.Fatal("userTask is core kind")
	}
}

func TestLaneForElement(t *testing.T) {
	reg, err := BuildRegistry(extensionProcess())
	if err != nil {
		t.Fatal(err)
	}
	lane, ok := reg.LaneForElement("review")
	if !ok || lane.ID != "lane-mgr" {
		t.Fatalf("lane lookup: ok=%v lane=%+v", ok, lane)
	}
}
