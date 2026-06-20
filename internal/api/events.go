package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/event"
)

type inboundEventRequest struct {
	TenantID string            `json:"tenantId"`
	Topic    string            `json:"topic,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Payload  json.RawMessage   `json:"payload"`
}

func handleStreamInboundEvent(publisher event.Publisher, stream event.Stream) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if publisher == nil {
			writeBadRequest(w, "events_disabled", "event consumer not enabled")
			return
		}
		ctx := r.Context()
		source := chi.URLParam(r, "source")
		if source == "" {
			writeBadRequest(w, "invalid_source", "source path param required")
			return
		}

		var req inboundEventRequest
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeBadRequest(w, "invalid_body", "failed to read body")
			return
		}
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				writeBadRequest(w, "invalid_json", "invalid json body")
				return
			}
		}

		payload := req.Payload
		if len(payload) == 0 && len(body) > 0 {
			payload = body
		}

		tenantID := req.TenantID
		if tenantID == "" {
			tenantID = r.Header.Get("X-Tenant-Id")
		}
		if tenantID == "" {
			tenantID = r.Header.Get("X-Tenant-ID")
		}

		evt := event.InboundEvent{
			ID:         uuid.NewString(),
			Stream:     stream,
			Source:     source,
			Topic:      req.Topic,
			TenantID:   tenantID,
			Headers:    mergeHeaders(req.Headers, r.Header),
			Payload:    payload,
			ReceivedAt: time.Now().UTC(),
		}
		if err := publisher.Publish(ctx, stream, evt); err != nil {
			writeInternalError(w, "publish_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{
			"status":   "accepted",
			"stream":   string(stream),
			"event_id": evt.ID,
		})
	}
}

func mergeHeaders(custom map[string]string, h http.Header) map[string]string {
	out := make(map[string]string, len(custom)+4)
	for k, v := range custom {
		out[k] = v
	}
	for _, k := range []string{"Content-Type", "User-Agent", "X-Request-Id"} {
		if v := h.Get(k); v != "" {
			out[k] = v
		}
	}
	return out
}
