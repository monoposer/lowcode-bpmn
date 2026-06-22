package definition

import "fmt"

// Registry indexes a validated process definition for fast traversal.
type Registry struct {
	Def            ProcessDefinition
	Elements       map[string]Element
	Outgoing       map[string][]SequenceFlow
	Incoming       map[string][]SequenceFlow
	StartEvents    []string
	EndEvents      map[string]struct{}
	BoundaryByHost map[string][]Element // attachedToRef → boundary events
	ElementLane    map[string]Lane      // element id → lane
}

// BuildRegistry validates and indexes a process definition.
func BuildRegistry(def ProcessDefinition) (*Registry, error) {
	if err := Validate(def); err != nil {
		return nil, err
	}
	r := &Registry{
		Def:            def,
		Elements:       make(map[string]Element, len(def.Elements)),
		Outgoing:       make(map[string][]SequenceFlow),
		Incoming:       make(map[string][]SequenceFlow),
		EndEvents:      make(map[string]struct{}),
		BoundaryByHost: make(map[string][]Element),
		ElementLane:    make(map[string]Lane),
	}
	for _, el := range def.Elements {
		r.Elements[el.ID] = el
		switch el.Kind {
		case KindStartEvent:
			r.StartEvents = append(r.StartEvents, el.ID)
		case KindEndEvent:
			r.EndEvents[el.ID] = struct{}{}
		case KindBoundaryEvent:
			if el.AttachedToRef != "" {
				r.BoundaryByHost[el.AttachedToRef] = append(r.BoundaryByHost[el.AttachedToRef], el)
			}
		}
	}
	for _, lane := range def.LaneSet {
		for _, ref := range lane.FlowNodeRefs {
			r.ElementLane[ref] = lane
		}
	}
	for _, flow := range def.Flows {
		if _, ok := r.Elements[flow.SourceRef]; !ok {
			return nil, fmt.Errorf("flow %s: unknown source %s", flow.ID, flow.SourceRef)
		}
		if _, ok := r.Elements[flow.TargetRef]; !ok {
			return nil, fmt.Errorf("flow %s: unknown target %s", flow.ID, flow.TargetRef)
		}
		r.Outgoing[flow.SourceRef] = append(r.Outgoing[flow.SourceRef], flow)
		r.Incoming[flow.TargetRef] = append(r.Incoming[flow.TargetRef], flow)
	}
	return r, nil
}

// Element returns an element by ID.
func (r *Registry) Element(id string) (Element, bool) {
	el, ok := r.Elements[id]
	return el, ok
}

// OutgoingFlows returns sequence flows leaving an element.
func (r *Registry) OutgoingFlows(id string) []SequenceFlow {
	return r.Outgoing[id]
}

// IncomingFlows returns sequence flows entering an element.
func (r *Registry) IncomingFlows(id string) []SequenceFlow {
	return r.Incoming[id]
}

// IsEndEvent reports whether the element is an end event.
func (r *Registry) IsEndEvent(id string) bool {
	_, ok := r.EndEvents[id]
	return ok
}

// IsJoinGateway reports whether a parallel/inclusive gateway has multiple incoming flows.
func (r *Registry) IsJoinGateway(id string) bool {
	el, ok := r.Elements[id]
	if !ok {
		return false
	}
	switch el.Kind {
	case KindParallelGateway, KindInclusiveGateway:
		return len(r.Incoming[id]) > 1
	default:
		return false
	}
}

// BoundaryEvents returns boundary events attached to the given activity id.
func (r *Registry) BoundaryEvents(hostElementID string) []Element {
	return r.BoundaryByHost[hostElementID]
}

// LaneForElement returns the lane containing a flow node, if any.
func (r *Registry) LaneForElement(elementID string) (Lane, bool) {
	lane, ok := r.ElementLane[elementID]
	return lane, ok
}
