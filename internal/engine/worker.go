package engine

import (
	"context"
	"time"

	"github.com/monoposer/lowcode-bpmn/internal/telemetry"
)

// Worker polls and executes async BPMN jobs.
type Worker struct {
	engine   *Engine
	interval time.Duration
}

// NewWorker constructs a job worker.
func NewWorker(e *Engine, interval time.Duration) *Worker {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	return &Worker{engine: e, interval: interval}
}

// Run processes jobs until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.engine.ProcessNextJob(ctx); err != nil {
				telemetry.Ctx(ctx).Error("worker job failed", "error", err)
			}
		}
	}
}
