package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"lowcode-automation/internal/engine"
)

// Store is a Postgres implementation of engine.Store.
type Store struct {
	db *sql.DB
}

// NewStore constructs a new Store and ensures schema exists.
func NewStore(ctx context.Context, db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.initSchema(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) initSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS flows (
			id UUID PRIMARY KEY,
			workspace_id UUID NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL,
			definition JSONB,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS runs (
			id UUID PRIMARY KEY,
			flow_id UUID NOT NULL,
			workspace_id UUID NOT NULL,
			status TEXT NOT NULL,
			trigger_type TEXT NOT NULL,
			input JSONB,
			error_message TEXT,
			started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS node_executions (
			id UUID PRIMARY KEY,
			run_id UUID NOT NULL,
			node_id TEXT NOT NULL,
			status TEXT NOT NULL,
			input JSONB,
			output JSONB,
			error_message TEXT,
			retry_count INT NOT NULL,
			started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ
		);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateFlow(ctx context.Context, f *engine.Flow) error {
	now := time.Now().UTC()
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
	f.UpdatedAt = now

	var defJSON []byte
	var err error
	if f.Definition != nil {
		defJSON, err = json.Marshal(f.Definition)
		if err != nil {
			return err
		}
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO flows (id, workspace_id, name, description, status, definition, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		f.ID, f.WorkspaceID, f.Name, f.Description, string(f.Status), defJSON, f.CreatedAt, f.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateFlow(ctx context.Context, f *engine.Flow) error {
	f.UpdatedAt = time.Now().UTC()

	var defJSON []byte
	var err error
	if f.Definition != nil {
		defJSON, err = json.Marshal(f.Definition)
		if err != nil {
			return err
		}
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE flows
		 SET workspace_id=$2, name=$3, description=$4, status=$5, definition=$6, updated_at=$7
		 WHERE id=$1`,
		f.ID, f.WorkspaceID, f.Name, f.Description, string(f.Status), defJSON, f.UpdatedAt,
	)
	return err
}

func (s *Store) GetFlow(ctx context.Context, id uuid.UUID) (*engine.Flow, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, workspace_id, name, description, status, definition, created_at, updated_at
		 FROM flows WHERE id=$1`, id)

	var f engine.Flow
	var status string
	var defBytes []byte
	if err := row.Scan(&f.ID, &f.WorkspaceID, &f.Name, &f.Description, &status, &defBytes, &f.CreatedAt, &f.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	f.Status = engine.FlowStatus(status)
	if len(defBytes) > 0 {
		var def engine.FlowDefinition
		if err := json.Unmarshal(defBytes, &def); err != nil {
			return nil, err
		}
		f.Definition = &def
	}
	return &f, nil
}

func (s *Store) ListFlows(ctx context.Context, workspaceID uuid.UUID) ([]*engine.Flow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, workspace_id, name, description, status, definition, created_at, updated_at
		 FROM flows WHERE workspace_id=$1 ORDER BY created_at ASC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*engine.Flow
	for rows.Next() {
		var f engine.Flow
		var status string
		var defBytes []byte
		if err := rows.Scan(&f.ID, &f.WorkspaceID, &f.Name, &f.Description, &status, &defBytes, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		f.Status = engine.FlowStatus(status)
		if len(defBytes) > 0 {
			var def engine.FlowDefinition
			if err := json.Unmarshal(defBytes, &def); err != nil {
				return nil, err
			}
			f.Definition = &def
		}
		res = append(res, &f)
	}
	return res, rows.Err()
}

func (s *Store) CreateRun(ctx context.Context, r *engine.Run) error {
	now := time.Now().UTC()
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}

	inputBytes, err := json.Marshal(r.Input)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO runs (id, flow_id, workspace_id, status, trigger_type, input, error_message, started_at, finished_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		r.ID, r.FlowID, r.WorkspaceID, string(r.Status), string(r.TriggerType), inputBytes,
		r.ErrorMessage, r.StartedAt, r.FinishedAt, r.CreatedAt,
	)
	return err
}

func (s *Store) GetRun(ctx context.Context, id uuid.UUID) (*engine.Run, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, flow_id, workspace_id, status, trigger_type, input, error_message, started_at, finished_at, created_at
		 FROM runs WHERE id=$1`, id)

	var r engine.Run
	var status, trig string
	var inputBytes []byte
	if err := row.Scan(&r.ID, &r.FlowID, &r.WorkspaceID, &status, &trig, &inputBytes,
		&r.ErrorMessage, &r.StartedAt, &r.FinishedAt, &r.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	r.Status = engine.RunStatus(status)
	r.TriggerType = engine.TriggerType(trig)
	if len(inputBytes) > 0 {
		if err := json.Unmarshal(inputBytes, &r.Input); err != nil {
			return nil, err
		}
	}
	return &r, nil
}

func (s *Store) ListRunsByFlow(ctx context.Context, flowID uuid.UUID) ([]*engine.Run, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, flow_id, workspace_id, status, trigger_type, input, error_message, started_at, finished_at, created_at
		 FROM runs WHERE flow_id=$1 ORDER BY created_at DESC`, flowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*engine.Run
	for rows.Next() {
		var r engine.Run
		var status, trig string
		var inputBytes []byte
		if err := rows.Scan(&r.ID, &r.FlowID, &r.WorkspaceID, &status, &trig, &inputBytes,
			&r.ErrorMessage, &r.StartedAt, &r.FinishedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Status = engine.RunStatus(status)
		r.TriggerType = engine.TriggerType(trig)
		if len(inputBytes) > 0 {
			if err := json.Unmarshal(inputBytes, &r.Input); err != nil {
				return nil, err
			}
		}
		res = append(res, &r)
	}
	return res, rows.Err()
}

func (s *Store) CreateNodeExecution(ctx context.Context, ne *engine.NodeExecution) error {
	if ne.ID == uuid.Nil {
		ne.ID = uuid.New()
	}

	inputBytes, err := json.Marshal(ne.Input)
	if err != nil {
		return err
	}
	outputBytes, err := json.Marshal(ne.Output)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO node_executions (id, run_id, node_id, status, input, output, error_message, retry_count, started_at, finished_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		ne.ID, ne.RunID, ne.NodeID, string(ne.Status), inputBytes, outputBytes,
		ne.ErrorMessage, ne.RetryCount, ne.StartedAt, ne.FinishedAt,
	)
	return err
}

func (s *Store) ListNodeExecutionsByRun(ctx context.Context, runID uuid.UUID) ([]*engine.NodeExecution, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, run_id, node_id, status, input, output, error_message, retry_count, started_at, finished_at
		 FROM node_executions WHERE run_id=$1 ORDER BY started_at ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*engine.NodeExecution
	for rows.Next() {
		var ne engine.NodeExecution
		var status string
		var inBytes, outBytes []byte
		if err := rows.Scan(&ne.ID, &ne.RunID, &ne.NodeID, &status, &inBytes, &outBytes,
			&ne.ErrorMessage, &ne.RetryCount, &ne.StartedAt, &ne.FinishedAt); err != nil {
			return nil, err
		}
		ne.Status = engine.RunStatus(status)
		if len(inBytes) > 0 {
			if err := json.Unmarshal(inBytes, &ne.Input); err != nil {
				return nil, err
			}
		}
		if len(outBytes) > 0 {
			if err := json.Unmarshal(outBytes, &ne.Output); err != nil {
				return nil, err
			}
		}
		res = append(res, &ne)
	}
	return res, rows.Err()
}

var _ engine.Store = (*Store)(nil)

