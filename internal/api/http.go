package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"lowcode-automation/internal/engine"
)

// NewHTTPRouter constructs the main HTTP router for the platform.
func NewHTTPRouter() http.Handler {
	r := chi.NewRouter()

	// middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// health
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// minimal demo workspace (in a real system this is multi-tenant & authenticated)
	r.Route("/api/v1/workspaces/{workspaceID}", func(r chi.Router) {
		r.Route("/flows", func(r chi.Router) {
			r.Post("/", handleCreateFlow)
			r.Get("/", handleListFlows)
			r.Route("/{flowID}", func(r chi.Router) {
				r.Get("/", handleGetFlow)
				r.Put("/definition", handleUpdateFlowDefinition)
				r.Post("/runs", handleCreateRun)
				r.Get("/runs", handleListRuns)
				r.Get("/runs/{runID}", handleGetRun)
			})
		})
	})

	return r
}

// For now we keep a global engine instance with in-memory store for demo.
var defaultEngine = engine.NewEngine(nil)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Flow DTOs for HTTP layer

type createFlowRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type createRunRequest struct {
	Input map[string]any `json:"input"`
}

type updateFlowDefinitionRequest struct {
	Definition *engine.FlowDefinition `json:"definition"`
}

func handleCreateFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workspaceIDStr := chi.URLParam(r, "workspaceID")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspaceID"})
		return
	}

	var req createFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	f := &engine.Flow{
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Description: req.Description,
		Status:      engine.FlowStatusDraft,
	}

	if err := getEngine(ctx).CreateFlow(ctx, f); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, f)
}

func handleGetFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	flowIDStr := chi.URLParam(r, "flowID")
	flowID, err := uuid.Parse(flowIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid flowID"})
		return
	}

	f, err := getEngine(ctx).GetFlow(ctx, flowID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if f == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "flow not found"})
		return
	}

	writeJSON(w, http.StatusOK, f)
}

// handleUpdateFlowDefinition updates the definition (graph) of a flow.
func handleUpdateFlowDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	flowIDStr := chi.URLParam(r, "flowID")
	flowID, err := uuid.Parse(flowIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid flowID"})
		return
	}

	var req updateFlowDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Definition == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "definition is required"})
		return
	}

	f, err := getEngine(ctx).GetFlow(ctx, flowID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if f == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "flow not found"})
		return
	}

	f.Definition = req.Definition

	if err := getEngine(ctx).UpdateFlow(ctx, f); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, f)
}

func handleListFlows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workspaceIDStr := chi.URLParam(r, "workspaceID")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspaceID"})
		return
	}

	flows, err := getEngine(ctx).ListFlows(ctx, workspaceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, flows)
}

func handleCreateRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workspaceIDStr := chi.URLParam(r, "workspaceID")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspaceID"})
		return
	}

	flowIDStr := chi.URLParam(r, "flowID")
	flowID, err := uuid.Parse(flowIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid flowID"})
		return
	}

	var req createRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	run, err := getEngine(ctx).StartRun(ctx, flowID, workspaceID, req.Input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, run)
}

func handleGetRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	runIDStr := chi.URLParam(r, "runID")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid runID"})
		return
	}

	run, err := getEngine(ctx).GetRun(ctx, runID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if run == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
		return
	}

	writeJSON(w, http.StatusOK, run)
}

func handleListRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	flowIDStr := chi.URLParam(r, "flowID")
	flowID, err := uuid.Parse(flowIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid flowID"})
		return
	}

	runs, err := getEngine(ctx).ListRunsByFlow(ctx, flowID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, runs)
}

// In the future, the engine may be injected via context or DI container.
// For now we keep it simple.
type engineKey struct{}

func WithEngine(ctx context.Context, e *engine.Engine) context.Context {
	return context.WithValue(ctx, engineKey{}, e)
}

func getEngine(ctx context.Context) *engine.Engine {
	if e, ok := ctx.Value(engineKey{}).(*engine.Engine); ok && e != nil {
		return e
	}
	// default global engine must be initialised by main.
	return defaultEngine
}


