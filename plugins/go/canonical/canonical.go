package canonical

import (
	"context"
	"strings"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/contract"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/sdk"
)

type AssigneeAdapter struct{}

func (AssigneeAdapter) Name() string         { return "canonical" }
func (AssigneeAdapter) Stream() event.Stream { return event.StreamAssignee }
func (AssigneeAdapter) Supports(evt event.InboundEvent) bool {
	return strings.HasPrefix(evt.Topic, "canonical.assignee") || evt.Source == "canonical-assignee"
}

func (AssigneeAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	var action sdk.Action
	if err := sdk.ParseJSON(evt.Payload, &action); err != nil {
		return err
	}
	if action.TenantID == "" {
		action.TenantID = sdk.TenantOrDefault(evt.TenantID, "demo")
	}
	return sdk.ApplyAssigneeAction(ctx, host, action)
}

type TriggerAdapter struct{}

func (TriggerAdapter) Name() string         { return "canonical" }
func (TriggerAdapter) Stream() event.Stream { return event.StreamTrigger }
func (TriggerAdapter) Supports(evt event.InboundEvent) bool {
	return strings.HasPrefix(evt.Topic, "canonical.trigger") || evt.Source == "canonical-trigger"
}

func (TriggerAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	var action sdk.Action
	if err := sdk.ParseJSON(evt.Payload, &action); err != nil {
		return err
	}
	if action.TenantID == "" {
		action.TenantID = sdk.TenantOrDefault(evt.TenantID, "demo")
	}
	return sdk.ApplyTriggerAction(ctx, host, action)
}

type TaskAdapter struct{}

func (TaskAdapter) Name() string         { return "canonical" }
func (TaskAdapter) Stream() event.Stream { return event.StreamTask }
func (TaskAdapter) Supports(evt event.InboundEvent) bool {
	return strings.HasPrefix(evt.Topic, "canonical.task") || evt.Source == "canonical-task"
}

func (TaskAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	var action sdk.Action
	if err := sdk.ParseJSON(evt.Payload, &action); err != nil {
		return err
	}
	if action.TenantID == "" {
		action.TenantID = sdk.TenantOrDefault(evt.TenantID, "demo")
	}
	return sdk.ApplyTaskAction(ctx, host, action)
}

type ControlAdapter struct{}

func (ControlAdapter) Name() string         { return "canonical" }
func (ControlAdapter) Stream() event.Stream { return event.StreamControl }
func (ControlAdapter) Supports(evt event.InboundEvent) bool {
	return strings.HasPrefix(evt.Topic, "canonical.control") || evt.Source == "canonical-control"
}

func (ControlAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	var action sdk.Action
	if err := sdk.ParseJSON(evt.Payload, &action); err != nil {
		return err
	}
	if action.TenantID == "" {
		action.TenantID = sdk.TenantOrDefault(evt.TenantID, "demo")
	}
	return sdk.ApplyControlAction(ctx, host, action)
}
