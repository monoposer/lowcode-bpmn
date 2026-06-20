package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

type Store struct {
	mu                sync.RWMutex
	processes         map[string]*engine.DeployedProcess // tenant/key/vN
	processInstances  map[uuid.UUID]*engine.ProcessInstance
	activityInstances map[uuid.UUID]*engine.ActivityInstance
	jobs              map[uuid.UUID]*engine.Job
}

func NewStore() *Store {
	return &Store{
		processes:         make(map[string]*engine.DeployedProcess),
		processInstances:  make(map[uuid.UUID]*engine.ProcessInstance),
		activityInstances: make(map[uuid.UUID]*engine.ActivityInstance),
		jobs:              make(map[uuid.UUID]*engine.Job),
	}
}

func (s *Store) WithTx(ctx context.Context, fn func(engine.Store) error) error {
	return fn(s)
}

func versionedProcessKey(tenantID, key string, version int) string {
	return tenantID + "/" + key + "/v" + itoa(version)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func (s *Store) InsertProcessVersion(ctx context.Context, p *engine.DeployedProcess) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if p.Version == 0 {
		p.Version = s.latestVersionLocked(p.TenantID, p.Key) + 1
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	copy := *p
	s.processes[versionedProcessKey(p.TenantID, p.Key, p.Version)] = &copy
	return nil
}

func (s *Store) latestVersionLocked(tenantID, key string) int {
	prefix := tenantID + "/" + key + "/v"
	max := 0
	for k, p := range s.processes {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix && p.Version > max {
			max = p.Version
		}
	}
	return max
}

func (s *Store) DeleteProcess(ctx context.Context, tenantID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := tenantID + "/" + key + "/v"
	for k := range s.processes {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(s.processes, k)
		}
	}
	return nil
}

func (s *Store) GetProcess(ctx context.Context, tenantID, key string) (*engine.DeployedProcess, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getLatestProcessLocked(tenantID, key), nil
}

func (s *Store) GetProcessVersion(ctx context.Context, tenantID, key string, version int) (*engine.DeployedProcess, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if p, ok := s.processes[versionedProcessKey(tenantID, key, version)]; ok {
		copy := *p
		return &copy, nil
	}
	return nil, nil
}

func (s *Store) getLatestProcessLocked(tenantID, key string) *engine.DeployedProcess {
	prefix := tenantID + "/" + key + "/v"
	var latest *engine.DeployedProcess
	for k, p := range s.processes {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			if latest == nil || p.Version > latest.Version {
				copy := *p
				latest = &copy
			}
		}
	}
	return latest
}

func (s *Store) ListProcesses(ctx context.Context, tenantID string) ([]*engine.DeployedProcess, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	latestByKey := make(map[string]*engine.DeployedProcess)
	prefix := tenantID + "/"
	for k, p := range s.processes {
		if len(k) < len(prefix) || k[:len(prefix)] != prefix {
			continue
		}
		if cur, ok := latestByKey[p.Key]; !ok || p.Version > cur.Version {
			copy := *p
			latestByKey[p.Key] = &copy
		}
	}
	res := make([]*engine.DeployedProcess, 0, len(latestByKey))
	for _, p := range latestByKey {
		res = append(res, p)
	}
	return res, nil
}

func (s *Store) CreateProcessInstance(ctx context.Context, inst *engine.ProcessInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inst.ID == uuid.Nil {
		inst.ID = uuid.New()
	}
	now := time.Now().UTC()
	if inst.StartedAt.IsZero() {
		inst.StartedAt = now
	}
	inst.UpdatedAt = now
	copy := *inst
	s.processInstances[inst.ID] = &copy
	return nil
}

func (s *Store) UpdateProcessInstance(ctx context.Context, inst *engine.ProcessInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.processInstances[inst.ID]
	if !ok {
		return engine.ErrVersionConflict
	}
	if existing.LockVersion != inst.LockVersion {
		return engine.ErrVersionConflict
	}
	inst.UpdatedAt = time.Now().UTC()
	inst.LockVersion++
	copy := *inst
	s.processInstances[inst.ID] = &copy
	return nil
}

func (s *Store) GetProcessInstance(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if inst, ok := s.processInstances[id]; ok {
		copy := *inst
		return &copy, nil
	}
	return nil, nil
}

func (s *Store) GetProcessInstanceForUpdate(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error) {
	return s.GetProcessInstance(ctx, id)
}

func (s *Store) FindRunningInstanceByBusinessKey(ctx context.Context, tenantID, processKey, businessKey string) (*engine.ProcessInstance, error) {
	if businessKey == "" {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, inst := range s.processInstances {
		if inst.TenantID == tenantID && inst.ProcessKey == processKey &&
			inst.BusinessKey == businessKey && inst.Status == engine.ProcessStatusRunning {
			copy := *inst
			return &copy, nil
		}
	}
	return nil, nil
}

func (s *Store) CreateActivityInstance(ctx context.Context, act *engine.ActivityInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if act.ID == uuid.Nil {
		act.ID = uuid.New()
	}
	if act.StartedAt.IsZero() {
		act.StartedAt = time.Now().UTC()
	}
	copy := *act
	s.activityInstances[act.ID] = &copy
	return nil
}

func (s *Store) UpdateActivityInstance(ctx context.Context, act *engine.ActivityInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := *act
	s.activityInstances[act.ID] = &copy
	return nil
}

func (s *Store) GetActivityInstance(ctx context.Context, id uuid.UUID) (*engine.ActivityInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if act, ok := s.activityInstances[id]; ok {
		copy := *act
		return &copy, nil
	}
	return nil, nil
}

func (s *Store) ListActivitiesByProcess(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []*engine.ActivityInstance
	for _, act := range s.activityInstances {
		if act.ProcessInstanceID == processInstanceID {
			copy := *act
			res = append(res, &copy)
		}
	}
	return res, nil
}

func (s *Store) ListActiveActivities(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []*engine.ActivityInstance
	for _, act := range s.activityInstances {
		if act.ProcessInstanceID == processInstanceID && act.Status == engine.ActivityStatusActive {
			copy := *act
			res = append(res, &copy)
		}
	}
	return res, nil
}

func (s *Store) ListActiveUserTasks(ctx context.Context, tenantID, assignee string) ([]*engine.UserTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []*engine.UserTask
	for _, act := range s.activityInstances {
		if act.Status != engine.ActivityStatusActive || act.ElementKind != bpmn.KindUserTask {
			continue
		}
		inst, ok := s.processInstances[act.ProcessInstanceID]
		if !ok || inst.TenantID != tenantID {
			continue
		}
		if assignee != "" && !engine.TaskVisibleToAssignee(act, assignee) {
			continue
		}
		res = append(res, &engine.UserTask{
			ActivityInstance: *act,
			TenantID:         inst.TenantID,
			ProcessKey:       inst.ProcessKey,
			BusinessKey:      inst.BusinessKey,
			ProcessVersion:   inst.ProcessVersion,
		})
	}
	return res, nil
}

func (s *Store) EnqueueJob(ctx context.Context, job *engine.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	job.Status = engine.JobStatusPending
	copy := *job
	s.jobs[job.ID] = &copy
	return nil
}

func (s *Store) ClaimNextJob(ctx context.Context) (*engine.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var pending []*engine.Job
	for _, j := range s.jobs {
		if j.Status == engine.JobStatusPending {
			copy := *j
			pending = append(pending, &copy)
		}
	}
	if len(pending) == 0 {
		return nil, nil
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].CreatedAt.Before(pending[j].CreatedAt)
	})
	job := pending[0]
	now := time.Now().UTC()
	job.Status = engine.JobStatusRunning
	job.Attempts++
	job.LockedAt = &now
	s.jobs[job.ID] = job
	return job, nil
}

func (s *Store) CompleteJob(ctx context.Context, jobID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[jobID]; ok {
		now := time.Now().UTC()
		job.Status = engine.JobStatusDone
		job.CompletedAt = &now
	}
	return nil
}

func (s *Store) FailJob(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[jobID]; ok {
		job.Status = engine.JobStatusFailed
		job.ErrorMsg = errMsg
	}
	return nil
}

func containsAssignee(assignees []string, assignee string) bool {
	for _, a := range assignees {
		if a == assignee {
			return true
		}
	}
	return false
}

var _ engine.Store = (*Store)(nil)
