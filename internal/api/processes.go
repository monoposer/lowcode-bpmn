package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/bpmnxml"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

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

		def, err := decodeProcessDefinition(r)
		if err != nil {
			writeBadRequest(w, "invalid_body", err.Error())
			return
		}
		if def.ID == "" {
			def.ID = key
		}

		eng := getEngine(ctx, deps)
		if eng == nil {
			writeInternalError(w, "engine_unavailable", "engine not configured")
			return
		}
		dp, err := eng.DeployProcess(ctx, tenantID, key, def)
		if err != nil {
			writeBadRequest(w, "deploy_failed", err.Error())
			return
		}
		if wantsXML(r) {
			writeProcessXML(w, dp.Definition)
			return
		}
		writeJSON(w, http.StatusOK, dp)
	}
}

func handleGetProcessDefinition(deps RouterDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tenantID := chi.URLParam(r, "tenantId")
		key := chi.URLParam(r, "key")
		eng := getEngine(ctx, deps)
		if eng == nil {
			writeInternalError(w, "engine_unavailable", "engine not configured")
			return
		}
		dp, err := eng.GetProcess(ctx, tenantID, key)
		if err != nil || dp == nil {
			writeBadRequest(w, "not_found", "process not found")
			return
		}
		if wantsXML(r) {
			writeProcessXML(w, dp.Definition)
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

func decodeProcessDefinition(r *http.Request) (bpmn.ProcessDefinition, error) {
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(ct, "xml") {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			return bpmn.ProcessDefinition{}, err
		}
		return bpmnxml.Parse(raw)
	}
	var req deployProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return bpmn.ProcessDefinition{}, err
	}
	return req.Definition, nil
}

func wantsXML(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "xml")
}

func writeProcessXML(w http.ResponseWriter, def bpmn.ProcessDefinition) {
	data, err := bpmnxml.Marshal(def)
	if err != nil {
		writeInternalError(w, "marshal_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
