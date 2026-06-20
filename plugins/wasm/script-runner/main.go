//go:build ignore

// TinyGo WASM guest — build with plugins/wasm/script-runner/build.sh (not the Go toolchain).

package main

import (
	"encoding/json"
	"unsafe"
)

//go:wasmimport script_host log
func hostLog(ptr, size uint32) uint32

//go:wasmimport script_host http_fetch
func hostHTTPFetch(urlPtr, urlLen, outPtr, outMax uint32) uint32

var heap []byte

//export alloc
func alloc(size uint32) *byte {
	if int(size) > len(heap) {
		heap = make([]byte, size)
	}
	return &heap[0]
}

type runRequest struct {
	Script    string         `json:"script"`
	Variables map[string]any `json:"variables"`
}

type runResponse struct {
	Variables map[string]any `json:"variables"`
	Error     string         `json:"error,omitempty"`
}

//export run
func run(inPtr, inLen, outPtr, outMax uint32) uint32 {
	input := readMem(inPtr, inLen)
	var req runRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return writeOut(outPtr, outMax, runResponse{Error: "invalid input json"})
	}

	out := req.Variables
	if out == nil {
		out = map[string]any{}
	}

	script := req.Script
	if script == "" {
		return writeOut(outPtr, outMax, runResponse{Error: "script is empty"})
	}

	_ = hostLog(strPtr("wasm script-runner start"), uint32(len("wasm script-runner start")))

	// Convention: script body is JSON object to merge, e.g. {"flag":true}
	// Prefix "http:" triggers host http_fetch of the remainder URL.
	if rest, ok := cutPrefix(script, "http:"); ok {
		url := trimSpace(rest)
		respLen := hostHTTPFetch(strPtr(url), uint32(len(url)), outPtr+512, outMax-512)
		if respLen == 0 {
			return writeOut(outPtr, outMax, runResponse{Error: "http_fetch failed"})
		}
		respJSON := readMem(outPtr+512, respLen)
		var fetched map[string]any
		_ = json.Unmarshal(respJSON, &fetched)
		out["http"] = fetched
	} else {
		var extra map[string]any
		if err := json.Unmarshal([]byte(script), &extra); err != nil {
			return writeOut(outPtr, outMax, runResponse{Error: "script must be JSON object or http:URL"})
		}
		for k, v := range extra {
			out[k] = v
		}
	}

	out["via"] = "wasm"
	return writeOut(outPtr, outMax, runResponse{Variables: out})
}

func writeOut(outPtr, outMax uint32, resp runResponse) uint32 {
	raw, err := json.Marshal(resp)
	if err != nil {
		errRaw, _ := json.Marshal(runResponse{Error: "marshal failed"})
		raw = errRaw
	}
	if uint32(len(raw)) > outMax {
		return 0
	}
	copyMem(outPtr, raw)
	return uint32(len(raw))
}

func readMem(ptr, size uint32) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
}

func copyMem(ptr uint32, b []byte) {
	dst := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), len(b))
	copy(dst, b)
}

func strPtr(s string) uint32 {
	b := []byte(s)
	if len(b) == 0 {
		return 0
	}
	p := alloc(uint32(len(b)))
	copyMem(uint32(uintptr(unsafe.Pointer(p))), b)
	return uint32(uintptr(unsafe.Pointer(p)))
}

func cutPrefix(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return s, false
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}

func main() {}
