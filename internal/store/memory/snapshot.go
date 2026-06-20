package memory

import (
	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

// Snapshot is a serializable copy of in-memory store state.
type Snapshot struct {
	Processes         []*engine.DeployedProcess  `yaml:"processes,omitempty"`
	ProcessInstances  []*engine.ProcessInstance  `yaml:"process_instances,omitempty"`
	ActivityInstances []*engine.ActivityInstance `yaml:"activity_instances,omitempty"`
	Jobs              []*engine.Job              `yaml:"jobs,omitempty"`
}

// Snapshot exports the current store state.
func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap := Snapshot{}
	for _, p := range s.processes {
		copy := *p
		snap.Processes = append(snap.Processes, &copy)
	}
	for _, inst := range s.processInstances {
		copy := *inst
		snap.ProcessInstances = append(snap.ProcessInstances, &copy)
	}
	for _, act := range s.activityInstances {
		copy := *act
		snap.ActivityInstances = append(snap.ActivityInstances, &copy)
	}
	for _, job := range s.jobs {
		copy := *job
		snap.Jobs = append(snap.Jobs, &copy)
	}
	return snap
}

// Restore replaces store state from a snapshot.
func (s *Store) Restore(snap Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.processes = make(map[string]*engine.DeployedProcess, len(snap.Processes))
	s.processInstances = make(map[uuid.UUID]*engine.ProcessInstance, len(snap.ProcessInstances))
	s.activityInstances = make(map[uuid.UUID]*engine.ActivityInstance, len(snap.ActivityInstances))
	s.jobs = make(map[uuid.UUID]*engine.Job, len(snap.Jobs))

	for _, p := range snap.Processes {
		if p == nil {
			continue
		}
		copy := *p
		s.processes[versionedProcessKey(p.TenantID, p.Key, p.Version)] = &copy
	}
	for _, inst := range snap.ProcessInstances {
		if inst == nil {
			continue
		}
		copy := *inst
		s.processInstances[inst.ID] = &copy
	}
	for _, act := range snap.ActivityInstances {
		if act == nil {
			continue
		}
		copy := *act
		s.activityInstances[act.ID] = &copy
	}
	for _, job := range snap.Jobs {
		if job == nil {
			continue
		}
		copy := *job
		s.jobs[job.ID] = &copy
	}
}
