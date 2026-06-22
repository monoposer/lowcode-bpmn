package definition

import "fmt"

// Validate checks structural correctness of a BPMN process definition.
func Validate(def ProcessDefinition) error {
	if def.ID == "" {
		return fmt.Errorf("process id is required")
	}
	if len(def.Elements) == 0 {
		return fmt.Errorf("process has no elements")
	}

	ids := make(map[string]Element, len(def.Elements))
	var startCount int
	for _, el := range def.Elements {
		if el.ID == "" {
			return fmt.Errorf("element id is required")
		}
		if _, dup := ids[el.ID]; dup {
			return fmt.Errorf("duplicate element id: %s", el.ID)
		}
		ids[el.ID] = el
		if !isSupportedKind(el.Kind) {
			return fmt.Errorf("unsupported element type %q on %s", el.Kind, el.ID)
		}
		switch el.Kind {
		case KindStartEvent:
			startCount++
			if err := ValidateStartEventEventDefinition(el); err != nil {
				return err
			}
		case KindScriptTask, KindServiceTask, KindSendTask, KindReceiveTask, KindBusinessRuleTask, KindUserTask:
			if err := ValidateTaskElement(el); err != nil {
				return err
			}
		default:
			if IsExtensionKind(el.Kind) {
				if err := validateExtensionElement(el, ids); err != nil {
					return err
				}
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

	for _, lane := range def.LaneSet {
		if lane.ID == "" {
			return fmt.Errorf("lane id is required")
		}
		for _, ref := range lane.FlowNodeRefs {
			if _, ok := ids[ref]; !ok {
				return fmt.Errorf("lane %s: unknown flowNodeRef %s", lane.ID, ref)
			}
		}
	}

	return ValidateCollaboration(def)
}

func isSupportedKind(k ElementKind) bool {
	switch k {
	case KindStartEvent, KindEndEvent,
		KindUserTask, KindScriptTask,
		KindServiceTask, KindSendTask, KindReceiveTask, KindBusinessRuleTask,
		KindExclusiveGateway, KindParallelGateway, KindInclusiveGateway,
		KindSubProcess,
		KindBoundaryEvent, KindIntermediateCatchEvent, KindIntermediateThrowEvent,
		KindEventBasedGateway, KindComplexGateway, KindCallActivity:
		return true
	default:
		return false
	}
}
