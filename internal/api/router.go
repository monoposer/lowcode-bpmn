package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/monoposer/lowcode-bpmn/internal/telemetry"
	"github.com/monoposer/lowcode-bpmn/pkg/event"
)

// NewHTTPRouter constructs the HTTP router for lowcode-bpmn.
func NewHTTPRouter(deps RouterDeps, auth AuthConfig) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(telemetry.OTelMiddleware("lowcode-bpmn"))
	r.Use(telemetry.HTTPLogMiddleware)
	r.Use(telemetry.RecoverMiddleware)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(metricsMiddleware)
	r.Use(AuthMiddleware(auth))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Handle("/metrics", MetricsHandler())

	r.Get("/api/v1/tasks", handleListUserTasks(deps))

	r.Route("/api/v1/assignee-sync", func(r chi.Router) {
		r.Post("/remove-user", handleAssigneeSyncRemoveUser(deps))
		r.Post("/replace", handleAssigneeSyncReplace(deps))
	})

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
			r.Patch("/tasks/{activityID}/assignees", handleUpdateAssignees(deps))
			r.Post("/terminate", handleTerminateInstance(deps))
		})
	})

	r.Route("/api/v1/triggers", func(r chi.Router) {
		r.Post("/message", handleTriggerMessage(deps))
	})

	r.Post("/api/v1/events/assignee/{source}", handleStreamInboundEvent(deps.Events, event.StreamAssignee))
	r.Post("/api/v1/events/trigger/{source}", handleStreamInboundEvent(deps.Events, event.StreamTrigger))
	r.Post("/api/v1/events/task/{source}", handleStreamInboundEvent(deps.Events, event.StreamTask))
	r.Post("/api/v1/events/control/{source}", handleStreamInboundEvent(deps.Events, event.StreamControl))

	return r
}
