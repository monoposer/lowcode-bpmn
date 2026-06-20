package feishu

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/monoposer/lowcode-bpmn/internal/event"
	"github.com/monoposer/lowcode-bpmn/internal/plugin/contract"
	"github.com/monoposer/lowcode-bpmn/plugins/sdk"
)

// TriggerAdapter handles Feishu business events → process start (not approval result).
type TriggerAdapter struct {
	DefaultTenant string
}

func (a TriggerAdapter) Name() string         { return "feishu" }
func (a TriggerAdapter) Stream() event.Stream { return event.StreamTrigger }
func (a TriggerAdapter) Supports(evt event.InboundEvent) bool {
	return Source(evt)
}

func (a TriggerAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	tenant := Tenant(evt, a.DefaultTenant)
	var env Envelope
	if err := sdk.ParseJSON(evt.Payload, &env); err != nil {
		return err
	}
	if env.Header.TenantKey != "" && evt.TenantID == "" {
		tenant = env.Header.TenantKey
	}
	switch env.Header.EventType {
	case "contact.user.deleted_v3", "contact.user.resigned_v1",
		"approval.instance.status_changed_v1", "approval.approval_updated_v1":
		return nil
	default:
		if env.Header.EventType == "" {
			return nil
		}
		vars := map[string]any{
			"feishu": map[string]any{
				"event_type": env.Header.EventType,
				"event":      json.RawMessage(env.Event),
			},
		}
		return sdk.ApplyTriggerAction(ctx, host, sdk.Action{
			Kind:       "trigger_message",
			TenantID:   tenant,
			MessageRef: "feishu." + strings.ReplaceAll(env.Header.EventType, ".", "_"),
			Variables:  vars,
		})
	}
}
