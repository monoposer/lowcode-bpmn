package definition

import (
	"fmt"

	pkgvars "github.com/monoposer/lowcode-bpmn/pkg/vars"
)

// EventDefinitionType maps to BPMN 2.0 start event definitions (JSON form).
// See schemas/process-definition.schema.json for the canonical protocol.
type EventDefinitionType string

const (
	EventTypeNone        EventDefinitionType = "none"        // blank / manual start (default)
	EventTypeMessage     EventDefinitionType = "message"     // messageEventDefinition
	EventTypeSignal      EventDefinitionType = "signal"      // signalEventDefinition
	EventTypeTimer       EventDefinitionType = "timer"       // timerEventDefinition (metadata only; scheduler external)
	EventTypeConditional EventDefinitionType = "conditional" // conditionalEventDefinition
	EventTypeError       EventDefinitionType = "error"       // errorEventDefinition (boundary)
)

// EventDefinition describes how a startEvent is triggered (BPMN 2.0 eventDefinition).
// This is NOT a sequenceFlow condition — those apply only at gateways when leaving a node.
type EventDefinition struct {
	Type           EventDefinitionType `json:"type,omitempty"`
	MessageRef     string              `json:"messageRef,omitempty"`
	SignalRef      string              `json:"signalRef,omitempty"`
	CorrelationKey string              `json:"correlationKey,omitempty"` // dot path → businessKey
	Condition      string              `json:"condition,omitempty"`      // evaluated against trigger payload
	TimerCycle     string              `json:"timerCycle,omitempty"`     // ISO-8601 repeat or cron (external scheduler)
	ErrorRef       string              `json:"errorRef,omitempty"`       // errorEventDefinition
}

// EffectiveEventType returns the event definition type, defaulting to none.
func (e *EventDefinition) EffectiveEventType() EventDefinitionType {
	if e == nil || e.Type == "" {
		return EventTypeNone
	}
	return e.Type
}

// ValidateStartEventEventDefinition checks BPMN start event definition rules.
func ValidateStartEventEventDefinition(el Element) error {
	return validateEventDefinitionForElement(el, KindStartEvent)
}

// ValidateEventDefinition checks eventDefinition on boundary and intermediate events.
func ValidateEventDefinition(el Element) error {
	if el.EventDefinition == nil {
		switch el.Kind {
		case KindBoundaryEvent, KindIntermediateCatchEvent:
			return fmt.Errorf("%s %s requires eventDefinition", el.Kind, el.ID)
		}
		return nil
	}
	return validateEventDefinitionBody(el)
}

func validateEventDefinitionForElement(el Element, expectKind ElementKind) error {
	ed := el.EventDefinition
	if ed == nil {
		return nil
	}
	if el.Kind != expectKind {
		return fmt.Errorf("element %s: unexpected kind for event validation", el.ID)
	}
	return validateEventDefinitionBody(el)
}

func validateEventDefinitionBody(el Element) error {
	ed := el.EventDefinition
	if ed == nil {
		return nil
	}
	switch ed.EffectiveEventType() {
	case EventTypeNone:
		if el.Kind == KindBoundaryEvent || el.Kind == KindIntermediateCatchEvent {
			return fmt.Errorf("%s %s requires a non-none eventDefinition", el.Kind, el.ID)
		}
		return nil
	case EventTypeMessage:
		if ed.MessageRef == "" {
			return fmt.Errorf("%s %s: message event requires messageRef", el.Kind, el.ID)
		}
	case EventTypeSignal:
		if ed.SignalRef == "" {
			return fmt.Errorf("%s %s: signal event requires signalRef", el.Kind, el.ID)
		}
	case EventTypeTimer:
		if ed.TimerCycle == "" {
			return fmt.Errorf("%s %s: timer event requires timerCycle", el.Kind, el.ID)
		}
	case EventTypeConditional:
		if ed.Condition == "" {
			return fmt.Errorf("%s %s: conditional event requires condition", el.Kind, el.ID)
		}
	case EventTypeError:
		if ed.ErrorRef == "" {
			return fmt.Errorf("%s %s: error event requires errorRef", el.Kind, el.ID)
		}
	default:
		return fmt.Errorf("%s %s: unsupported event type %q", el.Kind, el.ID, ed.Type)
	}
	return nil
}

// SignalStartMatch reports whether a startEvent should fire for signalRef and payload.
func SignalStartMatch(el Element, signalRef string, vars map[string]any) (bool, error) {
	if el.Kind != KindStartEvent || el.EventDefinition == nil {
		return false, nil
	}
	ed := el.EventDefinition
	if ed.EffectiveEventType() != EventTypeSignal {
		return false, nil
	}
	if ed.SignalRef != signalRef {
		return false, nil
	}
	return EvalCondition(ed.Condition, vars)
}

// ConditionalStartMatch reports whether a conditional startEvent should fire.
func ConditionalStartMatch(el Element, vars map[string]any) (bool, error) {
	if el.Kind != KindStartEvent || el.EventDefinition == nil {
		return false, nil
	}
	ed := el.EventDefinition
	if ed.EffectiveEventType() != EventTypeConditional {
		return false, nil
	}
	return EvalCondition(ed.Condition, vars)
}

// MessageStartMatch reports whether a startEvent should fire for the given messageRef and payload.
func MessageStartMatch(el Element, messageRef string, vars map[string]any) (bool, error) {
	if el.Kind != KindStartEvent || el.EventDefinition == nil {
		return false, nil
	}
	ed := el.EventDefinition
	if ed.EffectiveEventType() != EventTypeMessage {
		return false, nil
	}
	if ed.MessageRef != messageRef {
		return false, nil
	}
	ok, err := EvalCondition(ed.Condition, vars)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// BusinessKeyFromCorrelation reads businessKey from variables using correlationKey dot path.
func BusinessKeyFromCorrelation(vars map[string]any, correlationKey string) string {
	if correlationKey == "" {
		return ""
	}
	v, ok := pkgvars.ResolvePath(vars, correlationKey)
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}
