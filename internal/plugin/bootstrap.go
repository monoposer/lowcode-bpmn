package plugin

import (
	"context"
	"os"
	"strings"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/internal/event"
	"github.com/monoposer/lowcode-bpmn/internal/event/setup"
	"github.com/monoposer/lowcode-bpmn/internal/plugin/wasm"
	"github.com/monoposer/lowcode-bpmn/plugins/registry"
)

// Config controls triple-stream consumers and adapter registration.
type Config struct {
	ConsumerKind     string
	Consumer         setup.Config
	AssigneeAdapters []string
	TriggerAdapters  []string
	TaskAdapters     []string
	WASMDir          string
	DefaultTenant    string
}

// Bootstrap wires assignee, trigger, and task consumers to separate runtimes.
type Bootstrap struct {
	AssigneeConsumer event.Consumer
	TriggerConsumer  event.Consumer
	TaskConsumer     event.Consumer
	AssigneeRuntime  *Runtime
	TriggerRuntime   *Runtime
	TaskRuntime      *Runtime
	Host             Host
	router           *event.RouterPublisher
	wasmClosers      []func(context.Context) error
}

// LoadConfigFromEnv reads PLUGIN_* and EVENT_* settings.
func LoadConfigFromEnv() Config {
	assignee := splitCSV(os.Getenv("PLUGIN_ASSIGNEE_ADAPTERS"))
	if len(assignee) == 0 {
		assignee = []string{"generic", "canonical", "feishu", "wecom"}
	}
	trigger := splitCSV(os.Getenv("PLUGIN_TRIGGER_ADAPTERS"))
	if len(trigger) == 0 {
		trigger = []string{"generic", "canonical", "feishu", "wecom", "airtable"}
	}
	task := splitCSV(os.Getenv("PLUGIN_TASK_ADAPTERS"))
	if len(task) == 0 {
		task = []string{"generic", "canonical", "feishu", "wecom"}
	}
	return Config{
		ConsumerKind:     getenv("EVENT_CONSUMER", "memory"),
		Consumer:         setup.LoadConfigFromEnv(),
		AssigneeAdapters: assignee,
		TriggerAdapters:  trigger,
		TaskAdapters:     task,
		WASMDir:          os.Getenv("PLUGIN_WASM_DIR"),
		DefaultTenant:    getenv("PLUGIN_DEFAULT_TENANT", "demo"),
	}
}

// BootstrapFromEngine creates triple consumers, host SDK, and stream-specific runtimes.
func BootstrapFromEngine(ctx context.Context, eng *engine.Engine, cfg Config) (*Bootstrap, error) {
	host := NewHost(eng)
	ccfg := cfg.Consumer
	if ccfg.Kind == "" {
		ccfg.Kind = cfg.ConsumerKind
	}
	streams, err := setup.NewStreams(ccfg)
	if err != nil {
		return nil, err
	}

	plugCfg := registry.Config{DefaultTenant: cfg.DefaultTenant}
	assigneeGo := registry.Pick(event.StreamAssignee, cfg.AssigneeAdapters, plugCfg)
	triggerGo := registry.Pick(event.StreamTrigger, cfg.TriggerAdapters, plugCfg)
	taskGo := registry.Pick(event.StreamTask, cfg.TaskAdapters, plugCfg)

	wasmAll, err := wasm.LoadAdapters(ctx, cfg.WASMDir, host)
	if err != nil {
		return nil, err
	}

	assigneeAdapters := append(assigneeGo, wasm.FilterByStream(wasmAll, event.StreamAssignee)...)
	triggerAdapters := append(triggerGo, wasm.FilterByStream(wasmAll, event.StreamTrigger)...)
	taskAdapters := append(taskGo, wasm.FilterByStream(wasmAll, event.StreamTask)...)

	b := &Bootstrap{
		AssigneeConsumer: streams.Assignee,
		TriggerConsumer:  streams.Trigger,
		TaskConsumer:     streams.Task,
		AssigneeRuntime:  NewRuntime(host, assigneeAdapters...),
		TriggerRuntime:   NewRuntime(host, triggerAdapters...),
		TaskRuntime:      NewRuntime(host, taskAdapters...),
		Host:             host,
		router:           event.NewRouterPublisher(streams.Assignee, streams.Trigger, streams.Task),
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
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
