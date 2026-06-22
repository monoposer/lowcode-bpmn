package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

// TriggerBoundaryMatch describes a boundary event fired on a running instance.
type TriggerBoundaryMatch struct {
	ProcessInstanceID string `json:"process_instance_id"`
	ProcessKey        string `json:"process_key,omitempty"`
	HostElementID     string `json:"host_element_id"`
	BoundaryElementID string `json:"boundary_element_id"`
	CancelledHost     bool   `json:"cancelled_host,omitempty"`
	Error             string `json:"error,omitempty"`
}

// TriggerBoundaryRequest fires a timer or explicitly addressed boundary event.
type TriggerBoundaryRequest struct {
	TenantID          string
	ProcessInstanceID uuid.UUID
	HostElementID     string // optional when BoundaryElementID is set
	BoundaryElementID string
	Variables         map[string]any
}

func boundaryFiredKey(hostActivityID uuid.UUID, boundaryElementID string) string {
	return fmt.Sprintf("boundary_fired:%s:%s", hostActivityID, boundaryElementID)
}

func boundaryAlreadyFired(state *execState, hostActivityID uuid.UUID, boundaryElementID string) bool {
	if state.inst.InternalState == nil {
		return false
	}
	_, ok := state.inst.InternalState[boundaryFiredKey(hostActivityID, boundaryElementID)]
	return ok
}

func markBoundaryFired(state *execState, hostActivityID uuid.UUID, boundaryElementID string) {
	if state.inst.InternalState == nil {
		state.inst.InternalState = make(map[string]any)
	}
	state.inst.InternalState[boundaryFiredKey(hostActivityID, boundaryElementID)] = true
}

func (e *Engine) matchBoundaryEventsOnInstances(ctx context.Context, tenantID, messageRef string, vars map[string]any) ([]TriggerBoundaryMatch, error) {
	instances, err := e.store.ListRunningInstances(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var matches []TriggerBoundaryMatch
	for _, inst := range instances {
		if inst == nil {
			continue
		}
		m, err := e.matchBoundaryEventsOnInstance(ctx, inst, messageRef, vars)
		if err != nil {
			return matches, err
		}
		matches = append(matches, m...)
	}
	return matches, nil
}

func (e *Engine) matchBoundaryEventsOnInstance(ctx context.Context, inst *ProcessInstance, messageRef string, vars map[string]any) ([]TriggerBoundaryMatch, error) {
	reg, err := registryForInstance(inst)
	if err != nil {
		return nil, err
	}
	active, err := e.store.ListActiveActivities(ctx, inst.ID)
	if err != nil {
		return nil, err
	}
	var matches []TriggerBoundaryMatch
	for _, hostAct := range active {
		if hostAct == nil || !bpmn.IsBoundaryHostKind(hostAct.ElementKind) {
			continue
		}
		for _, boundaryEl := range reg.BoundaryEvents(hostAct.ElementID) {
			ok, matchErr := bpmn.BoundaryMessageMatch(boundaryEl, messageRef, vars)
			if matchErr != nil {
				matches = append(matches, TriggerBoundaryMatch{
					ProcessInstanceID: inst.ID.String(),
					ProcessKey:        inst.ProcessKey,
					HostElementID:     hostAct.ElementID,
					BoundaryElementID: boundaryEl.ID,
					Error:             matchErr.Error(),
				})
				continue
			}
			if !ok {
				continue
			}
			fired, fireErr := e.fireBoundaryEvent(ctx, inst, reg, hostAct, boundaryEl, vars)
			m := TriggerBoundaryMatch{
				ProcessInstanceID: inst.ID.String(),
				ProcessKey:        inst.ProcessKey,
				HostElementID:     hostAct.ElementID,
				BoundaryElementID: boundaryEl.ID,
				CancelledHost:     fired.cancelledHost,
			}
			if fireErr != nil {
				m.Error = fireErr.Error()
			}
			matches = append(matches, m)
		}
	}
	return matches, nil
}

type boundaryFireResult struct {
	cancelledHost bool
}

func (e *Engine) fireBoundaryEvent(ctx context.Context, inst *ProcessInstance, reg *bpmn.Registry, hostAct *ActivityInstance, boundaryEl bpmn.Element, vars map[string]any) (boundaryFireResult, error) {
	var result boundaryFireResult
	err := e.store.WithTx(ctx, func(tx Store) error {
		runner := &Engine{store: tx, script: e.script, async: false}
		instCopy, err := tx.GetProcessInstanceForUpdate(ctx, inst.ID)
		if err != nil || instCopy == nil {
			return fmt.Errorf("instance not found: %s", inst.ID)
		}
		hostCopy, err := tx.GetActivityInstance(ctx, hostAct.ID)
		if err != nil || hostCopy == nil {
			return fmt.Errorf("host activity not found")
		}
		if hostCopy.Status != ActivityStatusActive {
			return nil
		}
		state := &execState{inst: instCopy, reg: reg}
		if boundaryAlreadyFired(state, hostCopy.ID, boundaryEl.ID) {
			return nil
		}
		markBoundaryFired(state, hostCopy.ID, boundaryEl.ID)
		for k, v := range vars {
			state.inst.Variables[k] = v
		}
		now := time.Now().UTC()
		bAct := &ActivityInstance{
			ID:                uuid.New(),
			ProcessInstanceID: state.inst.ID,
			ElementID:         boundaryEl.ID,
			ElementKind:       boundaryEl.Kind,
			Status:            ActivityStatusCompleted,
			ScopeID:           hostCopy.ScopeID,
			BranchFlowID:      hostCopy.BranchFlowID,
			Input: map[string]any{
				"hostElementId":     hostCopy.ElementID,
				"hostActivityId":    hostCopy.ID.String(),
				"boundaryTriggered": true,
			},
			StartedAt: now,
			EndedAt:   &now,
		}
		if err := tx.CreateActivityInstance(ctx, bAct); err != nil {
			return err
		}
		if bpmn.BoundaryCancelsActivity(boundaryEl) {
			result.cancelledHost = true
			hostCopy.Status = ActivityStatusCancelled
			hostCopy.Outcome = OutcomeCancelled
			hostCopy.EndedAt = &now
			state.inst.ActiveElements = removeString(state.inst.ActiveElements, hostCopy.ElementID)
			if err := tx.UpdateActivityInstance(ctx, hostCopy); err != nil {
				return err
			}
		}
		state.inst.UpdatedAt = now
		if err := tx.UpdateProcessInstance(ctx, state.inst); err != nil {
			return err
		}
		return runner.followBoundaryOutgoing(ctx, state, boundaryEl.ID, hostCopy.BranchFlowID)
	})
	return result, err
}

func (e *Engine) followBoundaryOutgoing(ctx context.Context, state *execState, boundaryElementID, branchFlowID string) error {
	outgoing := state.reg.OutgoingFlows(boundaryElementID)
	if len(outgoing) == 0 {
		return e.tryCompleteProcess(ctx, state)
	}
	for _, flow := range outgoing {
		if err := e.activateViaFlow(ctx, state, flow, branchFlowID); err != nil {
			return err
		}
	}
	return nil
}

// TriggerBoundary fires a boundary event by element id (timer scheduler / explicit API).
func (e *Engine) TriggerBoundary(ctx context.Context, req TriggerBoundaryRequest) (*TriggerBoundaryMatch, error) {
	if e.store == nil {
		return nil, fmt.Errorf("engine: store not configured")
	}
	if req.TenantID == "" || req.ProcessInstanceID == uuid.Nil || req.BoundaryElementID == "" {
		return nil, fmt.Errorf("tenantId, processInstanceId, and boundaryElementId required")
	}
	inst, err := e.store.GetProcessInstance(ctx, req.ProcessInstanceID)
	if err != nil || inst == nil {
		return nil, fmt.Errorf("process instance not found")
	}
	if inst.TenantID != req.TenantID {
		return nil, fmt.Errorf("tenant mismatch")
	}
	if inst.Status != ProcessStatusRunning {
		return nil, fmt.Errorf("process not running: %s", inst.Status)
	}
	reg, err := registryForInstance(inst)
	if err != nil {
		return nil, err
	}
	boundaryEl, ok := reg.Element(req.BoundaryElementID)
	if !ok || boundaryEl.Kind != bpmn.KindBoundaryEvent {
		return nil, fmt.Errorf("boundary event not found: %s", req.BoundaryElementID)
	}
	active, err := e.store.ListActiveActivities(ctx, inst.ID)
	if err != nil {
		return nil, err
	}
	var hostAct *ActivityInstance
	for _, act := range active {
		if act == nil || act.Status != ActivityStatusActive {
			continue
		}
		if req.HostElementID != "" && act.ElementID != req.HostElementID {
			continue
		}
		if act.ElementID == boundaryEl.AttachedToRef {
			hostAct = act
			break
		}
	}
	if hostAct == nil {
		return nil, fmt.Errorf("no active host activity for boundary %s", req.BoundaryElementID)
	}
	fired, fireErr := e.fireBoundaryEvent(ctx, inst, reg, hostAct, boundaryEl, req.Variables)
	match := &TriggerBoundaryMatch{
		ProcessInstanceID: inst.ID.String(),
		ProcessKey:        inst.ProcessKey,
		HostElementID:     hostAct.ElementID,
		BoundaryElementID: boundaryEl.ID,
		CancelledHost:     fired.cancelledHost,
	}
	if fireErr != nil {
		match.Error = fireErr.Error()
		return match, fireErr
	}
	return match, nil
}
