package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"lowcode-automation/internal/engine"
)

type Store struct {
	mu       sync.RWMutex
	flows    map[uuid.UUID]*engine.Flow
	runs     map[uuid.UUID]*engine.Run
	nodeExec map[uuid.UUID]*engine.NodeExecution
}

func NewStore() *Store {
	return &Store{
		flows:   make(map[uuid.UUID]*engine.Flow),
		runs:    make(map[uuid.UUID]*engine.Run),
		nodeExec: make(map[uuid.UUID]*engine.NodeExecution),
	}
}

func (s *Store) CreateFlow(ctx context.Context, f *engine.Flow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
	f.UpdatedAt = now

	s.flows[f.ID] = f
	return nil
}

func (s *Store) UpdateFlow(ctx context.Context, f *engine.Flow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.flows[f.ID]; !ok {
		return nil
	}

	f.UpdatedAt = time.Now().UTC()
	s.flows[f.ID] = f
	return nil
}

func (s *Store) GetFlow(ctx context.Context, id uuid.UUID) (*engine.Flow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if f, ok := s.flows[id]; ok {
		copy := *f
		return &copy, nil
	}
	return nil, nil
}

func (s *Store) ListFlows(ctx context.Context, workspaceID uuid.UUID) ([]*engine.Flow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []*engine.Flow
	for _, f := range s.flows {
		if f.WorkspaceID == workspaceID {
			copy := *f
			res = append(res, &copy)
		}
	}
	return res, nil
}

func (s *Store) CreateRun(ctx context.Context, r *engine.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}

	s.runs[r.ID] = r
	return nil
}

func (s *Store) GetRun(ctx context.Context, id uuid.UUID) (*engine.Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if run, ok := s.runs[id]; ok {
		copy := *run
		return &copy, nil
	}
	return nil, nil
}

func (s *Store) ListRunsByFlow(ctx context.Context, flowID uuid.UUID) ([]*engine.Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []*engine.Run
	for _, run := range s.runs {
		if run.FlowID == flowID {
			copy := *run
			res = append(res, &copy)
		}
	}
	return res, nil
}

func (s *Store) CreateNodeExecution(ctx context.Context, ne *engine.NodeExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ne.ID == uuid.Nil {
		ne.ID = uuid.New()
	}
	s.nodeExec[ne.ID] = ne
	return nil
}

func (s *Store) ListNodeExecutionsByRun(ctx context.Context, runID uuid.UUID) ([]*engine.NodeExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []*engine.NodeExecution
	for _, ne := range s.nodeExec {
		if ne.RunID == runID {
			copy := *ne
			res = append(res, &copy)
		}
	}
	return res, nil
}

var _ engine.Store = (*Store)(nil)

