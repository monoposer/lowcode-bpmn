package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/internal/telemetry"
)

// RouterDeps holds shared handlers for HTTP routes.
type RouterDeps struct {
	Engine *engine.Engine
}

// NewHTTPRouter constructs the HTTP router for lowcode-bpmn.
func NewHTTPRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(telemetry.OTelMiddleware("lowcode-bpmn"))
	r.Use(telemetry.HTTPLogMiddleware)
	r.Use(telemetry.RecoverMiddleware)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(metricsMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Handle("/metrics", MetricsHandler())

	r.Get("/api/v1/tasks", handleListUserTasks(deps))

	r.Route("/api/v1/tenants/{tenantId}/processes", func(r chi.Router) {
		r.Put("/{key}", handleDeployProcess(deps))
		r.Get("/", handleListProcesses(deps))
		r.Delete("/{key}", handleDeleteProcess(deps))
	})

	r.Route("/api/v1/process-instances", func(r chi.Router) {
		r.Post("/", handleStartProcessInstance(deps))
		r.Route("/{instanceID}", func(r chi.Router) {
			r.Get("/", handleGetProcessInstance(deps))
			r.Get("/activities", handleListProcessActivities(deps))
			r.Post("/tasks/{activityID}/complete", handleCompleteUserTask(deps))
		})
	})

	return r
}

type engineKey struct{}

func WithEngine(ctx context.Context, e *engine.Engine) context.Context {
	return context.WithValue(ctx, engineKey{}, e)
}

func getEngine(ctx context.Context, deps RouterDeps) *engine.Engine {
	if e, ok := ctx.Value(engineKey{}).(*engine.Engine); ok && e != nil {
		return e
	}
	return deps.Engine
}

type deployProcessRequest struct {
	Definition bpmn.ProcessDefinition `json:"definition"`
}

func handleDeployProcess(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tenantID := chi.URLParam(r, "tenantId")
		key := chi.URLParam(r, "key")
		if tenantID == "" || key == "" {
			writeBadRequest(w, "invalid_request", "tenantId and key required")
			return
		}
		var req deployProcessRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "invalid_json", "invalid json body")
			return
		}
		if req.Definition.ID == "" {
			req.Definition.ID = key
		}
		eng := getEngine(ctx, deps)
		if eng == nil {
			writeInternalError(w, "engine_unavailable", "engine not configured")
			return
		}
		dp, err := eng.DeployProcess(ctx, tenantID, key, req.Definition)
		if err != nil {
			writeBadRequest(w, "deploy_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, dp)
	}
}

func handleDeleteProcess(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tenantID := chi.URLParam(r, "tenantId")
		key := chi.URLParam(r, "key")
		eng := getEngine(ctx, deps)
		if err := eng.DeleteProcess(ctx, tenantID, key); err != nil {
			writeInternalError(w, "delete_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func handleListProcesses(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tenantID := chi.URLParam(r, "tenantId")
		eng := getEngine(ctx, deps)
		list, err := eng.ListProcesses(ctx, tenantID)
		if err != nil {
			writeInternalError(w, "list_failed", err.Error())
			return
		}
		if list == nil {
			list = []*engine.DeployedProcess{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

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

type completeTaskRequest struct {
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
