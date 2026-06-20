package bpmn

import "fmt"

// ResolveReturnTarget picks the element to re-activate after reject.
// Explicit returnTo wins; otherwise walk upstream to the previous userTask,
// stopping at parallel join gateways (uses scope entryRef as fallback).
func ResolveReturnTarget(reg *Registry, fromElementID, explicitReturnTo string) (string, error) {
	if explicitReturnTo != "" {
		if _, ok := reg.Element(explicitReturnTo); !ok {
			return "", fmt.Errorf("returnTo element not found: %s", explicitReturnTo)
		}
		return explicitReturnTo, nil
	}

	fromEl, ok := reg.Element(fromElementID)
	if !ok {
		return "", fmt.Errorf("element not found: %s", fromElementID)
	}

	if target := upstreamUserTask(reg, fromElementID); target != "" {
		return target, nil
	}

	scopeID := fromEl.ScopeID
	if scopeID == "" {
		return "", fmt.Errorf("no return target for %s (set returnTo on userTask)", fromElementID)
	}

	for _, el := range reg.Def.Elements {
		if el.Kind == KindSubProcess && (el.ScopeID == scopeID || el.ID == scopeID) && el.EntryRef != "" {
			if _, ok := reg.Element(el.EntryRef); ok {
				if el.EntryRef != fromElementID {
					return el.EntryRef, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no return target for %s in scope %s", fromElementID, scopeID)
}

func upstreamUserTask(reg *Registry, startID string) string {
	seen := map[string]struct{}{startID: {}}
	queue := reg.IncomingFlows(startID)

	for len(queue) > 0 {
		f := queue[0]
		queue = queue[1:]

		srcID := f.SourceRef
		if _, dup := seen[srcID]; dup {
			continue
		}
		seen[srcID] = struct{}{}

		el, ok := reg.Element(srcID)
		if !ok {
			continue
		}

		switch el.Kind {
		case KindUserTask:
			return srcID
		case KindParallelGateway, KindInclusiveGateway:
			if len(reg.IncomingFlows(srcID)) > 1 {
				// join gateway — stop walking; caller may use scope entryRef
				continue
			}
		case KindStartEvent, KindEndEvent:
			continue
		}

		for _, in := range reg.IncomingFlows(srcID) {
			queue = append(queue, in)
		}
	}
	return ""
}

// ScopeElementIDs returns element ids belonging to a scope (including the subProcess marker).
func ScopeElementIDs(reg *Registry, scopeID string) []string {
	if scopeID == "" {
		return nil
	}
	var ids []string
	for _, el := range reg.Def.Elements {
		if el.ID == scopeID || el.ScopeID == scopeID {
			ids = append(ids, el.ID)
		}
	}
	return ids
}

// ScopeGatewayIDs returns gateway ids inside a scope.
func ScopeGatewayIDs(reg *Registry, scopeID string) []string {
	var ids []string
	for _, el := range reg.Def.Elements {
		if el.ScopeID != scopeID && el.ID != scopeID {
			continue
		}
		switch el.Kind {
		case KindParallelGateway, KindInclusiveGateway, KindExclusiveGateway:
			ids = append(ids, el.ID)
		}
	}
	return ids
}
