package definition

import "testing"

func TestValidateCollaboration(t *testing.T) {
	def := ProcessDefinition{
		ID: "p1",
		Elements: []Element{
			{ID: "start", Kind: KindStartEvent},
			{ID: "task", Kind: KindUserTask, Assignees: []string{"a"}},
			{ID: "end", Kind: KindEndEvent},
		},
		Flows: []SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "task"},
			{ID: "f2", SourceRef: "task", TargetRef: "end"},
		},
		Collaboration: &Collaboration{
			Pools: []Pool{{ID: "Pool_A", ProcessRef: "p1"}},
			MessageFlows: []MessageFlow{
				{ID: "mf1", SourceRef: "Pool_A", TargetRef: "task", MessageRef: "handoff"},
			},
		},
	}
	if err := Validate(def); err != nil {
		t.Fatal(err)
	}
}
