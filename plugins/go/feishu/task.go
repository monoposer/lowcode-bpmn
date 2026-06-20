package feishu

import (
	"context"
	"encoding/json"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/contract"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/sdk"
)

// TaskAdapter handles Feishu approval result → CompleteTask (approve/reject).
type TaskAdapter struct {
	DefaultTenant string
}

func (a TaskAdapter) Name() string         { return "feishu" }
func (a TaskAdapter) Stream() event.Stream { return event.StreamTask }
func (a TaskAdapter) Supports(evt event.InboundEvent) bool {
	return Source(evt)
}

func (a TaskAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	tenant := Tenant(evt, a.DefaultTenant)

	var action sdk.Action
	if err := json.Unmarshal(evt.Payload, &action); err == nil && action.Kind == "complete_task" {
		if action.TenantID == "" {
			action.TenantID = tenant
		}
		return sdk.ApplyTaskAction(ctx, host, action)
	}

	var env Envelope
	if err := sdk.ParseJSON(evt.Payload, &env); err != nil {
		return err
	}
	if env.Header.TenantKey != "" && evt.TenantID == "" {
		tenant = env.Header.TenantKey
	}
	switch env.Header.EventType {
	case "approval.instance.status_changed_v1", "approval.approval_updated_v1":
		var body ApprovalEvent
		if err := json.Unmarshal(env.Event, &body); err != nil {
			return err
		}
		approvalAction := sdk.NormalizeApprovalAction(body.Status)
		if approvalAction == "" {
			return nil
		}
		assignee := body.UserID
		if assignee == "" {
			assignee = body.OpenID
		}
		if body.InstanceCode != "" && assignee != "" {
			return sdk.CompleteTaskByBusinessKey(ctx, host, tenant, assignee, body.InstanceCode, approvalAction, body.Comment)
		}
	}
	return nil
}
