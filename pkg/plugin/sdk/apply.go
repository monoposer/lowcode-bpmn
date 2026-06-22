package sdk

import (
	"context"
	"fmt"

	"github.com/monoposer/lowcode-bpmn/pkg/plugin/contract"
)

func ApplyAssigneeAction(ctx context.Context, host contract.Host, action Action) error {
	switch action.Kind {
	case "remove_user":
		if action.UserID == "" {
			return nil
		}
		return host.RemoveUserFromActiveTasks(ctx, contract.RemoveUserRequest{
			TenantID: action.TenantID,
			UserID:   action.UserID,
			Reason:   action.Reason,
			Operator: action.Operator,
		})
	case "replace_assignees":
		instID, err := action.InstanceID()
		if err != nil {
			return fmt.Errorf("replace_assignees: invalid process_instance_id")
		}
		actID, err := action.ActivityUUID()
		if err != nil {
			return fmt.Errorf("replace_assignees: invalid activity_id")
		}
		return host.ReplaceTaskAssignees(ctx, contract.ReplaceAssigneesRequest{
			TenantID:          action.TenantID,
			ProcessInstanceID: instID,
			ActivityID:        actID,
			Assignees:         action.Assignees,
			Operator:          action.Operator,
			Reason:            action.Reason,
		})
	default:
		return nil
	}
}

func ApplyTriggerAction(ctx context.Context, host contract.Host, action Action) error {
	switch action.Kind {
	case "trigger_message":
		if action.MessageRef == "" {
			return nil
		}
		return host.TriggerMessage(ctx, contract.TriggerMessageRequest{
			TenantID:    action.TenantID,
			MessageRef:  action.MessageRef,
			BusinessKey: action.BusinessKey,
			Variables:   action.Variables,
		})
	case "start_process":
		if action.ProcessKey == "" {
			return nil
		}
		return host.StartProcess(ctx, contract.StartProcessRequest{
			TenantID:    action.TenantID,
			ProcessKey:  action.ProcessKey,
			BusinessKey: action.BusinessKey,
			Variables:   action.Variables,
		})
	case "trigger_boundary":
		instID, err := action.InstanceID()
		if err != nil {
			return fmt.Errorf("trigger_boundary: invalid process_instance_id")
		}
		if action.BoundaryElementID == "" {
			return nil
		}
		return host.TriggerBoundary(ctx, contract.TriggerBoundaryRequest{
			TenantID:          action.TenantID,
			ProcessInstanceID: instID,
			HostElementID:     action.HostElementID,
			BoundaryElementID: action.BoundaryElementID,
			Variables:         action.Variables,
		})
	default:
		return nil
	}
}

func ApplyTaskAction(ctx context.Context, host contract.Host, action Action) error {
	switch action.Kind {
	case "complete_task":
		instID, err := action.InstanceID()
		if err != nil {
			return fmt.Errorf("complete_task: invalid process_instance_id")
		}
		actID, err := action.ActivityUUID()
		if err != nil {
			return fmt.Errorf("complete_task: invalid activity_id")
		}
		if action.Action == "" {
			action.Action = "approve"
		}
		return host.CompleteTask(ctx, contract.CompleteTaskRequest{
			ProcessInstanceID: instID,
			ActivityID:        actID,
			Assignee:          action.Assignee,
			Action:            action.Action,
			Comment:           action.Comment,
			Variables:         action.Variables,
			LockVersion:       action.LockVersion,
		})
	case "complete_activity":
		instID, err := action.InstanceID()
		if err != nil {
			return fmt.Errorf("complete_activity: invalid process_instance_id")
		}
		actID, err := action.ActivityUUID()
		if err != nil {
			return fmt.Errorf("complete_activity: invalid activity_id")
		}
		return host.CompleteActivity(ctx, contract.CompleteActivityRequest{
			ProcessInstanceID: instID,
			ActivityID:        actID,
			Assignee:          action.Assignee,
			Action:            action.Action,
			Comment:           action.Comment,
			Variables:         action.Variables,
			LockVersion:       action.LockVersion,
			SelectedFlowID:    action.SelectedFlowID,
		})
	default:
		return nil
	}
}

func ApplyControlAction(ctx context.Context, host contract.Host, action Action) error {
	switch action.Kind {
	case "terminate":
		instID, err := action.InstanceID()
		if err != nil {
			return fmt.Errorf("terminate: invalid process_instance_id")
		}
		return host.Terminate(ctx, contract.TerminateRequest{
			ProcessInstanceID: instID,
			ScopeID:           action.ScopeID,
			Reason:            action.Reason,
			Operator:          action.Operator,
			LockVersion:       action.LockVersion,
		})
	default:
		return nil
	}
}

func CompleteTaskByBusinessKey(ctx context.Context, host contract.Host, tenant, assignee, businessKey, approvalAction, comment string) error {
	if assignee == "" || businessKey == "" {
		return nil
	}
	tasks, err := host.ListUserTasks(ctx, tenant, assignee)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if t.BusinessKey != businessKey {
			continue
		}
		return host.CompleteTask(ctx, contract.CompleteTaskRequest{
			ProcessInstanceID: t.ProcessInstanceID,
			ActivityID:        t.ID,
			Assignee:          assignee,
			Action:            approvalAction,
			Comment:           comment,
		})
	}
	return nil
}
