package api

import (
	"encoding/json"
	"net/http"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

type triggerMessageRequest struct {
	TenantID    string         `json:"tenantId"`
	MessageRef  string         `json:"messageRef"`
	BusinessKey string         `json:"businessKey,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
}

func handleTriggerMessage(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req triggerMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		if req.TenantID == "" || req.MessageRef == "" {
			writeBadRequest(w, "invalid_request", "tenantId and messageRef required")
			return
		}
		eng := getEngine(ctx, deps)
		result, err := eng.TriggerMessage(ctx, engine.TriggerMessageRequest{
			TenantID:    req.TenantID,
			MessageRef:  req.MessageRef,
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
