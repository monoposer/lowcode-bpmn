package script

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const scriptManifestName = "script.json"

// ScriptManifest describes a WASM ScriptTask module and its host capabilities.
type ScriptManifest struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
	WASM         string   `json:"wasm"`
}

func (m ScriptManifest) CapabilitySet() ScriptCapSet {
	if len(m.Capabilities) == 0 {
		return DefaultScriptCaps
	}
	return ParseScriptCapabilities(m.Capabilities)
}

func ParseScriptManifest(raw []byte) (ScriptManifest, error) {
	var m ScriptManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return ScriptManifest{}, err
	}
	if m.WASM == "" {
		m.WASM = "script.wasm"
	}
	return m, nil
}

type wasmModuleBundle struct {
	manifest ScriptManifest
	wasm     []byte
	key      string
}

func loadWASMBundle(dir, wasmPath string) (*wasmModuleBundle, error) {
	wasmPath = strings.TrimSpace(wasmPath)
	if wasmPath == "" {
		return nil, nil
	}
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("read wasm %s: %w", wasmPath, err)
	}
	manifestPath := filepath.Join(filepath.Dir(wasmPath), scriptManifestName)
	manifest := ScriptManifest{Name: filepath.Base(filepath.Dir(wasmPath)), WASM: filepath.Base(wasmPath)}
	if raw, err := os.ReadFile(manifestPath); err == nil {
		if m, err := ParseScriptManifest(raw); err == nil {
			manifest = m
		}
	}
	return &wasmModuleBundle{
		manifest: manifest,
		wasm:     wasmBytes,
		key:      wasmPath,
	}, nil
}

func loadTenantWASMBundles(dir string) map[string]*wasmModuleBundle {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make(map[string]*wasmModuleBundle)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		tenantDir := filepath.Join(dir, e.Name())
		wasmPath := filepath.Join(tenantDir, "script.wasm")
		b, err := loadWASMBundle(tenantDir, wasmPath)
		if err != nil || b == nil {
			continue
		}
		out[e.Name()] = b
	}
	return out
}
