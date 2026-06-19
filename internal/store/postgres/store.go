package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"lowcode-bpmn/internal/bpmn"
	"lowcode-bpmn/internal/engine"
)

// Store is a Postgres implementation of engine.Store.
type Store struct {
	db *sql.DB
	tx *sql.Tx
}

// NewStore constructs a Store and applies pending migrations.
func NewStore(ctx context.Context, db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.runMigrations(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) InsertProcessVersion(ctx context.Context, p *engine.DeployedProcess) error {
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.Version == 0 {
		latest, err := s.GetProcess(ctx, p.TenantID, p.Key)
		if err != nil {
			return err
		}
		p.Version = 1
		if latest != nil {
			p.Version = latest.Version + 1
		}
	}
	defJSON, err := json.Marshal(p.Definition)
	if err != nil {
		return err
	}
	name := p.Name
	if name == "" {
		name = p.Key
	}
	_, err = s.execContext(ctx,
		`INSERT INTO bpmn_processes (tenant_id, process_key, version, name, definition, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		p.TenantID, p.Key, p.Version, name, defJSON, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (s *Store) DeleteProcess(ctx context.Context, tenantID, key string) error {
	_, err := s.execContext(ctx,
		`DELETE FROM bpmn_processes WHERE tenant_id=$1 AND process_key=$2`, tenantID, key)
	return err
}

func (s *Store) GetProcess(ctx context.Context, tenantID, key string) (*engine.DeployedProcess, error) {
	row := s.queryRowContext(ctx,
		`SELECT tenant_id, process_key, version, name, definition, created_at, updated_at
		 FROM bpmn_processes WHERE tenant_id=$1 AND process_key=$2
		 ORDER BY version DESC LIMIT 1`, tenantID, key)
	return scanDeployedProcess(row)
}

func (s *Store) GetProcessVersion(ctx context.Context, tenantID, key string, version int) (*engine.DeployedProcess, error) {
	row := s.queryRowContext(ctx,
		`SELECT tenant_id, process_key, version, name, definition, created_at, updated_at
		 FROM bpmn_processes WHERE tenant_id=$1 AND process_key=$2 AND version=$3`,
		tenantID, key, version)
	return scanDeployedProcess(row)
}

func (s *Store) ListProcesses(ctx context.Context, tenantID string) ([]*engine.DeployedProcess, error) {
	rows, err := s.queryContext(ctx,
		`SELECT DISTINCT ON (process_key) tenant_id, process_key, version, name, definition, created_at, updated_at
		 FROM bpmn_processes WHERE tenant_id=$1
		 ORDER BY process_key, version DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*engine.DeployedProcess
	for rows.Next() {
		p, err := scanDeployedProcess(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, p)
	}
	return res, rows.Err()
}

func (s *Store) CreateProcessInstance(ctx context.Context, inst *engine.ProcessInstance) error {
	if inst.ID == uuid.Nil {
		inst.ID = uuid.New()
	}
	now := time.Now().UTC()
	if inst.StartedAt.IsZero() {
		inst.StartedAt = now
	}
	inst.UpdatedAt = now
	varsJSON, err := json.Marshal(inst.Variables)
	if err != nil {
		return err
	}
	internalJSON, err := json.Marshal(inst.InternalState)
	if err != nil {
		return err
	}
	activeJSON, err := json.Marshal(inst.ActiveElements)
	if err != nil {
		return err
	}
	snapJSON, err := json.Marshal(inst.DefinitionSnapshot)
	if err != nil {
		return err
	}
	_, err = s.execContext(ctx,
		`INSERT INTO bpmn_instances (id, tenant_id, process_key, process_version, business_key, status, variables, internal_state, active_elements, definition_snapshot, lock_version, error_message, started_at, ended_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		inst.ID, inst.TenantID, inst.ProcessKey, inst.ProcessVersion, inst.BusinessKey, string(inst.Status),
		varsJSON, internalJSON, activeJSON, snapJSON, inst.LockVersion,
		inst.ErrorMsg, inst.StartedAt, inst.EndedAt, inst.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateProcessInstance(ctx context.Context, inst *engine.ProcessInstance) error {
	inst.UpdatedAt = time.Now().UTC()
	varsJSON, err := json.Marshal(inst.Variables)
	if err != nil {
		return err
	}
	internalJSON, err := json.Marshal(inst.InternalState)
	if err != nil {
		return err
	}
	activeJSON, err := json.Marshal(inst.ActiveElements)
	if err != nil {
		return err
	}
	res, err := s.execContext(ctx,
		`UPDATE bpmn_instances SET status=$2, variables=$3, internal_state=$4, active_elements=$5, error_message=$6, ended_at=$7, updated_at=$8, lock_version=lock_version+1
		 WHERE id=$1 AND lock_version=$9`,
		inst.ID, string(inst.Status), varsJSON, internalJSON, activeJSON, inst.ErrorMsg, inst.EndedAt, inst.UpdatedAt, inst.LockVersion,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return engine.ErrVersionConflict
	}
	inst.LockVersion++
	return nil
}

func (s *Store) GetProcessInstance(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error) {
	row := s.queryRowContext(ctx, instanceSelectSQL+` WHERE id=$1`, id)
	return scanProcessInstance(row)
}

func (s *Store) GetProcessInstanceForUpdate(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error) {
	row := s.queryRowContext(ctx, instanceSelectSQL+` WHERE id=$1 FOR UPDATE`, id)
	return scanProcessInstance(row)
}

const instanceSelectSQL = `
	SELECT id, tenant_id, process_key, process_version, business_key, status, variables, internal_state, active_elements, definition_snapshot, lock_version, error_message, started_at, ended_at, updated_at
	FROM bpmn_instances`

func (s *Store) CreateActivityInstance(ctx context.Context, act *engine.ActivityInstance) error {
	if act.ID == uuid.Nil {
		act.ID = uuid.New()
	}
	if act.StartedAt.IsZero() {
		act.StartedAt = time.Now().UTC()
	}
	assignJSON, err := json.Marshal(act.Assignees)
	if err != nil {
		return err
	}
	inJSON, err := json.Marshal(act.Input)
	if err != nil {
		return err
	}
	outJSON, err := json.Marshal(act.Output)
	if err != nil {
		return err
	}
	_, err = s.execContext(ctx,
		`INSERT INTO bpmn_activities (id, process_instance_id, element_id, element_kind, status, assignees, input, output, error_message, started_at, ended_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		act.ID, act.ProcessInstanceID, act.ElementID, string(act.ElementKind), string(act.Status),
		assignJSON, inJSON, outJSON, act.ErrorMsg, act.StartedAt, act.EndedAt,
	)
	return err
}

func (s *Store) UpdateActivityInstance(ctx context.Context, act *engine.ActivityInstance) error {
	assignJSON, err := json.Marshal(act.Assignees)
	if err != nil {
		return err
	}
	inJSON, err := json.Marshal(act.Input)
	if err != nil {
		return err
	}
	outJSON, err := json.Marshal(act.Output)
	if err != nil {
		return err
	}
	_, err = s.execContext(ctx,
		`UPDATE bpmn_activities SET status=$2, assignees=$3, input=$4, output=$5, error_message=$6, ended_at=$7 WHERE id=$1`,
		act.ID, string(act.Status), assignJSON, inJSON, outJSON, act.ErrorMsg, act.EndedAt,
	)
	return err
}

func (s *Store) GetActivityInstance(ctx context.Context, id uuid.UUID) (*engine.ActivityInstance, error) {
	row := s.queryRowContext(ctx,
		`SELECT id, process_instance_id, element_id, element_kind, status, assignees, input, output, error_message, started_at, ended_at
		 FROM bpmn_activities WHERE id=$1`, id)
	return scanActivityInstance(row)
}

func (s *Store) ListActivitiesByProcess(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error) {
	rows, err := s.queryContext(ctx,
		`SELECT id, process_instance_id, element_id, element_kind, status, assignees, input, output, error_message, started_at, ended_at
		 FROM bpmn_activities WHERE process_instance_id=$1 ORDER BY started_at ASC`, processInstanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivityRows(rows)
}

func (s *Store) ListActiveActivities(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error) {
	rows, err := s.queryContext(ctx,
		`SELECT id, process_instance_id, element_id, element_kind, status, assignees, input, output, error_message, started_at, ended_at
		 FROM bpmn_activities WHERE process_instance_id=$1 AND status=$2 ORDER BY started_at ASC`,
		processInstanceID, string(engine.ActivityStatusActive))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivityRows(rows)
}

func (s *Store) ListActiveUserTasks(ctx context.Context, tenantID, assignee string) ([]*engine.UserTask, error) {
	rows, err := s.queryContext(ctx,
		`SELECT a.id, a.process_instance_id, a.element_id, a.element_kind, a.status, a.assignees, a.input, a.output, a.error_message, a.started_at, a.ended_at,
		        i.tenant_id, i.process_key, i.business_key, i.process_version
		 FROM bpmn_activities a
		 JOIN bpmn_instances i ON i.id = a.process_instance_id
		 WHERE i.tenant_id = $1 AND a.status = $2 AND a.element_kind = $3
		 ORDER BY a.started_at ASC`,
		tenantID, string(engine.ActivityStatusActive), string(bpmn.KindUserTask))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*engine.UserTask
	for rows.Next() {
		ut, err := scanUserTaskRow(rows)
		if err != nil {
			return nil, err
		}
		if assignee != "" && !containsAssignee(ut.Assignees, assignee) {
			continue
		}
		res = append(res, ut)
	}
	return res, rows.Err()
}

func (s *Store) EnqueueJob(ctx context.Context, job *engine.Job) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	payloadJSON, err := json.Marshal(job.Payload)
	if err != nil {
		return err
	}
	_, err = s.execContext(ctx,
		`INSERT INTO bpmn_jobs (id, process_instance_id, job_type, payload, status, attempts, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		job.ID, job.ProcessInstanceID, string(job.Type), payloadJSON, string(engine.JobStatusPending), job.Attempts, job.CreatedAt,
	)
	return err
}

func (s *Store) ClaimNextJob(ctx context.Context) (*engine.Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx,
		`SELECT id, process_instance_id, job_type, payload, status, attempts, error_message, created_at, locked_at, completed_at
		 FROM bpmn_jobs WHERE status = $1
		 ORDER BY created_at ASC
		 LIMIT 1 FOR UPDATE SKIP LOCKED`, string(engine.JobStatusPending))

	job, err := scanJob(row)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, nil
	}

	now := time.Now().UTC()
	job.Status = engine.JobStatusRunning
	job.Attempts++
	job.LockedAt = &now
	_, err = tx.ExecContext(ctx,
		`UPDATE bpmn_jobs SET status=$2, attempts=$3, locked_at=$4 WHERE id=$1`,
		job.ID, string(job.Status), job.Attempts, job.LockedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Store) CompleteJob(ctx context.Context, jobID uuid.UUID) error {
	now := time.Now().UTC()
	_, err := s.execContext(ctx,
		`UPDATE bpmn_jobs SET status=$2, completed_at=$3 WHERE id=$1`,
		jobID, string(engine.JobStatusDone), now)
	return err
}

func (s *Store) FailJob(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	_, err := s.execContext(ctx,
		`UPDATE bpmn_jobs SET status=$2, error_message=$3 WHERE id=$1`,
		jobID, string(engine.JobStatusFailed), errMsg)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDeployedProcess(row rowScanner) (*engine.DeployedProcess, error) {
	var p engine.DeployedProcess
	var defBytes []byte
	if err := row.Scan(&p.TenantID, &p.Key, &p.Version, &p.Name, &defBytes, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(defBytes, &p.Definition); err != nil {
		return nil, err
	}
	return &p, nil
}

func scanProcessInstance(row rowScanner) (*engine.ProcessInstance, error) {
	var inst engine.ProcessInstance
	var status string
	var varsBytes, internalBytes, activeBytes, snapBytes []byte
	if err := row.Scan(&inst.ID, &inst.TenantID, &inst.ProcessKey, &inst.ProcessVersion, &inst.BusinessKey, &status,
		&varsBytes, &internalBytes, &activeBytes, &snapBytes, &inst.LockVersion,
		&inst.ErrorMsg, &inst.StartedAt, &inst.EndedAt, &inst.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	inst.Status = engine.ProcessInstanceStatus(status)
	if len(varsBytes) > 0 {
		_ = json.Unmarshal(varsBytes, &inst.Variables)
	}
	if len(internalBytes) > 0 {
		_ = json.Unmarshal(internalBytes, &inst.InternalState)
	}
	if len(activeBytes) > 0 {
		_ = json.Unmarshal(activeBytes, &inst.ActiveElements)
	}
	if len(snapBytes) > 0 {
		_ = json.Unmarshal(snapBytes, &inst.DefinitionSnapshot)
	}
	return &inst, nil
}

func scanActivityInstance(row rowScanner) (*engine.ActivityInstance, error) {
	var act engine.ActivityInstance
	var kind, status string
	var assignBytes, inBytes, outBytes []byte
	if err := row.Scan(&act.ID, &act.ProcessInstanceID, &act.ElementID, &kind, &status, &assignBytes, &inBytes, &outBytes,
		&act.ErrorMsg, &act.StartedAt, &act.EndedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	act.ElementKind = bpmn.ElementKind(kind)
	act.Status = engine.ActivityStatus(status)
	if len(assignBytes) > 0 {
		_ = json.Unmarshal(assignBytes, &act.Assignees)
	}
	if len(inBytes) > 0 {
		_ = json.Unmarshal(inBytes, &act.Input)
	}
	if len(outBytes) > 0 {
		_ = json.Unmarshal(outBytes, &act.Output)
	}
	return &act, nil
}

func scanActivityRows(rows *sql.Rows) ([]*engine.ActivityInstance, error) {
	var res []*engine.ActivityInstance
	for rows.Next() {
		act, err := scanActivityInstance(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, act)
	}
	return res, rows.Err()
}

func scanUserTaskRow(row rowScanner) (*engine.UserTask, error) {
	var ut engine.UserTask
	var kind, status string
	var assignBytes, inBytes, outBytes []byte
	if err := row.Scan(
		&ut.ID, &ut.ProcessInstanceID, &ut.ElementID, &kind, &status, &assignBytes, &inBytes, &outBytes,
		&ut.ErrorMsg, &ut.StartedAt, &ut.EndedAt,
		&ut.TenantID, &ut.ProcessKey, &ut.BusinessKey, &ut.ProcessVersion,
	); err != nil {
		return nil, err
	}
	ut.ElementKind = bpmn.ElementKind(kind)
	ut.Status = engine.ActivityStatus(status)
	if len(assignBytes) > 0 {
		_ = json.Unmarshal(assignBytes, &ut.Assignees)
	}
	if len(inBytes) > 0 {
		_ = json.Unmarshal(inBytes, &ut.Input)
	}
	if len(outBytes) > 0 {
		_ = json.Unmarshal(outBytes, &ut.Output)
	}
	return &ut, nil
}

func scanJob(row rowScanner) (*engine.Job, error) {
	var job engine.Job
	var jobType, status string
	var payloadBytes []byte
	if err := row.Scan(&job.ID, &job.ProcessInstanceID, &jobType, &payloadBytes, &status, &job.Attempts,
		&job.ErrorMsg, &job.CreatedAt, &job.LockedAt, &job.CompletedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	job.Type = engine.JobType(jobType)
	job.Status = engine.JobStatus(status)
	if len(payloadBytes) > 0 {
		_ = json.Unmarshal(payloadBytes, &job.Payload)
	}
	return &job, nil
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

// DB exposes the underlying connection for health checks.
func (s *Store) DB() *sql.DB { return s.db }

// Ping verifies database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("postgres store: db not configured")
	}
	return s.db.PingContext(ctx)
}
