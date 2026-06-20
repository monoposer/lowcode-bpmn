package canonical

import (
	"context"
	"strings"

	"github.com/monoposer/lowcode-bpmn/internal/event"
	"github.com/monoposer/lowcode-bpmn/internal/plugin/contract"
	"github.com/monoposer/lowcode-bpmn/plugins/sdk"
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
	if err := sdk.ApplyTaskAction(ctx, host, action); err != nil {
		return err
	}
	return sdk.ApplyControlAction(ctx, host, action)
}
