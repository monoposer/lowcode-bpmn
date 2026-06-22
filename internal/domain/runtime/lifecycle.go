package runtime

// IsTerminal reports whether the process instance has reached a final status.
func (s ProcessInstanceStatus) IsTerminal() bool {
	switch s {
	case ProcessStatusCompleted, ProcessStatusFailed, ProcessStatusCancelled:
		return true
	default:
		return false
	}
}

// IsRunning reports whether the instance is actively executing.
func (pi *ProcessInstance) IsRunning() bool {
	return pi.Status == ProcessStatusRunning
}

// IsActive reports whether the activity is still open (waiting or in progress).
func (a *ActivityInstance) IsActive() bool {
	return a.Status == ActivityStatusActive
}

// LockConflict returns ErrVersionConflict when clientVersion does not match the
// instance's current lock version (optimistic concurrency check).
func (pi *ProcessInstance) LockConflict(clientVersion int) error {
	if clientVersion != pi.LockVersion {
		return ErrVersionConflict
	}
	return nil
}
