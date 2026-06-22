package engine

import (
	"context"
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

func (e *Engine) activateExtensionElement(ctx context.Context, state *execState, el bpmn.Element, act *ActivityInstance) error {
	act.Input = extensionActivityInput(state, el)
	if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
		return err
	}
	state.inst.ActiveElements = appendUnique(state.inst.ActiveElements, el.ID)
	state.inst.UpdatedAt = time.Now().UTC()
	return e.store.UpdateProcessInstance(ctx, state.inst)
}

func extensionActivityInput(state *execState, el bpmn.Element) map[string]any {
	in := map[string]any{
		"extensionRequired": true,
		"kind":              string(el.Kind),
	}
	if el.CalledElement != "" {
		in["calledElement"] = el.CalledElement
	}
	if el.ExtensionHandler != "" {
		in["extensionHandler"] = el.ExtensionHandler
	}
	if el.EventDefinition != nil {
		in["eventDefinition"] = el.EventDefinition
	}
	if el.MultiInstance != nil {
		in["multiInstance"] = el.MultiInstance
	}
	if lane, ok := state.reg.LaneForElement(el.ID); ok {
		in["laneId"] = lane.ID
		in["laneName"] = lane.Name
	}
	return in
}

func attachBoundaryMetadata(act *ActivityInstance, reg *bpmn.Registry, elementID string) {
	boundary := reg.BoundaryEvents(elementID)
	if len(boundary) == 0 {
		return
	}
	refs := make([]string, len(boundary))
	for i, b := range boundary {
		refs[i] = b.ID
	}
	if act.Input == nil {
		act.Input = map[string]any{}
	}
	act.Input["boundaryEvents"] = refs
}
