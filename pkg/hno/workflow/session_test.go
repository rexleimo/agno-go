package workflow

import (
	"errors"
	"sync"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestNewWorkflowSession(t *testing.T) {
	sessionID := "session-1"
	workflowID := "workflow-1"
	userID := "user-1"

	session := NewWorkflowSession(sessionID, workflowID, userID)

	if session.SessionID != sessionID {
		t.Errorf("expected SessionID %s, got %s", sessionID, session.SessionID)
	}

	if session.WorkflowID != workflowID {
		t.Errorf("expected WorkflowID %s, got %s", workflowID, session.WorkflowID)
	}

	if session.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, session.UserID)
	}

	if session.Runs == nil {
		t.Error("expected Runs to be initialized")
	}

	if len(session.Runs) != 0 {
		t.Errorf("expected empty Runs, got %d", len(session.Runs))
	}

	if session.Metadata == nil {
		t.Error("expected Metadata to be initialized")
	}

	if session.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if session.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestWorkflowSession_AddRun(t *testing.T) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")

	run1 := NewWorkflowRun("run-1", "session-1", "workflow-1", "input-1")
	run2 := NewWorkflowRun("run-2", "session-1", "workflow-1", "input-2")

	session.AddRun(run1)
	session.AddRun(run2)

	if session.CountRuns() != 2 {
		t.Errorf("expected 2 runs, got %d", session.CountRuns())
	}

	runs := session.GetRuns()
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}

	if runs[0].RunID != "run-1" {
		t.Errorf("expected first run ID 'run-1', got '%s'", runs[0].RunID)
	}

	if runs[1].RunID != "run-2" {
		t.Errorf("expected second run ID 'run-2', got '%s'", runs[1].RunID)
	}
}

func TestWorkflowSession_GetLastRun(t *testing.T) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")

	// Empty session
	if session.GetLastRun() != nil {
		t.Error("expected nil for empty session")
	}

	// Add runs
	run1 := NewWorkflowRun("run-1", "session-1", "workflow-1", "input-1")
	run2 := NewWorkflowRun("run-2", "session-1", "workflow-1", "input-2")

	session.AddRun(run1)
	lastRun := session.GetLastRun()
	if lastRun == nil || lastRun.RunID != "run-1" {
		t.Error("expected last run to be run-1")
	}

	session.AddRun(run2)
	lastRun = session.GetLastRun()
	if lastRun == nil || lastRun.RunID != "run-2" {
		t.Error("expected last run to be run-2")
	}
}

func TestWorkflowSession_GetHistory(t *testing.T) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")

	// Empty session
	history := session.GetHistory(3)
	if history != nil {
		t.Error("expected nil history for empty session")
	}

	// Add completed runs
	for i := 1; i <= 5; i++ {
		run := NewWorkflowRun(
			"run-"+string(rune('0'+i)),
			"session-1",
			"workflow-1",
			"input-"+string(rune('0'+i)),
		)
		run.MarkCompleted("output-" + string(rune('0'+i)))
		session.AddRun(run)
	}

	// Add a pending run (should be excluded)
	pendingRun := NewWorkflowRun("run-6", "session-1", "workflow-1", "input-6")
	session.AddRun(pendingRun)

	// Test get last 3
	history = session.GetHistory(3)
	if len(history) != 3 {
		t.Errorf("expected 3 history entries, got %d", len(history))
	}

	// Verify order (most recent last 3)
	for i, entry := range history {
		expectedInput := "input-" + string(rune('0'+3+i))
		expectedOutput := "output-" + string(rune('0'+3+i))
		if entry.Input != expectedInput {
			t.Errorf("expected input '%s', got '%s'", expectedInput, entry.Input)
		}
		if entry.Output != expectedOutput {
			t.Errorf("expected output '%s', got '%s'", expectedOutput, entry.Output)
		}
	}

	// Test get all (numRuns = 0 or < 0)
	history = session.GetHistory(0)
	if len(history) != 5 {
		t.Errorf("expected 5 history entries, got %d", len(history))
	}

	history = session.GetHistory(-1)
	if len(history) != 5 {
		t.Errorf("expected 5 history entries for negative numRuns, got %d", len(history))
	}

	// Test get more than available
	history = session.GetHistory(10)
	if len(history) != 5 {
		t.Errorf("expected 5 history entries, got %d", len(history))
	}
}

func TestWorkflowSession_GetHistoryContext(t *testing.T) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")

	// Empty session
	context := session.GetHistoryContext(3)
	if context != "" {
		t.Error("expected empty context for empty session")
	}

	// Add completed runs
	run1 := NewWorkflowRun("run-1", "session-1", "workflow-1", "Hello")
	run1.MarkCompleted("Hi there!")
	session.AddRun(run1)

	run2 := NewWorkflowRun("run-2", "session-1", "workflow-1", "How are you?")
	run2.MarkCompleted("I'm good, thanks!")
	session.AddRun(run2)

	context = session.GetHistoryContext(2)

	// Verify context format
	if context == "" {
		t.Error("expected non-empty context")
	}

	// Check for expected content
	expectedParts := []string{
		"<workflow_history_context>",
		"[run-1]",
		"input: Hello",
		"output: Hi there!",
		"[run-2]",
		"input: How are you?",
		"output: I'm good, thanks!",
		"</workflow_history_context>",
	}

	for _, part := range expectedParts {
		if !contains(context, part) {
			t.Errorf("expected context to contain '%s'", part)
		}
	}
}

func TestWorkflowSession_GetHistoryMessages(t *testing.T) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")

	// Add runs with messages
	run1 := NewWorkflowRun("run-1", "session-1", "workflow-1", "input-1")
	run1.AddMessage(types.NewUserMessage("user message 1"))
	run1.AddMessage(types.NewAssistantMessage("assistant message 1"))
	run1.MarkCompleted("output-1")
	session.AddRun(run1)

	run2 := NewWorkflowRun("run-2", "session-1", "workflow-1", "input-2")
	run2.AddMessage(types.NewUserMessage("user message 2"))
	run2.AddMessage(types.NewAssistantMessage("assistant message 2"))
	run2.MarkCompleted("output-2")
	session.AddRun(run2)

	// Get messages from last 2 runs
	messages := session.GetHistoryMessages(2)

	if len(messages) != 4 {
		t.Errorf("expected 4 messages, got %d", len(messages))
	}

	// Verify message order
	expectedContents := []string{
		"user message 1",
		"assistant message 1",
		"user message 2",
		"assistant message 2",
	}

	for i, msg := range messages {
		if msg.Content != expectedContents[i] {
			t.Errorf("expected message %d to be '%s', got '%s'",
				i, expectedContents[i], msg.Content)
		}
	}
}

func TestWorkflowSession_CountMethods(t *testing.T) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")

	// Add various runs
	run1 := NewWorkflowRun("run-1", "session-1", "workflow-1", "input-1")
	run1.MarkCompleted("output-1") // Successful
	session.AddRun(run1)

	run2 := NewWorkflowRun("run-2", "session-1", "workflow-1", "input-2")
	run2.MarkFailed(errors.New("test error")) // Failed
	session.AddRun(run2)

	run3 := NewWorkflowRun("run-3", "session-1", "workflow-1", "input-3")
	run3.MarkCancelled() // Cancelled
	session.AddRun(run3)

	run4 := NewWorkflowRun("run-4", "session-1", "workflow-1", "input-4")
	run4.MarkStarted() // Running (not completed)
	session.AddRun(run4)

	// Test counts
	if session.CountRuns() != 4 {
		t.Errorf("expected 4 total runs, got %d", session.CountRuns())
	}

	if session.CountCompletedRuns() != 3 {
		t.Errorf("expected 3 completed runs, got %d", session.CountCompletedRuns())
	}

	if session.CountSuccessfulRuns() != 1 {
		t.Errorf("expected 1 successful run, got %d", session.CountSuccessfulRuns())
	}
}

func TestWorkflowSession_Clear(t *testing.T) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")

	// Add runs
	for i := 1; i <= 3; i++ {
		run := NewWorkflowRun("run-"+string(rune('0'+i)), "session-1", "workflow-1", "input")
		session.AddRun(run)
	}

	if session.CountRuns() != 3 {
		t.Errorf("expected 3 runs, got %d", session.CountRuns())
	}

	// Clear
	session.Clear()

	if session.CountRuns() != 0 {
		t.Errorf("expected 0 runs after clear, got %d", session.CountRuns())
	}

	if session.GetLastRun() != nil {
		t.Error("expected nil last run after clear")
	}
}

func TestWorkflowSession_Metadata(t *testing.T) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")

	// Set metadata
	session.SetMetadata("key1", "value1")
	session.SetMetadata("key2", 42)
	session.SetMetadata("key3", map[string]interface{}{"nested": "data"})

	// Get metadata
	value1, exists1 := session.GetMetadata("key1")
	if !exists1 || value1 != "value1" {
		t.Error("expected key1 to exist with value 'value1'")
	}

	value2, exists2 := session.GetMetadata("key2")
	if !exists2 || value2 != 42 {
		t.Error("expected key2 to exist with value 42")
	}

	_, exists3 := session.GetMetadata("key3")
	if !exists3 {
		t.Error("expected key3 to exist")
	}

	// Non-existent key
	_, exists4 := session.GetMetadata("key4")
	if exists4 {
		t.Error("expected key4 to not exist")
	}
}

func TestWorkflowSession_ConcurrentAccess(t *testing.T) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")

	// Concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 10
	numRunsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numRunsPerGoroutine; j++ {
				run := NewWorkflowRun(
					"run-"+string(rune('0'+goroutineID))+"-"+string(rune('0'+j)),
					"session-1",
					"workflow-1",
					"input",
				)
				run.MarkCompleted("output")
				session.AddRun(run)
			}
		}(i)
	}

	wg.Wait()

	// Verify all runs were added
	expectedCount := numGoroutines * numRunsPerGoroutine
	if session.CountRuns() != expectedCount {
		t.Errorf("expected %d runs, got %d", expectedCount, session.CountRuns())
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = session.GetRuns()
			_ = session.GetHistory(5)
			_ = session.GetHistoryContext(3)
			_ = session.GetLastRun()
		}()
	}

	wg.Wait()
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkWorkflowSession_AddRun benchmarks adding runs to a session
func BenchmarkWorkflowSession_AddRun(b *testing.B) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")
	run := NewWorkflowRun("run-1", "session-1", "workflow-1", "input")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		session.AddRun(run)
	}
}

// BenchmarkWorkflowSession_GetHistory benchmarks getting history
func BenchmarkWorkflowSession_GetHistory(b *testing.B) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")

	// Add 100 completed runs
	for i := 0; i < 100; i++ {
		run := NewWorkflowRun("run-"+string(rune('0'+i)), "session-1", "workflow-1", "input")
		run.MarkCompleted("output")
		session.AddRun(run)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = session.GetHistory(10)
	}
}

// BenchmarkWorkflowSession_GetHistoryContext benchmarks getting history context
func BenchmarkWorkflowSession_GetHistoryContext(b *testing.B) {
	session := NewWorkflowSession("session-1", "workflow-1", "user-1")

	// Add 100 completed runs
	for i := 0; i < 100; i++ {
		run := NewWorkflowRun("run-"+string(rune('0'+i)), "session-1", "workflow-1", "input")
		run.MarkCompleted("output")
		session.AddRun(run)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = session.GetHistoryContext(10)
	}
}
