package bpmn

import "fmt"

// Validate checks structural correctness of a BPMN process definition.
func Validate(def ProcessDefinition) error {
	if def.ID == "" {
		return fmt.Errorf("process id is required")
	}
	if len(def.Elements) == 0 {
		return fmt.Errorf("process has no elements")
	}

	ids := make(map[string]struct{}, len(def.Elements))
	var startCount int
	for _, el := range def.Elements {
		if el.ID == "" {
			return fmt.Errorf("element id is required")
		}
		if _, dup := ids[el.ID]; dup {
			return fmt.Errorf("duplicate element id: %s", el.ID)
		}
		ids[el.ID] = struct{}{}
		if !isSupportedKind(el.Kind) {
			return fmt.Errorf("unsupported element type %q on %s", el.Kind, el.ID)
		}
		switch el.Kind {
		case KindStartEvent:
			startCount++
			if err := ValidateStartEventEventDefinition(el); err != nil {
				return err
			}
		case KindScriptTask:
			if el.Script == "" {
				return fmt.Errorf("scriptTask %s requires script", el.ID)
			}
		case KindUserTask:
			if err := ValidateUserTaskApproval(el); err != nil {
				return err
			}
		}
	}
	if startCount == 0 {
		return fmt.Errorf("process must have at least one startEvent")
	}

	flowIDs := make(map[string]struct{}, len(def.Flows))
	for _, flow := range def.Flows {
		if flow.ID == "" {
			return fmt.Errorf("sequence flow id is required")
		}
		if _, dup := flowIDs[flow.ID]; dup {
			return fmt.Errorf("duplicate flow id: %s", flow.ID)
		}
		flowIDs[flow.ID] = struct{}{}
		if flow.SourceRef == "" || flow.TargetRef == "" {
			return fmt.Errorf("flow %s requires sourceRef and targetRef", flow.ID)
		}
		if _, ok := ids[flow.SourceRef]; !ok {
			return fmt.Errorf("flow %s: unknown sourceRef %s", flow.ID, flow.SourceRef)
		}
		if _, ok := ids[flow.TargetRef]; !ok {
			return fmt.Errorf("flow %s: unknown targetRef %s", flow.ID, flow.TargetRef)
		}
	}

	return nil
}

func isSupportedKind(k ElementKind) bool {
	switch k {
	case KindStartEvent, KindEndEvent,
		KindUserTask, KindScriptTask,
		KindExclusiveGateway, KindParallelGateway, KindInclusiveGateway,
		KindSubProcess:
		return true
	default:
		return false
	}
}
