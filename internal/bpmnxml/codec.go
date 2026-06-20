package bpmnxml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

// Parse reads BPMN 2.0 XML (.bpmn / .bpmn20.xml) into the engine process IR.
func Parse(data []byte) (bpmn.ProcessDefinition, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	var root xmlDefinitions
	if err := dec.Decode(&root); err != nil {
		return bpmn.ProcessDefinition{}, fmt.Errorf("bpmn xml: %w", err)
	}
	if len(root.Processes) == 0 {
		return bpmn.ProcessDefinition{}, fmt.Errorf("bpmn xml: no process element")
	}
	return mapProcess(root.Processes[0])
}

// Marshal writes BPMN 2.0 XML for a process definition.
func Marshal(def bpmn.ProcessDefinition) ([]byte, error) {
	root := xmlDefinitions{
		Xmlns:    BPMNNS,
		XmlnsXSI: XSINS,
		XmlnsLC:  LCNS,
		TargetNS: "http://bpmn.io/schema/bpmn",
		Processes: []xmlProcess{buildXMLProcess(def)},
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
func ParseReader(r io.Reader) (bpmn.ProcessDefinition, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return bpmn.ProcessDefinition{}, err
	}
	return Parse(raw)
}

type xmlDefinitions struct {
	XMLName   xml.Name     `xml:"definitions"`
	Xmlns     string       `xml:"xmlns,attr"`
	XmlnsXSI  string       `xml:"xmlns:xsi,attr,omitempty"`
	XmlnsLC   string       `xml:"xmlns:lc,attr,omitempty"`
	TargetNS  string       `xml:"targetNamespace,attr,omitempty"`
	Processes []xmlProcess `xml:"process"`
}

type xmlProcess struct {
	XMLName xml.Name `xml:"process"`
	ID      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr,omitempty"`
	Executable bool  `xml:"isExecutable,attr,omitempty"`

	FlowElements []xmlFlowElement `xml:",any"`
}

type xmlFlowElement struct {
	XMLName  xml.Name
	ID       string `xml:"id,attr"`
	Name     string `xml:"name,attr,omitempty"`
	SourceRef string `xml:"sourceRef,attr,omitempty"`
	TargetRef string `xml:"targetRef,attr,omitempty"`

	MessageEventDef *xmlMessageEventDef `xml:"messageEventDefinition"`
	SignalEventDef  *xmlSignalEventDef  `xml:"signalEventDefinition"`
	TimerEventDef   *xmlTimerEventDef   `xml:"timerEventDefinition"`
	ConditionalDef  *xmlConditionalDef  `xml:"conditionalEventDefinition"`

	ConditionExpr *xmlConditionExpr `xml:"conditionExpression"`
	Default       *xmlDefaultFlow   `xml:"default"`

	ExtensionElements *xmlExtensionElements `xml:"extensionElements"`

	Script *xmlScript `xml:"script"`
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

func mapProcess(p xmlProcess) (bpmn.ProcessDefinition, error) {
	def := bpmn.ProcessDefinition{
		ID:   p.ID,
		Name: p.Name,
	}
	var elements []bpmn.Element
	var flows []bpmn.SequenceFlow

	for _, fe := range p.FlowElements {
		local := fe.XMLName.Local
		switch local {
		case "sequenceFlow":
			flows = append(flows, bpmn.SequenceFlow{
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
			el := bpmn.Element{ID: fe.ID, Kind: kind, Name: fe.Name}
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
			if kind == bpmn.KindStartEvent {
				el.EventDefinition = mapStartEventDef(fe)
			}
			if kind == bpmn.KindReceiveTask && el.MessageRef != "" {
				// receiveTask message correlation
			}
			elements = append(elements, el)
		}
	}

	if def.ID == "" {
		return bpmn.ProcessDefinition{}, fmt.Errorf("bpmn xml: process id required")
	}
	def.Elements = elements
	def.Flows = flows
	return def, nil
}

func xmlLocalToKind(local string) (bpmn.ElementKind, bool) {
	switch local {
	case "startEvent":
		return bpmn.KindStartEvent, true
	case "endEvent":
		return bpmn.KindEndEvent, true
	case "userTask":
		return bpmn.KindUserTask, true
	case "scriptTask":
		return bpmn.KindScriptTask, true
	case "serviceTask":
		return bpmn.KindServiceTask, true
	case "sendTask":
		return bpmn.KindSendTask, true
	case "receiveTask":
		return bpmn.KindReceiveTask, true
	case "businessRuleTask":
		return bpmn.KindBusinessRuleTask, true
	case "exclusiveGateway":
		return bpmn.KindExclusiveGateway, true
	case "parallelGateway":
		return bpmn.KindParallelGateway, true
	case "inclusiveGateway":
		return bpmn.KindInclusiveGateway, true
	case "subProcess":
		return bpmn.KindSubProcess, true
	default:
		return "", false
	}
}

func mapStartEventDef(fe xmlFlowElement) *bpmn.EventDefinition {
	ed := &bpmn.EventDefinition{}
	switch {
	case fe.MessageEventDef != nil:
		ed.Type = bpmn.EventTypeMessage
		ed.MessageRef = fe.MessageEventDef.MessageRef
		if fe.ExtensionElements != nil {
			ed.CorrelationKey = fe.ExtensionElements.CorrelationKey
		}
	case fe.SignalEventDef != nil:
		ed.Type = bpmn.EventTypeSignal
		ed.SignalRef = fe.SignalEventDef.SignalRef
	case fe.TimerEventDef != nil:
		ed.Type = bpmn.EventTypeTimer
		if fe.TimerEventDef.TimeCycle != "" {
			ed.TimerCycle = strings.TrimSpace(fe.TimerEventDef.TimeCycle)
		} else {
			ed.TimerCycle = strings.TrimSpace(fe.TimerEventDef.TimeDate)
		}
	case fe.ConditionalDef != nil && fe.ConditionalDef.Condition != nil:
		ed.Type = bpmn.EventTypeConditional
		ed.Condition = strings.TrimSpace(conditionBody(fe.ConditionalDef.Condition))
	default:
		ed.Type = bpmn.EventTypeNone
	}
	if fe.ExtensionElements != nil {
		ed.CorrelationKey = firstNonEmpty(fe.ExtensionElements.CorrelationKey, fe.ExtensionElements.LCCorrelationKey)
		if ed.Condition == "" {
			ed.Condition = strings.TrimSpace(fe.ExtensionElements.PropertiesValue("condition"))
		}
	}
	return ed
}

func applyExtensions(el *bpmn.Element, ext *xmlExtensionElements) {
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

func buildXMLProcess(def bpmn.ProcessDefinition) xmlProcess {
	p := xmlProcess{
		ID:         def.ID,
		Name:       def.Name,
		Executable: true,
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

func elementToXML(el bpmn.Element) xmlFlowElement {
	fe := xmlFlowElement{
		XMLName: xml.Name{Space: BPMNNS, Local: kindToXMLLocal(el.Kind)},
		ID:      el.ID,
		Name:    el.Name,
	}
	if el.Script != "" {
		fe.Script = &xmlScript{Format: el.ScriptLang, Body: el.Script}
	}
	fe.ExtensionElements = extensionsFromElement(el)
	if el.Kind == bpmn.KindStartEvent {
		attachStartEventDef(&fe, el.EventDefinition)
	}
	return fe
}

func kindToXMLLocal(k bpmn.ElementKind) string {
	return string(k)
}

func extensionsFromElement(el bpmn.Element) *xmlExtensionElements {
	if el.TaskType == "" && el.Implementation == "" && el.ServiceURL == "" &&
		el.MessageRef == "" && el.DecisionRef == "" && len(el.Assignees) == 0 &&
		el.AssigneesVariable == "" && el.ApprovalMode == "" && !el.AutoComplete &&
		el.ReturnTo == "" && el.OnReject == "" && el.ScopeID == "" &&
		el.EntryRef == "" && el.ExitRef == "" && el.ScriptLang == "" && len(el.Properties) == 0 {
		return nil
	}
	ext := &xmlExtensionElements{
		TaskType:       el.TaskType,
		Implementation: el.Implementation,
		ServiceURL:     el.ServiceURL,
		ServiceMethod:  el.ServiceMethod,
		MessageRef:     el.MessageRef,
		DecisionRef:    el.DecisionRef,
		ScriptLang:     el.ScriptLang,
		AssigneesVar:   el.AssigneesVariable,
		ApprovalMode:   el.ApprovalMode,
		ReturnTo:       el.ReturnTo,
		OnReject:       el.OnReject,
		ScopeID:        el.ScopeID,
		EntryRef:       el.EntryRef,
		ExitRef:        el.ExitRef,
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

func attachStartEventDef(fe *xmlFlowElement, ed *bpmn.EventDefinition) {
	if ed == nil {
		return
	}
	switch ed.EffectiveEventType() {
	case bpmn.EventTypeMessage:
		fe.MessageEventDef = &xmlMessageEventDef{MessageRef: ed.MessageRef}
	case bpmn.EventTypeSignal:
		fe.SignalEventDef = &xmlSignalEventDef{SignalRef: ed.SignalRef}
	case bpmn.EventTypeTimer:
		fe.TimerEventDef = &xmlTimerEventDef{TimeCycle: ed.TimerCycle}
	case bpmn.EventTypeConditional:
		fe.ConditionalDef = &xmlConditionalDef{
			Condition: &xmlConditionExpr{Body: ed.Condition},
		}
	}
	if ed.CorrelationKey != "" && fe.ExtensionElements == nil {
		fe.ExtensionElements = &xmlExtensionElements{CorrelationKey: ed.CorrelationKey}
	} else if ed.CorrelationKey != "" && fe.ExtensionElements != nil {
		fe.ExtensionElements.CorrelationKey = ed.CorrelationKey
	}
}
