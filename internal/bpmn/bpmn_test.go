package bpmn

import "testing"

func TestValidate_Process(t *testing.T) {
	def := ProcessDefinition{
		ID: "p1",
		Elements: []Element{
			{ID: "start", Kind: KindStartEvent},
			{ID: "task", Kind: KindUserTask},
			{ID: "end", Kind: KindEndEvent},
		},
		Flows: []SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "task"},
			{ID: "f2", SourceRef: "task", TargetRef: "end"},
		},
	}
	if err := Validate(def); err != nil {
		t.Fatal(err)
	}
	reg, err := BuildRegistry(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.StartEvents) != 1 {
		t.Fatalf("expected 1 start event, got %d", len(reg.StartEvents))
	}
}

func TestEvalCondition(t *testing.T) {
	vars := map[string]any{"approved": true, "amount": 500, "status": "ok"}
	ok, err := EvalCondition("approved == true", vars)
	if err != nil || !ok {
		t.Fatalf("expected true, got %v err=%v", ok, err)
	}
	ok, err = EvalCondition("amount >= 1000", vars)
	if err != nil || ok {
		t.Fatalf("expected amount >= 1000 false, got %v err=%v", ok, err)
	}
	ok, err = EvalCondition("status == ok", vars)
	if err != nil || !ok {
		t.Fatalf("expected status match")
	}
}

func TestEvalCondition_nestedPath(t *testing.T) {
	vars := map[string]any{
		"amount": 500,
		"item": map[string]any{
			"kk": 11,
			"name": "widget",
		},
	}

	ok, err := EvalCondition("item.kk >= 10", vars)
	if err != nil || !ok {
		t.Fatalf("expected item.kk >= 10 true, got %v err=%v", ok, err)
	}
	ok, err = EvalCondition("item.kk >= 20", vars)
	if err != nil || ok {
		t.Fatalf("expected item.kk >= 20 false, got %v err=%v", ok, err)
	}
	ok, err = EvalCondition("item.kk == 11", vars)
	if err != nil || !ok {
		t.Fatalf("expected item.kk == 11 true, got %v err=%v", ok, err)
	}
	ok, err = EvalCondition("item.name == widget", vars)
	if err != nil || !ok {
		t.Fatalf("expected item.name match, got %v err=%v", ok, err)
	}
	ok, err = EvalCondition("item.missing == x", vars)
	if err != nil || ok {
		t.Fatalf("expected missing path false, got %v err=%v", ok, err)
	}
	ok, err = EvalCondition("item.missing != x", vars)
	if err != nil || !ok {
		t.Fatalf("expected missing path != true, got %v err=%v", ok, err)
	}
	ok, err = EvalCondition("item.kk", vars)
	if err != nil || !ok {
		t.Fatalf("expected item.kk truthy, got %v err=%v", ok, err)
	}
}
