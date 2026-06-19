package bpmn

// ElementKind identifies BPMN 2.0 element types supported by the engine.
type ElementKind string

const (
	KindStartEvent         ElementKind = "startEvent"
	KindEndEvent           ElementKind = "endEvent"
	KindUserTask           ElementKind = "userTask"
	KindScriptTask         ElementKind = "scriptTask"
	KindExclusiveGateway   ElementKind = "exclusiveGateway"
	KindParallelGateway    ElementKind = "parallelGateway"
	KindInclusiveGateway   ElementKind = "inclusiveGateway"
)

// ProcessDefinition is a deployable BPMN process (JSON form, designer-friendly).
type ProcessDefinition struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Elements    []Element      `json:"elements"`
	Flows       []SequenceFlow `json:"flows"`
}

// Element is a node in the process graph.
type Element struct {
	ID          string         `json:"id"`
	Kind        ElementKind    `json:"type"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Assignees   []string       `json:"assignees,omitempty"`
	Script      string         `json:"script,omitempty"`
	ScriptLang  string         `json:"scriptLang,omitempty"`
	AutoComplete bool          `json:"autoComplete,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
}

// SequenceFlow connects two elements with an optional condition expression.
type SequenceFlow struct {
	ID          string `json:"id"`
	SourceRef   string `json:"sourceRef"`
	TargetRef   string `json:"targetRef"`
	Name        string `json:"name,omitempty"`
	Condition   string `json:"condition,omitempty"`
	IsDefault   bool   `json:"isDefault,omitempty"`
}
