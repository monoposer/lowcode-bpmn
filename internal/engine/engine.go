package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/script"
)

// Engine executes BPMN 2.0 process definitions.
type Engine struct {
	store  Store
	script script.Runner
	async  bool
}

// NewEngine constructs a BPMN engine. Async execution is off by default (sync).
// scriptExec may be nil (uses script.NewExecutor()), script.DefaultRunner(), or a custom
// script.Runner (HTTP remote, tenant router, sandbox wrapper) without changing engine logic.
func NewEngine(store Store, scriptExec script.Runner) *Engine {
	if scriptExec == nil {
		scriptExec = script.NewExecutor()
	}
	return &Engine{store: store, script: scriptExec}
}

// SetAsync enables background job execution for StartProcess and CompleteTask continuations.
func (e *Engine) SetAsync(v bool) { e.async = v }

// DeployProcess validates and stores a new BPMN definition version for a tenant.
func (e *Engine) DeployProcess(ctx context.Context, tenantID, key string, def bpmn.ProcessDefinition) (*DeployedProcess, error) {
	if e.store == nil {
		return nil, errors.New("engine: store not configured")
	}
	if tenantID == "" || key == "" {
		return nil, errors.New("tenantID and process key are required")
	}
	if _, err := bpmn.BuildRegistry(def); err != nil {
		return nil, fmt.Errorf("invalid process: %w", err)
	}
	now := time.Now().UTC()
	dp := &DeployedProcess{
		TenantID:   tenantID,
		Key:        key,
		Name:       def.Name,
		Definition: def,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := e.store.InsertProcessVersion(ctx, dp); err != nil {
		return nil, err
	}
	return dp, nil
}

func (e *Engine) DeleteProcess(ctx context.Context, tenantID, key string) error {
	if e.store == nil {
		return errors.New("engine: store not configured")
	}
	return e.store.DeleteProcess(ctx, tenantID, key)
}

func (e *Engine) ListProcesses(ctx context.Context, tenantID string) ([]*DeployedProcess, error) {
	if e.store == nil {
		return nil, errors.New("engine: store not configured")
	}
	return e.store.ListProcesses(ctx, tenantID)
}

// GetProcess returns the latest deployed process version for a tenant/key.
func (e *Engine) GetProcess(ctx context.Context, tenantID, key string) (*DeployedProcess, error) {
	if e.store == nil {
		return nil, errors.New("engine: store not configured")
	}
	return e.store.GetProcess(ctx, tenantID, key)
}

type StartProcessRequest struct {
	TenantID          string
	ProcessKey        string
	BusinessKey       string
	Variables         map[string]any
	StartElementIDs   []string // optional: activate only these startEvents (message/signal trigger)
}

func (e *Engine) StartProcess(ctx context.Context, req StartProcessRequest) (*ProcessInstance, error) {
	if e.store == nil {
		return nil, errors.New("engine: store not configured")
	}
	dp, err := e.store.GetProcess(ctx, req.TenantID, req.ProcessKey)
	if err != nil {
		return nil, err
	}
	if dp == nil {
		return nil, fmt.Errorf("process not found: %s/%s", req.TenantID, req.ProcessKey)
	}

	if e.async {
		inst := e.newInstance(dp, req.BusinessKey, req.Variables)
		inst.Status = ProcessStatusPending
		if len(req.StartElementIDs) > 0 {
			inst.InternalState["start_element_ids"] = req.StartElementIDs
		}
		err = e.store.WithTx(ctx, func(tx Store) error {
			if err := tx.CreateProcessInstance(ctx, inst); err != nil {
				return err
			}
			return tx.EnqueueJob(ctx, &Job{
				ProcessInstanceID: inst.ID,
				Type:              JobTypeStart,
			})
		})
		if err != nil {
			return nil, err
		}
		return e.store.GetProcessInstance(ctx, inst.ID)
	}

	var inst *ProcessInstance
	err = e.store.WithTx(ctx, func(tx Store) error {
		runner := &Engine{store: tx, script: e.script, async: false}
		var runErr error
		inst, runErr = runner.startWithDefinition(ctx, dp, req.BusinessKey, req.Variables, req.StartElementIDs)
		return runErr
	})
	if err != nil {
		return inst, err
	}
	return e.store.GetProcessInstance(ctx, inst.ID)
}

func (e *Engine) StartProcessWithDefinition(ctx context.Context, tenantID, key, businessKey string, def bpmn.ProcessDefinition, vars map[string]any) (*ProcessInstance, error) {
	dp := &DeployedProcess{TenantID: tenantID, Key: key, Version: 1, Definition: def}
	return e.startWithDefinition(ctx, dp, businessKey, vars, nil)
}

func (e *Engine) GetProcessInstance(ctx context.Context, id uuid.UUID) (*ProcessInstance, error) {
	if e.store == nil {
		return nil, errors.New("engine: store not configured")
	}
	return e.store.GetProcessInstance(ctx, id)
}

func (e *Engine) ListActivities(ctx context.Context, processInstanceID uuid.UUID) ([]*ActivityInstance, error) {
	if e.store == nil {
		return nil, errors.New("engine: store not configured")
	}
	return e.store.ListActivitiesByProcess(ctx, processInstanceID)
}

func (e *Engine) ListUserTasks(ctx context.Context, tenantID, assignee string) ([]*UserTask, error) {
	if e.store == nil {
		return nil, errors.New("engine: store not configured")
	}
	return e.store.ListActiveUserTasks(ctx, tenantID, assignee)
}

func (e *Engine) ProcessNextJob(ctx context.Context) error {
	job, err := e.store.ClaimNextJob(ctx)
	if err != nil || job == nil {
		return err
	}

	runErr := e.store.WithTx(ctx, func(tx Store) error {
		runner := &Engine{store: tx, script: e.script, async: true}
		switch job.Type {
		case JobTypeStart:
			return runner.runStartJob(ctx, job.ProcessInstanceID)
		case JobTypeContinue:
			elementID, _ := job.Payload["element_id"].(string)
			if elementID == "" {
				return errors.New("continue job missing element_id")
			}
			return runner.runContinueJob(ctx, job.ProcessInstanceID, elementID)
		default:
			return fmt.Errorf("unknown job type: %s", job.Type)
		}
	})

	if runErr != nil {
		_ = e.store.FailJob(ctx, job.ID, runErr.Error())
		return runErr
	}
	return e.store.CompleteJob(ctx, job.ID)
}

func (e *Engine) newInstance(dp *DeployedProcess, businessKey string, vars map[string]any) *ProcessInstance {
	now := time.Now().UTC()
	return &ProcessInstance{
		ID:                 uuid.New(),
		TenantID:           dp.TenantID,
		ProcessKey:         dp.Key,
		ProcessVersion:     dp.Version,
		BusinessKey:        businessKey,
		Status:             ProcessStatusRunning,
		Variables:          cloneVars(vars),
		InternalState:      make(map[string]any),
		DefinitionSnapshot: dp.Definition,
		StartedAt:          now,
		UpdatedAt:          now,
	}
}

func (e *Engine) startWithDefinition(ctx context.Context, dp *DeployedProcess, businessKey string, vars map[string]any, startElementIDs []string) (*ProcessInstance, error) {
	reg, err := bpmn.BuildRegistry(dp.Definition)
	if err != nil {
		return nil, err
	}
	inst := e.newInstance(dp, businessKey, vars)
	if err := e.store.CreateProcessInstance(ctx, inst); err != nil {
		return nil, err
	}

	startIDs := startElementIDs
	if len(startIDs) == 0 {
		startIDs = reg.StartEvents
	}

	state := &execState{inst: inst, reg: reg}
	for _, startID := range startIDs {
		if err := e.activateElement(ctx, state, startID, ""); err != nil {
			return e.failInstance(ctx, inst, err)
		}
	}
	return e.store.GetProcessInstance(ctx, inst.ID)
}

func (e *Engine) runStartJob(ctx context.Context, instanceID uuid.UUID) error {
	inst, err := e.store.GetProcessInstanceForUpdate(ctx, instanceID)
	if err != nil || inst == nil {
		return fmt.Errorf("instance not found: %s", instanceID)
	}
	reg, err := registryForInstance(inst)
	if err != nil {
		return err
	}
	inst.Status = ProcessStatusRunning
	if err := e.store.UpdateProcessInstance(ctx, inst); err != nil {
		return err
	}
	state := &execState{inst: inst, reg: reg}
	startIDs := reg.StartEvents
	if raw, ok := inst.InternalState["start_element_ids"]; ok {
		if ids, ok := raw.([]string); ok && len(ids) > 0 {
			startIDs = ids
		} else if arr, ok := raw.([]any); ok && len(arr) > 0 {
			ids := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok && s != "" {
					ids = append(ids, s)
				}
			}
			if len(ids) > 0 {
				startIDs = ids
			}
		}
	}
	for _, startID := range startIDs {
		if err := e.activateElement(ctx, state, startID, ""); err != nil {
			_, failErr := e.failInstance(ctx, inst, err)
			return failErr
		}
	}
	return nil
}

func (e *Engine) runContinueJob(ctx context.Context, instanceID uuid.UUID, elementID string) error {
	inst, err := e.store.GetProcessInstanceForUpdate(ctx, instanceID)
	if err != nil || inst == nil {
		return fmt.Errorf("instance not found: %s", instanceID)
	}
	if inst.Status != ProcessStatusRunning {
		return fmt.Errorf("process not running: %s", inst.Status)
	}
	reg, err := registryForInstance(inst)
	if err != nil {
		return err
	}
	state := &execState{inst: inst, reg: reg}
	if err := e.completeElement(ctx, state, elementID); err != nil {
		_, failErr := e.failInstance(ctx, inst, err)
		return failErr
	}
	return nil
}

func (e *Engine) CompleteTask(ctx context.Context, req CompleteTaskRequest) (*ProcessInstance, error) {
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
	if act.ElementKind != bpmn.KindUserTask {
		return nil, fmt.Errorf("element is not a userTask: %s", act.ElementKind)
	}

	reg, err := registryForInstance(inst)
	if err != nil {
		return nil, err
	}
	el, ok := reg.Element(act.ElementID)
	if !ok {
		return nil, fmt.Errorf("element not found: %s", act.ElementID)
	}

	step, err := applyApprovalStep(act, el, req)
	if err != nil {
		return nil, err
	}
	for k, v := range req.Variables {
		inst.Variables[k] = v
	}
	if inst.InternalState == nil {
		inst.InternalState = make(map[string]any)
	}
	act.Output = mergeApprovalOutput(act)

	if !step.done {
		inst.UpdatedAt = time.Now().UTC()
		err = e.store.WithTx(ctx, func(tx Store) error {
			if err := tx.UpdateActivityInstance(ctx, act); err != nil {
				return err
			}
			return tx.UpdateProcessInstance(ctx, inst)
		})
		if err != nil {
			return nil, err
		}
		return e.store.GetProcessInstance(ctx, inst.ID)
	}

	if step.rejected {
		act.Output = mergeApprovalOutput(act)
		state := &execState{inst: inst, reg: reg}
		err = e.store.WithTx(ctx, func(tx Store) error {
			runner := &Engine{store: tx, script: e.script, async: false}
			return runner.handleTaskReject(ctx, state, act, el, req)
		})
		if err != nil {
			return nil, err
		}
		return e.store.GetProcessInstance(ctx, inst.ID)
	}

	now := time.Now().UTC()
	act.Status = ActivityStatusCompleted
	act.Outcome = OutcomeApprove
	act.EndedAt = &now
	act.PendingAssignees = nil
	elementID := act.ElementID

	if e.async {
		err = e.store.WithTx(ctx, func(tx Store) error {
			if err := tx.UpdateActivityInstance(ctx, act); err != nil {
				return err
			}
			inst.ActiveElements = removeString(inst.ActiveElements, elementID)
			if err := tx.UpdateProcessInstance(ctx, inst); err != nil {
				return err
			}
			return tx.EnqueueJob(ctx, &Job{
				ProcessInstanceID: inst.ID,
				Type:              JobTypeContinue,
				Payload:           map[string]any{"element_id": elementID},
			})
		})
		if err != nil {
			return nil, err
		}
		return e.store.GetProcessInstance(ctx, inst.ID)
	}

	var result *ProcessInstance
	err = e.store.WithTx(ctx, func(tx Store) error {
		runner := &Engine{store: tx, script: e.script, async: false}
		if err := tx.UpdateActivityInstance(ctx, act); err != nil {
			return err
		}
		state := &execState{inst: inst, reg: reg}
		if err := runner.completeElement(ctx, state, elementID); err != nil {
			_, failErr := runner.failInstance(ctx, inst, err)
			return failErr
		}
		var getErr error
		result, getErr = tx.GetProcessInstance(ctx, inst.ID)
		return getErr
	})
	if err != nil {
		return result, err
	}
	return e.store.GetProcessInstance(ctx, inst.ID)
}

func registryForInstance(inst *ProcessInstance) (*bpmn.Registry, error) {
	if inst.DefinitionSnapshot.ID == "" && len(inst.DefinitionSnapshot.Elements) == 0 {
		return nil, errors.New("process definition snapshot missing on instance")
	}
	return bpmn.BuildRegistry(inst.DefinitionSnapshot)
}

type execState struct {
	inst         *ProcessInstance
	reg          *bpmn.Registry
	scopeID      string
	branchFlowID string
}

func (e *Engine) activateElement(ctx context.Context, state *execState, elementID, branchFlowID string) error {
	el, ok := state.reg.Element(elementID)
	if !ok {
		return fmt.Errorf("element not found: %s", elementID)
	}

	scopeID := scopeForElement(state, el)
	if el.Kind == bpmn.KindSubProcess {
		state.scopeID = scopeID
		if el.EntryRef != "" {
			return e.activateElement(ctx, state, el.EntryRef, branchFlowID)
		}
		return nil
	}

	now := time.Now().UTC()
	act := &ActivityInstance{
		ID:                uuid.New(),
		ProcessInstanceID: state.inst.ID,
		ElementID:         elementID,
		ElementKind:       el.Kind,
		Status:            ActivityStatusActive,
		ScopeID:           scopeID,
		BranchFlowID:      branchFlowID,
		Assignees:         el.Assignees,
		Input:             cloneVars(state.inst.Variables),
		StartedAt:         now,
	}
	if err := e.store.CreateActivityInstance(ctx, act); err != nil {
		return err
	}

	switch el.Kind {
	case bpmn.KindStartEvent:
		act.Status = ActivityStatusCompleted
		end := time.Now().UTC()
		act.EndedAt = &end
		if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
			return err
		}
		return e.completeElement(ctx, state, elementID)

	case bpmn.KindEndEvent:
		act.Status = ActivityStatusCompleted
		end := time.Now().UTC()
		act.EndedAt = &end
		if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
			return err
		}
		return e.tryCompleteProcess(ctx, state)

	case bpmn.KindServiceTask, bpmn.KindSendTask, bpmn.KindBusinessRuleTask:
		out, err := e.runAutomatedTask(ctx, state, el, act)
		return e.completeAutomatedTask(ctx, state, elementID, act, out, err)

	case bpmn.KindReceiveTask:
		return e.activateReceiveTask(ctx, state, el, act)

	case bpmn.KindScriptTask:
		out, err := e.script.Run(ctx, script.RunRequest{
			Script:     el.Script,
			Lang:       el.ScriptLang,
			Variables:  state.inst.Variables,
			InstanceID: state.inst.ID.String(),
			ElementID:  elementID,
			TenantID:   state.inst.TenantID,
			ProcessKey: state.inst.ProcessKey,
		})
		if err != nil {
			act.Status = ActivityStatusFailed
			act.ErrorMsg = err.Error()
			end := time.Now().UTC()
			act.EndedAt = &end
			_ = e.store.UpdateActivityInstance(ctx, act)
			return err
		}
		for k, v := range out {
			state.inst.Variables[k] = v
		}
		state.inst.UpdatedAt = time.Now().UTC()
		if err := e.store.UpdateProcessInstance(ctx, state.inst); err != nil {
			return err
		}
		act.Status = ActivityStatusCompleted
		act.Output = out
		end := time.Now().UTC()
		act.EndedAt = &end
		if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
			return err
		}
		return e.completeElement(ctx, state, elementID)

	case bpmn.KindUserTask:
		if el.AutoComplete {
			act.Status = ActivityStatusCompleted
			end := time.Now().UTC()
			act.EndedAt = &end
			if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
				return err
			}
			return e.completeElement(ctx, state, elementID)
		}
		initUserTaskApproval(act, el, resolvedAssignees(state, elementID, el))
		stampAssigneeSyncMeta(act, resolvedAssigneeSource(state, elementID, el), "")
		attachBoundaryMetadata(act, state.reg, elementID)
		if lane, ok := state.reg.LaneForElement(elementID); ok {
			if act.Input == nil {
				act.Input = map[string]any{}
			}
			act.Input["laneId"] = lane.ID
			act.Input["laneName"] = lane.Name
		}
		if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
			return err
		}
		state.inst.ActiveElements = appendUnique(state.inst.ActiveElements, elementID)
		state.inst.UpdatedAt = time.Now().UTC()
		return e.store.UpdateProcessInstance(ctx, state.inst)

	case bpmn.KindExclusiveGateway, bpmn.KindParallelGateway, bpmn.KindInclusiveGateway:
		act.Status = ActivityStatusCompleted
		end := time.Now().UTC()
		act.EndedAt = &end
		if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
			return err
		}
		return e.completeElement(ctx, state, elementID)

	default:
		if bpmn.IsExtensionKind(el.Kind) {
			return e.activateExtensionElement(ctx, state, el, act)
		}
		return fmt.Errorf("unsupported element kind: %s", el.Kind)
	}
}

func (e *Engine) completeElement(ctx context.Context, state *execState, elementID string) error {
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
	default:
		if len(outgoing) > 1 {
			return fmt.Errorf("element %s has multiple outgoing flows but is not a gateway", elementID)
		}
		return e.activateViaFlow(ctx, state, outgoing[0], state.branchFlowID)
	}
}

func (e *Engine) activateViaFlow(ctx context.Context, state *execState, flow bpmn.SequenceFlow, branchFlowID string) error {
	srcEl, srcOK := state.reg.Element(flow.SourceRef)
	if srcOK && srcEl.Kind == bpmn.KindParallelGateway && len(state.reg.OutgoingFlows(flow.SourceRef)) > 1 {
		branchFlowID = flow.ID
	}
	targetID := flow.TargetRef
	if state.reg.IsJoinGateway(targetID) {
		incoming := state.reg.IncomingFlows(targetID)
		if len(incoming) > 1 {
			if state.inst.InternalState == nil {
				state.inst.InternalState = make(map[string]any)
			}
			key := joinArrivalsKey(targetID)
			arrived := toStringSet(state.inst.InternalState[key])
			arrived[flow.ID] = struct{}{}
			state.inst.InternalState[key] = stringSetKeys(arrived)
			if len(arrived) < len(incoming) {
				state.inst.UpdatedAt = time.Now().UTC()
				return e.store.UpdateProcessInstance(ctx, state.inst)
			}
			delete(state.inst.InternalState, key)
		}
	}
	return e.activateElement(ctx, state, targetID, branchFlowID)
}

func (e *Engine) followExclusive(ctx context.Context, state *execState, flows []bpmn.SequenceFlow) error {
	var defaultFlow *bpmn.SequenceFlow
	for i := range flows {
		f := &flows[i]
		if f.IsDefault {
			defaultFlow = f
			continue
		}
		if f.Condition == "" {
			defaultFlow = f
			continue
		}
		ok, err := bpmn.EvalCondition(f.Condition, state.inst.Variables)
		if err != nil {
			return err
		}
		if ok {
			return e.activateViaFlow(ctx, state, *f, state.branchFlowID)
		}
	}
	if defaultFlow != nil {
		return e.activateViaFlow(ctx, state, *defaultFlow, state.branchFlowID)
	}
	return errors.New("exclusive gateway: no matching outgoing flow")
}

func (e *Engine) followInclusive(ctx context.Context, state *execState, flows []bpmn.SequenceFlow) error {
	var matched []bpmn.SequenceFlow
	var defaultFlow *bpmn.SequenceFlow
	for _, f := range flows {
		if f.IsDefault {
			defaultFlow = &f
			continue
		}
		ok, err := bpmn.EvalCondition(f.Condition, state.inst.Variables)
		if err != nil {
			return err
		}
		if ok {
			matched = append(matched, f)
		}
	}
	if len(matched) == 0 {
		if defaultFlow != nil {
			return e.activateViaFlow(ctx, state, *defaultFlow, state.branchFlowID)
		}
		return errors.New("inclusive gateway: no matching outgoing flow")
	}
	for _, f := range matched {
		if err := e.activateViaFlow(ctx, state, f, state.branchFlowID); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) followAll(ctx context.Context, state *execState, flows []bpmn.SequenceFlow) error {
	for _, f := range flows {
		if err := e.activateViaFlow(ctx, state, f, state.branchFlowID); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) tryCompleteProcess(ctx context.Context, state *execState) error {
	active, err := e.store.ListActiveActivities(ctx, state.inst.ID)
	if err != nil {
		return err
	}
	if len(active) > 0 {
		return nil
	}
	if len(state.inst.ActiveElements) > 0 {
		return nil
	}
	now := time.Now().UTC()
	state.inst.Status = ProcessStatusCompleted
	state.inst.EndedAt = &now
	state.inst.UpdatedAt = now
	state.inst.ActiveElements = nil
	return e.store.UpdateProcessInstance(ctx, state.inst)
}

func (e *Engine) failInstance(ctx context.Context, inst *ProcessInstance, err error) (*ProcessInstance, error) {
	now := time.Now().UTC()
	inst.Status = ProcessStatusFailed
	inst.ErrorMsg = err.Error()
	inst.EndedAt = &now
	inst.UpdatedAt = now
	_ = e.store.UpdateProcessInstance(ctx, inst)
	return inst, err
}

func joinArrivalsKey(gatewayID string) string {
	return "join_arrivals_" + gatewayID
}

func toStringSet(v any) map[string]struct{} {
	out := make(map[string]struct{})
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			if s, ok := item.(string); ok {
				out[s] = struct{}{}
			}
		}
	case []string:
		for _, s := range t {
			out[s] = struct{}{}
		}
	}
	return out
}

func stringSetKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func cloneVars(m map[string]any) map[string]any {
	if m == nil {
		return make(map[string]any)
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func appendUnique(list []string, s string) []string {
	for _, v := range list {
		if v == s {
			return list
		}
	}
	return append(list, s)
}

func removeString(list []string, s string) []string {
	out := list[:0]
	for _, v := range list {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}
