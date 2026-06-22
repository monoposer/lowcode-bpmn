package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
)

// CompleteActivityRequest completes a waiting activity (userTask, extension element, receiveTask).
type CompleteActivityRequest struct {
	ProcessInstanceID uuid.UUID
	ActivityID        uuid.UUID
	Assignee          string // userTask only
	Action            string // userTask: approve|reject
	Comment           string
	Variables         map[string]any
	LockVersion       int
	SelectedFlowID    string // eventBasedGateway: chosen outgoing sequence flow id
}

// CompleteActivity completes an active activity and continues token flow.
func (e *Engine) CompleteActivity(ctx context.Context, req CompleteActivityRequest) (*ProcessInstance, error) {
	inst, err := e.store.GetProcessInstance(ctx, req.ProcessInstanceID)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, errors.New("process instance not found")
	}
	if inst.Status != ProcessStatusRunning {
		return nil, fmt.Errorf("process not running: %s", inst.Status)
	}
	if req.LockVersion > 0 && req.LockVersion != inst.LockVersion {
		return nil, ErrVersionConflict
	}

	act, err := e.store.GetActivityInstance(ctx, req.ActivityID)
	if err != nil {
		return nil, err
	}
	if act == nil {
		return nil, errors.New("activity instance not found")
	}
	if act.ProcessInstanceID != inst.ID {
		return nil, errors.New("activity does not belong to process instance")
	}
	if act.Status != ActivityStatusActive {
		return nil, fmt.Errorf("activity not active: %s", act.Status)
	}

	switch act.ElementKind {
	case bpmn.KindUserTask:
		return e.CompleteTask(ctx, CompleteTaskRequest{
			ProcessInstanceID: req.ProcessInstanceID,
			ActivityID:        req.ActivityID,
			Assignee:          req.Assignee,
			Action:            req.Action,
			Comment:           req.Comment,
			Variables:         req.Variables,
			LockVersion:       req.LockVersion,
		})
	case bpmn.KindReceiveTask:
		return e.completeReceiveActivity(ctx, inst, act, req)
	default:
		if bpmn.IsExtensionKind(act.ElementKind) {
			return e.completeExtensionActivity(ctx, inst, act, req)
		}
		return nil, fmt.Errorf("complete not supported for element kind: %s", act.ElementKind)
	}
}

func (e *Engine) completeReceiveActivity(ctx context.Context, inst *ProcessInstance, act *ActivityInstance, req CompleteActivityRequest) (*ProcessInstance, error) {
	var result *ProcessInstance
	err := e.store.WithTx(ctx, func(tx Store) error {
		runner := &Engine{store: tx, script: e.script, async: false}
		instCopy, err := tx.GetProcessInstanceForUpdate(ctx, inst.ID)
		if err != nil {
			return err
		}
		actCopy, err := tx.GetActivityInstance(ctx, act.ID)
		if err != nil {
			return err
		}
		for k, v := range req.Variables {
			instCopy.Variables[k] = v
		}
		now := time.Now().UTC()
		actCopy.Status = ActivityStatusCompleted
		actCopy.EndedAt = &now
		if err := tx.UpdateActivityInstance(ctx, actCopy); err != nil {
			return err
		}
		reg, err := registryForInstance(instCopy)
		if err != nil {
			return err
		}
		state := &execState{inst: instCopy, reg: reg}
		if err := runner.completeElementWithSelection(ctx, state, actCopy.ElementID, req.SelectedFlowID); err != nil {
			_, failErr := runner.failInstance(ctx, instCopy, err)
			return failErr
		}
		result, err = tx.GetProcessInstance(ctx, inst.ID)
		return err
	})
	if err != nil {
		return result, err
	}
	return e.store.GetProcessInstance(ctx, inst.ID)
}

func (e *Engine) completeExtensionActivity(ctx context.Context, inst *ProcessInstance, act *ActivityInstance, req CompleteActivityRequest) (*ProcessInstance, error) {
	reg, err := registryForInstance(inst)
	if err != nil {
		return nil, err
	}
	el, ok := reg.Element(act.ElementID)
	if !ok {
		return nil, fmt.Errorf("element not found: %s", act.ElementID)
	}

	var result *ProcessInstance
	err = e.store.WithTx(ctx, func(tx Store) error {
		runner := &Engine{store: tx, script: e.script, async: false}
		instCopy, err := tx.GetProcessInstanceForUpdate(ctx, inst.ID)
		if err != nil {
			return err
		}
		actCopy, err := tx.GetActivityInstance(ctx, act.ID)
		if err != nil {
			return err
		}
		for k, v := range req.Variables {
			instCopy.Variables[k] = v
		}
		if el.Kind == bpmn.KindCallActivity {
			state := &execState{inst: instCopy, reg: reg}
			if err := runner.invokeCallActivity(ctx, state, el); err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		actCopy.Status = ActivityStatusCompleted
		actCopy.Output = req.Variables
		actCopy.EndedAt = &now
		if err := tx.UpdateActivityInstance(ctx, actCopy); err != nil {
			return err
		}
		instCopy.ActiveElements = removeString(instCopy.ActiveElements, actCopy.ElementID)
		state := &execState{inst: instCopy, reg: reg}
		if err := runner.completeElementWithSelection(ctx, state, actCopy.ElementID, req.SelectedFlowID); err != nil {
			_, failErr := runner.failInstance(ctx, instCopy, err)
			return failErr
		}
		result, err = tx.GetProcessInstance(ctx, inst.ID)
		return err
	})
	if err != nil {
		return result, err
	}
	return e.store.GetProcessInstance(ctx, inst.ID)
}

func (e *Engine) completeElementWithSelection(ctx context.Context, state *execState, elementID, selectedFlowID string) error {
	el, ok := state.reg.Element(elementID)
	if !ok {
		return fmt.Errorf("element not found: %s", elementID)
	}

	state.inst.ActiveElements = removeString(state.inst.ActiveElements, elementID)
	state.inst.UpdatedAt = time.Now().UTC()
	if err := e.store.UpdateProcessInstance(ctx, state.inst); err != nil {
		return err
	}

	outgoing := state.reg.OutgoingFlows(elementID)
	if len(outgoing) == 0 {
		if state.reg.IsEndEvent(elementID) {
			return e.tryCompleteProcess(ctx, state)
		}
		return nil
	}

	switch el.Kind {
	case bpmn.KindExclusiveGateway:
		return e.followExclusive(ctx, state, outgoing)
	case bpmn.KindParallelGateway:
		return e.followAll(ctx, state, outgoing)
	case bpmn.KindInclusiveGateway:
		if len(state.reg.IncomingFlows(elementID)) <= 1 {
			return e.followInclusive(ctx, state, outgoing)
		}
		return e.followAll(ctx, state, outgoing)
	case bpmn.KindEventBasedGateway:
		return e.followEventBasedGateway(ctx, state, outgoing, selectedFlowID)
	case bpmn.KindComplexGateway:
		return e.followComplexGateway(ctx, state, outgoing)
	default:
		if len(outgoing) > 1 {
			return fmt.Errorf("element %s has multiple outgoing flows but is not a gateway", elementID)
		}
		return e.activateViaFlow(ctx, state, outgoing[0], state.branchFlowID)
	}
}

func (e *Engine) followEventBasedGateway(ctx context.Context, state *execState, flows []bpmn.SequenceFlow, selectedFlowID string) error {
	if selectedFlowID == "" {
		for _, f := range flows {
			ok, err := bpmn.EvalCondition(f.Condition, state.inst.Variables)
			if err != nil {
				return err
			}
			if ok {
				return e.activateViaFlow(ctx, state, f, state.branchFlowID)
			}
		}
		if len(flows) == 1 {
			return e.activateViaFlow(ctx, state, flows[0], state.branchFlowID)
		}
		return errors.New("eventBasedGateway: no matching flow; pass selectedFlowId")
	}
	for _, f := range flows {
		if f.ID == selectedFlowID {
			return e.activateViaFlow(ctx, state, f, state.branchFlowID)
		}
	}
	return fmt.Errorf("eventBasedGateway: selected flow %q not found", selectedFlowID)
}

func (e *Engine) followComplexGateway(ctx context.Context, state *execState, flows []bpmn.SequenceFlow) error {
	var matched []bpmn.SequenceFlow
	for _, f := range flows {
		cond := f.Condition
		if cond == "" {
			continue
		}
		ok, err := bpmn.EvalCondition(cond, state.inst.Variables)
		if err != nil {
			return err
		}
		if ok {
			matched = append(matched, f)
		}
	}
	if len(matched) == 0 {
		return errors.New("complexGateway: no activation condition matched")
	}
	for _, f := range matched {
		if err := e.activateViaFlow(ctx, state, f, state.branchFlowID); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) invokeCallActivity(ctx context.Context, state *execState, el bpmn.Element) error {
	if el.CalledElement == "" {
		return nil
	}
	bk := fmt.Sprintf("%s:%s", state.inst.ID, el.ID)
	if state.inst.BusinessKey != "" {
		bk = state.inst.BusinessKey + ":" + el.ID
	}
	_, err := e.StartProcess(ctx, StartProcessRequest{
		TenantID:    state.inst.TenantID,
		ProcessKey:  el.CalledElement,
		BusinessKey: bk,
		Variables:   cloneVars(state.inst.Variables),
	})
	return err
}

// EvaluateComplexGateway evaluates outgoing activation conditions (for WASM/plugin adapters).
func (e *Engine) EvaluateComplexGateway(ctx context.Context, instanceID uuid.UUID, gatewayElementID string) ([]string, error) {
	inst, err := e.store.GetProcessInstance(ctx, instanceID)
	if err != nil || inst == nil {
		return nil, fmt.Errorf("instance not found")
	}
	reg, err := registryForInstance(inst)
	if err != nil {
		return nil, err
	}
	el, ok := reg.Element(gatewayElementID)
	if !ok || el.Kind != bpmn.KindComplexGateway {
		return nil, fmt.Errorf("complex gateway not found: %s", gatewayElementID)
	}
	var matched []string
	for _, f := range reg.OutgoingFlows(gatewayElementID) {
		if f.Condition == "" {
			continue
		}
		ok, err := bpmn.EvalCondition(f.Condition, inst.Variables)
		if err != nil {
			return nil, err
		}
		if ok {
			matched = append(matched, f.ID)
		}
	}
	return matched, nil
}
