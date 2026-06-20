package feishu

import (
	"encoding/json"
	"strings"

	"github.com/monoposer/lowcode-bpmn/internal/event"
	"github.com/monoposer/lowcode-bpmn/plugins/sdk"
)

func Source(evt event.InboundEvent) bool {
	if evt.Source == "feishu" || evt.Source == "lark" {
		return true
	}
	return strings.Contains(strings.ToLower(sdk.HeaderGet(evt.Headers, "X-Feishu-Source", "User-Agent")), "feishu") ||
		strings.Contains(strings.ToLower(sdk.HeaderGet(evt.Headers, "User-Agent")), "lark")
}

func Tenant(evt event.InboundEvent, def string) string {
	t := sdk.TenantOrDefault(evt.TenantID, def)
	if t == "" {
		return "demo"
	}
	return t
}

type Envelope struct {
	Header struct {
		EventType string `json:"event_type"`
		TenantKey string `json:"tenant_key"`
	} `json:"header"`
	Event json.RawMessage `json:"event"`
}

type ContactDeleted struct {
	Object struct {
		UserID string `json:"user_id"`
		OpenID string `json:"open_id"`
	} `json:"object"`
}

type ApprovalEvent struct {
	InstanceCode string `json:"instance_code"`
	Status       string `json:"status"`
	UserID       string `json:"user_id"`
	OpenID       string `json:"open_id"`
	Comment      string `json:"comment"`
}
