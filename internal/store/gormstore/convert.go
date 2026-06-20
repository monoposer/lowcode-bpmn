package gormstore

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
)

func toJSON(v any) (datatypes.JSON, error) {
	if v == nil {
		return datatypes.JSON("null"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

func fromJSON[T any](raw []byte, dst *T) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, dst)
}

func processFromModel(m *BpmnProcess) (*engine.DeployedProcess, error) {
	var def bpmn.ProcessDefinition
	if err := fromJSON(m.Definition, &def); err != nil {
		return nil, fmt.Errorf("decode process definition: %w", err)
	}
	return &engine.DeployedProcess{
		TenantID:   m.TenantID,
		Key:        m.ProcessKey,
		Version:    m.Version,
		Name:       m.Name,
		Definition: def,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}, nil
}

func processToModel(p *engine.DeployedProcess) (*BpmnProcess, error) {
	defJSON, err := toJSON(p.Definition)
	if err != nil {
		return nil, err
	}
	name := p.Name
	if name == "" {
		name = p.Key
	}
	return &BpmnProcess{
		TenantID:   p.TenantID,
		ProcessKey: p.Key,
		Version:    p.Version,
		Name:       name,
		Definition: defJSON,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}, nil
}

func instanceFromModel(m *BpmnInstance) (*engine.ProcessInstance, error) {
	inst := &engine.ProcessInstance{
		ID:             m.ID,
		TenantID:       m.TenantID,
		ProcessKey:     m.ProcessKey,
		ProcessVersion: m.ProcessVersion,
		BusinessKey:    m.BusinessKey,
		Status:         engine.ProcessInstanceStatus(m.Status),
		ErrorMsg:       m.ErrorMessage,
		StartedAt:      m.StartedAt,
		EndedAt:        m.EndedAt,
		UpdatedAt:      m.UpdatedAt,
		LockVersion:    m.LockVersion,
	}
	if err := fromJSON(m.Variables, &inst.Variables); err != nil {
		return nil, err
	}
	if err := fromJSON(m.InternalState, &inst.InternalState); err != nil {
		return nil, err
	}
	if err := fromJSON(m.ActiveElements, &inst.ActiveElements); err != nil {
		return nil, err
	}
	if len(m.DefinitionSnapshot) > 0 {
		if err := fromJSON(m.DefinitionSnapshot, &inst.DefinitionSnapshot); err != nil {
			return nil, err
		}
	}
	return inst, nil
}

func instanceToModel(inst *engine.ProcessInstance) (*BpmnInstance, error) {
	varsJSON, err := toJSON(inst.Variables)
	if err != nil {
		return nil, err
	}
	internalJSON, err := toJSON(inst.InternalState)
	if err != nil {
		return nil, err
	}
	activeJSON, err := toJSON(inst.ActiveElements)
	if err != nil {
		return nil, err
	}
	snapJSON, err := toJSON(inst.DefinitionSnapshot)
	if err != nil {
		return nil, err
	}
	return &BpmnInstance{
		ID:                 inst.ID,
		TenantID:           inst.TenantID,
		ProcessKey:         inst.ProcessKey,
		ProcessVersion:     inst.ProcessVersion,
		BusinessKey:        inst.BusinessKey,
		Status:             string(inst.Status),
		Variables:          varsJSON,
		InternalState:      internalJSON,
		ActiveElements:     activeJSON,
		DefinitionSnapshot: snapJSON,
		LockVersion:        inst.LockVersion,
		ErrorMessage:       inst.ErrorMsg,
		StartedAt:          inst.StartedAt,
		EndedAt:            inst.EndedAt,
		UpdatedAt:          inst.UpdatedAt,
	}, nil
}

func activityFromModel(m *BpmnActivity) (*engine.ActivityInstance, error) {
	act := &engine.ActivityInstance{
		ID:                m.ID,
		ProcessInstanceID: m.ProcessInstanceID,
		ElementID:         m.ElementID,
		ElementKind:       bpmn.ElementKind(m.ElementKind),
		Status:            engine.ActivityStatus(m.Status),
		ErrorMsg:          m.ErrorMessage,
		StartedAt:         m.StartedAt,
		EndedAt:           m.EndedAt,
	}
	if err := fromJSON(m.Assignees, &act.Assignees); err != nil {
		return nil, err
	}
	if err := fromJSON(m.Input, &act.Input); err != nil {
		return nil, err
	}
	if err := fromJSON(m.Output, &act.Output); err != nil {
		return nil, err
	}
	return act, nil
}

func activityToModel(act *engine.ActivityInstance) (*BpmnActivity, error) {
	assignJSON, err := toJSON(act.Assignees)
	if err != nil {
		return nil, err
	}
	inJSON, err := toJSON(act.Input)
	if err != nil {
		return nil, err
	}
	outJSON, err := toJSON(act.Output)
	if err != nil {
		return nil, err
	}
	return &BpmnActivity{
		ID:                act.ID,
		ProcessInstanceID: act.ProcessInstanceID,
		ElementID:         act.ElementID,
		ElementKind:       string(act.ElementKind),
		Status:            string(act.Status),
		Assignees:         assignJSON,
		Input:             inJSON,
		Output:            outJSON,
		ErrorMessage:      act.ErrorMsg,
		StartedAt:         act.StartedAt,
		EndedAt:           act.EndedAt,
	}, nil
}

func jobFromModel(m *BpmnJob) (*engine.Job, error) {
	job := &engine.Job{
		ID:                m.ID,
		ProcessInstanceID: m.ProcessInstanceID,
		Type:              engine.JobType(m.JobType),
		Status:            engine.JobStatus(m.Status),
		Attempts:          m.Attempts,
		ErrorMsg:          m.ErrorMessage,
		CreatedAt:         m.CreatedAt,
		LockedAt:          m.LockedAt,
		CompletedAt:       m.CompletedAt,
	}
	if err := fromJSON(m.Payload, &job.Payload); err != nil {
		return nil, err
	}
	return job, nil
}

func jobToModel(job *engine.Job) (*BpmnJob, error) {
	payloadJSON, err := toJSON(job.Payload)
	if err != nil {
		return nil, err
	}
	return &BpmnJob{
		ID:                job.ID,
		ProcessInstanceID: job.ProcessInstanceID,
		JobType:           string(job.Type),
		Payload:           payloadJSON,
		Status:            string(job.Status),
		Attempts:          job.Attempts,
		ErrorMessage:      job.ErrorMsg,
		CreatedAt:         job.CreatedAt,
		LockedAt:          job.LockedAt,
		CompletedAt:       job.CompletedAt,
	}, nil
}

func containsAssignee(assignees []string, assignee string) bool {
	for _, a := range assignees {
		if a == assignee {
			return true
		}
	}
	return false
}

func newUUID() uuid.UUID {
	return uuid.New()
}
