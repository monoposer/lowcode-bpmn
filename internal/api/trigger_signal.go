package api

import (
	"encoding/json"
	"net/http"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

type triggerSignalRequest struct {
	TenantID    string         `json:"tenantId"`
	SignalRef   string         `json:"signalRef"`
	BusinessKey string         `json:"businessKey,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
}

func handleTriggerSignal(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req triggerSignalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		if req.TenantID == "" || req.SignalRef == "" {
			writeBadRequest(w, "invalid_request", "tenantId and signalRef required")
			return
		}
		eng := getEngine(ctx, deps)
		result, err := eng.TriggerSignal(ctx, engine.TriggerSignalRequest{
			TenantID:    req.TenantID,
			SignalRef:   req.SignalRef,
			BusinessKey: req.BusinessKey,
			Variables:   req.Variables,
		})
		if err != nil {
			writeInternalError(w, "trigger_failed", err.Error())
			return
		}
		status := http.StatusOK
		if len(result.Matches) == 0 {
			status = http.StatusAccepted
		}
		writeJSON(w, status, result)
	}
}

type triggerConditionalRequest struct {
	TenantID    string         `json:"tenantId"`
	BusinessKey string         `json:"businessKey,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
}

func handleTriggerConditional(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req triggerConditionalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		if req.TenantID == "" {
			writeBadRequest(w, "invalid_request", "tenantId required")
			return
		}
		eng := getEngine(ctx, deps)
		result, err := eng.TriggerConditional(ctx, engine.TriggerConditionalRequest{
			TenantID:    req.TenantID,
			BusinessKey: req.BusinessKey,
			Variables:   req.Variables,
		})
		if err != nil {
			writeInternalError(w, "trigger_failed", err.Error())
			return
		}
		status := http.StatusOK
		if len(result.Matches) == 0 {
			status = http.StatusAccepted
		}
		writeJSON(w, status, result)
	}
}
