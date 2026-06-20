package script

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dop251/goja"
)

const defaultHTTPTimeout = 10 * time.Second

const defaultHTTPMaxBody = 1 << 20 // 1 MiB

// bindHTTP exposes http.get/post/request to ScriptTask JavaScript.
func bindHTTP(vm *goja.Runtime, ctx context.Context, client *http.Client, maxBody int64) error {
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	if maxBody <= 0 {
		maxBody = defaultHTTPMaxBody
	}

	httpObj := vm.NewObject()
	call := func(method, url string, opts goja.Value) (goja.Value, error) {
		return doHTTP(vm, ctx, client, maxBody, method, url, opts)
	}
	if err := httpObj.Set("request", call); err != nil {
		return err
	}
	if err := httpObj.Set("get", func(url string, opts goja.Value) (goja.Value, error) {
		return call(http.MethodGet, url, opts)
	}); err != nil {
		return err
	}
	if err := httpObj.Set("post", func(url, body string, opts goja.Value) (goja.Value, error) {
		optVal := opts
		if body != "" {
			obj := vm.NewObject()
			_ = obj.Set("body", body)
			if opts != nil && !goja.IsUndefined(opts) && !goja.IsNull(opts) {
				if m, ok := opts.Export().(map[string]any); ok {
					if h, ok := m["headers"]; ok {
						_ = obj.Set("headers", h)
					}
				}
			}
			optVal = obj
		}
		return call(http.MethodPost, url, optVal)
	}); err != nil {
		return err
	}
	return vm.Set("http", httpObj)
}

func doHTTP(vm *goja.Runtime, ctx context.Context, client *http.Client, maxBody int64, method, url string, opts goja.Value) (goja.Value, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return goja.Undefined(), fmt.Errorf("http: method required")
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return goja.Undefined(), fmt.Errorf("http: url required")
	}
	if err := httpAllowed(ctx, url); err != nil {
		return goja.Undefined(), err
	}

	var body []byte
	headers := make(http.Header)
	if opts != nil && !goja.IsUndefined(opts) && !goja.IsNull(opts) {
		if m, ok := opts.Export().(map[string]any); ok {
			if raw, ok := m["body"]; ok && raw != nil {
				switch b := raw.(type) {
				case string:
					body = []byte(b)
				default:
					body = []byte(fmt.Sprint(b))
				}
			}
			if h, ok := m["headers"].(map[string]any); ok {
				for k, v := range h {
					headers.Set(k, fmt.Sprint(v))
				}
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return goja.Undefined(), err
	}
	req.Header = headers
	if len(body) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return goja.Undefined(), fmt.Errorf("http %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxBody+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return goja.Undefined(), err
	}
	if int64(len(raw)) > maxBody {
		return goja.Undefined(), fmt.Errorf("http: response body exceeds %d bytes", maxBody)
	}

	hdr := vm.NewObject()
	for k, vals := range resp.Header {
		if len(vals) == 1 {
			_ = hdr.Set(k, vals[0])
		} else {
			_ = hdr.Set(k, vals)
		}
	}
	out := vm.NewObject()
	_ = out.Set("status", resp.StatusCode)
	_ = out.Set("statusText", resp.Status)
	_ = out.Set("headers", hdr)
	_ = out.Set("body", string(raw))
	return out, nil
}
