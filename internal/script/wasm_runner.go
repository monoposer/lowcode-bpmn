package script

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/monoposer/lowcode-bpmn/pkg/env"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WASMRunner executes scriptLang wasm/webassembly modules via wazero.
type WASMRunner struct {
	defaultBundle *wasmModuleBundle
	tenantBundles map[string]*wasmModuleBundle

	mu    sync.Mutex
	cache map[string]*loadedWASM
}

type loadedWASM struct {
	runtime wazero.Runtime
	module  api.Module
	host    *wasmHost
	caps    ScriptCapSet
}

// NewWASMRunner loads WASM modules from SCRIPT_WASM_PATH and SCRIPT_WASM_DIR.
func NewWASMRunner() *WASMRunner {
	defaultPath := strings.TrimSpace(env.Get("SCRIPT_WASM_PATH", ""))
	dir := strings.TrimSpace(env.Get("SCRIPT_WASM_DIR", ""))

	var defaultBundle *wasmModuleBundle
	if defaultPath != "" {
		b, err := loadWASMBundle("", defaultPath)
		if err != nil {
			slog.Warn("script wasm default module load failed", slog.String("path", defaultPath), slog.String("error", err.Error()))
		} else {
			defaultBundle = b
		}
	}

	tenantBundles := loadTenantWASMBundles(dir)
	if defaultBundle == nil && len(tenantBundles) == 0 {
		return nil
	}
	return &WASMRunner{
		defaultBundle: defaultBundle,
		tenantBundles: tenantBundles,
		cache:         make(map[string]*loadedWASM),
	}
}

func (w *WASMRunner) Run(ctx context.Context, req RunRequest) (map[string]any, error) {
	if w == nil {
		return nil, errRunnerNotConfigured
	}
	if req.Script == "" {
		return nil, fmt.Errorf("script is empty")
	}

	bundle := w.bundleFor(req.TenantID)
	if bundle == nil {
		return nil, fmt.Errorf("wasm script module not configured for tenant %q", req.TenantID)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	loaded, err := w.loadLocked(ctx, bundle)
	if err != nil {
		attrs := append(slogAttrs(req), slog.String("error", err.Error()))
		slog.WarnContext(ctx, "wasm script load failed", attrs...)
		return nil, fmt.Errorf("wasm script: %w", err)
	}
	loaded.host.req = req

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
		return nil, fmt.Errorf("wasm script encode: %w", err)
	}

	inPtr, err := writeGuestBytes(ctx, loaded.module, payload)
	if err != nil {
		return nil, err
	}

	const outMax = 1 << 20
	outPtr, err := guestAlloc(ctx, loaded.module, outMax)
	if err != nil {
		return nil, err
	}

	runFn := loaded.module.ExportedFunction("run")
	if runFn == nil {
		return nil, fmt.Errorf("wasm export run not found")
	}
	written, err := runFn.Call(ctx, uint64(inPtr), uint64(len(payload)), uint64(outPtr), uint64(outMax))
	if err != nil {
		attrs := append(slogAttrs(req), slog.String("error", err.Error()))
		slog.WarnContext(ctx, "wasm script execution failed", attrs...)
		return nil, fmt.Errorf("wasm script error: %w", err)
	}
	if len(written) == 0 || written[0] == 0 {
		return nil, fmt.Errorf("wasm script returned no output")
	}

	var resp remoteRunResponse
	if err := readGuestJSON(loaded.module, outPtr, uint32(written[0]), &resp); err != nil {
		return nil, fmt.Errorf("wasm script decode: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("wasm script error: %s", resp.Error)
	}
	if resp.Variables == nil {
		return map[string]any{}, nil
	}
	return resp.Variables, nil
}

func (w *WASMRunner) bundleFor(tenantID string) *wasmModuleBundle {
	if tenantID != "" {
		if b, ok := w.tenantBundles[tenantID]; ok {
			return b
		}
	}
	return w.defaultBundle
}

func (w *WASMRunner) loadLocked(ctx context.Context, bundle *wasmModuleBundle) (*loadedWASM, error) {
	if bundle == nil {
		return nil, errRunnerNotConfigured
	}
	if loaded, ok := w.cache[bundle.key]; ok {
		return loaded, nil
	}

	caps := bundle.manifest.CapabilitySet()
	r := wazero.NewRuntime(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("wasi: %w", err)
	}
	host, err := attachScriptHost(ctx, r, caps, RunRequest{})
	if err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("script host module: %w", err)
	}

	mod, err := r.Instantiate(ctx, bundle.wasm)
	if err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("instantiate: %w", err)
	}

	loaded := &loadedWASM{runtime: r, module: mod, host: host, caps: caps}
	w.cache[bundle.key] = loaded
	return loaded, nil
}

// Close releases all cached runtimes.
func (w *WASMRunner) Close(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	var err error
	for k, loaded := range w.cache {
		if closeErr := loaded.runtime.Close(ctx); closeErr != nil && err == nil {
			err = closeErr
		}
		delete(w.cache, k)
	}
	return err
}

var _ Runner = (*WASMRunner)(nil)

func guestAlloc(ctx context.Context, mod api.Module, size uint32) (uint32, error) {
	alloc := mod.ExportedFunction("alloc")
	if alloc == nil {
		return 0, fmt.Errorf("wasm export alloc required")
	}
	results, err := alloc.Call(ctx, uint64(size))
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("wasm alloc returned no value")
	}
	return uint32(results[0]), nil
}

func writeGuestBytes(ctx context.Context, mod api.Module, b []byte) (uint32, error) {
	ptr, err := guestAlloc(ctx, mod, uint32(len(b)))
	if err != nil {
		return 0, err
	}
	if !mod.Memory().Write(ptr, b) {
		return 0, fmt.Errorf("wasm memory write failed")
	}
	return ptr, nil
}

func readGuestJSON(mod api.Module, ptr, size uint32, dest any) error {
	raw, ok := mod.Memory().Read(ptr, size)
	if !ok {
		return fmt.Errorf("wasm memory read failed")
	}
	return json.Unmarshal(raw, dest)
}
