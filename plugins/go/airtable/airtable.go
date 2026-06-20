package airtable

import (
	"context"

	"github.com/monoposer/lowcode-bpmn/internal/event"
	"github.com/monoposer/lowcode-bpmn/internal/plugin/contract"
	"github.com/monoposer/lowcode-bpmn/plugins/sdk"
)

// Adapter adapts Airtable automation/webhook payloads to message start events.
type Adapter struct {
	DefaultTenant string
	MessageRef    string
}

func (a Adapter) Name() string         { return "airtable" }
func (a Adapter) Stream() event.Stream { return event.StreamTrigger }
func (a Adapter) Supports(evt event.InboundEvent) bool {
	return evt.Source == "airtable"
}

type webhook struct {
	Base struct {
		ID string `json:"id"`
	} `json:"base"`
	Webhook struct {
		ID string `json:"id"`
	} `json:"webhook"`
	Timestamp string         `json:"timestamp"`
	Event     map[string]any `json:"event,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}

func (a Adapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	tenant := sdk.TenantOrDefault(evt.TenantID, a.DefaultTenant)
	if tenant == "" {
		tenant = "demo"
	}
	msgRef := a.MessageRef
	if msgRef == "" {
		msgRef = "airtable.orders.updated"
	}

	var body webhook
	if err := sdk.ParseJSON(evt.Payload, &body); err != nil {
		vars := map[string]any{"raw": string(evt.Payload)}
		return sdk.ApplyTriggerAction(ctx, host, sdk.Action{
			Kind:       "trigger_message",
			TenantID:   tenant,
			MessageRef: msgRef,
			Variables:  vars,
		})
	}

	vars := map[string]any{
		"event": map[string]any{
			"baseId":    body.Base.ID,
			"webhookId": body.Webhook.ID,
			"timestamp": body.Timestamp,
		},
	}
	if body.Event != nil {
		vars["event"] = sdk.MergeMaps(vars["event"].(map[string]any), body.Event)
	}
	if body.Payload != nil {
		if recID, ok := body.Payload["recordId"].(string); ok {
			vars["event"].(map[string]any)["recordId"] = recID
		}
		if fields, ok := body.Payload["fields"].(map[string]any); ok {
			vars["event"].(map[string]any)["fields"] = fields
		}
	}

	return sdk.ApplyTriggerAction(ctx, host, sdk.Action{
		Kind:       "trigger_message",
		TenantID:   tenant,
		MessageRef: msgRef,
		Variables:  vars,
	})
}
