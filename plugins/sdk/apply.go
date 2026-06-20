package sdk

import (
	"context"
	"fmt"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
	"github.com/monoposer/lowcode-bpmn/internal/plugin/contract"
)

func ApplyAssigneeAction(ctx context.Context, host contract.Host, action Action) error {
	switch action.Kind {
	case "remove_user":
		if action.UserID == "" {
			return nil
		}
		_, err := host.RemoveUserFromActiveTasks(ctx, engine.RemoveUserSyncRequest{
			TenantID: action.TenantID,
			UserID:   action.UserID,
			Reason:   action.Reason,
			Operator: action.Operator,
		})
		return err
	case "replace_assignees":
		instID, err := action.InstanceID()
		if err != nil {
			return fmt.Errorf("replace_assignees: invalid process_instance_id")
		}
		actID, err := action.ActivityUUID()
		if err != nil {
			return fmt.Errorf("replace_assignees: invalid activity_id")
		}
		_, err = host.ReplaceTaskAssignees(ctx, engine.ReplaceAssigneesSyncRequest{
			TenantID:          action.TenantID,
			ProcessInstanceID: instID,
			ActivityID:        actID,
			Assignees:         action.Assignees,
			Operator:          action.Operator,
			Reason:            action.Reason,
		})
		return err
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
		_, err := host.TriggerMessage(ctx, engine.TriggerMessageRequest{
			TenantID:    action.TenantID,
			MessageRef:  action.MessageRef,
			BusinessKey: action.BusinessKey,
			Variables:   action.Variables,
		})
		return err
	case "start_process":
		if action.ProcessKey == "" {
			return nil
		}
		_, err := host.StartProcess(ctx, engine.StartProcessRequest{
			TenantID:    action.TenantID,
			ProcessKey:  action.ProcessKey,
			BusinessKey: action.BusinessKey,
			Variables:   action.Variables,
		})
		return err
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
		_, err = host.CompleteTask(ctx, engine.CompleteTaskRequest{
			ProcessInstanceID: instID,
			ActivityID:        actID,
			Assignee:          action.Assignee,
			Action:            action.Action,
			Comment:           action.Comment,
			Variables:         action.Variables,
			LockVersion:       action.LockVersion,
		})
		return err
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
		_, err = host.Terminate(ctx, engine.TerminateRequest{
			ProcessInstanceID: instID,
			ScopeID:           action.ScopeID,
			Reason:            action.Reason,
			Operator:          action.Operator,
			LockVersion:       action.LockVersion,
		})
		return err
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
		_, err = host.CompleteTask(ctx, engine.CompleteTaskRequest{
			ProcessInstanceID: t.ProcessInstanceID,
			ActivityID:        t.ID,
			Assignee:          assignee,
			Action:            approvalAction,
			Comment:           comment,
		})
		return err
	}
	return nil
}
