package wasm

// Capability declares which Host SDK functions a WASM plugin may call.
type Capability string

const (
	// Query extension point
	CapReadInstances Capability = "read_instances"
	CapReadTasks     Capability = "read_tasks"
	CapReadActivities Capability = "read_activities"

	// Start / trigger extension point
	CapTriggerMessage Capability = "trigger_message"
	CapStartProcess   Capability = "start_process"

	// Assignee extension point
	CapRemoveUser       Capability = "remove_user"
	CapReplaceAssignees Capability = "replace_assignees"

	// Task extension point (approve / reject)
	CapCompleteTask Capability = "complete_task"

	// Control extension point
	CapTerminate Capability = "terminate"
)

// Set is a capability permission set (Paca-style).
type Set map[Capability]struct{}

func ParseCapabilities(list []string) Set {
	s := make(Set, len(list))
	for _, c := range list {
		s[Capability(c)] = struct{}{}
	}
	return s
}

func (s Set) Has(c Capability) bool {
	if s == nil {
		return false
	}
	_, ok := s[c]
	return ok
}

var AllAssignee = Set{
	CapRemoveUser:       {},
	CapReplaceAssignees: {},
	CapReadTasks:        {},
}

var AllTrigger = Set{
	CapTriggerMessage: {},
	CapStartProcess:   {},
	CapReadInstances:  {},
	CapReadTasks:      {},
}

var AllTask = Set{
	CapCompleteTask:  {},
	CapReadInstances: {},
	CapReadTasks:     {},
	CapReadActivities: {},
}

var AllControl = Set{
	CapTerminate:     {},
	CapReadInstances: {},
}
