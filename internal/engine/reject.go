package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

const (
	OutcomeApprove   = "approve"
	OutcomeReject    = "reject"
	OutcomeCancelled = "cancelled"

	OnRejectReturn         = "return"
	OnRejectTerminateScope = "terminateScope"
)

func parseOnReject(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "terminatescope", "terminate_scope", "terminate":
		return OnRejectTerminateScope
	default:
		return OnRejectReturn
	}
}

func (e *Engine) handleTaskReject(
	ctx context.Context,
	state *execState,
	act *ActivityInstance,
	el bpmn.Element,
	req CompleteTaskRequest,
) error {
	now := time.Now().UTC()
	act.Status = ActivityStatusCompleted
	act.Outcome = OutcomeReject
	act.EndedAt = &now
	act.PendingAssignees = nil

	state.inst.Variables["approved"] = false
	if req.Variables == nil {
		req.Variables = map[string]any{}
	}
	req.Variables["approved"] = false
	for k, v := range req.Variables {
		state.inst.Variables[k] = v
	}

	if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
		return err
	}
	state.inst.ActiveElements = removeString(state.inst.ActiveElements, act.ElementID)

	onReject := parseOnReject(el.OnReject)
	if onReject == OnRejectTerminateScope {
		scopeID := act.ScopeID
		if scopeID == "" {
			scopeID = el.ScopeID
		}
		if scopeID == "" {
			return e.cancelInstance(ctx, state.inst, req.Comment)
		}
		return e.terminateScope(ctx, state, scopeID, req.Comment)
	}

	if err := e.cancelBranchPeers(ctx, state, act); err != nil {
		return err
	}

	target, err := bpmn.ResolveReturnTarget(state.reg, act.ElementID, el.ReturnTo)
	if err != nil {
		return err
	}

	clearJoinStateInScope(state.inst, state.reg, act.ScopeID)

	state.inst.UpdatedAt = now
	if err := e.store.UpdateProcessInstance(ctx, state.inst); err != nil {
		return err
	}

	return e.activateElement(ctx, state, target, act.BranchFlowID)
}

func (e *Engine) cancelBranchPeers(ctx context.Context, state *execState, rejected *ActivityInstance) error {
	if rejected.BranchFlowID == "" {
		return nil
	}
	active, err := e.store.ListActiveActivities(ctx, state.inst.ID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, peer := range active {
		if peer.ID == rejected.ID {
			continue
		}
		if peer.BranchFlowID != rejected.BranchFlowID {
			continue
		}
		peer.Status = ActivityStatusCancelled
		peer.Outcome = OutcomeCancelled
		peer.EndedAt = &now
		peer.PendingAssignees = nil
		if err := e.store.UpdateActivityInstance(ctx, peer); err != nil {
			return err
		}
		state.inst.ActiveElements = removeString(state.inst.ActiveElements, peer.ElementID)
	}
	return nil
}

func clearJoinStateInScope(inst *ProcessInstance, reg *bpmn.Registry, scopeID string) {
	if inst.InternalState == nil || scopeID == "" {
		return
	}
	for _, gwID := range bpmn.ScopeGatewayIDs(reg, scopeID) {
		delete(inst.InternalState, joinArrivalsKey(gwID))
	}
}

func (e *Engine) Terminate(ctx context.Context, req TerminateRequest) (*ProcessInstance, error) {
	inst, err := e.store.GetProcessInstance(ctx, req.ProcessInstanceID)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, fmt.Errorf("process instance not found")
	}
	if inst.Status != ProcessStatusRunning {
		return nil, fmt.Errorf("process not running: %s", inst.Status)
	}
	if req.LockVersion > 0 && req.LockVersion != inst.LockVersion {
		return nil, ErrVersionConflict
	}

	reg, err := registryForInstance(inst)
	if err != nil {
		return nil, err
	}
	state := &execState{inst: inst, reg: reg}

	if req.ScopeID == "" {
		if err := e.cancelInstance(ctx, inst, req.Reason); err != nil {
			return nil, err
		}
		return e.store.GetProcessInstance(ctx, inst.ID)
	}

	if err := e.terminateScope(ctx, state, req.ScopeID, req.Reason); err != nil {
		return nil, err
	}
	return e.store.GetProcessInstance(ctx, inst.ID)
}

func (e *Engine) terminateScope(ctx context.Context, state *execState, scopeID, reason string) error {
	active, err := e.store.ListActiveActivities(ctx, state.inst.ID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	scopeSet := toScopeSet(state.reg, scopeID)

	for _, act := range active {
		_, inScope := scopeSet[act.ElementID]
		if !inScope && act.ScopeID != scopeID {
			continue
		}
		act.Status = ActivityStatusCancelled
		act.Outcome = OutcomeCancelled
		act.EndedAt = &now
		act.PendingAssignees = nil
		if reason != "" {
			act.ErrorMsg = reason
		}
		if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
			return err
		}
		state.inst.ActiveElements = removeString(state.inst.ActiveElements, act.ElementID)
	}

	clearJoinStateInScope(state.inst, state.reg, scopeID)
	state.inst.Variables["scopeTerminated"] = scopeID
	if reason != "" {
		state.inst.Variables["terminateReason"] = reason
	}
	state.inst.UpdatedAt = now
	if err := e.store.UpdateProcessInstance(ctx, state.inst); err != nil {
		return err
	}
	return e.tryCompleteProcess(ctx, state)
}

func (e *Engine) cancelInstance(ctx context.Context, inst *ProcessInstance, reason string) error {
	active, err := e.store.ListActiveActivities(ctx, inst.ID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, act := range active {
		act.Status = ActivityStatusCancelled
		act.Outcome = OutcomeCancelled
		act.EndedAt = &now
		act.PendingAssignees = nil
		if reason != "" {
			act.ErrorMsg = reason
		}
		if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
			return err
		}
	}
	inst.Status = ProcessStatusCancelled
	inst.ActiveElements = nil
	inst.EndedAt = &now
	inst.UpdatedAt = now
	if reason != "" {
		inst.ErrorMsg = reason
		inst.Variables["terminateReason"] = reason
	}
	return e.store.UpdateProcessInstance(ctx, inst)
}

func toScopeSet(reg *bpmn.Registry, scopeID string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, id := range bpmn.ScopeElementIDs(reg, scopeID) {
		set[id] = struct{}{}
	}
	return set
}

func scopeForElement(state *execState, el bpmn.Element) string {
	if el.ScopeID != "" {
		return el.ScopeID
	}
	return state.scopeID
}
