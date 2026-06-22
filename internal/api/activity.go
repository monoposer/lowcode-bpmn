package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

type triggerBoundaryRequest struct {
	TenantID          string         `json:"tenantId"`
	ProcessInstanceID string         `json:"processInstanceId"`
	HostElementID     string         `json:"hostElementId,omitempty"`
	BoundaryElementID string         `json:"boundaryElementId"`
	Variables         map[string]any `json:"variables,omitempty"`
}

func handleTriggerBoundary(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req triggerBoundaryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		if req.TenantID == "" || req.ProcessInstanceID == "" || req.BoundaryElementID == "" {
			writeBadRequest(w, "invalid_request", "tenantId, processInstanceId, and boundaryElementId required")
			return
		}
		instID, err := uuid.Parse(req.ProcessInstanceID)
		if err != nil {
			writeBadRequest(w, "invalid_request", "invalid processInstanceId")
			return
		}
		eng := getEngine(ctx, deps)
		match, err := eng.TriggerBoundary(ctx, engine.TriggerBoundaryRequest{
			TenantID:          req.TenantID,
			ProcessInstanceID: instID,
			HostElementID:     req.HostElementID,
			BoundaryElementID: req.BoundaryElementID,
			Variables:         req.Variables,
		})
		if err != nil {
			writeInternalError(w, "trigger_boundary_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, match)
	}
}

type completeActivityRequest struct {
	Assignee       string         `json:"assignee,omitempty"`
	Action         string         `json:"action,omitempty"`
	Comment        string         `json:"comment,omitempty"`
	Variables      map[string]any `json:"variables,omitempty"`
	LockVersion    int            `json:"lockVersion,omitempty"`
	SelectedFlowID string         `json:"selectedFlowId,omitempty"`
}

func handleCompleteActivity(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
		if err != nil {
			writeBadRequest(w, "invalid_request", "invalid instance id")
			return
		}
		activityID, err := uuid.Parse(chi.URLParam(r, "activityID"))
		if err != nil {
			writeBadRequest(w, "invalid_request", "invalid activity id")
			return
		}
		var req completeActivityRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		eng := getEngine(ctx, deps)
		inst, err := eng.CompleteActivity(ctx, engine.CompleteActivityRequest{
			ProcessInstanceID: instanceID,
			ActivityID:        activityID,
			Assignee:          req.Assignee,
			Action:            req.Action,
			Comment:           req.Comment,
			Variables:         req.Variables,
			LockVersion:       req.LockVersion,
			SelectedFlowID:    req.SelectedFlowID,
		})
		if err != nil {
			if err == engine.ErrVersionConflict {
				writeError(w, http.StatusConflict, "version_conflict", err.Error())
				return
			}
			writeInternalError(w, "complete_activity_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, inst)
	}
}
