package bpmn

import "testing"

func TestResolveReturnTargetExplicit(t *testing.T) {
	reg := mustReg(t, ProcessDefinition{
		ID: "p",
		Elements: []Element{
			{ID: "start", Kind: KindStartEvent},
			{ID: "submit", Kind: KindUserTask, ScopeID: "sub1"},
			{ID: "review", Kind: KindUserTask, ScopeID: "sub1"},
			{ID: "end", Kind: KindEndEvent},
		},
		Flows: []SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "submit"},
			{ID: "f2", SourceRef: "submit", TargetRef: "review"},
			{ID: "f3", SourceRef: "review", TargetRef: "end"},
		},
	})
	got, err := ResolveReturnTarget(reg, "review", "submit")
	if err != nil || got != "submit" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestResolveReturnTargetDefaultUpstream(t *testing.T) {
	reg := mustReg(t, ProcessDefinition{
		ID: "p",
		Elements: []Element{
			{ID: "start", Kind: KindStartEvent},
			{ID: "submit", Kind: KindUserTask},
			{ID: "review", Kind: KindUserTask},
			{ID: "end", Kind: KindEndEvent},
		},
		Flows: []SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "submit"},
			{ID: "f2", SourceRef: "submit", TargetRef: "review"},
			{ID: "f3", SourceRef: "review", TargetRef: "end"},
		},
	})
	got, err := ResolveReturnTarget(reg, "review", "")
	if err != nil || got != "submit" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func mustReg(t *testing.T, def ProcessDefinition) *Registry {
	t.Helper()
	reg, err := BuildRegistry(def)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}
