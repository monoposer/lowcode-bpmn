package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

type startProcessInstanceRequest struct {
	TenantID    string         `json:"tenantId"`
	ProcessKey  string         `json:"processKey"`
	BusinessKey string         `json:"businessKey,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
}

func handleStartProcessInstance(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req startProcessInstanceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		if req.TenantID == "" || req.ProcessKey == "" {
			writeBadRequest(w, "invalid_request", "tenantId and processKey required")
			return
		}
		eng := getEngine(ctx, deps)
		inst, err := eng.StartProcess(ctx, engine.StartProcessRequest{
			TenantID:    req.TenantID,
			ProcessKey:  req.ProcessKey,
			BusinessKey: req.BusinessKey,
			Variables:   req.Variables,
		})
		if err != nil {
			writeInternalError(w, "start_failed", err.Error())
			return
		}
		incProcessStarts()
		writeJSON(w, http.StatusCreated, inst)
	}
}

func handleGetProcessInstance(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id, err := uuid.Parse(chi.URLParam(r, "instanceID"))
		if err != nil {
			writeBadRequest(w, "invalid_instance_id", "invalid instanceID")
			return
		}
		eng := getEngine(ctx, deps)
		inst, err := eng.GetProcessInstance(ctx, id)
		if err != nil {
			writeInternalError(w, "get_failed", err.Error())
			return
		}
		if inst == nil {
			writeNotFound(w, "instance_not_found", "instance not found")
			return
		}
		writeJSON(w, http.StatusOK, inst)
	}
}

func handleListProcessActivities(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id, err := uuid.Parse(chi.URLParam(r, "instanceID"))
		if err != nil {
			writeBadRequest(w, "invalid_instance_id", "invalid instanceID")
			return
		}
		eng := getEngine(ctx, deps)
		acts, err := eng.ListActivities(ctx, id)
		if err != nil {
			writeInternalError(w, "list_activities_failed", err.Error())
			return
		}
		if acts == nil {
			acts = []*engine.ActivityInstance{}
		}
		writeJSON(w, http.StatusOK, acts)
	}
}

type terminateRequest struct {
	ScopeID     string `json:"scopeId,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Operator    string `json:"operator,omitempty"`
	LockVersion int    `json:"lockVersion,omitempty"`
}

func handleTerminateInstance(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
		if err != nil {
			writeBadRequest(w, "invalid_instance_id", "invalid instanceID")
			return
		}
		var req terminateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		eng := getEngine(ctx, deps)
		inst, err := eng.Terminate(ctx, engine.TerminateRequest{
			ProcessInstanceID: instanceID,
			ScopeID:           req.ScopeID,
			Reason:            req.Reason,
			Operator:          req.Operator,
			LockVersion:       req.LockVersion,
		})
		if err != nil {
			if err == engine.ErrVersionConflict {
				writeError(w, http.StatusConflict, "version_conflict", err.Error())
				return
			}
			writeBadRequest(w, "terminate_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, inst)
	}
}

type updateAssigneesRequest struct {
	Assignees        []string `json:"assignees"`
	PendingAssignees []string `json:"pendingAssignees,omitempty"`
	Operator         string   `json:"operator,omitempty"`
	LockVersion      int      `json:"lockVersion,omitempty"`
}

func handleUpdateAssignees(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
		if err != nil {
			writeBadRequest(w, "invalid_instance_id", "invalid instanceID")
			return
		}
		activityID, err := uuid.Parse(chi.URLParam(r, "activityID"))
		if err != nil {
			writeBadRequest(w, "invalid_activity_id", "invalid activityID")
			return
		}
		var req updateAssigneesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		eng := getEngine(ctx, deps)
		act, err := eng.UpdateActivityAssignees(ctx, engine.UpdateAssigneesRequest{
			ProcessInstanceID: instanceID,
			ActivityID:        activityID,
			Assignees:         req.Assignees,
			PendingAssignees:  req.PendingAssignees,
			Operator:          req.Operator,
			LockVersion:       req.LockVersion,
		})
		if err != nil {
			if err == engine.ErrVersionConflict {
				writeError(w, http.StatusConflict, "version_conflict", err.Error())
				return
			}
			writeBadRequest(w, "update_assignees_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, act)
	}
}
