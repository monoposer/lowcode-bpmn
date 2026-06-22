package definition

import "fmt"

// ValidateTaskElement checks BPMN task-type-specific rules.
func ValidateTaskElement(el Element) error {
	switch el.Kind {
	case KindScriptTask:
		if el.Script == "" {
			return fmt.Errorf("scriptTask %s requires script", el.ID)
		}
	case KindServiceTask:
		if el.ServiceURL == "" && el.Implementation == "" && el.TaskType == "" {
			return fmt.Errorf("serviceTask %s requires serviceUrl, implementation, or taskType", el.ID)
		}
	case KindSendTask:
		if el.MessageRef == "" && el.TaskType == "" {
			return fmt.Errorf("sendTask %s requires messageRef or taskType", el.ID)
		}
	case KindReceiveTask:
		if el.MessageRef == "" && !el.AutoComplete {
			return fmt.Errorf("receiveTask %s requires messageRef or autoComplete", el.ID)
		}
	case KindBusinessRuleTask:
		if el.DecisionRef == "" && el.Script == "" && el.TaskType == "" {
			return fmt.Errorf("businessRuleTask %s requires decisionRef, script, or taskType", el.ID)
		}
	case KindUserTask:
		return ValidateUserTaskApproval(el)
	}
	return nil
}

// IsAutomatedTask reports whether the element runs without human assignees.
func IsAutomatedTask(k ElementKind) bool {
	switch k {
	case KindScriptTask, KindServiceTask, KindSendTask, KindReceiveTask, KindBusinessRuleTask:
		return true
	default:
		return false
	}
}
