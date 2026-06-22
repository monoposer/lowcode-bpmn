package definition

// ElementKind identifies BPMN 2.0 element types supported by the engine.
type ElementKind string

const (
	KindStartEvent         ElementKind = "startEvent"
	KindEndEvent           ElementKind = "endEvent"
	KindUserTask           ElementKind = "userTask"
	KindScriptTask         ElementKind = "scriptTask"
	KindServiceTask        ElementKind = "serviceTask"
	KindSendTask           ElementKind = "sendTask"
	KindReceiveTask        ElementKind = "receiveTask"
	KindBusinessRuleTask   ElementKind = "businessRuleTask"
	KindExclusiveGateway   ElementKind = "exclusiveGateway"
	KindParallelGateway    ElementKind = "parallelGateway"
	KindInclusiveGateway   ElementKind = "inclusiveGateway"
	KindSubProcess         ElementKind = "subProcess"
	// Extension-backed kinds (see docs/ddd/extensions.md).
	KindBoundaryEvent            ElementKind = "boundaryEvent"
	KindIntermediateCatchEvent   ElementKind = "intermediateCatchEvent"
	KindIntermediateThrowEvent   ElementKind = "intermediateThrowEvent"
	KindEventBasedGateway        ElementKind = "eventBasedGateway"
	KindComplexGateway           ElementKind = "complexGateway"
	KindCallActivity             ElementKind = "callActivity"
)

// AutomatedTaskKinds are task types that need no human assignee (BPMN 2.0 automation).
var AutomatedTaskKinds = []ElementKind{
	KindScriptTask, KindServiceTask, KindSendTask, KindReceiveTask, KindBusinessRuleTask,
}

// ProcessDefinition is the internal IR for a deployable BPMN 2.0 process.
// File storage and interchange use BPMN XML (.bpmn20.xml); JSON is supported for API/designer.
type ProcessDefinition struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Elements    []Element      `json:"elements"`
	Flows       []SequenceFlow `json:"flows"`
	LaneSet     []Lane         `json:"laneSet,omitempty"`
	DataObjects []DataObject   `json:"dataObjects,omitempty"`
	DataStores  []DataStore    `json:"dataStores,omitempty"`
	Collaboration *Collaboration `json:"collaboration,omitempty"`
}

// Lane is a swimlane within a process (BPMN laneSet).
type Lane struct {
	ID           string   `json:"id"`
	Name         string   `json:"name,omitempty"`
	FlowNodeRefs []string `json:"flowNodeRefs,omitempty"`
}

// DataObject is a data artifact reference in the process diagram.
type DataObject struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// DataStore is a persistent data store reference in the process diagram.
type DataStore struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// Pool is a BPMN collaboration pool (may reference a process).
type Pool struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	ProcessRef string `json:"processRef,omitempty"`
}

// MessageFlow connects pools or elements across collaboration boundaries.
type MessageFlow struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	SourceRef  string `json:"sourceRef"`
	TargetRef  string `json:"targetRef"`
	MessageRef string `json:"messageRef,omitempty"`
}

// Collaboration holds multi-pool diagram metadata (extension-backed message dispatch).
type Collaboration struct {
	Pools        []Pool        `json:"pools,omitempty"`
	MessageFlows []MessageFlow `json:"messageFlows,omitempty"`
}

// MultiInstanceLoopCharacteristics configures multi-instance extension execution.
type MultiInstanceLoopCharacteristics struct {
	IsSequential    bool   `json:"isSequential,omitempty"`
	Collection      string `json:"collection,omitempty"`
	ElementVariable string `json:"elementVariable,omitempty"`
}

// Element is a node in the process graph.
type Element struct {
	ID          string         `json:"id"`
	Kind        ElementKind    `json:"type"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Assignees          []string       `json:"assignees,omitempty"`
	AssigneesVariable  string         `json:"assigneesVariable,omitempty"` // dot path in instance variables
	ApprovalMode       string         `json:"approvalMode,omitempty"`
	RequiredApprovals  int            `json:"requiredApprovals,omitempty"`
	ReturnTo     string         `json:"returnTo,omitempty"`
	OnReject     string         `json:"onReject,omitempty"` // return (default) | terminateScope
	ScopeID      string         `json:"scopeId,omitempty"`
	EntryRef     string         `json:"entryRef,omitempty"` // subProcess inner entry
	ExitRef      string         `json:"exitRef,omitempty"`  // subProcess inner exit (join/end)
	Script       string         `json:"script,omitempty"`
	ScriptLang  string         `json:"scriptLang,omitempty"`
	// BPMN 2.0 task extensions (also mapped to extensionElements in XML).
	TaskType       string `json:"taskType,omitempty"`       // business subtype e.g. data-sync, export
	Implementation string `json:"implementation,omitempty"` // serviceTask: http, delegate
	ServiceURL     string `json:"serviceUrl,omitempty"`
	ServiceMethod  string `json:"serviceMethod,omitempty"`
	MessageRef     string `json:"messageRef,omitempty"`     // receiveTask / sendTask
	DecisionRef    string `json:"decisionRef,omitempty"`  // businessRuleTask
	AutoComplete bool             `json:"autoComplete,omitempty"`
	EventDefinition *EventDefinition `json:"eventDefinition,omitempty"`
	// Extension fields (boundary, call activity, collaboration, forms).
	AttachedToRef     string                            `json:"attachedToRef,omitempty"`
	CancelActivity    *bool                             `json:"cancelActivity,omitempty"`
	CalledElement     string                            `json:"calledElement,omitempty"`
	MultiInstance     *MultiInstanceLoopCharacteristics `json:"multiInstance,omitempty"`
	LaneRef           string                            `json:"laneRef,omitempty"`
	DataInputRefs     []string                          `json:"dataInputRefs,omitempty"`
	DataOutputRefs    []string                          `json:"dataOutputRefs,omitempty"`
	FormKey           string                            `json:"formKey,omitempty"`
	FormURL           string                            `json:"formUrl,omitempty"`
	ExtensionHandler  string                            `json:"extensionHandler,omitempty"` // explicit adapter id
	EventSubProcess   bool                              `json:"eventSubProcess,omitempty"`
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
