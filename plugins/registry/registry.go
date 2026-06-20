package registry

import (
	"log/slog"
	"strings"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/contract"
	"github.com/monoposer/lowcode-bpmn/plugins/go/airtable"
	"github.com/monoposer/lowcode-bpmn/plugins/go/canonical"
	"github.com/monoposer/lowcode-bpmn/plugins/go/feishu"
	genericplugin "github.com/monoposer/lowcode-bpmn/plugins/go/generic"
	"github.com/monoposer/lowcode-bpmn/plugins/go/wecom"
)

// Config is passed when constructing Go plugin adapters.
type Config struct {
	DefaultTenant string
}

func Resolve(stream event.Stream, name string, cfg Config) (contract.EventAdapter, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	switch stream {
	case event.StreamAssignee:
		switch key {
		case "generic":
			return genericplugin.AssigneeAdapter{}, true
		case "canonical":
			return canonical.AssigneeAdapter{}, true
		case "feishu", "lark":
			return feishu.AssigneeAdapter{DefaultTenant: cfg.DefaultTenant}, true
		case "wecom", "wework":
			return wecom.AssigneeAdapter{DefaultTenant: cfg.DefaultTenant}, true
		}
	case event.StreamTrigger:
		switch key {
		case "generic":
			return genericplugin.TriggerAdapter{}, true
		case "canonical":
			return canonical.TriggerAdapter{}, true
		case "feishu", "lark":
			return feishu.TriggerAdapter{DefaultTenant: cfg.DefaultTenant}, true
		case "wecom", "wework":
			return wecom.TriggerAdapter{DefaultTenant: cfg.DefaultTenant}, true
		case "airtable":
			return airtable.Adapter{DefaultTenant: cfg.DefaultTenant}, true
		}
	case event.StreamTask:
		switch key {
		case "generic":
			return genericplugin.TaskAdapter{}, true
		case "canonical":
			return canonical.TaskAdapter{}, true
		case "feishu", "lark":
			return feishu.TaskAdapter{DefaultTenant: cfg.DefaultTenant}, true
		case "wecom", "wework":
			return wecom.TaskAdapter{DefaultTenant: cfg.DefaultTenant}, true
		}
	case event.StreamControl:
		switch key {
		case "generic":
			return genericplugin.ControlAdapter{}, true
		case "canonical":
			return canonical.ControlAdapter{}, true
		}
	}
	return nil, false
}

func Pick(stream event.Stream, names []string, cfg Config) []contract.EventAdapter {
	var out []contract.EventAdapter
	seen := map[string]struct{}{}
	for _, name := range names {
		ad, ok := Resolve(stream, name, cfg)
		if !ok {
			slog.Warn("unknown go plugin skipped", slog.String("stream", string(stream)), slog.String("name", name))
			continue
		}
		if _, dup := seen[ad.Name()]; dup {
			continue
		}
		seen[ad.Name()] = struct{}{}
		out = append(out, ad)
	}
	return out
}
