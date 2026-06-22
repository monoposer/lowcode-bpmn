package definition

// BoundaryCancelsActivity reports whether a fired boundary event cancels its host activity.
// BPMN default is interrupting (cancel) when cancelActivity is omitted.
func BoundaryCancelsActivity(el Element) bool {
	if el.CancelActivity != nil {
		return *el.CancelActivity
	}
	return true
}

// BoundaryMessageMatch reports whether a boundaryEvent should fire for messageRef and payload.
func BoundaryMessageMatch(el Element, messageRef string, vars map[string]any) (bool, error) {
	if el.Kind != KindBoundaryEvent || el.EventDefinition == nil {
		return false, nil
	}
	ed := el.EventDefinition
	if ed.EffectiveEventType() != EventTypeMessage {
		return false, nil
	}
	if ed.MessageRef != messageRef {
		return false, nil
	}
	return EvalCondition(ed.Condition, vars)
}

// BoundarySignalMatch reports whether a boundaryEvent should fire for signalRef.
func BoundarySignalMatch(el Element, signalRef string, vars map[string]any) (bool, error) {
	if el.Kind != KindBoundaryEvent || el.EventDefinition == nil {
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

// IsBoundaryHostKind reports whether an element kind can host boundary events.
func IsBoundaryHostKind(k ElementKind) bool {
	switch k {
	case KindUserTask, KindScriptTask, KindReceiveTask, KindServiceTask,
		KindSendTask, KindBusinessRuleTask, KindSubProcess, KindCallActivity:
		return true
	default:
		return false
	}
}
