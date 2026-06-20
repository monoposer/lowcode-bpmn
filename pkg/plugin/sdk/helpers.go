package sdk

import (
	"encoding/json"
	"strings"
)

func TenantOrDefault(tenant, def string) string {
	if tenant != "" {
		return tenant
	}
	return def
}

func HeaderGet(headers map[string]string, keys ...string) string {
	if headers == nil {
		return ""
	}
	for _, k := range keys {
		for hk, v := range headers {
			if strings.EqualFold(hk, k) && v != "" {
				return v
			}
		}
	}
	return ""
}

func ParseJSON(payload []byte, out any) error {
	if len(payload) == 0 {
		return nil
	}
	return json.Unmarshal(payload, out)
}

func NormalizeApprovalAction(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "APPROVED", "APPROVE", "PASS", "PASSED", "AGREE", "AGREED":
		return "approve"
	case "REJECTED", "REJECT", "DENY", "DENIED", "REFUSE", "REFUSED":
		return "reject"
	default:
		return ""
	}
}

func MergeMaps(base map[string]any, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
