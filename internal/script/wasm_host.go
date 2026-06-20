package script

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const scriptHostModule = "script_host"

type wasmHost struct {
	caps   ScriptCapSet
	client *http.Client
	req    RunRequest
}

func attachScriptHost(ctx context.Context, r wazero.Runtime, caps ScriptCapSet, req RunRequest) (*wasmHost, error) {
	host := &wasmHost{
		caps:   caps,
		client: &http.Client{Timeout: 10 * time.Second},
		req:    req,
	}
	b := r.NewHostModuleBuilder(scriptHostModule)
	if caps.Has(ScriptCapLog) {
		b.NewFunctionBuilder().WithFunc(host.hostLog).Export("log")
	}
	if caps.Has(ScriptCapHTTPFetch) {
		b.NewFunctionBuilder().WithFunc(host.hostHTTPFetch).Export("http_fetch")
	}
	if _, err := b.Instantiate(ctx); err != nil {
		return nil, err
	}
	return host, nil
}

type httpFetchResponse struct {
	Status int               `json:"status"`
	Body   string            `json:"body"`
	Header map[string]string `json:"headers,omitempty"`
}

func (h *wasmHost) hostLog(ctx context.Context, m api.Module, ptr, size uint32) uint32 {
	if !h.caps.Has(ScriptCapLog) {
		return 403
	}
	msg, err := readGuestString(m, ptr, size)
	if err != nil {
		return 400
	}
	attrs := append(slogAttrs(h.req), slog.String("message", msg))
	slog.InfoContext(ctx, "wasm script log", attrs...)
	return 0
}

func (h *wasmHost) hostHTTPFetch(ctx context.Context, m api.Module, urlPtr, urlLen, outPtr, outMax uint32) uint32 {
	if !h.caps.Has(ScriptCapHTTPFetch) {
		return 403
	}
	urlBytes, ok := m.Memory().Read(urlPtr, urlLen)
	if !ok {
		return 400
	}
	url := strings.TrimSpace(string(urlBytes))
	if err := httpAllowed(ctx, url); err != nil {
		return 0
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 400
	}
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return 502
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, defaultHTTPMaxBody+1))
	if err != nil {
		return 500
	}
	if int64(len(raw)) > defaultHTTPMaxBody {
		return 413
	}

	hdr := make(map[string]string, len(resp.Header))
	for k, vals := range resp.Header {
		if len(vals) > 0 {
			hdr[k] = vals[0]
		}
	}
	out, err := json.Marshal(httpFetchResponse{
		Status: resp.StatusCode,
		Body:   string(raw),
		Header: hdr,
	})
	if err != nil {
		return 500
	}
	if uint32(len(out)) > outMax {
		return 413
	}
	if !m.Memory().Write(outPtr, out) {
		return 500
	}
	return uint32(len(out))
}

func readGuestString(m api.Module, ptr, size uint32) (string, error) {
	b, ok := m.Memory().Read(ptr, size)
	if !ok {
		return "", fmt.Errorf("wasm memory read failed")
	}
	return string(b), nil
}
