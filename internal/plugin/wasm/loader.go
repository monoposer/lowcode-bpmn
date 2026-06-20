package wasm

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/contract"
)

// LoadAdapters scans dir for subdirectories containing plugin.json + plugin.wasm.
func LoadAdapters(ctx context.Context, dir string, host contract.Host) ([]contract.EventAdapter, error) {
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []contract.EventAdapter
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		plugDir := filepath.Join(dir, e.Name())
		manifestPath := filepath.Join(plugDir, "plugin.json")
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		manifest, err := ParseManifest(raw)
		if err != nil {
			slog.Warn("wasm plugin manifest invalid", slog.String("dir", plugDir), slog.String("error", err.Error()))
			continue
		}
		wasmPath := filepath.Join(plugDir, manifest.WASM)
		wasmBytes, err := os.ReadFile(wasmPath)
		if err != nil {
			slog.Warn("wasm plugin binary missing", slog.String("path", wasmPath))
			continue
		}
		ad, err := NewAdapter(ctx, wasmBytes, manifest, host)
		if err != nil {
			slog.Warn("wasm plugin load failed", slog.String("name", manifest.Name), slog.String("error", err.Error()))
			continue
		}
		out = append(out, ad)
		slog.Info("wasm plugin loaded",
			slog.String("name", manifest.Name),
			slog.String("stream", manifest.Stream),
			slog.String("capabilities", strings.Join(manifest.Capabilities, ",")),
		)
	}
	return out, nil
}

// FilterByStream returns adapters matching a stream.
func FilterByStream(adapters []contract.EventAdapter, stream event.Stream) []contract.EventAdapter {
	var out []contract.EventAdapter
	for _, ad := range adapters {
		if ad.Stream() == stream {
			out = append(out, ad)
		}
	}
	return out
}
