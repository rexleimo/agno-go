package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestNewWorkflowRun(t *testing.T) {
	runID := "test-run-1"
	sessionID := "session-1"
	workflowID := "workflow-1"
	input := "test input"

	run := NewWorkflowRun(runID, sessionID, workflowID, input)

	if run.RunID != runID {
		t.Errorf("expected RunID %s, got %s", runID, run.RunID)
	}

	if run.SessionID != sessionID {
		t.Errorf("expected SessionID %s, got %s", sessionID, run.SessionID)
	}

	if run.WorkflowID != workflowID {
		t.Errorf("expected WorkflowID %s, got %s", workflowID, run.WorkflowID)
	}

	if run.Input != input {
		t.Errorf("expected Input %s, got %s", input, run.Input)
	}

	if run.Status != RunStatusPending {
		t.Errorf("expected Status %s, got %s", RunStatusPending, run.Status)
	}

	if run.Messages == nil {
		t.Error("expected Messages to be initialized")
	}

	if run.Metadata == nil {
		t.Error("expected Metadata to be initialized")
	}

	if run.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}
}

func TestWorkflowRun_MarkStarted(t *testing.T) {
	run := NewWorkflowRun("run-1", "session-1", "workflow-1", "input")

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	originalStartTime := run.StartedAt
	run.MarkStarted()

	if run.Status != RunStatusRunning {
		t.Errorf("expected Status %s, got %s", RunStatusRunning, run.Status)
	}

	// StartedAt should be updated
	if !run.StartedAt.After(originalStartTime) {
		t.Error("expected StartedAt to be updated")
	}
}

func TestWorkflowRun_MarkCompleted(t *testing.T) {
	run := NewWorkflowRun("run-1", "session-1", "workflow-1", "input")
	run.MarkStarted()

	output := "test output"
	run.MarkCompleted(output)

	if run.Status != RunStatusCompleted {
		t.Errorf("expected Status %s, got %s", RunStatusCompleted, run.Status)
	}

	if run.Output != output {
		t.Errorf("expected Output %s, got %s", output, run.Output)
	}

	if run.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set")
	}

	if !run.CompletedAt.After(run.StartedAt) {
		t.Error("expected CompletedAt to be after StartedAt")
	}
}

func TestWorkflowRun_MarkFailed(t *testing.T) {
	run := NewWorkflowRun("run-1", "session-1", "workflow-1", "input")
	run.MarkStarted()

	testError := errors.New("test error")
	run.MarkFailed(testError)

	if run.Status != RunStatusFailed {
		t.Errorf("expected Status %s, got %s", RunStatusFailed, run.Status)
	}

	if run.Error != testError.Error() {
		t.Errorf("expected Error %s, got %s", testError.Error(), run.Error)
	}

	if run.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set")
	}
}

func TestWorkflowRun_MarkCancelled(t *testing.T) {
	run := NewWorkflowRun("run-1", "session-1", "workflow-1", "input")
	run.MarkStarted()

	run.MarkCancelled()

	if run.Status != RunStatusCancelled {
		t.Errorf("expected Status %s, got %s", RunStatusCancelled, run.Status)
	}

	if run.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set")
	}
}

func TestWorkflowRun_AddMessage(t *testing.T) {
	run := NewWorkflowRun("run-1", "session-1", "workflow-1", "input")

	msg1 := types.NewUserMessage("hello")
	msg2 := types.NewAssistantMessage("hi there")

	run.AddMessage(msg1)
	run.AddMessage(msg2)

	if len(run.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(run.Messages))
	}

	if run.Messages[0] != msg1 {
		t.Error("expected first message to match")
	}

	if run.Messages[1] != msg2 {
		t.Error("expected second message to match")
	}
}

func TestWorkflowRun_Duration(t *testing.T) {
	run := NewWorkflowRun("run-1", "session-1", "workflow-1", "input")
	run.MarkStarted()

	// Sleep to ensure measurable duration
	time.Sleep(50 * time.Millisecond)

	// Test duration while running
	duration := run.Duration()
	if duration < 50*time.Millisecond {
		t.Errorf("expected duration >= 50ms, got %v", duration)
	}

	// Mark as completed and test duration
	run.MarkCompleted("output")
	completedDuration := run.Duration()

	if completedDuration <= 0 {
		t.Error("expected positive duration")
	}

	// Duration should remain stable after completion
	time.Sleep(10 * time.Millisecond)
	if run.Duration() != completedDuration {
		t.Error("expected duration to remain stable after completion")
	}
}

func TestWorkflowRun_IsCompleted(t *testing.T) {
	tests := []struct {
		name     string
		status   RunStatus
		expected bool
	}{
		{"pending", RunStatusPending, false},
		{"running", RunStatusRunning, false},
		{"completed", RunStatusCompleted, true},
		{"failed", RunStatusFailed, true},
		{"cancelled", RunStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := NewWorkflowRun("run-1", "session-1", "workflow-1", "input")
			run.Status = tt.status

			if run.IsCompleted() != tt.expected {
				t.Errorf("expected IsCompleted() to be %v for status %s", tt.expected, tt.status)
			}
		})
	}
}

func TestWorkflowRun_IsSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		status   RunStatus
		expected bool
	}{
		{"pending", RunStatusPending, false},
		{"running", RunStatusRunning, false},
		{"completed", RunStatusCompleted, true},
		{"failed", RunStatusFailed, false},
		{"cancelled", RunStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := NewWorkflowRun("run-1", "session-1", "workflow-1", "input")
			run.Status = tt.status

			if run.IsSuccessful() != tt.expected {
				t.Errorf("expected IsSuccessful() to be %v for status %s", tt.expected, tt.status)
			}
		})
	}
}

func TestWorkflowRun_Lifecycle(t *testing.T) {
	// Test complete lifecycle: pending -> running -> completed
	run := NewWorkflowRun("run-1", "session-1", "workflow-1", "test input")

	// Initial state
	if run.Status != RunStatusPending {
		t.Errorf("expected initial status %s, got %s", RunStatusPending, run.Status)
	}
	if run.IsCompleted() {
		t.Error("expected run not to be completed initially")
	}

	// Start
	run.MarkStarted()
	if run.Status != RunStatusRunning {
		t.Errorf("expected status %s after start, got %s", RunStatusRunning, run.Status)
	}
	if run.IsCompleted() {
		t.Error("expected run not to be completed while running")
	}

	// Add some messages
	run.AddMessage(types.NewUserMessage("hello"))
	run.AddMessage(types.NewAssistantMessage("hi"))

	// Complete
	run.MarkCompleted("final output")
	if run.Status != RunStatusCompleted {
		t.Errorf("expected status %s after completion, got %s", RunStatusCompleted, run.Status)
	}
	if !run.IsCompleted() {
		t.Error("expected run to be completed")
	}
	if !run.IsSuccessful() {
		t.Error("expected run to be successful")
	}

	// Verify data integrity
	if run.Output != "final output" {
		t.Errorf("expected output 'final output', got '%s'", run.Output)
	}
	if len(run.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(run.Messages))
	}
	if run.Duration() <= 0 {
		t.Error("expected positive duration")
	}
}

// BenchmarkNewWorkflowRun benchmarks the creation of WorkflowRun
func BenchmarkNewWorkflowRun(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = NewWorkflowRun("run-1", "session-1", "workflow-1", "input")
	}
}

// BenchmarkWorkflowRun_AddMessage benchmarks adding messages to a run
func BenchmarkWorkflowRun_AddMessage(b *testing.B) {
	run := NewWorkflowRun("run-1", "session-1", "workflow-1", "input")
	msg := types.NewUserMessage("test message")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		run.AddMessage(msg)
	}
}
