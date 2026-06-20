package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/contract"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const hostModuleName = "bpmn_host"

// Adapter runs a WASM module as an EventAdapter with capability-gated host imports.
type Adapter struct {
	Manifest    Manifest
	eventStream event.Stream
	Sources     map[string]struct{}
	Caps        Set

	runtime wazero.Runtime
	module  api.Module
	mu      sync.Mutex
	host    contract.Host
}

// NewAdapter compiles and instantiates WASM with only permitted host functions.
func NewAdapter(ctx context.Context, wasmBytes []byte, manifest Manifest, host contract.Host) (*Adapter, error) {
	stream, err := manifest.EventStream()
	if err != nil {
		return nil, err
	}
	caps := manifest.CapabilitySet()
	if len(caps) == 0 {
		switch stream {
		case event.StreamAssignee:
			caps = AllAssignee
		case event.StreamTrigger:
			caps = AllTrigger
		case event.StreamTask:
			caps = AllTask
		case event.StreamControl:
			caps = AllControl
		}
	}
	sources := make(map[string]struct{}, len(manifest.Sources))
	for _, s := range manifest.Sources {
		sources[s] = struct{}{}
	}

	r := wazero.NewRuntime(ctx)
	capHost := &Adapter{Manifest: manifest, eventStream: stream, Sources: sources, Caps: caps, runtime: r, host: host}

	hmb := r.NewHostModuleBuilder(hostModuleName)
	if caps.Has(CapTriggerMessage) {
		hmb.NewFunctionBuilder().WithFunc(capHost.hostTriggerMessage).Export("trigger_message")
	}
	if caps.Has(CapStartProcess) {
		hmb.NewFunctionBuilder().WithFunc(capHost.hostStartProcess).Export("start_process")
	}
	if caps.Has(CapRemoveUser) {
		hmb.NewFunctionBuilder().WithFunc(capHost.hostRemoveUser).Export("remove_user")
	}
	if caps.Has(CapReplaceAssignees) {
		hmb.NewFunctionBuilder().WithFunc(capHost.hostReplaceAssignees).Export("replace_assignees")
	}
	if caps.Has(CapReadTasks) {
		hmb.NewFunctionBuilder().WithFunc(capHost.hostListTasks).Export("list_tasks")
	}
	if caps.Has(CapReadInstances) {
		hmb.NewFunctionBuilder().WithFunc(capHost.hostGetInstance).Export("get_process_instance")
	}
	if caps.Has(CapReadActivities) {
		hmb.NewFunctionBuilder().WithFunc(capHost.hostListActivities).Export("list_activities")
	}
	if caps.Has(CapCompleteTask) {
		hmb.NewFunctionBuilder().WithFunc(capHost.hostCompleteTask).Export("complete_task")
	}
	if caps.Has(CapTerminate) {
		hmb.NewFunctionBuilder().WithFunc(capHost.hostTerminate).Export("terminate")
	}
	if _, err := hmb.Instantiate(ctx); err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("wasm host module: %w", err)
	}

	mod, err := r.Instantiate(ctx, wasmBytes)
	if err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("wasm instantiate: %w", err)
	}
	capHost.module = mod
	return capHost, nil
}

func (a *Adapter) Name() string         { return "wasm:" + a.Manifest.Name }
func (a *Adapter) Stream() event.Stream { return a.eventStream }

func (a *Adapter) Supports(evt event.InboundEvent) bool {
	if len(a.Sources) == 0 {
		return true
	}
	_, ok := a.Sources[evt.Source]
	return ok
}

func (a *Adapter) Handle(ctx context.Context, evt event.InboundEvent, _ contract.Host) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.module == nil {
		return fmt.Errorf("wasm module not loaded")
	}
	raw, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	ptr, err := a.writeString(ctx, raw)
	if err != nil {
		return err
	}
	handle := a.module.ExportedFunction("handle")
	if handle == nil {
		return fmt.Errorf("wasm export handle not found")
	}
	results, err := handle.Call(ctx, uint64(ptr), uint64(len(raw)))
	if err != nil {
		return err
	}
	if len(results) > 0 && results[0] != 0 {
		return fmt.Errorf("wasm handle returned %d", results[0])
	}
	return nil
}

func (a *Adapter) Close(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.runtime != nil {
		return a.runtime.Close(ctx)
	}
	return nil
}

func (a *Adapter) writeString(ctx context.Context, b []byte) (uint32, error) {
	alloc := a.module.ExportedFunction("alloc")
	if alloc == nil {
		return 0, fmt.Errorf("wasm export alloc(size) required")
	}
	results, err := alloc.Call(ctx, uint64(len(b)))
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("alloc returned no value")
	}
	ptr := uint32(results[0])
	mem := a.module.Memory()
	if !mem.Write(ptr, b) {
		return 0, fmt.Errorf("wasm memory write failed")
	}
	return ptr, nil
}

func (a *Adapter) readJSON(ptr, size uint32, dest any) error {
	mem := a.module.Memory()
	b, ok := mem.Read(ptr, size)
	if !ok {
		return fmt.Errorf("wasm memory read failed")
	}
	return json.Unmarshal(b, dest)
}

func (a *Adapter) hostTriggerMessage(ctx context.Context, m api.Module, ptr, size uint32) uint32 {
	if !a.Caps.Has(CapTriggerMessage) {
		return 403
	}
	var req contract.TriggerMessageRequest
	if err := a.readJSON(ptr, size, &req); err != nil {
		return 400
	}
	if err := a.host.TriggerMessage(ctx, req); err != nil {
		return 500
	}
	return 0
}

func (a *Adapter) hostStartProcess(ctx context.Context, m api.Module, ptr, size uint32) uint32 {
	if !a.Caps.Has(CapStartProcess) {
		return 403
	}
	var req contract.StartProcessRequest
	if err := a.readJSON(ptr, size, &req); err != nil {
		return 400
	}
	if err := a.host.StartProcess(ctx, req); err != nil {
		return 500
	}
	return 0
}

func (a *Adapter) hostRemoveUser(ctx context.Context, m api.Module, ptr, size uint32) uint32 {
	if !a.Caps.Has(CapRemoveUser) {
		return 403
	}
	var req contract.RemoveUserRequest
	if err := a.readJSON(ptr, size, &req); err != nil {
		return 400
	}
	if err := a.host.RemoveUserFromActiveTasks(ctx, req); err != nil {
		return 500
	}
	return 0
}

func (a *Adapter) hostReplaceAssignees(ctx context.Context, m api.Module, ptr, size uint32) uint32 {
	if !a.Caps.Has(CapReplaceAssignees) {
		return 403
	}
	var req contract.ReplaceAssigneesRequest
	if err := a.readJSON(ptr, size, &req); err != nil {
		return 400
	}
	if err := a.host.ReplaceTaskAssignees(ctx, req); err != nil {
		return 500
	}
	return 0
}

func (a *Adapter) hostListTasks(ctx context.Context, m api.Module, tenantPtr, tenantLen, assigneePtr, assigneeLen, outPtr, outMaxLen uint32) uint32 {
	if !a.Caps.Has(CapReadTasks) {
		return 403
	}
	tenant, ok := readMem(m, tenantPtr, tenantLen)
	if !ok {
		return 400
	}
	assignee, ok := readMem(m, assigneePtr, assigneeLen)
	if !ok {
		return 400
	}
	tasks, err := a.host.ListUserTasks(ctx, string(tenant), string(assignee))
	if err != nil {
		return 500
	}
	raw, err := json.Marshal(tasks)
	if err != nil {
		return 500
	}
	return writeBounded(m, outPtr, outMaxLen, raw)
}

func readMem(m api.Module, ptr, size uint32) ([]byte, bool) {
	return m.Memory().Read(ptr, size)
}

func writeMem(m api.Module, ptr uint32, b []byte) bool {
	return m.Memory().Write(ptr, b)
}

const wasmStatusPayloadTooLarge uint32 = 413

func writeBounded(m api.Module, ptr, maxLen uint32, b []byte) uint32 {
	if maxLen == 0 {
		return 400
	}
	if len(b) > int(maxLen) {
		return wasmStatusPayloadTooLarge
	}
	if !writeMem(m, ptr, b) {
		return 500
	}
	return 0
}

func (a *Adapter) hostGetInstance(ctx context.Context, m api.Module, idPtr, idLen, outPtr, outMaxLen uint32) uint32 {
	if !a.Caps.Has(CapReadInstances) {
		return 403
	}
	idRaw, ok := readMem(m, idPtr, idLen)
	if !ok {
		return 400
	}
	id, err := parseUUID(string(idRaw))
	if err != nil {
		return 400
	}
	inst, err := a.host.GetProcessInstance(ctx, id)
	if err != nil {
		return 500
	}
	raw, err := json.Marshal(inst)
	if err != nil {
		return 500
	}
	return writeBounded(m, outPtr, outMaxLen, raw)
}

func (a *Adapter) hostListActivities(ctx context.Context, m api.Module, idPtr, idLen, outPtr, outMaxLen uint32) uint32 {
	if !a.Caps.Has(CapReadActivities) {
		return 403
	}
	idRaw, ok := readMem(m, idPtr, idLen)
	if !ok {
		return 400
	}
	id, err := parseUUID(string(idRaw))
	if err != nil {
		return 400
	}
	acts, err := a.host.ListActivities(ctx, id)
	if err != nil {
		return 500
	}
	raw, err := json.Marshal(acts)
	if err != nil {
		return 500
	}
	return writeBounded(m, outPtr, outMaxLen, raw)
}

func (a *Adapter) hostCompleteTask(ctx context.Context, m api.Module, ptr, size uint32) uint32 {
	if !a.Caps.Has(CapCompleteTask) {
		return 403
	}
	var req contract.CompleteTaskRequest
	if err := a.readJSON(ptr, size, &req); err != nil {
		return 400
	}
	if err := a.host.CompleteTask(ctx, req); err != nil {
		return 500
	}
	return 0
}

func (a *Adapter) hostTerminate(ctx context.Context, m api.Module, ptr, size uint32) uint32 {
	if !a.Caps.Has(CapTerminate) {
		return 403
	}
	var req contract.TerminateRequest
	if err := a.readJSON(ptr, size, &req); err != nil {
		return 400
	}
	if err := a.host.Terminate(ctx, req); err != nil {
		return 500
	}
	return 0
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(s))
}
