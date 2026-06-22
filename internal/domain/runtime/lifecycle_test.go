package runtime

import (
	"errors"
	"testing"
)

// TDD domain example: table-driven tests for pure runtime lifecycle rules.
// No I/O, no mocks — write the test first, then implement lifecycle.go.

func TestProcessInstanceStatus_IsTerminal(t *testing.T) {
	// TDD domain example — Red-Green-Refactor on status predicates.
	cases := []struct {
		status ProcessInstanceStatus
		want   bool
	}{
		{ProcessStatusPending, false},
		{ProcessStatusRunning, false},
		{ProcessStatusCompleted, true},
		{ProcessStatusFailed, true},
		{ProcessStatusCancelled, true},
	}
	for _, c := range cases {
		t.Run(string(c.status), func(t *testing.T) {
			if got := c.status.IsTerminal(); got != c.want {
				t.Fatalf("IsTerminal(%q) = %v, want %v", c.status, got, c.want)
			}
		})
	}
}

func TestProcessInstance_IsRunning(t *testing.T) {
	// TDD domain example — instance aggregate behavior without persistence.
	cases := []struct {
		status ProcessInstanceStatus
		want   bool
	}{
		{ProcessStatusPending, false},
		{ProcessStatusRunning, true},
		{ProcessStatusCompleted, false},
		{ProcessStatusFailed, false},
		{ProcessStatusCancelled, false},
	}
	for _, c := range cases {
		t.Run(string(c.status), func(t *testing.T) {
			inst := &ProcessInstance{Status: c.status}
			if got := inst.IsRunning(); got != c.want {
				t.Fatalf("IsRunning() with status %q = %v, want %v", c.status, got, c.want)
			}
		})
	}
}

func TestActivityInstance_IsActive(t *testing.T) {
	// TDD domain example — activity state predicate.
	cases := []struct {
		status ActivityStatus
		want   bool
	}{
		{ActivityStatusActive, true},
		{ActivityStatusCompleted, false},
		{ActivityStatusFailed, false},
		{ActivityStatusCancelled, false},
	}
	for _, c := range cases {
		t.Run(string(c.status), func(t *testing.T) {
			act := &ActivityInstance{Status: c.status}
			if got := act.IsActive(); got != c.want {
				t.Fatalf("IsActive() with status %q = %v, want %v", c.status, got, c.want)
			}
		})
	}
}

func TestProcessInstance_LockConflict(t *testing.T) {
	// TDD domain example — optimistic locking rule on the aggregate.
	inst := &ProcessInstance{LockVersion: 3}

	t.Run("matching version", func(t *testing.T) {
		if err := inst.LockConflict(3); err != nil {
			t.Fatalf("LockConflict(3) = %v, want nil", err)
		}
	})

	t.Run("stale version", func(t *testing.T) {
		err := inst.LockConflict(2)
		if !errors.Is(err, ErrVersionConflict) {
			t.Fatalf("LockConflict(2) = %v, want ErrVersionConflict", err)
		}
	})
}
