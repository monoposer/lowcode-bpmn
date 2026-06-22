package definition

import "fmt"

// Extension element kinds are modeled in the IR and executed via integration adapters.
var ExtensionKinds = []ElementKind{
	KindBoundaryEvent,
	KindIntermediateCatchEvent,
	KindIntermediateThrowEvent,
	KindEventBasedGateway,
	KindComplexGateway,
	KindCallActivity,
}

// IsExtensionKind reports whether k is extension-backed (not core-native execution).
func IsExtensionKind(k ElementKind) bool {
	switch k {
	case KindBoundaryEvent, KindIntermediateCatchEvent, KindIntermediateThrowEvent,
		KindEventBasedGateway, KindComplexGateway, KindCallActivity:
		return true
	default:
		return false
	}
}

func validateExtensionElement(el Element, ids map[string]Element) error {
	switch el.Kind {
	case KindBoundaryEvent:
		if el.AttachedToRef == "" {
			return fmt.Errorf("boundaryEvent %s requires attachedToRef", el.ID)
		}
		target, ok := ids[el.AttachedToRef]
		if !ok {
			return fmt.Errorf("boundaryEvent %s: unknown attachedToRef %s", el.ID, el.AttachedToRef)
		}
		if target.Kind == KindBoundaryEvent || target.Kind == KindStartEvent || target.Kind == KindEndEvent {
			return fmt.Errorf("boundaryEvent %s: cannot attach to %s", el.ID, target.Kind)
		}
		return ValidateEventDefinition(el)
	case KindIntermediateCatchEvent:
		return ValidateEventDefinition(el)
	case KindIntermediateThrowEvent:
		if el.EventDefinition != nil {
			return ValidateEventDefinition(el)
		}
		return nil
	case KindCallActivity:
		if el.CalledElement == "" {
			return fmt.Errorf("callActivity %s requires calledElement", el.ID)
		}
		return nil
	case KindEventBasedGateway, KindComplexGateway:
		return nil
	default:
		return nil
	}
}
