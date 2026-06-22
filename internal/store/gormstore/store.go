package gormstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

// Store is a GORM implementation of engine.Store supporting postgres, mysql, and sqlite.
type Store struct {
	db      *gorm.DB
	tx      *gorm.DB
	dialect string
}

func (s *Store) conn(ctx context.Context) *gorm.DB {
	if s.tx != nil {
		return s.tx.WithContext(ctx)
	}
	return s.db.WithContext(ctx)
}

// WithTx runs fn inside a database transaction.
func (s *Store) WithTx(ctx context.Context, fn func(engine.Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txStore := &Store{db: s.db, tx: tx, dialect: s.dialect}
		return fn(txStore)
	})
}

// Ping verifies database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("gorm store: get sql db: %w", err)
	}
	return sqlDB.PingContext(ctx)
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
	row, err := processToModel(p)
	if err != nil {
		return err
	}
	return s.conn(ctx).Create(row).Error
}

func (s *Store) DeleteProcess(ctx context.Context, tenantID, key string) error {
	return s.conn(ctx).
		Where("tenant_id = ? AND process_key = ?", tenantID, key).
		Delete(&BpmnProcess{}).Error
}

func (s *Store) GetProcess(ctx context.Context, tenantID, key string) (*engine.DeployedProcess, error) {
	var row BpmnProcess
	err := s.conn(ctx).
		Where("tenant_id = ? AND process_key = ?", tenantID, key).
		Order("version DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return processFromModel(&row)
}

func (s *Store) GetProcessVersion(ctx context.Context, tenantID, key string, version int) (*engine.DeployedProcess, error) {
	var row BpmnProcess
	err := s.conn(ctx).
		Where("tenant_id = ? AND process_key = ? AND version = ?", tenantID, key, version).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return processFromModel(&row)
}

func (s *Store) ListProcesses(ctx context.Context, tenantID string) ([]*engine.DeployedProcess, error) {
	var rows []BpmnProcess
	err := s.conn(ctx).Raw(`
		SELECT tenant_id, process_key, version, name, definition, created_at, updated_at
		FROM (
			SELECT *, ROW_NUMBER() OVER (PARTITION BY process_key ORDER BY version DESC) AS rn
			FROM bpmn_processes
			WHERE tenant_id = ?
		) ranked
		WHERE rn = 1
	`, tenantID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	res := make([]*engine.DeployedProcess, 0, len(rows))
	for i := range rows {
		p, err := processFromModel(&rows[i])
		if err != nil {
			return nil, err
		}
		res = append(res, p)
	}
	return res, nil
}

func (s *Store) CreateProcessInstance(ctx context.Context, inst *engine.ProcessInstance) error {
	if inst.ID == uuid.Nil {
		inst.ID = newUUID()
	}
	now := time.Now().UTC()
	if inst.StartedAt.IsZero() {
		inst.StartedAt = now
	}
	inst.UpdatedAt = now
	row, err := instanceToModel(inst)
	if err != nil {
		return err
	}
	return s.conn(ctx).Create(row).Error
}

func (s *Store) UpdateProcessInstance(ctx context.Context, inst *engine.ProcessInstance) error {
	inst.UpdatedAt = time.Now().UTC()
	row, err := instanceToModel(inst)
	if err != nil {
		return err
	}
	res := s.conn(ctx).Model(&BpmnInstance{}).
		Where("id = ? AND lock_version = ?", inst.ID, inst.LockVersion).
		Updates(map[string]any{
			"status":          row.Status,
			"variables":       row.Variables,
			"internal_state":  row.InternalState,
			"active_elements": row.ActiveElements,
			"error_message":   row.ErrorMessage,
			"ended_at":        row.EndedAt,
			"updated_at":      row.UpdatedAt,
			"lock_version":    gorm.Expr("lock_version + 1"),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return engine.ErrVersionConflict
	}
	inst.LockVersion++
	return nil
}

func (s *Store) GetProcessInstance(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error) {
	return s.loadProcessInstance(ctx, id, false)
}

func (s *Store) GetProcessInstanceForUpdate(ctx context.Context, id uuid.UUID) (*engine.ProcessInstance, error) {
	return s.loadProcessInstance(ctx, id, true)
}

func (s *Store) loadProcessInstance(ctx context.Context, id uuid.UUID, forUpdate bool) (*engine.ProcessInstance, error) {
	var row BpmnInstance
	q := s.conn(ctx).Where("id = ?", id)
	if forUpdate {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return instanceFromModel(&row)
}

func (s *Store) FindRunningInstanceByBusinessKey(ctx context.Context, tenantID, processKey, businessKey string) (*engine.ProcessInstance, error) {
	if businessKey == "" {
		return nil, nil
	}
	var row BpmnInstance
	err := s.conn(ctx).
		Where("tenant_id = ? AND process_key = ? AND business_key = ? AND status = ?",
			tenantID, processKey, businessKey, engine.ProcessStatusRunning).
		Order("started_at DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return instanceFromModel(&row)
}

func (s *Store) ListRunningInstances(ctx context.Context, tenantID string) ([]*engine.ProcessInstance, error) {
	var rows []BpmnInstance
	err := s.conn(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, engine.ProcessStatusRunning).
		Order("started_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*engine.ProcessInstance, 0, len(rows))
	for i := range rows {
		inst, err := instanceFromModel(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, inst)
	}
	return out, nil
}

func (s *Store) CreateActivityInstance(ctx context.Context, act *engine.ActivityInstance) error {
	if act.ID == uuid.Nil {
		act.ID = newUUID()
	}
	if act.StartedAt.IsZero() {
		act.StartedAt = time.Now().UTC()
	}
	row, err := activityToModel(act)
	if err != nil {
		return err
	}
	return s.conn(ctx).Create(row).Error
}

func (s *Store) UpdateActivityInstance(ctx context.Context, act *engine.ActivityInstance) error {
	row, err := activityToModel(act)
	if err != nil {
		return err
	}
	return s.conn(ctx).Model(&BpmnActivity{}).Where("id = ?", act.ID).Updates(map[string]any{
		"status":            row.Status,
		"scope_id":          row.ScopeID,
		"branch_flow_id":    row.BranchFlowID,
		"outcome":           row.Outcome,
		"assignees":         row.Assignees,
		"approval_mode":      row.ApprovalMode,
		"required_approvals": row.RequiredApprovals,
		"pending_assignees": row.PendingAssignees,
		"approval_records":  row.ApprovalRecords,
		"input":             row.Input,
		"output":            row.Output,
		"error_message":     row.ErrorMessage,
		"ended_at":          row.EndedAt,
	}).Error
}

func (s *Store) GetActivityInstance(ctx context.Context, id uuid.UUID) (*engine.ActivityInstance, error) {
	var row BpmnActivity
	err := s.conn(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return activityFromModel(&row)
}

func (s *Store) ListActivitiesByProcess(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error) {
	var rows []BpmnActivity
	err := s.conn(ctx).
		Where("process_instance_id = ?", processInstanceID).
		Order("started_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return activitiesFromModels(rows)
}

func (s *Store) ListActiveActivities(ctx context.Context, processInstanceID uuid.UUID) ([]*engine.ActivityInstance, error) {
	var rows []BpmnActivity
	err := s.conn(ctx).
		Where("process_instance_id = ? AND status = ?", processInstanceID, string(engine.ActivityStatusActive)).
		Order("started_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return activitiesFromModels(rows)
}

func activitiesFromModels(rows []BpmnActivity) ([]*engine.ActivityInstance, error) {
	res := make([]*engine.ActivityInstance, 0, len(rows))
	for i := range rows {
		act, err := activityFromModel(&rows[i])
		if err != nil {
			return nil, err
		}
		res = append(res, act)
	}
	return res, nil
}

type userTaskRow struct {
	BpmnActivity
	TenantID       string `gorm:"column:tenant_id"`
	ProcessKey     string `gorm:"column:process_key"`
	BusinessKey    string `gorm:"column:business_key"`
	ProcessVersion int    `gorm:"column:process_version"`
}

func (s *Store) ListActiveUserTasks(ctx context.Context, tenantID, assignee string) ([]*engine.UserTask, error) {
	var rows []userTaskRow
	err := s.conn(ctx).
		Table("bpmn_activities a").
		Select(`a.id, a.process_instance_id, a.element_id, a.element_kind, a.status,
		        a.assignees, a.approval_mode, a.required_approvals, a.pending_assignees, a.approval_records,
		        a.input, a.output, a.error_message, a.started_at, a.ended_at,
		        i.tenant_id, i.process_key, i.business_key, i.process_version`).
		Joins("JOIN bpmn_instances i ON i.id = a.process_instance_id").
		Where("i.tenant_id = ? AND a.status = ? AND a.element_kind = ?",
			tenantID, string(engine.ActivityStatusActive), string(bpmn.KindUserTask)).
		Order("a.started_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	var res []*engine.UserTask
	for i := range rows {
		act, err := activityFromModel(&rows[i].BpmnActivity)
		if err != nil {
			return nil, err
		}
		if assignee != "" && !engine.TaskVisibleToAssignee(act, assignee) {
			continue
		}
		res = append(res, &engine.UserTask{
			ActivityInstance: *act,
			TenantID:         rows[i].TenantID,
			ProcessKey:       rows[i].ProcessKey,
			BusinessKey:      rows[i].BusinessKey,
			ProcessVersion:   rows[i].ProcessVersion,
		})
	}
	return res, nil
}

func (s *Store) EnqueueJob(ctx context.Context, job *engine.Job) error {
	if job.ID == uuid.Nil {
		job.ID = newUUID()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.Status == "" {
		job.Status = engine.JobStatusPending
	}
	row, err := jobToModel(job)
	if err != nil {
		return err
	}
	return s.conn(ctx).Create(row).Error
}

func (s *Store) ClaimNextJob(ctx context.Context) (*engine.Job, error) {
	var claimed *engine.Job
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row BpmnJob
		q := tx.Where("status = ?", string(engine.JobStatusPending)).Order("created_at ASC")
		if s.supportsSkipLocked() {
			q = q.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		} else {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := q.First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		now := time.Now().UTC()
		row.Status = string(engine.JobStatusRunning)
		row.Attempts++
		row.LockedAt = &now
		if err := tx.Model(&BpmnJob{}).Where("id = ?", row.ID).Updates(map[string]any{
			"status":    row.Status,
			"attempts":  row.Attempts,
			"locked_at": row.LockedAt,
		}).Error; err != nil {
			return err
		}

		job, err := jobFromModel(&row)
		if err != nil {
			return err
		}
		claimed = job
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

func (s *Store) supportsSkipLocked() bool {
	return s.dialect == "postgres" || s.dialect == "mysql"
}

func (s *Store) CompleteJob(ctx context.Context, jobID uuid.UUID) error {
	now := time.Now().UTC()
	return s.conn(ctx).Model(&BpmnJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":       string(engine.JobStatusDone),
		"completed_at": now,
	}).Error
}

func (s *Store) FailJob(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	return s.conn(ctx).Model(&BpmnJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":        string(engine.JobStatusFailed),
		"error_message": errMsg,
	}).Error
}

var _ engine.Store = (*Store)(nil)
