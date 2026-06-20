package wecom

import (
	"context"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/contract"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/sdk"
)

type callback struct {
	MsgType    string `json:"MsgType"`
	Event      string `json:"Event"`
	ChangeType string `json:"ChangeType"`
	UserID     string `json:"UserID"`
}

func Source(evt event.InboundEvent) bool {
	if evt.Source == "wecom" || evt.Source == "wework" {
		return true
	}
	return containsIgnoreCase(sdk.HeaderGet(evt.Headers, "User-Agent"), "wxwork")
}

func Tenant(evt event.InboundEvent, def string) string {
	t := sdk.TenantOrDefault(evt.TenantID, def)
	if t == "" {
		return "demo"
	}
	return t
}

// AssigneeAdapter handles member delete → assignee sync.
type AssigneeAdapter struct {
	DefaultTenant string
}

func (a AssigneeAdapter) Name() string         { return "wecom" }
func (a AssigneeAdapter) Stream() event.Stream { return event.StreamAssignee }
func (a AssigneeAdapter) Supports(evt event.InboundEvent) bool {
	return Source(evt)
}

func (a AssigneeAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	tenant := Tenant(evt, a.DefaultTenant)
	var body callback
	if err := sdk.ParseJSON(evt.Payload, &body); err != nil {
		return err
	}
	if body.MsgType == "event" && body.Event == "change_contact" {
		switch body.ChangeType {
		case "delete_user", "delete_party":
			if body.UserID == "" {
				return nil
			}
			return sdk.ApplyAssigneeAction(ctx, host, sdk.Action{
				Kind:     "remove_user",
				TenantID: tenant,
				UserID:   body.UserID,
				Reason:   "wecom:change_contact:" + body.ChangeType,
				Operator: "wecom-adapter",
			})
		}
	}
	return nil
}

// TriggerAdapter forwards WeCom business events → process trigger.
type TriggerAdapter struct {
	DefaultTenant string
}

func (a TriggerAdapter) Name() string         { return "wecom" }
func (a TriggerAdapter) Stream() event.Stream { return event.StreamTrigger }
func (a TriggerAdapter) Supports(evt event.InboundEvent) bool {
	return Source(evt)
}

func (a TriggerAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	tenant := Tenant(evt, a.DefaultTenant)
	var body callback
	if err := sdk.ParseJSON(evt.Payload, &body); err != nil {
		return err
	}
	if body.MsgType == "event" && body.Event == "change_contact" {
		return nil
	}
	vars := map[string]any{"wecom": body}
	msgRef := "wecom.event"
	if body.Event != "" {
		msgRef = "wecom." + body.Event
	}
	return sdk.ApplyTriggerAction(ctx, host, sdk.Action{
		Kind:       "trigger_message",
		TenantID:   tenant,
		MessageRef: msgRef,
		Variables:  vars,
	})
}

func containsIgnoreCase(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexFold(s, sub) >= 0)
}

func indexFold(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if equalFold(s[i:i+len(sub)], sub) {
			return i
		}
	}
	return -1
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
