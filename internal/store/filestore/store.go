package filestore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/monoposer/lowcode-bpmn/internal/engine"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

const defaultStateFile = "state.yaml"

// Store persists BPMN state as YAML files on disk.
type Store struct {
	mu   sync.Mutex
	dir  string
	file string
	mem  *memstore.Store
}

// Open creates or loads a file-backed store from dir.
func Open(dir string) (*Store, error) {
	if dir == "" {
		dir = "./data"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	s := &Store{
		dir:  dir,
		file: filepath.Join(dir, defaultStateFile),
		mem:  memstore.NewStore(),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read store file: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	var snap memstore.Snapshot
	if err := yaml.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("decode store file: %w", err)
	}
	s.mem.Restore(snap)
	if err := s.loadProcessDefinitionsFromXML(); err != nil {
		return err
	}
	return nil
}

func (s *Store) persistLocked() error {
	data, err := yaml.Marshal(s.mem.Snapshot())
	if err != nil {
		return fmt.Errorf("encode store file: %w", err)
	}

	tmp := s.file + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write store temp file: %w", err)
	}
	if err := os.Rename(tmp, s.file); err != nil {
		return fmt.Errorf("replace store file: %w", err)
	}
	return nil
}

// Ping verifies the store directory is accessible.
func (s *Store) Ping(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	info, err := os.Stat(s.dir)
	if err != nil {
		return fmt.Errorf("stat store dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("store path is not a directory: %s", s.dir)
	}
	return nil
}

// Path returns the directory containing persisted YAML state.
func (s *Store) Path() string { return s.dir }

func (s *Store) WithTx(ctx context.Context, fn func(engine.Store) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&txStore{mem: s.mem}); err != nil {
		return err
	}
	return s.persistLocked()
}

func (s *Store) InsertProcessVersion(ctx context.Context, p *engine.DeployedProcess) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.mem.InsertProcessVersion(ctx, p); err != nil {
		return err
	}
	if err := s.writeProcessXML(p); err != nil {
		return err
	}
	return s.persistLocked()
}

func (s *Store) DeleteProcess(ctx context.Context, tenantID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.mem.DeleteProcess(ctx, tenantID, key); err != nil {
		return err
	}
	if err := s.deleteProcessXML(tenantID, key); err != nil {
		return err
	}
	return s.persistLocked()
}

func (s *Store) GetProcess(ctx context.Context, tenantID, key string) (*engine.DeployedProcess, error) {
	return s.mem.GetProcess(ctx, tenantID, key)
}

func (s *Store) GetProcessVersion(ctx context.Context, tenantID, key string, version int) (*engine.DeployedProcess, error) {
	return s.mem.GetProcessVersion(ctx, tenantID, key, version)
}

func (s *Store) ListProcesses(ctx context.Context, tenantID string) ([]*engine.DeployedProcess, error) {
	return s.mem.ListProcesses(ctx, tenantID)
}

func (s *Store) CreateProcessInstance(ctx context.Context, inst *engine.ProcessInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.mem.CreateProcessInstance(ctx, inst); err != nil {
		return err
	}
	return s.persistLocked()
}

func (s *Store) UpdateProcessInstance(ctx context.Context, inst *engine.ProcessInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.mem.UpdateProcessInstance(ctx, inst); err != nil {
		return err
	}
	return s.persistLocked()
}

func (s *Store) GetProcessInstance(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error) {
	return s.mem.GetProcessInstance(ctx, id)
}

func (s *Store) GetProcessInstanceForUpdate(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error) {
	return s.mem.GetProcessInstanceForUpdate(ctx, id)
}

func (s *Store) FindRunningInstanceByBusinessKey(ctx context.Context, tenantID, processKey, businessKey string) (*engine.ProcessInstance, error) {
	return s.mem.FindRunningInstanceByBusinessKey(ctx, tenantID, processKey, businessKey)
}

func (s *Store) ListRunningInstances(ctx context.Context, tenantID string) ([]*engine.ProcessInstance, error) {
	return s.mem.ListRunningInstances(ctx, tenantID)
}

func (s *Store) CreateActivityInstance(ctx context.Context, act *engine.ActivityInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.mem.CreateActivityInstance(ctx, act); err != nil {
		return err
	}
	return s.persistLocked()
}

func (s *Store) UpdateActivityInstance(ctx context.Context, act *engine.ActivityInstance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.mem.UpdateActivityInstance(ctx, act); err != nil {
		return err
	}
	return s.persistLocked()
}

func (s *Store) GetActivityInstance(ctx context.Context, id uuid.UUID) (*engine.ActivityInstance, error) {
	return s.mem.GetActivityInstance(ctx, id)
}

func (s *Store) ListActivitiesByProcess(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error) {
	return s.mem.ListActivitiesByProcess(ctx, processInstanceID)
}

func (s *Store) ListActiveActivities(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error) {
	return s.mem.ListActiveActivities(ctx, processInstanceID)
}

func (s *Store) ListActiveUserTasks(ctx context.Context, tenantID, assignee string) ([]*engine.UserTask, error) {
	return s.mem.ListActiveUserTasks(ctx, tenantID, assignee)
}

func (s *Store) EnqueueJob(ctx context.Context, job *engine.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.mem.EnqueueJob(ctx, job); err != nil {
		return err
	}
	return s.persistLocked()
}

func (s *Store) ClaimNextJob(ctx context.Context) (*engine.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, err := s.mem.ClaimNextJob(ctx)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, nil
	}
	if err := s.persistLocked(); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Store) CompleteJob(ctx context.Context, jobID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.mem.CompleteJob(ctx, jobID); err != nil {
		return err
	}
	return s.persistLocked()
}

func (s *Store) FailJob(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.mem.FailJob(ctx, jobID, errMsg); err != nil {
		return err
	}
	return s.persistLocked()
}

type txStore struct {
	mem *memstore.Store
}

func (t *txStore) WithTx(ctx context.Context, fn func(engine.Store) error) error {
	return fn(t)
}

func (t *txStore) InsertProcessVersion(ctx context.Context, p *engine.DeployedProcess) error {
	return t.mem.InsertProcessVersion(ctx, p)
}

func (t *txStore) DeleteProcess(ctx context.Context, tenantID, key string) error {
	return t.mem.DeleteProcess(ctx, tenantID, key)
}

func (t *txStore) GetProcess(ctx context.Context, tenantID, key string) (*engine.DeployedProcess, error) {
	return t.mem.GetProcess(ctx, tenantID, key)
}

func (t *txStore) GetProcessVersion(ctx context.Context, tenantID, key string, version int) (*engine.DeployedProcess, error) {
	return t.mem.GetProcessVersion(ctx, tenantID, key, version)
}

func (t *txStore) ListProcesses(ctx context.Context, tenantID string) ([]*engine.DeployedProcess, error) {
	return t.mem.ListProcesses(ctx, tenantID)
}

func (t *txStore) CreateProcessInstance(ctx context.Context, inst *engine.ProcessInstance) error {
	return t.mem.CreateProcessInstance(ctx, inst)
}

func (t *txStore) UpdateProcessInstance(ctx context.Context, inst *engine.ProcessInstance) error {
	return t.mem.UpdateProcessInstance(ctx, inst)
}

func (t *txStore) GetProcessInstance(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error) {
	return t.mem.GetProcessInstance(ctx, id)
}

func (t *txStore) GetProcessInstanceForUpdate(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error) {
	return t.mem.GetProcessInstanceForUpdate(ctx, id)
}

func (t *txStore) FindRunningInstanceByBusinessKey(ctx context.Context, tenantID, processKey, businessKey string) (*engine.ProcessInstance, error) {
	return t.mem.FindRunningInstanceByBusinessKey(ctx, tenantID, processKey, businessKey)
}

func (t *txStore) ListRunningInstances(ctx context.Context, tenantID string) ([]*engine.ProcessInstance, error) {
	return t.mem.ListRunningInstances(ctx, tenantID)
}

func (t *txStore) CreateActivityInstance(ctx context.Context, act *engine.ActivityInstance) error {
	return t.mem.CreateActivityInstance(ctx, act)
}

func (t *txStore) UpdateActivityInstance(ctx context.Context, act *engine.ActivityInstance) error {
	return t.mem.UpdateActivityInstance(ctx, act)
}

func (t *txStore) GetActivityInstance(ctx context.Context, id uuid.UUID) (*engine.ActivityInstance, error) {
	return t.mem.GetActivityInstance(ctx, id)
}

func (t *txStore) ListActivitiesByProcess(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error) {
	return t.mem.ListActivitiesByProcess(ctx, processInstanceID)
}

func (t *txStore) ListActiveActivities(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error) {
	return t.mem.ListActiveActivities(ctx, processInstanceID)
}

func (t *txStore) ListActiveUserTasks(ctx context.Context, tenantID, assignee string) ([]*engine.UserTask, error) {
	return t.mem.ListActiveUserTasks(ctx, tenantID, assignee)
}

func (t *txStore) EnqueueJob(ctx context.Context, job *engine.Job) error {
	return t.mem.EnqueueJob(ctx, job)
}

func (t *txStore) ClaimNextJob(ctx context.Context) (*engine.Job, error) {
	return t.mem.ClaimNextJob(ctx)
}

func (t *txStore) CompleteJob(ctx context.Context, jobID uuid.UUID) error {
	return t.mem.CompleteJob(ctx, jobID)
}

func (t *txStore) FailJob(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	return t.mem.FailJob(ctx, jobID, errMsg)
}

var _ engine.Store = (*Store)(nil)
