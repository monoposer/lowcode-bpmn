package bpmn

import "testing"

func TestParseApprovalMode(t *testing.T) {
	cases := []struct {
		in   string
		want ApprovalMode
	}{
		{"", ApprovalAny},
		{"any", ApprovalAny},
		{"或签", ApprovalAny},
		{"all", ApprovalAll},
		{"会签", ApprovalAll},
		{"sequential", ApprovalSequential},
		{"顺签", ApprovalSequential},
	}
	for _, c := range cases {
		if got := ParseApprovalMode(c.in); got != c.want {
			t.Fatalf("ParseApprovalMode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
