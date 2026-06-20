package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

type assigneeSyncRemoveUserBody struct {
	TenantID    string   `json:"tenantId"`
	UserID      string   `json:"userId"`
	ProcessKeys []string `json:"processKeys,omitempty"`
	ElementIDs  []string `json:"elementIds,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	Operator    string   `json:"operator,omitempty"`
}

func handleAssigneeSyncRemoveUser(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req assigneeSyncRemoveUserBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		if req.TenantID == "" || req.UserID == "" {
			writeBadRequest(w, "invalid_request", "tenantId and userId required")
			return
		}
		eng := getEngine(ctx, deps)
		result, err := eng.RemoveUserFromActiveTasks(ctx, engine.RemoveUserSyncRequest{
			TenantID:    req.TenantID,
			UserID:      req.UserID,
			ProcessKeys: req.ProcessKeys,
			ElementIDs:  req.ElementIDs,
			Reason:      req.Reason,
			Operator:    req.Operator,
		})
		if err != nil {
			writeBadRequest(w, "sync_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

type assigneeSyncReplaceBody struct {
	TenantID          string   `json:"tenantId"`
	ProcessInstanceID string   `json:"processInstanceId"`
	ActivityID        string   `json:"activityId"`
	Assignees         []string `json:"assignees"`
	PendingAssignees  []string `json:"pendingAssignees,omitempty"`
	Operator          string   `json:"operator,omitempty"`
	Reason            string   `json:"reason,omitempty"`
}

func handleAssigneeSyncReplace(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req assigneeSyncReplaceBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		if req.TenantID == "" || req.ProcessInstanceID == "" || req.ActivityID == "" || len(req.Assignees) == 0 {
			writeBadRequest(w, "invalid_request", "tenantId, processInstanceId, activityId, assignees required")
			return
		}
		instID, err := uuid.Parse(req.ProcessInstanceID)
		if err != nil {
			writeBadRequest(w, "invalid_instance_id", "invalid processInstanceId")
			return
		}
		actID, err := uuid.Parse(req.ActivityID)
		if err != nil {
			writeBadRequest(w, "invalid_activity_id", "invalid activityId")
			return
		}
		eng := getEngine(ctx, deps)
		act, err := eng.ReplaceTaskAssigneesSync(ctx, engine.ReplaceAssigneesSyncRequest{
			TenantID:          req.TenantID,
			ProcessInstanceID: instID,
			ActivityID:        actID,
			Assignees:         req.Assignees,
			PendingAssignees:  req.PendingAssignees,
			Operator:          req.Operator,
			Reason:            req.Reason,
		})
		if err != nil {
			writeBadRequest(w, "sync_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, act)
	}
}
