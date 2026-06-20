package generic

import (
	"context"
	"encoding/json"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/contract"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/sdk"
)

type AssigneeAdapter struct{}

func (AssigneeAdapter) Name() string         { return "generic" }
func (AssigneeAdapter) Stream() event.Stream { return event.StreamAssignee }
func (AssigneeAdapter) Supports(evt event.InboundEvent) bool {
	return evt.Source == "" || evt.Source == "generic" || evt.Source == "webhook"
}

func (AssigneeAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	var action sdk.Action
	if err := json.Unmarshal(evt.Payload, &action); err != nil {
		return err
	}
	if action.Kind == "" {
		action.Kind = "remove_user"
	}
	if action.TenantID == "" {
		action.TenantID = sdk.TenantOrDefault(evt.TenantID, "demo")
	}
	return sdk.ApplyAssigneeAction(ctx, host, action)
}

type TriggerAdapter struct{}

func (TriggerAdapter) Name() string         { return "generic" }
func (TriggerAdapter) Stream() event.Stream { return event.StreamTrigger }
func (TriggerAdapter) Supports(evt event.InboundEvent) bool {
	return evt.Source == "" || evt.Source == "generic" || evt.Source == "webhook"
}

func (TriggerAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	var action sdk.Action
	if err := json.Unmarshal(evt.Payload, &action); err != nil {
		return err
	}
	if action.Kind == "" && action.MessageRef == "" {
		var direct struct {
			TenantID    string         `json:"tenantId"`
			MessageRef  string         `json:"messageRef"`
			BusinessKey string         `json:"businessKey"`
			Variables   map[string]any `json:"variables"`
		}
		if err := json.Unmarshal(evt.Payload, &direct); err == nil && direct.MessageRef != "" {
			action.Kind = "trigger_message"
			action.TenantID = direct.TenantID
			action.MessageRef = direct.MessageRef
			action.BusinessKey = direct.BusinessKey
			action.Variables = direct.Variables
		}
	}
	if action.Kind == "" && action.ProcessKey != "" {
		action.Kind = "start_process"
	}
	if action.TenantID == "" {
		action.TenantID = sdk.TenantOrDefault(evt.TenantID, "demo")
	}
	return sdk.ApplyTriggerAction(ctx, host, action)
}

type TaskAdapter struct{}

func (TaskAdapter) Name() string         { return "generic" }
func (TaskAdapter) Stream() event.Stream { return event.StreamTask }
func (TaskAdapter) Supports(evt event.InboundEvent) bool {
	return evt.Source == "" || evt.Source == "generic" || evt.Source == "webhook"
}

func (TaskAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	var action sdk.Action
	if err := json.Unmarshal(evt.Payload, &action); err != nil {
		return err
	}
	if action.Kind == "" {
		action.Kind = "complete_task"
	}
	if action.TenantID == "" {
		action.TenantID = sdk.TenantOrDefault(evt.TenantID, "demo")
	}
	return sdk.ApplyTaskAction(ctx, host, action)
}

type ControlAdapter struct{}

func (ControlAdapter) Name() string         { return "generic" }
func (ControlAdapter) Stream() event.Stream { return event.StreamControl }
func (ControlAdapter) Supports(evt event.InboundEvent) bool {
	return evt.Source == "" || evt.Source == "generic" || evt.Source == "webhook"
}

func (ControlAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	var action sdk.Action
	if err := json.Unmarshal(evt.Payload, &action); err != nil {
		return err
	}
	if action.Kind == "" {
		action.Kind = "terminate"
	}
	if action.TenantID == "" {
		action.TenantID = sdk.TenantOrDefault(evt.TenantID, "demo")
	}
	return sdk.ApplyControlAction(ctx, host, action)
}
