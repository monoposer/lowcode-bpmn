package definition

import "fmt"

// ValidateCollaboration checks pool and message flow references when collaboration metadata is present.
func ValidateCollaboration(def ProcessDefinition) error {
	if def.Collaboration == nil {
		return nil
	}
	c := def.Collaboration
	poolIDs := make(map[string]struct{}, len(c.Pools))
	for _, p := range c.Pools {
		if p.ID == "" {
			return fmt.Errorf("pool id is required")
		}
		poolIDs[p.ID] = struct{}{}
	}
	elementIDs := make(map[string]struct{}, len(def.Elements))
	for _, el := range def.Elements {
		elementIDs[el.ID] = struct{}{}
	}
	for _, mf := range c.MessageFlows {
		if mf.ID == "" {
			return fmt.Errorf("messageFlow id is required")
		}
		if mf.SourceRef == "" || mf.TargetRef == "" {
			return fmt.Errorf("messageFlow %s requires sourceRef and targetRef", mf.ID)
		}
		if !refExists(mf.SourceRef, poolIDs, elementIDs) {
			return fmt.Errorf("messageFlow %s: unknown sourceRef %s", mf.ID, mf.SourceRef)
		}
		if !refExists(mf.TargetRef, poolIDs, elementIDs) {
			return fmt.Errorf("messageFlow %s: unknown targetRef %s", mf.ID, mf.TargetRef)
		}
	}
	return nil
}

func refExists(ref string, pools, elements map[string]struct{}) bool {
	if _, ok := pools[ref]; ok {
		return true
	}
	_, ok := elements[ref]
	return ok
}
