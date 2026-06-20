package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

type completeTaskRequest struct {
	Assignee    string         `json:"assignee,omitempty"`
	Action      string         `json:"action,omitempty"`
	Comment     string         `json:"comment,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
	LockVersion int            `json:"lockVersion,omitempty"`
}

func handleCompleteUserTask(deps RouterDeps) http.HandlerFunc {
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
		var req completeTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		eng := getEngine(ctx, deps)
		inst, err := eng.CompleteTask(ctx, engine.CompleteTaskRequest{
			ProcessInstanceID: instanceID,
			ActivityID:        activityID,
			Assignee:          req.Assignee,
			Action:            req.Action,
			Comment:           req.Comment,
			Variables:         req.Variables,
			LockVersion:       req.LockVersion,
		})
		if err != nil {
			if err == engine.ErrVersionConflict {
				writeError(w, http.StatusConflict, "version_conflict", err.Error())
				return
			}
			writeBadRequest(w, "complete_failed", err.Error())
			return
		}
		incTaskCompletions()
		writeJSON(w, http.StatusOK, inst)
	}
}

func handleListUserTasks(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tenantID := r.URL.Query().Get("tenantId")
		if tenantID == "" {
			writeBadRequest(w, "invalid_request", "tenantId query parameter required")
			return
		}
		assignee := r.URL.Query().Get("assignee")
		eng := getEngine(ctx, deps)
		tasks, err := eng.ListUserTasks(ctx, tenantID, assignee)
		if err != nil {
			writeInternalError(w, "list_tasks_failed", err.Error())
			return
		}
		if tasks == nil {
			tasks = []*engine.UserTask{}
		}
		writeJSON(w, http.StatusOK, tasks)
	}
}
