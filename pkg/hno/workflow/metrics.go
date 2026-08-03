package workflow

import (
	"sync"
	"time"
)

// WorkflowMetrics tracks runtime metrics for a workflow execution.
// The structure is safe for concurrent access as workflow steps may run in
// parallel and attempt to query duration information while the workflow is still
// running.
type WorkflowMetrics struct {
	mu        sync.RWMutex
	startedAt time.Time
	completed time.Time
	duration  time.Duration
	isRunning bool
}

// NewWorkflowMetrics constructs a metrics tracker with zeroed values.
func NewWorkflowMetrics() *WorkflowMetrics {
	return &WorkflowMetrics{}
}

// Start marks the workflow as started. Subsequent calls are no-ops so callers
// do not need to guard repeated invocations.
func (m *WorkflowMetrics) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.isRunning {
		return
	}
	if m.startedAt.IsZero() {
		m.startedAt = time.Now()
	}
	m.isRunning = true
}

// Stop marks the workflow as completed and returns the final duration. The
// method is idempotent; calling Stop multiple times returns the same duration.
func (m *WorkflowMetrics) Stop() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.isRunning && !m.completed.IsZero() {
		return m.duration
	}
	if m.startedAt.IsZero() {
		m.startedAt = time.Now()
	}
	m.completed = time.Now()
	m.duration = m.completed.Sub(m.startedAt)
	m.isRunning = false
	return m.duration
}

// Duration returns the current or final duration. When the workflow is still
// running it reports the elapsed time since Start was called.
func (m *WorkflowMetrics) Duration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.startedAt.IsZero() {
		return 0
	}
	if m.isRunning {
		return time.Since(m.startedAt)
	}
	return m.duration
}

// Snapshot returns a copy of the metrics in a map that can be safely embedded
// into metadata payloads.
func (m *WorkflowMetrics) Snapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := map[string]interface{}{}
	if !m.startedAt.IsZero() {
		result["started_at"] = m.startedAt.UTC()
	}
	if !m.completed.IsZero() {
		result["completed_at"] = m.completed.UTC()
	}
	if duration := m.duration; duration > 0 {
		result["duration_seconds"] = duration.Seconds()
	} else if m.isRunning {
		result["duration_seconds"] = time.Since(m.startedAt).Seconds()
	}
	return result
}
