package engine

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/script"
)

func (e *Engine) runAutomatedTask(ctx context.Context, state *execState, el bpmn.Element, act *ActivityInstance) (map[string]any, error) {
	switch el.Kind {
	case bpmn.KindServiceTask:
		return e.runServiceTask(ctx, state, el)
	case bpmn.KindSendTask:
		return e.runSendTask(ctx, state, el)
	case bpmn.KindBusinessRuleTask:
		return e.runBusinessRuleTask(ctx, state, el)
	default:
		return nil, fmt.Errorf("unsupported automated task %s", el.Kind)
	}
}

func (e *Engine) runServiceTask(ctx context.Context, state *execState, el bpmn.Element) (map[string]any, error) {
	out := map[string]any{"taskType": el.TaskType}
	if el.ServiceURL == "" {
		out["serviceTask"] = "completed"
		return out, nil
	}
	method := strings.ToUpper(strings.TrimSpace(el.ServiceMethod))
	if method == "" {
		method = http.MethodGet
	}
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, method, el.ServiceURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("serviceTask http: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	out["serviceStatus"] = resp.StatusCode
	out["serviceBody"] = string(body)
	return out, nil
}

func (e *Engine) runSendTask(ctx context.Context, state *execState, el bpmn.Element) (map[string]any, error) {
	attrs := []any{
		slog.String("tenantId", state.inst.TenantID),
		slog.String("processKey", state.inst.ProcessKey),
		slog.String("elementId", el.ID),
		slog.String("messageRef", el.MessageRef),
		slog.String("taskType", el.TaskType),
	}
	slog.InfoContext(ctx, "sendTask dispatched", attrs...)
	return map[string]any{
		"sent":       true,
		"messageRef": el.MessageRef,
		"taskType":   el.TaskType,
	}, nil
}

func (e *Engine) runBusinessRuleTask(ctx context.Context, state *execState, el bpmn.Element) (map[string]any, error) {
	if el.Script != "" && e.script != nil {
		return e.script.Run(ctx, script.RunRequest{
			Script:     el.Script,
			Lang:       el.ScriptLang,
			Variables:  state.inst.Variables,
			InstanceID: state.inst.ID.String(),
			ElementID:  el.ID,
			TenantID:   state.inst.TenantID,
			ProcessKey: state.inst.ProcessKey,
		})
	}
	return map[string]any{
		"decisionRef": el.DecisionRef,
		"taskType":    el.TaskType,
		"evaluated":   true,
	}, nil
}

func (e *Engine) activateReceiveTask(ctx context.Context, state *execState, el bpmn.Element, act *ActivityInstance) error {
	if el.AutoComplete || el.MessageRef == "" {
		act.Status = ActivityStatusCompleted
		now := time.Now().UTC()
		act.EndedAt = &now
		if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
			return err
		}
		return e.completeElement(ctx, state, el.ID)
	}
	act.Input = map[string]any{"messageRef": el.MessageRef}
	if err := e.store.UpdateActivityInstance(ctx, act); err != nil {
		return err
	}
	state.inst.ActiveElements = appendUnique(state.inst.ActiveElements, el.ID)
	state.inst.UpdatedAt = time.Now().UTC()
	return e.store.UpdateProcessInstance(ctx, state.inst)
}

func (e *Engine) completeAutomatedTask(ctx context.Context, state *execState, elementID string, act *ActivityInstance, out map[string]any, runErr error) error {
	if runErr != nil {
		act.Status = ActivityStatusFailed
		act.ErrorMsg = runErr.Error()
		end := time.Now().UTC()
		act.EndedAt = &end
		_ = e.store.UpdateActivityInstance(ctx, act)
		return runErr
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
}
