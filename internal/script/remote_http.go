package script

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/monoposer/lowcode-bpmn/pkg/env"
)

const defaultRemoteTimeout = 30 * time.Second

// HTTPRemoteRunner executes scripts via an HTTP script service.
// POST {BaseURL}/run with JSON body; expects 200 and {"variables":{...}}.
type HTTPRemoteRunner struct {
	BaseURL string
	Client  *http.Client
}

type remoteRunRequest struct {
	Script     string         `json:"script"`
	Lang       string         `json:"lang"`
	Variables  map[string]any `json:"variables,omitempty"`
	InstanceID string         `json:"instanceId,omitempty"`
	ElementID  string         `json:"elementId,omitempty"`
	TenantID   string         `json:"tenantId,omitempty"`
	ProcessKey string         `json:"processKey,omitempty"`
}

type remoteRunResponse struct {
	Variables map[string]any `json:"variables"`
	Error     string         `json:"error,omitempty"`
}

// NewHTTPRemoteRunner creates a remote script client. BaseURL may include a path prefix.
func NewHTTPRemoteRunner(baseURL string, client *http.Client) *HTTPRemoteRunner {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if client == nil {
		timeout := defaultRemoteTimeout
		if d := env.Get("SCRIPT_REMOTE_TIMEOUT", ""); d != "" {
			if parsed, err := time.ParseDuration(d); err == nil && parsed > 0 {
				timeout = parsed
			}
		}
		client = &http.Client{Timeout: timeout}
	}
	return &HTTPRemoteRunner{BaseURL: baseURL, Client: client}
}

func (r *HTTPRemoteRunner) Run(ctx context.Context, req RunRequest) (map[string]any, error) {
	if r == nil || r.BaseURL == "" {
		return nil, errRunnerNotConfigured
	}
	if req.Script == "" {
		return nil, fmt.Errorf("script is empty")
	}

	payload, err := json.Marshal(remoteRunRequest{
		Script:     req.Script,
		Lang:       req.Lang,
		Variables:  req.Variables,
		InstanceID: req.InstanceID,
		ElementID:  req.ElementID,
		TenantID:   req.TenantID,
		ProcessKey: req.ProcessKey,
	})
	if err != nil {
		return nil, fmt.Errorf("remote script encode: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.BaseURL+"/run", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("remote script request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if req.TenantID != "" {
		httpReq.Header.Set("X-Tenant-Id", req.TenantID)
	}

	resp, err := r.Client.Do(httpReq)
	if err != nil {
		attrs := append(slogAttrs(req), slog.String("error", err.Error()))
		slog.WarnContext(ctx, "remote script request failed", attrs...)
		return nil, fmt.Errorf("remote script: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("remote script read: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("remote script: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out remoteRunResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("remote script decode: %w", err)
	}
	if out.Error != "" {
		return nil, fmt.Errorf("remote script error: %s", out.Error)
	}
	if out.Variables == nil {
		return map[string]any{}, nil
	}
	return out.Variables, nil
}
