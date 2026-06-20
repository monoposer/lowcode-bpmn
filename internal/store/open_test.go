package store_test

import (
	"context"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/store"
)

func TestOpenMemoryBackend(t *testing.T) {
	st, err := store.Open(context.Background(), store.Config{Backend: store.BackendMemory})
	if err != nil {
		t.Fatal(err)
	}
	if st == nil {
		t.Fatal("expected store")
	}
}

func TestOpenFileBackend(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(context.Background(), store.Config{
		Backend:  store.BackendFile,
		FilePath: dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Ping(context.Background(), st); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfig(t *testing.T) {
	t.Setenv("STORE_BACKEND", "file")
	t.Setenv("STORE_PATH", "/tmp/bpmn-data")

	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Backend != store.BackendFile || cfg.FilePath != "/tmp/bpmn-data" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestOpenDBRequiresDSN(t *testing.T) {
	_, err := store.Open(context.Background(), store.Config{Backend: store.BackendDB})
	if err == nil {
		t.Fatal("expected error without DATABASE_URL")
	}
}

func TestParseBackend(t *testing.T) {
	cases := map[string]store.Backend{
		"db":     store.BackendDB,
		"yaml":   store.BackendFile,
		"memory": store.BackendMemory,
	}
	for raw, want := range cases {
		got, err := store.ParseBackend(raw)
		if err != nil || got != want {
			t.Fatalf("ParseBackend(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
}