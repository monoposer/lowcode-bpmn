package wasm_test

import (
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/event"
	"github.com/monoposer/lowcode-bpmn/internal/plugin/wasm"
)

func TestCapabilitySet(t *testing.T) {
	caps := wasm.ParseCapabilities([]string{"remove_user", "trigger_message"})
	if !caps.Has(wasm.CapRemoveUser) {
		t.Fatal("expected remove_user capability")
	}
	if caps.Has(wasm.CapStartProcess) {
		t.Fatal("start_process should not be granted")
	}
}

func TestManifestStreamTask(t *testing.T) {
	m := wasm.Manifest{Stream: "task"}
	s, err := m.EventStream()
	if err != nil || s != event.StreamTask {
		t.Fatalf("stream task: %v %v", s, err)
	}
}

func TestAllTaskCapabilities(t *testing.T) {
	if !wasm.AllTask.Has(wasm.CapCompleteTask) {
		t.Fatal("AllTask should include complete_task")
	}
	if wasm.AllTask.Has(wasm.CapTriggerMessage) {
		t.Fatal("AllTask should not include trigger_message")
	}
}
