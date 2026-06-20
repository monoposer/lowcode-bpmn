package plugin

import (
	"context"
	"os"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/internal/plugin/wasm"
	"github.com/monoposer/lowcode-bpmn/pkg/env"
	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/event/setup"
	"github.com/monoposer/lowcode-bpmn/plugins/registry"
)

// Config controls quad-stream consumers and adapter registration.
type Config struct {
	ConsumerKind     string
	Consumer         setup.Config
	AssigneeAdapters []string
	TriggerAdapters  []string
	TaskAdapters     []string
	ControlAdapters  []string
	WASMDir          string
	DefaultTenant    string
}

// Bootstrap wires assignee, trigger, task, and control consumers to separate runtimes.
type Bootstrap struct {
	AssigneeConsumer event.Consumer
	TriggerConsumer  event.Consumer
	TaskConsumer     event.Consumer
	ControlConsumer  event.Consumer
	AssigneeRuntime  *Runtime
	TriggerRuntime   *Runtime
	TaskRuntime      *Runtime
	ControlRuntime   *Runtime
	Host             Host
	router           *event.RouterPublisher
	wasmClosers      []func(context.Context) error
}

// LoadConfigFromEnv reads PLUGIN_* and EVENT_* settings.
func LoadConfigFromEnv() Config {
	assignee := env.CSV("PLUGIN_ASSIGNEE_ADAPTERS")
	if len(assignee) == 0 {
		assignee = []string{"generic", "canonical", "feishu", "wecom"}
	}
	trigger := env.CSV("PLUGIN_TRIGGER_ADAPTERS")
	if len(trigger) == 0 {
		trigger = []string{"generic", "canonical", "feishu", "wecom", "airtable"}
	}
	task := env.CSV("PLUGIN_TASK_ADAPTERS")
	if len(task) == 0 {
		task = []string{"generic", "canonical", "feishu", "wecom"}
	}
	control := env.CSV("PLUGIN_CONTROL_ADAPTERS")
	if len(control) == 0 {
		control = []string{"generic", "canonical"}
	}
	return Config{
		ConsumerKind:     env.Get("EVENT_CONSUMER", "memory"),
		Consumer:         setup.LoadConfigFromEnv(),
		AssigneeAdapters: assignee,
		TriggerAdapters:  trigger,
		TaskAdapters:     task,
		ControlAdapters:  control,
		WASMDir:          os.Getenv("PLUGIN_WASM_DIR"),
		DefaultTenant:    env.Get("PLUGIN_DEFAULT_TENANT", "demo"),
	}
}

// BootstrapFromEngine creates consumers, host SDK, and stream-specific runtimes.
func BootstrapFromEngine(ctx context.Context, eng *engine.Engine, cfg Config) (*Bootstrap, error) {
	host := NewHost(eng)
	ccfg := cfg.Consumer
	if ccfg.Driver == "" {
		ccfg.Driver = cfg.ConsumerKind
	}
	streams, err := setup.NewStreams(ccfg)
	if err != nil {
		return nil, err
	}

	plugCfg := registry.Config{DefaultTenant: cfg.DefaultTenant}
	assigneeGo := registry.Pick(event.StreamAssignee, cfg.AssigneeAdapters, plugCfg)
	triggerGo := registry.Pick(event.StreamTrigger, cfg.TriggerAdapters, plugCfg)
	taskGo := registry.Pick(event.StreamTask, cfg.TaskAdapters, plugCfg)
	controlGo := registry.Pick(event.StreamControl, cfg.ControlAdapters, plugCfg)

	wasmAll, err := wasm.LoadAdapters(ctx, cfg.WASMDir, host)
	if err != nil {
		return nil, err
	}

	assigneeAdapters := append(assigneeGo, wasm.FilterByStream(wasmAll, event.StreamAssignee)...)
	triggerAdapters := append(triggerGo, wasm.FilterByStream(wasmAll, event.StreamTrigger)...)
	taskAdapters := append(taskGo, wasm.FilterByStream(wasmAll, event.StreamTask)...)
	controlAdapters := append(controlGo, wasm.FilterByStream(wasmAll, event.StreamControl)...)

	b := &Bootstrap{
		AssigneeConsumer: streams.Assignee,
		TriggerConsumer:  streams.Trigger,
		TaskConsumer:     streams.Task,
		ControlConsumer:  streams.Control,
		AssigneeRuntime:  NewRuntime(host, assigneeAdapters...),
		TriggerRuntime:   NewRuntime(host, triggerAdapters...),
		TaskRuntime:      NewRuntime(host, taskAdapters...),
		ControlRuntime:   NewRuntime(host, controlAdapters...),
		Host:             host,
		router:           event.NewRouterPublisher(streams.Assignee, streams.Trigger, streams.Task, streams.Control),
	}
	for _, ad := range wasmAll {
		if a, ok := ad.(*wasm.Adapter); ok {
			mod := a
			b.wasmClosers = append(b.wasmClosers, mod.Close)
		}
	}
	return b, nil
}

func (b *Bootstrap) Publisher() event.Publisher { return b.router }

func (b *Bootstrap) Close(ctx context.Context) {
	for _, fn := range b.wasmClosers {
		_ = fn(ctx)
	}
	if b.AssigneeConsumer != nil {
		_ = b.AssigneeConsumer.Close()
	}
	if b.TriggerConsumer != nil {
		_ = b.TriggerConsumer.Close()
	}
	if b.TaskConsumer != nil {
		_ = b.TaskConsumer.Close()
	}
	if b.ControlConsumer != nil {
		_ = b.ControlConsumer.Close()
	}
}
