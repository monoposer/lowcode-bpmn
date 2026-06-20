package wecom

import (
	"context"
	"encoding/json"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/contract"
	"github.com/monoposer/lowcode-bpmn/pkg/plugin/sdk"
)

// TaskAdapter handles WeCom approval callbacks → CompleteTask.
type TaskAdapter struct {
	DefaultTenant string
}

func (a TaskAdapter) Name() string         { return "wecom" }
func (a TaskAdapter) Stream() event.Stream { return event.StreamTask }
func (a TaskAdapter) Supports(evt event.InboundEvent) bool {
	return Source(evt)
}

type approvalInfo struct {
	SpNo     string `json:"SpNo"`
	SpStatus int    `json:"SpStatus"`
	Applyer  struct {
		UserID string `json:"UserId"`
	} `json:"Applyer"`
	Comments []struct {
		CommentUserInfo struct {
			UserID string `json:"UserId"`
		} `json:"CommentUserInfo"`
		CommentContent string `json:"CommentContent"`
	} `json:"Comments"`
}

type approvalCallback struct {
	ApprovalInfo approvalInfo `json:"ApprovalInfo"`
	UserID       string       `json:"UserID"`
	Action       string       `json:"Action"`
}

func approvalAction(spStatus int, action string) string {
	if a := sdk.NormalizeApprovalAction(action); a != "" {
		return a
	}
	switch spStatus {
	case 2:
		return "approve"
	case 3:
		return "reject"
	default:
		return ""
	}
}

func (a TaskAdapter) Handle(ctx context.Context, evt event.InboundEvent, host contract.Host) error {
	tenant := Tenant(evt, a.DefaultTenant)

	var act sdk.Action
	if err := json.Unmarshal(evt.Payload, &act); err == nil && act.Kind == "complete_task" {
		if act.TenantID == "" {
			act.TenantID = tenant
		}
		return sdk.ApplyTaskAction(ctx, host, act)
	}

	var body approvalCallback
	if err := sdk.ParseJSON(evt.Payload, &body); err != nil {
		return err
	}
	outcome := approvalAction(body.ApprovalInfo.SpStatus, body.Action)
	if outcome == "" {
		return nil
	}
	assignee := body.UserID
	if assignee == "" && len(body.ApprovalInfo.Comments) > 0 {
		assignee = body.ApprovalInfo.Comments[0].CommentUserInfo.UserID
	}
	comment := ""
	if len(body.ApprovalInfo.Comments) > 0 {
		comment = body.ApprovalInfo.Comments[0].CommentContent
	}
	spNo := body.ApprovalInfo.SpNo
	if spNo != "" && assignee != "" {
		return sdk.CompleteTaskByBusinessKey(ctx, host, tenant, assignee, spNo, outcome, comment)
	}
	return nil
}
