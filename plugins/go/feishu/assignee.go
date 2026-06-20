package feishu

import (
	"context"
	"encoding/json"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/contract"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/sdk"
)

// AssigneeAdapter handles Feishu contact / HR events → assignee sync.
type AssigneeAdapter struct {
	DefaultTenant string
}

func (a AssigneeAdapter) Name() string         { return "feishu" }
func (a AssigneeAdapter) Stream() event.Stream { return event.StreamAssignee }
func (a AssigneeAdapter) Supports(evt event.InboundEvent) bool {
	return Source(evt)
}

func (a AssigneeAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	tenant := Tenant(evt, a.DefaultTenant)
	var env Envelope
	if err := sdk.ParseJSON(evt.Payload, &env); err != nil {
		return err
	}
	if env.Header.TenantKey != "" && evt.TenantID == "" {
		tenant = env.Header.TenantKey
	}
	switch env.Header.EventType {
	case "contact.user.deleted_v3", "contact.user.resigned_v1":
		var body ContactDeleted
		if err := json.Unmarshal(env.Event, &body); err != nil {
			return err
		}
		userID := body.Object.UserID
		if userID == "" {
			userID = body.Object.OpenID
		}
		if userID == "" {
			return nil
		}
		return sdk.ApplyAssigneeAction(ctx, host, sdk.Action{
			Kind:     "remove_user",
			TenantID: tenant,
			UserID:   userID,
			Reason:   "feishu:" + env.Header.EventType,
			Operator: "feishu-adapter",
		})
	default:
		return nil
	}
}
