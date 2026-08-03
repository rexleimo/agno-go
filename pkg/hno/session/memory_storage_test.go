package session

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
)

func TestNewMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()

	if storage == nil {
		t.Fatal("Storage should not be nil")
	}

	if storage.sessions == nil {
		t.Error("Sessions map should be initialized")
	}
}

func TestMemoryStorage_Create(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	session := NewSession("sess-1", "agent-1")

	err := storage.Create(ctx, session)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify session was stored
	retrieved, err := storage.Get(ctx, "sess-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.SessionID != "sess-1" {
		t.Errorf("SessionID = %v, want 'sess-1'", retrieved.SessionID)
	}
}

func TestMemoryStorage_Create_EmptyID(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	session := &Session{SessionID: ""}

	err := storage.Create(ctx, session)
	if err != ErrInvalidSessionID {
		t.Errorf("Create() error = %v, want ErrInvalidSessionID", err)
	}
}

func TestMemoryStorage_Create_Duplicate(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	session1 := NewSession("sess-1", "agent-1")
	session1.Name = "First"

	err := storage.Create(ctx, session1)
	if err != nil {
		t.Fatalf("First Create() error = %v", err)
	}

	// Try to create with same ID (should update)
	session2 := NewSession("sess-1", "agent-1")
	session2.Name = "Second"

	err = storage.Create(ctx, session2)
	if err != nil {
		t.Fatalf("Second Create() error = %v", err)
	}

	// Verify updated
	retrieved, _ := storage.Get(ctx, "sess-1")
	if retrieved.Name != "Second" {
		t.Errorf("Name = %v, want 'Second'", retrieved.Name)
	}
}

func TestMemoryStorage_Get(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	session := NewSession("sess-1", "agent-1")
	session.Name = "Test Session"

	storage.Create(ctx, session)

	retrieved, err := storage.Get(ctx, "sess-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.SessionID != "sess-1" {
		t.Errorf("SessionID = %v, want 'sess-1'", retrieved.SessionID)
	}

	if retrieved.Name != "Test Session" {
		t.Errorf("Name = %v, want 'Test Session'", retrieved.Name)
	}
}

func TestMemoryStorage_Get_NotFound(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_, err := storage.Get(ctx, "non-existent")
	if err != ErrSessionNotFound {
		t.Errorf("Get() error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStorage_Get_EmptyID(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_, err := storage.Get(ctx, "")
	if err != ErrInvalidSessionID {
		t.Errorf("Get() error = %v, want ErrInvalidSessionID", err)
	}
}

func TestMemoryStorage_Update(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	session := NewSession("sess-1", "agent-1")
	session.Name = "Original"

	storage.Create(ctx, session)

	// Update session
	session.Name = "Updated"
	session.UserID = "user-123"

	err := storage.Update(ctx, session)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	retrieved, _ := storage.Get(ctx, "sess-1")
	if retrieved.Name != "Updated" {
		t.Errorf("Name = %v, want 'Updated'", retrieved.Name)
	}

	if retrieved.UserID != "user-123" {
		t.Errorf("UserID = %v, want 'user-123'", retrieved.UserID)
	}
}

func TestMemoryStorage_Update_NotFound(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	session := NewSession("non-existent", "agent-1")

	err := storage.Update(ctx, session)
	if err != ErrSessionNotFound {
		t.Errorf("Update() error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	session := NewSession("sess-1", "agent-1")
	storage.Create(ctx, session)

	// Delete session
	err := storage.Delete(ctx, "sess-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err = storage.Get(ctx, "sess-1")
	if err != ErrSessionNotFound {
		t.Errorf("After delete, Get() error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStorage_Delete_NotFound(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	err := storage.Delete(ctx, "non-existent")
	if err != ErrSessionNotFound {
		t.Errorf("Delete() error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStorage_List(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Create multiple sessions
	storage.Create(ctx, NewSession("sess-1", "agent-1"))
	storage.Create(ctx, NewSession("sess-2", "agent-2"))
	storage.Create(ctx, NewSession("sess-3", "agent-1"))

	// List all sessions
	sessions, err := storage.List(ctx, nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("List count = %d, want 3", len(sessions))
	}
}

func TestMemoryStorage_ContextCancellation(t *testing.T) {
	storage := NewMemoryStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := storage.Create(ctx, NewSession("ctx-session", "agent"))
	if err == nil {
		t.Fatalf("expected context cancellation error, got nil")
	}

	if err != context.Canceled && err != context.DeadlineExceeded {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMemoryStorage_List_WithFilters(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Create sessions with different agents
	sess1 := NewSession("sess-1", "agent-1")
	sess1.UserID = "user-1"
	storage.Create(ctx, sess1)

	sess2 := NewSession("sess-2", "agent-2")
	sess2.UserID = "user-2"
	storage.Create(ctx, sess2)

	sess3 := NewSession("sess-3", "agent-1")
	sess3.UserID = "user-1"
	storage.Create(ctx, sess3)

	// Filter by agent_id
	sessions, err := storage.List(ctx, map[string]interface{}{
		"agent_id": "agent-1",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("List count with agent_id filter = %d, want 2", len(sessions))
	}

	// Filter by user_id
	sessions, err = storage.List(ctx, map[string]interface{}{
		"user_id": "user-2",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("List count with user_id filter = %d, want 1", len(sessions))
	}
}

func TestMemoryStorage_ListByAgent(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Create sessions with different agents
	storage.Create(ctx, NewSession("sess-1", "agent-1"))
	storage.Create(ctx, NewSession("sess-2", "agent-2"))
	storage.Create(ctx, NewSession("sess-3", "agent-1"))

	sessions, err := storage.ListByAgent(ctx, "agent-1")
	if err != nil {
		t.Fatalf("ListByAgent() error = %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("ListByAgent count = %d, want 2", len(sessions))
	}

	// Verify all sessions are for agent-1
	for _, sess := range sessions {
		if sess.AgentID != "agent-1" {
			t.Errorf("Session AgentID = %v, want 'agent-1'", sess.AgentID)
		}
	}
}

func TestMemoryStorage_ListByUser(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Create sessions with different users
	sess1 := NewSession("sess-1", "agent-1")
	sess1.UserID = "user-1"
	storage.Create(ctx, sess1)

	sess2 := NewSession("sess-2", "agent-1")
	sess2.UserID = "user-2"
	storage.Create(ctx, sess2)

	sess3 := NewSession("sess-3", "agent-1")
	sess3.UserID = "user-1"
	storage.Create(ctx, sess3)

	sessions, err := storage.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("ListByUser count = %d, want 2", len(sessions))
	}

	// Verify all sessions are for user-1
	for _, sess := range sessions {
		if sess.UserID != "user-1" {
			t.Errorf("Session UserID = %v, want 'user-1'", sess.UserID)
		}
	}
}

func TestMemoryStorage_Close(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Create some sessions
	storage.Create(ctx, NewSession("sess-1", "agent-1"))
	storage.Create(ctx, NewSession("sess-2", "agent-2"))

	err := storage.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Verify sessions are cleared
	sessions, _ := storage.List(ctx, nil)
	if len(sessions) != 0 {
		t.Errorf("After close, session count = %d, want 0", len(sessions))
	}
}

func TestMemoryStorage_DeepCopy(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Create session with runs
	session := NewSession("sess-1", "agent-1")
	session.Metadata["key"] = "value"
	session.AddRun(&agent.RunOutput{Content: "Test"})

	storage.Create(ctx, session)

	// Get session and modify it
	retrieved, _ := storage.Get(ctx, "sess-1")
	retrieved.Metadata["key"] = "modified"
	retrieved.AddRun(&agent.RunOutput{Content: "Modified"})

	// Get again and verify original is unchanged
	original, _ := storage.Get(ctx, "sess-1")
	if original.Metadata["key"] != "value" {
		t.Error("Deep copy failed: metadata was modified")
	}

	if len(original.Runs) != 1 {
		t.Error("Deep copy failed: runs were modified")
	}
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Create initial session
	storage.Create(ctx, NewSession("sess-1", "agent-1"))

	// Concurrent reads and writes
	done := make(chan bool, 10)

	for i := 0; i < 5; i++ {
		go func() {
			_, err := storage.Get(ctx, "sess-1")
			if err != nil {
				t.Errorf("Concurrent Get() error = %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		go func() {
			session := NewSession("sess-1", "agent-1")
			err := storage.Update(ctx, session)
			if err != nil {
				t.Errorf("Concurrent Update() error = %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
