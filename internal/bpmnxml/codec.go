package bpmnxml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/monoposer/lowcode-bpmn/internal/domain/definition"
)

// Parse reads BPMN 2.0 XML (.bpmn / .bpmn20.xml) into the engine process IR.
func Parse(data []byte) (definition.ProcessDefinition, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	var root xmlDefinitions
	if err := dec.Decode(&root); err != nil {
		return definition.ProcessDefinition{}, fmt.Errorf("bpmn xml: %w", err)
	}
	if len(root.Processes) == 0 {
		return definition.ProcessDefinition{}, fmt.Errorf("bpmn xml: no process element")
	}
	def, err := mapProcess(root.Processes[0])
	if err != nil {
		return def, err
	}
	if collab := mapCollaborationForProcess(root.Collaborations, root.Processes[0].ID); collab != nil {
		def.Collaboration = collab
	}
	return def, nil
}

// Marshal writes BPMN 2.0 XML for a process definition.
func Marshal(def definition.ProcessDefinition) ([]byte, error) {
	root := xmlDefinitions{
		Xmlns:    BPMNNS,
		XmlnsXSI: XSINS,
		XmlnsLC:  LCNS,
		TargetNS: "http://definition.io/schema/bpmn",
		Processes: []xmlProcess{buildXMLProcess(def)},
	}
	if def.Collaboration != nil {
		root.Collaborations = []xmlCollaboration{buildXMLCollaboration(def.Collaboration, def.ID)}
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ParseReader is Parse from an io.Reader.
func ParseReader(r io.Reader) (definition.ProcessDefinition, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return definition.ProcessDefinition{}, err
	}
	return Parse(raw)
}

type xmlDefinitions struct {
	XMLName        xml.Name            `xml:"definitions"`
	Xmlns          string              `xml:"xmlns,attr"`
	XmlnsXSI       string              `xml:"xmlns:xsi,attr,omitempty"`
	XmlnsLC        string              `xml:"xmlns:lc,attr,omitempty"`
	TargetNS       string              `xml:"targetNamespace,attr,omitempty"`
	Collaborations []xmlCollaboration  `xml:"collaboration"`
	Processes      []xmlProcess        `xml:"process"`
}

type xmlCollaboration struct {
	ID           string             `xml:"id,attr,omitempty"`
	Participants []xmlParticipant   `xml:"participant"`
	MessageFlows []xmlMessageFlowEl `xml:"messageFlow"`
}

type xmlParticipant struct {
	ID         string `xml:"id,attr"`
	Name       string `xml:"name,attr,omitempty"`
	ProcessRef string `xml:"processRef,attr,omitempty"`
}

type xmlMessageFlowEl struct {
	ID         string `xml:"id,attr"`
	Name       string `xml:"name,attr,omitempty"`
	SourceRef  string `xml:"sourceRef,attr"`
	TargetRef  string `xml:"targetRef,attr"`
	MessageRef string `xml:"messageRef,attr,omitempty"`
}

type xmlProcess struct {
	XMLName    xml.Name         `xml:"process"`
	ID         string           `xml:"id,attr"`
	Name       string           `xml:"name,attr,omitempty"`
	Executable bool             `xml:"isExecutable,attr,omitempty"`
	LaneSets   []xmlLaneSet     `xml:"laneSet"`
	DataObjects []xmlNamedRef   `xml:"dataObjectReference"`
	DataStores  []xmlNamedRef   `xml:"dataStoreReference"`
	FlowElements []xmlFlowElement `xml:",any"`
}

type xmlLaneSet struct {
	ID    string    `xml:"id,attr,omitempty"`
	Lanes []xmlLane `xml:"lane"`
}

type xmlLane struct {
	ID           string   `xml:"id,attr"`
	Name         string   `xml:"name,attr,omitempty"`
	FlowNodeRefs []string `xml:"flowNodeRef"`
}

type xmlNamedRef struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr,omitempty"`
}

type xmlFlowElement struct {
	XMLName   xml.Name
	ID        string `xml:"id,attr"`
	Name      string `xml:"name,attr,omitempty"`
	SourceRef string `xml:"sourceRef,attr,omitempty"`
	TargetRef string `xml:"targetRef,attr,omitempty"`

	AttachedToRef  string `xml:"attachedToRef,attr,omitempty"`
	CancelActivity *bool  `xml:"cancelActivity,attr,omitempty"`
	CalledElement  string `xml:"calledElement,attr,omitempty"`

	MessageEventDef *xmlMessageEventDef `xml:"messageEventDefinition"`
	SignalEventDef  *xmlSignalEventDef  `xml:"signalEventDefinition"`
	TimerEventDef   *xmlTimerEventDef   `xml:"timerEventDefinition"`
	ConditionalDef  *xmlConditionalDef  `xml:"conditionalEventDefinition"`
	ErrorEventDef   *xmlErrorEventDef   `xml:"errorEventDefinition"`

	MultiInstance *xmlMultiInstance `xml:"multiInstanceLoopCharacteristics"`

	ConditionExpr *xmlConditionExpr `xml:"conditionExpression"`
	Default       *xmlDefaultFlow   `xml:"default"`

	ExtensionElements *xmlExtensionElements `xml:"extensionElements"`

	Script *xmlScript `xml:"script"`
}

type xmlErrorEventDef struct {
	ErrorRef string `xml:"errorRef,attr"`
}

type xmlMultiInstance struct {
	IsSequential    bool   `xml:"isSequential,attr,omitempty"`
	Collection      string `xml:"collection,attr,omitempty"`
	ElementVariable string `xml:"elementVariable,attr,omitempty"`
}

type xmlMessageEventDef struct {
	MessageRef string `xml:"messageRef,attr"`
}

type xmlSignalEventDef struct {
	SignalRef string `xml:"signalRef,attr"`
}

type xmlTimerEventDef struct {
	TimeCycle string `xml:"timeCycle"`
	TimeDate  string `xml:"timeDate"`
}

type xmlConditionalDef struct {
	Condition *xmlConditionExpr `xml:"condition"`
}

type xmlConditionExpr struct {
	Body string `xml:",innerxml"`
}

type xmlDefaultFlow struct {
	ID string `xml:"id,attr"`
}

type xmlScript struct {
	Format string `xml:"scriptFormat,attr,omitempty"`
	Body   string `xml:",chardata"`
}

type xmlExtensionElements struct {
	XMLName xml.Name `xml:"extensionElements"`

	TaskType       string        `xml:"taskType"`
	Assignees      string        `xml:"assignees"`
	AssigneesVar   string        `xml:"assigneesVariable"`
	ApprovalMode   string        `xml:"approvalMode"`
	ScriptLang     string        `xml:"scriptLang"`
	AutoComplete   *bool         `xml:"autoComplete"`
	Implementation string        `xml:"implementation"`
	ServiceURL     string        `xml:"serviceUrl"`
	ServiceMethod  string        `xml:"serviceMethod"`
	MessageRef     string        `xml:"messageRef"`
	DecisionRef    string        `xml:"decisionRef"`
	CorrelationKey string        `xml:"correlationKey"`
	ReturnTo       string        `xml:"returnTo"`
	OnReject       string        `xml:"onReject"`
	ScopeID        string        `xml:"scopeId"`
	EntryRef       string        `xml:"entryRef"`
	ExitRef        string        `xml:"exitRef"`
	FormKey        string        `xml:"formKey"`
	FormURL        string        `xml:"formUrl"`
	ExtensionHandler string      `xml:"extensionHandler"`
	Properties     []xmlProperty `xml:"property"`

	LCTaskType       string `xml:"http://github.com/monoposer/lowcode-bpmn/extensions taskType"`
	LCAssignees      string `xml:"http://github.com/monoposer/lowcode-bpmn/extensions assignees"`
	LCImplementation string `xml:"http://github.com/monoposer/lowcode-bpmn/extensions implementation"`
	LCServiceURL     string `xml:"http://github.com/monoposer/lowcode-bpmn/extensions serviceUrl"`
	LCServiceMethod  string `xml:"http://github.com/monoposer/lowcode-bpmn/extensions serviceMethod"`
	LCMessageRef     string `xml:"http://github.com/monoposer/lowcode-bpmn/extensions messageRef"`
	LCDecisionRef    string `xml:"http://github.com/monoposer/lowcode-bpmn/extensions decisionRef"`
	LCCorrelationKey string `xml:"http://github.com/monoposer/lowcode-bpmn/extensions correlationKey"`
}

type xmlProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

func mapProcess(p xmlProcess) (definition.ProcessDefinition, error) {
	def := definition.ProcessDefinition{
		ID:   p.ID,
		Name: p.Name,
	}
	var elements []definition.Element
	var flows []definition.SequenceFlow

	for _, ls := range p.LaneSets {
		for _, lane := range ls.Lanes {
			def.LaneSet = append(def.LaneSet, definition.Lane{
				ID:           lane.ID,
				Name:         lane.Name,
				FlowNodeRefs: append([]string(nil), lane.FlowNodeRefs...),
			})
		}
	}
	for _, d := range p.DataObjects {
		def.DataObjects = append(def.DataObjects, definition.DataObject{ID: d.ID, Name: d.Name})
	}
	for _, d := range p.DataStores {
		def.DataStores = append(def.DataStores, definition.DataStore{ID: d.ID, Name: d.Name})
	}

	for _, fe := range p.FlowElements {
		local := fe.XMLName.Local
		switch local {
		case "sequenceFlow":
			flows = append(flows, definition.SequenceFlow{
				ID:        fe.ID,
				Name:      fe.Name,
				SourceRef: fe.SourceRef,
				TargetRef: fe.TargetRef,
				Condition: strings.TrimSpace(conditionBody(fe.ConditionExpr)),
				IsDefault: fe.Default != nil,
			})
		default:
			kind, ok := xmlLocalToKind(local)
			if !ok {
				continue
			}
			el := definition.Element{ID: fe.ID, Kind: kind, Name: fe.Name}
			el.AttachedToRef = fe.AttachedToRef
			el.CancelActivity = fe.CancelActivity
			el.CalledElement = fe.CalledElement
			if fe.MultiInstance != nil {
				el.MultiInstance = &definition.MultiInstanceLoopCharacteristics{
					IsSequential:    fe.MultiInstance.IsSequential,
					Collection:      fe.MultiInstance.Collection,
					ElementVariable: fe.MultiInstance.ElementVariable,
				}
			}
			if fe.Script != nil {
				el.Script = strings.TrimSpace(fe.Script.Body)
				el.ScriptLang = fe.Script.Format
				if el.ScriptLang == "" {
					el.ScriptLang = "javascript"
				}
			}
			if fe.ExtensionElements != nil {
				applyExtensions(&el, fe.ExtensionElements)
			}
			if kind == definition.KindStartEvent ||
				kind == definition.KindBoundaryEvent ||
				kind == definition.KindIntermediateCatchEvent ||
				kind == definition.KindIntermediateThrowEvent {
				el.EventDefinition = mapEventDef(fe)
			}
			if kind == definition.KindReceiveTask && el.MessageRef != "" {
				// receiveTask message correlation
			}
			elements = append(elements, el)
		}
	}

	if def.ID == "" {
		return definition.ProcessDefinition{}, fmt.Errorf("bpmn xml: process id required")
	}
	def.Elements = elements
	def.Flows = flows
	return def, nil
}

func xmlLocalToKind(local string) (definition.ElementKind, bool) {
	switch local {
	case "startEvent":
		return definition.KindStartEvent, true
	case "endEvent":
		return definition.KindEndEvent, true
	case "userTask":
		return definition.KindUserTask, true
	case "scriptTask":
		return definition.KindScriptTask, true
	case "serviceTask":
		return definition.KindServiceTask, true
	case "sendTask":
		return definition.KindSendTask, true
	case "receiveTask":
		return definition.KindReceiveTask, true
	case "businessRuleTask":
		return definition.KindBusinessRuleTask, true
	case "exclusiveGateway":
		return definition.KindExclusiveGateway, true
	case "parallelGateway":
		return definition.KindParallelGateway, true
	case "inclusiveGateway":
		return definition.KindInclusiveGateway, true
	case "subProcess":
		return definition.KindSubProcess, true
	case "boundaryEvent":
		return definition.KindBoundaryEvent, true
	case "intermediateCatchEvent":
		return definition.KindIntermediateCatchEvent, true
	case "intermediateThrowEvent":
		return definition.KindIntermediateThrowEvent, true
	case "eventBasedGateway":
		return definition.KindEventBasedGateway, true
	case "complexGateway":
		return definition.KindComplexGateway, true
	case "callActivity":
		return definition.KindCallActivity, true
	default:
		return "", false
	}
}

func mapEventDef(fe xmlFlowElement) *definition.EventDefinition {
	ed := &definition.EventDefinition{}
	switch {
	case fe.MessageEventDef != nil:
		ed.Type = definition.EventTypeMessage
		ed.MessageRef = fe.MessageEventDef.MessageRef
		if fe.ExtensionElements != nil {
			ed.CorrelationKey = fe.ExtensionElements.CorrelationKey
		}
	case fe.SignalEventDef != nil:
		ed.Type = definition.EventTypeSignal
		ed.SignalRef = fe.SignalEventDef.SignalRef
	case fe.TimerEventDef != nil:
		ed.Type = definition.EventTypeTimer
		if fe.TimerEventDef.TimeCycle != "" {
			ed.TimerCycle = strings.TrimSpace(fe.TimerEventDef.TimeCycle)
		} else {
			ed.TimerCycle = strings.TrimSpace(fe.TimerEventDef.TimeDate)
		}
	case fe.ConditionalDef != nil && fe.ConditionalDef.Condition != nil:
		ed.Type = definition.EventTypeConditional
		ed.Condition = strings.TrimSpace(conditionBody(fe.ConditionalDef.Condition))
	case fe.ErrorEventDef != nil:
		ed.Type = definition.EventTypeError
		ed.ErrorRef = fe.ErrorEventDef.ErrorRef
	default:
		ed.Type = definition.EventTypeNone
	}
	if fe.ExtensionElements != nil {
		ed.CorrelationKey = firstNonEmpty(fe.ExtensionElements.CorrelationKey, fe.ExtensionElements.LCCorrelationKey)
		if ed.Condition == "" {
			ed.Condition = strings.TrimSpace(fe.ExtensionElements.PropertiesValue("condition"))
		}
	}
	return ed
}

// mapStartEventDef is an alias for backward compatibility within this package.
func mapStartEventDef(fe xmlFlowElement) *definition.EventDefinition {
	return mapEventDef(fe)
}

func applyExtensions(el *definition.Element, ext *xmlExtensionElements) {
	el.TaskType = firstNonEmpty(ext.TaskType, ext.LCTaskType)
	el.Implementation = firstNonEmpty(ext.Implementation, ext.LCImplementation)
	el.ServiceURL = firstNonEmpty(ext.ServiceURL, ext.LCServiceURL)
	el.ServiceMethod = firstNonEmpty(ext.ServiceMethod, ext.LCServiceMethod)
	el.MessageRef = firstNonEmpty(ext.MessageRef, ext.LCMessageRef)
	el.DecisionRef = firstNonEmpty(ext.DecisionRef, ext.LCDecisionRef)
	el.ScriptLang = firstNonEmpty(ext.ScriptLang, el.ScriptLang)
	el.ReturnTo = ext.ReturnTo
	el.OnReject = ext.OnReject
	el.ScopeID = ext.ScopeID
	el.EntryRef = ext.EntryRef
	el.ExitRef = ext.ExitRef
	if ext.AutoComplete != nil {
		el.AutoComplete = *ext.AutoComplete
	}
	assignees := firstNonEmpty(ext.Assignees, ext.LCAssignees)
	if assignees != "" {
		el.Assignees = splitCSV(assignees)
	}
	el.AssigneesVariable = ext.AssigneesVar
	el.ApprovalMode = ext.ApprovalMode
	el.FormKey = ext.FormKey
	el.FormURL = ext.FormURL
	el.ExtensionHandler = ext.ExtensionHandler
	if len(ext.Properties) > 0 {
		if el.Properties == nil {
			el.Properties = make(map[string]any, len(ext.Properties))
		}
		for _, p := range ext.Properties {
			el.Properties[p.Name] = p.Value
		}
	}
}

func (x *xmlExtensionElements) PropertiesValue(name string) string {
	for _, p := range x.Properties {
		if p.Name == name {
			return p.Value
		}
	}
	return ""
}

func conditionBody(c *xmlConditionExpr) string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(stripCDATA(c.Body))
}

func stripCDATA(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "<![CDATA[") && strings.HasSuffix(s, "]]>") {
		return strings.TrimSuffix(strings.TrimPrefix(s, "<![CDATA["), "]]>")
	}
	return s
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func buildXMLProcess(def definition.ProcessDefinition) xmlProcess {
	p := xmlProcess{
		ID:         def.ID,
		Name:       def.Name,
		Executable: true,
	}
	if len(def.LaneSet) > 0 {
		ls := xmlLaneSet{}
		for _, lane := range def.LaneSet {
			ls.Lanes = append(ls.Lanes, xmlLane{
				ID:           lane.ID,
				Name:         lane.Name,
				FlowNodeRefs: append([]string(nil), lane.FlowNodeRefs...),
			})
		}
		p.LaneSets = []xmlLaneSet{ls}
	}
	for _, d := range def.DataObjects {
		p.DataObjects = append(p.DataObjects, xmlNamedRef{ID: d.ID, Name: d.Name})
	}
	for _, d := range def.DataStores {
		p.DataStores = append(p.DataStores, xmlNamedRef{ID: d.ID, Name: d.Name})
	}
	for _, el := range def.Elements {
		p.FlowElements = append(p.FlowElements, elementToXML(el))
	}
	for _, f := range def.Flows {
		fe := xmlFlowElement{
			XMLName:   xml.Name{Space: BPMNNS, Local: "sequenceFlow"},
			ID:        f.ID,
			Name:      f.Name,
			SourceRef: f.SourceRef,
			TargetRef: f.TargetRef,
		}
		if f.Condition != "" {
			fe.ConditionExpr = &xmlConditionExpr{Body: f.Condition}
		}
		if f.IsDefault {
			fe.Default = &xmlDefaultFlow{}
		}
		p.FlowElements = append(p.FlowElements, fe)
	}
	return p
}

func elementToXML(el definition.Element) xmlFlowElement {
	fe := xmlFlowElement{
		XMLName:       xml.Name{Space: BPMNNS, Local: kindToXMLLocal(el.Kind)},
		ID:            el.ID,
		Name:          el.Name,
		AttachedToRef: el.AttachedToRef,
		CancelActivity: el.CancelActivity,
		CalledElement: el.CalledElement,
	}
	if el.MultiInstance != nil {
		fe.MultiInstance = &xmlMultiInstance{
			IsSequential:    el.MultiInstance.IsSequential,
			Collection:      el.MultiInstance.Collection,
			ElementVariable: el.MultiInstance.ElementVariable,
		}
	}
	if el.Script != "" {
		fe.Script = &xmlScript{Format: el.ScriptLang, Body: el.Script}
	}
	fe.ExtensionElements = extensionsFromElement(el)
	if el.EventDefinition != nil && needsEventDefXML(el.Kind) {
		attachEventDef(&fe, el.EventDefinition)
	}
	return fe
}

func needsEventDefXML(k definition.ElementKind) bool {
	switch k {
	case definition.KindStartEvent, definition.KindBoundaryEvent,
		definition.KindIntermediateCatchEvent, definition.KindIntermediateThrowEvent:
		return true
	default:
		return false
	}
}

func kindToXMLLocal(k definition.ElementKind) string {
	return string(k)
}

func extensionsFromElement(el definition.Element) *xmlExtensionElements {
	if el.TaskType == "" && el.Implementation == "" && el.ServiceURL == "" &&
		el.MessageRef == "" && el.DecisionRef == "" && len(el.Assignees) == 0 &&
		el.AssigneesVariable == "" && el.ApprovalMode == "" && !el.AutoComplete &&
		el.ReturnTo == "" && el.OnReject == "" && el.ScopeID == "" &&
		el.EntryRef == "" && el.ExitRef == "" && el.ScriptLang == "" &&
		el.FormKey == "" && el.FormURL == "" && el.ExtensionHandler == "" &&
		len(el.Properties) == 0 {
		return nil
	}
	ext := &xmlExtensionElements{
		TaskType:           el.TaskType,
		Implementation:     el.Implementation,
		ServiceURL:         el.ServiceURL,
		ServiceMethod:      el.ServiceMethod,
		MessageRef:         el.MessageRef,
		DecisionRef:        el.DecisionRef,
		ScriptLang:         el.ScriptLang,
		AssigneesVar:       el.AssigneesVariable,
		ApprovalMode:       el.ApprovalMode,
		ReturnTo:           el.ReturnTo,
		OnReject:           el.OnReject,
		ScopeID:            el.ScopeID,
		EntryRef:           el.EntryRef,
		ExitRef:            el.ExitRef,
		FormKey:            el.FormKey,
		FormURL:            el.FormURL,
		ExtensionHandler:   el.ExtensionHandler,
	}
	if el.AutoComplete {
		v := true
		ext.AutoComplete = &v
	}
	if len(el.Assignees) > 0 {
		ext.Assignees = strings.Join(el.Assignees, ",")
	}
	for k, v := range el.Properties {
		ext.Properties = append(ext.Properties, xmlProperty{Name: k, Value: fmt.Sprint(v)})
	}
	return ext
}

func attachEventDef(fe *xmlFlowElement, ed *definition.EventDefinition) {
	if ed == nil {
		return
	}
	switch ed.EffectiveEventType() {
	case definition.EventTypeMessage:
		fe.MessageEventDef = &xmlMessageEventDef{MessageRef: ed.MessageRef}
	case definition.EventTypeSignal:
		fe.SignalEventDef = &xmlSignalEventDef{SignalRef: ed.SignalRef}
	case definition.EventTypeTimer:
		fe.TimerEventDef = &xmlTimerEventDef{TimeCycle: ed.TimerCycle}
	case definition.EventTypeConditional:
		fe.ConditionalDef = &xmlConditionalDef{
			Condition: &xmlConditionExpr{Body: ed.Condition},
		}
	case definition.EventTypeError:
		fe.ErrorEventDef = &xmlErrorEventDef{ErrorRef: ed.ErrorRef}
	}
	if ed.CorrelationKey != "" && fe.ExtensionElements == nil {
		fe.ExtensionElements = &xmlExtensionElements{CorrelationKey: ed.CorrelationKey}
	} else if ed.CorrelationKey != "" && fe.ExtensionElements != nil {
		fe.ExtensionElements.CorrelationKey = ed.CorrelationKey
	}
}

func attachStartEventDef(fe *xmlFlowElement, ed *definition.EventDefinition) {
	attachEventDef(fe, ed)
}

func mapCollaborationForProcess(collabs []xmlCollaboration, processID string) *definition.Collaboration {
	for _, c := range collabs {
		relevant := len(c.MessageFlows) > 0
		for _, p := range c.Participants {
			if p.ProcessRef == "" || p.ProcessRef == processID {
				relevant = true
				break
			}
		}
		if !relevant {
			continue
		}
		out := &definition.Collaboration{}
		for _, p := range c.Participants {
			out.Pools = append(out.Pools, definition.Pool{
				ID: p.ID, Name: p.Name, ProcessRef: p.ProcessRef,
			})
		}
		for _, mf := range c.MessageFlows {
			out.MessageFlows = append(out.MessageFlows, definition.MessageFlow{
				ID: mf.ID, Name: mf.Name, SourceRef: mf.SourceRef,
				TargetRef: mf.TargetRef, MessageRef: mf.MessageRef,
			})
		}
		return out
	}
	return nil
}

func buildXMLCollaboration(c *definition.Collaboration, processID string) xmlCollaboration {
	x := xmlCollaboration{}
	for _, p := range c.Pools {
		x.Participants = append(x.Participants, xmlParticipant{
			ID: p.ID, Name: p.Name, ProcessRef: firstNonEmpty(p.ProcessRef, processID),
		})
	}
	for _, mf := range c.MessageFlows {
		x.MessageFlows = append(x.MessageFlows, xmlMessageFlowEl{
			ID: mf.ID, Name: mf.Name, SourceRef: mf.SourceRef,
			TargetRef: mf.TargetRef, MessageRef: mf.MessageRef,
		})
	}
	return x
}
