package engine

import (
	"context"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

// matchCollaborationMessageFlows starts partner processes when a messageFlow messageRef matches.
func (e *Engine) matchCollaborationMessageFlows(ctx context.Context, tenantID, messageRef string, vars map[string]any) ([]TriggerMessageMatch, error) {
	processes, err := e.store.ListProcesses(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var matches []TriggerMessageMatch
	for _, dp := range processes {
		if dp == nil || dp.Definition.Collaboration == nil {
			continue
		}
		c := dp.Definition.Collaboration
		poolByID := make(map[string]bpmn.Pool, len(c.Pools))
		for _, p := range c.Pools {
			poolByID[p.ID] = p
		}
		for _, mf := range c.MessageFlows {
			if mf.MessageRef != messageRef {
				continue
			}
			targetPool, ok := poolByID[mf.TargetRef]
			if !ok || targetPool.ProcessRef == "" {
				continue
			}
			partnerKey := targetPool.ProcessRef
			var partnerDef *bpmn.ProcessDefinition
			for _, p2 := range processes {
				if p2 != nil && p2.Key == partnerKey {
					def := p2.Definition
					partnerDef = &def
					break
				}
			}
			if partnerDef == nil {
				matches = append(matches, TriggerMessageMatch{
					ProcessKey: partnerKey,
					Error:      "collaboration partner process not deployed",
				})
				continue
			}
			for _, el := range partnerDef.Elements {
				if el.Kind != bpmn.KindStartEvent {
					continue
				}
				okMatch, err := bpmn.MessageStartMatch(el, messageRef, vars)
				if err != nil {
					matches = append(matches, TriggerMessageMatch{
						ProcessKey: partnerKey, StartElementID: el.ID, Error: err.Error(),
					})
					continue
				}
				if !okMatch {
					continue
				}
				inst, startErr := e.StartProcess(ctx, StartProcessRequest{
					TenantID:        tenantID,
					ProcessKey:      partnerKey,
					BusinessKey:     bpmn.BusinessKeyFromCorrelation(vars, el.EventDefinition.CorrelationKey),
					Variables:       vars,
					StartElementIDs: []string{el.ID},
				})
				m := TriggerMessageMatch{ProcessKey: partnerKey, StartElementID: el.ID}
				if startErr != nil {
					m.Error = startErr.Error()
				} else if inst != nil {
					m.InstanceID = inst.ID.String()
				}
				matches = append(matches, m)
			}
		}
	}
	return matches, nil
}
