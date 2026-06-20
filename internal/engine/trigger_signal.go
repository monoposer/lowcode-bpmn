package engine

import (
	"context"
	"fmt"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

// TriggerSignalRequest fires BPMN signal start events (signalEventDefinition).
type TriggerSignalRequest struct {
	TenantID   string
	SignalRef  string
	Variables  map[string]any
	BusinessKey string
}

// TriggerSignal scans deployed processes and starts those with a matching signal startEvent.
func (e *Engine) TriggerSignal(ctx context.Context, req TriggerSignalRequest) (*TriggerMessageResult, error) {
	if e.store == nil {
		return nil, fmt.Errorf("engine: store not configured")
	}
	if req.TenantID == "" || req.SignalRef == "" {
		return nil, fmt.Errorf("tenantId and signalRef required")
	}
	processes, err := e.store.ListProcesses(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}

	result := &TriggerMessageResult{MessageRef: req.SignalRef}
	for _, dp := range processes {
		if dp == nil {
			continue
		}
		for _, el := range dp.Definition.Elements {
			if el.Kind != bpmn.KindStartEvent {
				continue
			}
			match, err := bpmn.SignalStartMatch(el, req.SignalRef, req.Variables)
			if err != nil {
				result.Matches = append(result.Matches, TriggerMessageMatch{
					ProcessKey: dp.Key, StartElementID: el.ID, Error: err.Error(),
				})
				continue
			}
			if !match {
				continue
			}
			inst, startErr := e.StartProcess(ctx, StartProcessRequest{
				TenantID:        req.TenantID,
				ProcessKey:      dp.Key,
				BusinessKey:     req.BusinessKey,
				Variables:       req.Variables,
				StartElementIDs: []string{el.ID},
			})
			m := TriggerMessageMatch{ProcessKey: dp.Key, StartElementID: el.ID}
			if startErr != nil {
				m.Error = startErr.Error()
			} else if inst != nil {
				m.InstanceID = inst.ID.String()
			}
			result.Matches = append(result.Matches, m)
		}
	}
	return result, nil
}

// TriggerConditionalRequest evaluates conditional start events against variables.
type TriggerConditionalRequest struct {
	TenantID    string
	Variables   map[string]any
	BusinessKey string
}

// TriggerConditional starts processes whose conditional startEvent matches variables.
func (e *Engine) TriggerConditional(ctx context.Context, req TriggerConditionalRequest) (*TriggerMessageResult, error) {
	if e.store == nil {
		return nil, fmt.Errorf("engine: store not configured")
	}
	if req.TenantID == "" {
		return nil, fmt.Errorf("tenantId required")
	}
	processes, err := e.store.ListProcesses(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}

	result := &TriggerMessageResult{MessageRef: "conditional"}
	for _, dp := range processes {
		if dp == nil {
			continue
		}
		for _, el := range dp.Definition.Elements {
			if el.Kind != bpmn.KindStartEvent {
				continue
			}
			match, err := bpmn.ConditionalStartMatch(el, req.Variables)
			if err != nil {
				result.Matches = append(result.Matches, TriggerMessageMatch{
					ProcessKey: dp.Key, StartElementID: el.ID, Error: err.Error(),
				})
				continue
			}
			if !match {
				continue
			}
			inst, startErr := e.StartProcess(ctx, StartProcessRequest{
				TenantID:        req.TenantID,
				ProcessKey:      dp.Key,
				BusinessKey:     req.BusinessKey,
				Variables:       req.Variables,
				StartElementIDs: []string{el.ID},
			})
			m := TriggerMessageMatch{ProcessKey: dp.Key, StartElementID: el.ID}
			if startErr != nil {
				m.Error = startErr.Error()
			} else if inst != nil {
				m.InstanceID = inst.ID.String()
			}
			result.Matches = append(result.Matches, m)
		}
	}
	return result, nil
}
