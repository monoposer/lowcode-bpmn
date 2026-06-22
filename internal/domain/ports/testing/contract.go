// Package porttesting provides shared contract tests for ProcessRepository implementations.
package porttesting

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/domain/definition"
	"github.com/monoposer/lowcode-bpmn/internal/domain/ports"
	"github.com/monoposer/lowcode-bpmn/internal/domain/runtime"
)

// RunProcessRepositoryContract exercises the persistence port against any backend.
// Call from memory, gormstore, or filestore tests to keep adapters aligned.
func RunProcessRepositoryContract(t *testing.T, repo ports.ProcessRepository) {
	t.Helper()
	ctx := context.Background()
	tenantID := "contract-tenant"
	processKey := "contract-process"

	def := definition.ProcessDefinition{
		ID:   processKey,
		Name: "Contract smoke process",
		Elements: []definition.Element{
			{ID: "start", Kind: definition.KindStartEvent},
			{ID: "review", Kind: definition.KindUserTask, Assignees: []string{"alice"}},
			{ID: "end", Kind: definition.KindEndEvent},
		},
		Flows: []definition.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
		},
	}

	t.Run("deploy", func(t *testing.T) {
		if err := repo.InsertProcessVersion(ctx, &runtime.DeployedProcess{
			TenantID: tenantID, Key: processKey, Version: 1, Definition: def,
		}); err != nil {
			t.Fatalf("InsertProcessVersion: %v", err)
		}
		got, err := repo.GetProcess(ctx, tenantID, processKey)
		if err != nil {
			t.Fatalf("GetProcess: %v", err)
		}
		if got == nil || got.Key != processKey || got.Version != 1 {
			t.Fatalf("GetProcess = %+v, want key=%s version=1", got, processKey)
		}
		list, err := repo.ListProcesses(ctx, tenantID)
		if err != nil {
			t.Fatalf("ListProcesses: %v", err)
		}
		if len(list) == 0 {
			t.Fatal("ListProcesses: expected at least one process")
		}
	})

	t.Run("process instance CRUD", func(t *testing.T) {
		inst := &runtime.ProcessInstance{
			TenantID:       tenantID,
			ProcessKey:     processKey,
			ProcessVersion: 1,
			BusinessKey:    "BK-1",
			Status:         runtime.ProcessStatusRunning,
			Variables:      map[string]any{"amount": 42},
			LockVersion:    0,
		}
		if err := repo.CreateProcessInstance(ctx, inst); err != nil {
			t.Fatalf("CreateProcessInstance: %v", err)
		}
		if inst.ID == uuid.Nil {
			t.Fatal("CreateProcessInstance: expected ID assigned")
		}

		got, err := repo.GetProcessInstance(ctx, inst.ID)
		if err != nil {
			t.Fatalf("GetProcessInstance: %v", err)
		}
		if got == nil || got.BusinessKey != "BK-1" {
			t.Fatalf("GetProcessInstance = %+v, want business_key BK-1", got)
		}

		got.Variables["amount"] = 99
		got.LockVersion = 0
		if err := repo.UpdateProcessInstance(ctx, got); err != nil {
			t.Fatalf("UpdateProcessInstance: %v", err)
		}
		updated, err := repo.GetProcessInstance(ctx, inst.ID)
		if err != nil {
			t.Fatalf("GetProcessInstance after update: %v", err)
		}
		if updated.LockVersion != 1 {
			t.Fatalf("LockVersion = %d, want 1 after update", updated.LockVersion)
		}
		if updated.Variables["amount"] != 99 {
			t.Fatalf("variables amount = %v, want 99", updated.Variables["amount"])
		}

		running, err := repo.FindRunningInstanceByBusinessKey(ctx, tenantID, processKey, "BK-1")
		if err != nil {
			t.Fatalf("FindRunningInstanceByBusinessKey: %v", err)
		}
		if running == nil || running.ID != inst.ID {
			t.Fatalf("FindRunningInstanceByBusinessKey = %+v, want instance %s", running, inst.ID)
		}
	})

	t.Run("activity create and list", func(t *testing.T) {
		inst := &runtime.ProcessInstance{
			TenantID:       tenantID,
			ProcessKey:     processKey,
			ProcessVersion: 1,
			Status:         runtime.ProcessStatusRunning,
		}
		if err := repo.CreateProcessInstance(ctx, inst); err != nil {
			t.Fatalf("CreateProcessInstance: %v", err)
		}

		act := &runtime.ActivityInstance{
			ProcessInstanceID: inst.ID,
			ElementID:         "review",
			ElementKind:       definition.KindUserTask,
			Status:            runtime.ActivityStatusActive,
			Assignees:         []string{"alice"},
		}
		if err := repo.CreateActivityInstance(ctx, act); err != nil {
			t.Fatalf("CreateActivityInstance: %v", err)
		}
		if act.ID == uuid.Nil {
			t.Fatal("CreateActivityInstance: expected ID assigned")
		}

		all, err := repo.ListActivitiesByProcess(ctx, inst.ID)
		if err != nil {
			t.Fatalf("ListActivitiesByProcess: %v", err)
		}
		if len(all) != 1 || all[0].ElementID != "review" {
			t.Fatalf("ListActivitiesByProcess = %+v, want one review activity", all)
		}

		active, err := repo.ListActiveActivities(ctx, inst.ID)
		if err != nil {
			t.Fatalf("ListActiveActivities: %v", err)
		}
		if len(active) != 1 {
			t.Fatalf("ListActiveActivities len = %d, want 1", len(active))
		}
	})

	t.Run("inbox query", func(t *testing.T) {
		inst := &runtime.ProcessInstance{
			TenantID:       tenantID,
			ProcessKey:     processKey,
			ProcessVersion: 1,
			Status:         runtime.ProcessStatusRunning,
		}
		if err := repo.CreateProcessInstance(ctx, inst); err != nil {
			t.Fatalf("CreateProcessInstance: %v", err)
		}
		if err := repo.CreateActivityInstance(ctx, &runtime.ActivityInstance{
			ProcessInstanceID: inst.ID,
			ElementID:         "review",
			ElementKind:       definition.KindUserTask,
			Status:            runtime.ActivityStatusActive,
			Assignees:         []string{"bob"},
		}); err != nil {
			t.Fatalf("CreateActivityInstance: %v", err)
		}

		tasks, err := repo.ListActiveUserTasks(ctx, tenantID, "bob")
		if err != nil {
			t.Fatalf("ListActiveUserTasks: %v", err)
		}
		found := false
		for _, task := range tasks {
			if task.ProcessInstanceID == inst.ID && task.ElementID == "review" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("ListActiveUserTasks = %+v, want task for bob on instance %s", tasks, inst.ID)
		}
	})
}
