package workflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage(100)

	if storage == nil {
		t.Fatal("expected non-nil storage")
	}

	if storage.sessions == nil {
		t.Error("expected sessions map to be initialized")
	}

	if storage.maxSize != 100 {
		t.Errorf("expected maxSize 100, got %d", storage.maxSize)
	}
}

func TestMemoryStorage_CreateSession(t *testing.T) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	// Create session
	session, err := storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.SessionID != "session-1" {
		t.Errorf("expected SessionID 'session-1', got '%s'", session.SessionID)
	}

	// Try to create duplicate session
	_, err = storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != ErrSessionExists {
		t.Errorf("expected ErrSessionExists, got %v", err)
	}

	// Test invalid session ID
	_, err = storage.CreateSession(ctx, "", "workflow-1", "user-1")
	if err != ErrInvalidSessionID {
		t.Errorf("expected ErrInvalidSessionID, got %v", err)
	}

	// Test invalid workflow ID
	_, err = storage.CreateSession(ctx, "session-2", "", "user-1")
	if err != ErrInvalidWorkflowID {
		t.Errorf("expected ErrInvalidWorkflowID, got %v", err)
	}
}

func TestMemoryStorage_GetSession(t *testing.T) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	// Create session
	created, err := storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get session
	retrieved, err := storage.GetSession(ctx, "session-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.SessionID != created.SessionID {
		t.Error("retrieved session does not match created session")
	}

	// Get non-existent session
	_, err = storage.GetSession(ctx, "non-existent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}

	// Test invalid session ID
	_, err = storage.GetSession(ctx, "")
	if err != ErrInvalidSessionID {
		t.Errorf("expected ErrInvalidSessionID, got %v", err)
	}
}

func TestMemoryStorage_UpdateSession(t *testing.T) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	// Create session
	session, err := storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Add a run to the session
	run := NewWorkflowRun("run-1", "session-1", "workflow-1", "input")
	session.AddRun(run)

	// Update session
	err = storage.UpdateSession(ctx, session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetSession(ctx, "session-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.CountRuns() != 1 {
		t.Errorf("expected 1 run, got %d", retrieved.CountRuns())
	}

	// Try to update non-existent session
	nonExistent := NewWorkflowSession("non-existent", "workflow-1", "user-1")
	err = storage.UpdateSession(ctx, nonExistent)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}

	// Test nil session
	err = storage.UpdateSession(ctx, nil)
	if err != ErrInvalidSessionID {
		t.Errorf("expected ErrInvalidSessionID, got %v", err)
	}
}

func TestMemoryStorage_DeleteSession(t *testing.T) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	// Create session
	_, err := storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Delete session
	err = storage.DeleteSession(ctx, "session-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify deletion
	_, err = storage.GetSession(ctx, "session-1")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}

	// Try to delete non-existent session
	err = storage.DeleteSession(ctx, "non-existent")
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}

	// Test invalid session ID
	err = storage.DeleteSession(ctx, "")
	if err != ErrInvalidSessionID {
		t.Errorf("expected ErrInvalidSessionID, got %v", err)
	}
}

func TestMemoryStorage_ListSessions(t *testing.T) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	// Create sessions for multiple workflows
	_, err := storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = storage.CreateSession(ctx, "session-2", "workflow-1", "user-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = storage.CreateSession(ctx, "session-3", "workflow-2", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// List sessions for workflow-1
	sessions, err := storage.ListSessions(ctx, "workflow-1", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions for workflow-1, got %d", len(sessions))
	}

	// List sessions for workflow-2
	sessions, err = storage.ListSessions(ctx, "workflow-2", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 session for workflow-2, got %d", len(sessions))
	}

	// Test pagination with limit
	sessions, err = storage.ListSessions(ctx, "workflow-1", 1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 session with limit=1, got %d", len(sessions))
	}

	// Test pagination with offset
	sessions, err = storage.ListSessions(ctx, "workflow-1", 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 session with offset=1, got %d", len(sessions))
	}

	// Test invalid workflow ID
	_, err = storage.ListSessions(ctx, "", 0, 0)
	if err != ErrInvalidWorkflowID {
		t.Errorf("expected ErrInvalidWorkflowID, got %v", err)
	}
}

func TestMemoryStorage_ListUserSessions(t *testing.T) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	// Create sessions for multiple users
	_, err := storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = storage.CreateSession(ctx, "session-2", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = storage.CreateSession(ctx, "session-3", "workflow-2", "user-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// List sessions for user-1
	sessions, err := storage.ListUserSessions(ctx, "user-1", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions for user-1, got %d", len(sessions))
	}

	// List sessions for user-2
	sessions, err = storage.ListUserSessions(ctx, "user-2", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 session for user-2, got %d", len(sessions))
	}

	// Test pagination
	sessions, err = storage.ListUserSessions(ctx, "user-1", 1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 session with limit=1, got %d", len(sessions))
	}
}

func TestMemoryStorage_Clear(t *testing.T) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	// Create sessions
	for i := 1; i <= 3; i++ {
		_, err := storage.CreateSession(ctx, "session-"+string(rune('0'+i)), "workflow-1", "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Clear sessions older than 5ms (should clear all)
	count, err := storage.Clear(ctx, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 3 {
		t.Errorf("expected to clear 3 sessions, got %d", count)
	}

	// Verify all sessions were cleared
	sessions, err := storage.ListSessions(ctx, "workflow-1", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions after clear, got %d", len(sessions))
	}
}

func TestMemoryStorage_MaxSize(t *testing.T) {
	storage := NewMemoryStorage(2) // Max 2 sessions
	ctx := context.Background()

	// Create 3 sessions (should evict oldest)
	_, err := storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = storage.CreateSession(ctx, "session-2", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = storage.CreateSession(ctx, "session-3", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// session-1 should have been evicted
	_, err = storage.GetSession(ctx, "session-1")
	if err != ErrSessionNotFound {
		t.Error("expected session-1 to be evicted")
	}

	// session-2 and session-3 should exist
	_, err = storage.GetSession(ctx, "session-2")
	if err != nil {
		t.Errorf("expected session-2 to exist, got error: %v", err)
	}

	_, err = storage.GetSession(ctx, "session-3")
	if err != nil {
		t.Errorf("expected session-3 to exist, got error: %v", err)
	}
}

func TestMemoryStorage_GetStats(t *testing.T) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	// Create session with runs
	session, err := storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Add completed run
	run1 := NewWorkflowRun("run-1", "session-1", "workflow-1", "input-1")
	run1.MarkStarted()
	time.Sleep(10 * time.Millisecond)
	run1.MarkCompleted("output-1")
	session.AddRun(run1)

	// Add failed run
	run2 := NewWorkflowRun("run-2", "session-1", "workflow-1", "input-2")
	run2.MarkStarted()
	run2.MarkFailed(errors.New("test error"))
	session.AddRun(run2)

	// Update session in storage
	err = storage.UpdateSession(ctx, session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get stats
	stats, err := storage.GetStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.TotalSessions != 1 {
		t.Errorf("expected 1 total session, got %d", stats.TotalSessions)
	}

	if stats.TotalRuns != 2 {
		t.Errorf("expected 2 total runs, got %d", stats.TotalRuns)
	}

	if stats.CompletedRuns != 2 {
		t.Errorf("expected 2 completed runs, got %d", stats.CompletedRuns)
	}

	if stats.SuccessfulRuns != 1 {
		t.Errorf("expected 1 successful run, got %d", stats.SuccessfulRuns)
	}

	if stats.FailedRuns != 1 {
		t.Errorf("expected 1 failed run, got %d", stats.FailedRuns)
	}

	if stats.AverageDuration <= 0 {
		t.Error("expected positive average duration")
	}
}

func TestMemoryStorage_GetWorkflowStats(t *testing.T) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	// Create sessions for different workflows
	session1, err := storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	session2, err := storage.CreateSession(ctx, "session-2", "workflow-2", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Add runs to workflow-1
	run1 := NewWorkflowRun("run-1", "session-1", "workflow-1", "input-1")
	run1.MarkCompleted("output-1")
	session1.AddRun(run1)
	storage.UpdateSession(ctx, session1)

	// Add runs to workflow-2
	run2 := NewWorkflowRun("run-2", "session-2", "workflow-2", "input-2")
	run2.MarkCompleted("output-2")
	session2.AddRun(run2)

	run3 := NewWorkflowRun("run-3", "session-2", "workflow-2", "input-3")
	run3.MarkCompleted("output-3")
	session2.AddRun(run3)
	storage.UpdateSession(ctx, session2)

	// Get stats for workflow-1
	stats1, err := storage.GetWorkflowStats(ctx, "workflow-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats1.TotalSessions != 1 {
		t.Errorf("expected 1 session for workflow-1, got %d", stats1.TotalSessions)
	}

	if stats1.TotalRuns != 1 {
		t.Errorf("expected 1 run for workflow-1, got %d", stats1.TotalRuns)
	}

	// Get stats for workflow-2
	stats2, err := storage.GetWorkflowStats(ctx, "workflow-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats2.TotalSessions != 1 {
		t.Errorf("expected 1 session for workflow-2, got %d", stats2.TotalSessions)
	}

	if stats2.TotalRuns != 2 {
		t.Errorf("expected 2 runs for workflow-2, got %d", stats2.TotalRuns)
	}

	// Test invalid workflow ID
	_, err = storage.GetWorkflowStats(ctx, "")
	if err != ErrInvalidWorkflowID {
		t.Errorf("expected ErrInvalidWorkflowID, got %v", err)
	}
}

func TestMemoryStorage_Close(t *testing.T) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	// Create sessions
	_, err := storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Close storage
	err = storage.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify sessions were cleared
	_, err = storage.GetSession(ctx, "session-1")
	if err != ErrSessionNotFound {
		t.Error("expected all sessions to be cleared after close")
	}
}

func TestMemoryStorage_ContextCancellation(t *testing.T) {
	storage := NewMemoryStorage(0)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// All operations should return context error
	_, err := storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Create a session with normal context
	normalCtx := context.Background()
	storage.CreateSession(normalCtx, "session-1", "workflow-1", "user-1")

	// Try operations with cancelled context
	_, err = storage.GetSession(ctx, "session-1")
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// BenchmarkMemoryStorage_CreateSession benchmarks session creation
func BenchmarkMemoryStorage_CreateSession(b *testing.B) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		storage.CreateSession(ctx, "session-"+string(rune('0'+i)), "workflow-1", "user-1")
	}
}

// BenchmarkMemoryStorage_GetSession benchmarks session retrieval
func BenchmarkMemoryStorage_GetSession(b *testing.B) {
	storage := NewMemoryStorage(0)
	ctx := context.Background()

	// Create test session
	storage.CreateSession(ctx, "session-1", "workflow-1", "user-1")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		storage.GetSession(ctx, "session-1")
	}
}
