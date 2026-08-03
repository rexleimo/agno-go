package workflow

import (
	"testing"
	"time"
)

func TestWorkflowMetrics_StartStop(t *testing.T) {
	m := NewWorkflowMetrics()
	m.Start()
	time.Sleep(10 * time.Millisecond)
	dur := m.Stop()
	if dur <= 0 {
		t.Fatalf("expected positive duration, got %v", dur)
	}
	if got := m.Duration(); got != dur {
		t.Fatalf("expected duration %v, got %v", dur, got)
	}

	snapshot := m.Snapshot()
	if snapshot == nil {
		t.Fatal("expected snapshot to be populated")
	}
	if _, ok := snapshot["duration_seconds"]; !ok {
		t.Fatal("snapshot missing duration_seconds")
	}
}
