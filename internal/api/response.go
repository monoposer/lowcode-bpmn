package api

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the standard API error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{Error: message, Code: code})
}

func writeBadRequest(w http.ResponseWriter, code, message string) {
	writeError(w, http.StatusBadRequest, code, message)
}

func writeNotFound(w http.ResponseWriter, code, message string) {
	writeError(w, http.StatusNotFound, code, message)
}

func writeInternalError(w http.ResponseWriter, code, message string) {
	writeError(w, http.StatusInternalServerError, code, message)
}
