package engine

import (
	"context"
	"fmt"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

// TriggerMessageRequest fires BPMN message start events (BPMN 2.0 messageEventDefinition).
type TriggerMessageRequest struct {
	TenantID    string
	MessageRef  string
	Variables   map[string]any
	BusinessKey string
}

// TriggerMessageMatch describes one process start attempt.
type TriggerMessageMatch struct {
	ProcessKey     string `json:"process_key"`
	StartElementID string `json:"start_element_id"`
	InstanceID     string `json:"instance_id,omitempty"`
	Skipped        bool   `json:"skipped,omitempty"`
	SkipReason     string `json:"skip_reason,omitempty"`
	Error          string `json:"error,omitempty"`
}

// TriggerMessageResult aggregates starts from one inbound message (e.g. Airtable webhook).
type TriggerMessageResult struct {
	MessageRef      string                 `json:"message_ref"`
	Matches         []TriggerMessageMatch  `json:"matches"`
	BoundaryMatches []TriggerBoundaryMatch `json:"boundary_matches,omitempty"`
}

// TriggerMessage scans deployed processes and starts those with a matching message startEvent.
// Upper adapters (Airtable, Feishu, etc.) POST canonical payload here after normalizing their webhook.
func (e *Engine) TriggerMessage(ctx context.Context, req TriggerMessageRequest) (*TriggerMessageResult, error) {
	if e.store == nil {
		return nil, fmt.Errorf("engine: store not configured")
	}
	if req.TenantID == "" || req.MessageRef == "" {
		return nil, fmt.Errorf("tenantId and messageRef required")
	}
	processes, err := e.store.ListProcesses(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}

	result := &TriggerMessageResult{MessageRef: req.MessageRef}
	for _, dp := range processes {
		if dp == nil {
			continue
		}
		for _, el := range dp.Definition.Elements {
			if el.Kind != bpmn.KindStartEvent {
				continue
			}
			match, err := bpmn.MessageStartMatch(el, req.MessageRef, req.Variables)
			if err != nil {
				result.Matches = append(result.Matches, TriggerMessageMatch{
					ProcessKey:     dp.Key,
					StartElementID: el.ID,
					Error:          err.Error(),
				})
				continue
			}
			if !match {
				continue
			}
			bk := req.BusinessKey
			if bk == "" && el.EventDefinition != nil {
				bk = bpmn.BusinessKeyFromCorrelation(req.Variables, el.EventDefinition.CorrelationKey)
			}
			if bk != "" {
				existing, findErr := e.store.FindRunningInstanceByBusinessKey(ctx, req.TenantID, dp.Key, bk)
				if findErr != nil {
					result.Matches = append(result.Matches, TriggerMessageMatch{
						ProcessKey:     dp.Key,
						StartElementID: el.ID,
						Error:          findErr.Error(),
					})
					continue
				}
				if existing != nil {
					result.Matches = append(result.Matches, TriggerMessageMatch{
						ProcessKey:     dp.Key,
						StartElementID: el.ID,
						InstanceID:     existing.ID.String(),
						Skipped:        true,
						SkipReason:     "idempotent_duplicate",
					})
					continue
				}
			}
			inst, startErr := e.StartProcess(ctx, StartProcessRequest{
				TenantID:        req.TenantID,
				ProcessKey:      dp.Key,
				BusinessKey:     bk,
				Variables:       req.Variables,
				StartElementIDs: []string{el.ID},
			})
			m := TriggerMessageMatch{
				ProcessKey:     dp.Key,
				StartElementID: el.ID,
			}
			if startErr != nil {
				m.Error = startErr.Error()
			} else if inst != nil {
				m.InstanceID = inst.ID.String()
			}
			result.Matches = append(result.Matches, m)
		}
	}

	boundaryMatches, err := e.matchBoundaryEventsOnInstances(ctx, req.TenantID, req.MessageRef, req.Variables)
	if err != nil {
		return result, err
	}
	result.BoundaryMatches = boundaryMatches

	collabMatches, err := e.matchCollaborationMessageFlows(ctx, req.TenantID, req.MessageRef, req.Variables)
	if err != nil {
		return result, err
	}
	result.Matches = append(result.Matches, collabMatches...)
	return result, nil
}
