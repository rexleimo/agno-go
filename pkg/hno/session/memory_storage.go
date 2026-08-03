package session

import (
	"context"
	"sync"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
)

// MemoryStorage implements in-memory session storage
type MemoryStorage struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session
func (m *MemoryStorage) Create(ctx context.Context, session *Session) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}

	if session.SessionID == "" {
		return ErrInvalidSessionID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session already exists
	if _, exists := m.sessions[session.SessionID]; exists {
		// Update instead of creating
		return m.update(session)
	}

	// Set timestamps if not set
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	if session.UpdatedAt.IsZero() {
		session.UpdatedAt = time.Now()
	}

	// Deep copy to avoid external modifications
	m.sessions[session.SessionID] = m.deepCopy(session)

	return nil
}

// Get retrieves a session by ID
func (m *MemoryStorage) Get(ctx context.Context, sessionID string) (*Session, error) {
	if err := ensureContext(ctx); err != nil {
		return nil, err
	}

	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	// Return a deep copy to prevent external modification
	return m.deepCopy(session), nil
}

// Update updates an existing session
func (m *MemoryStorage) Update(ctx context.Context, session *Session) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}

	if session.SessionID == "" {
		return ErrInvalidSessionID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.update(session)
}

// update is the internal update method (caller must hold lock)
func (m *MemoryStorage) update(session *Session) error {
	if _, exists := m.sessions[session.SessionID]; !exists {
		return ErrSessionNotFound
	}

	// Update timestamp
	session.UpdatedAt = time.Now()

	// Deep copy to avoid external modifications
	m.sessions[session.SessionID] = m.deepCopy(session)

	return nil
}

// Delete deletes a session by ID
func (m *MemoryStorage) Delete(ctx context.Context, sessionID string) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}

	if sessionID == "" {
		return ErrInvalidSessionID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionID]; !exists {
		return ErrSessionNotFound
	}

	delete(m.sessions, sessionID)
	return nil
}

// List lists all sessions (with optional filters)
func (m *MemoryStorage) List(ctx context.Context, filters map[string]interface{}) ([]*Session, error) {
	if err := ensureContext(ctx); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Session

	for _, session := range m.sessions {
		if m.matchesFilters(session, filters) {
			result = append(result, m.deepCopy(session))
		}
	}

	return result, nil
}

// ListByAgent lists all sessions for a specific agent
func (m *MemoryStorage) ListByAgent(ctx context.Context, agentID string) ([]*Session, error) {
	if err := ensureContext(ctx); err != nil {
		return nil, err
	}

	return m.List(ctx, map[string]interface{}{
		"agent_id": agentID,
	})
}

// ListByUser lists all sessions for a specific user
func (m *MemoryStorage) ListByUser(ctx context.Context, userID string) ([]*Session, error) {
	if err := ensureContext(ctx); err != nil {
		return nil, err
	}

	return m.List(ctx, map[string]interface{}{
		"user_id": userID,
	})
}

// Close closes the storage (no-op for memory storage)
func (m *MemoryStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear all sessions
	m.sessions = make(map[string]*Session)
	return nil
}

// matchesFilters checks if a session matches the given filters
func (m *MemoryStorage) matchesFilters(session *Session, filters map[string]interface{}) bool {
	if filters == nil || len(filters) == 0 {
		return true
	}

	for key, value := range filters {
		switch key {
		case "agent_id":
			if session.AgentID != value.(string) {
				return false
			}
		case "user_id":
			if session.UserID != value.(string) {
				return false
			}
		case "team_id":
			if session.TeamID != value.(string) {
				return false
			}
		case "workflow_id":
			if session.WorkflowID != value.(string) {
				return false
			}
		}
	}

	return true
}

// deepCopy creates a deep copy of a session
func (m *MemoryStorage) deepCopy(session *Session) *Session {
	// Create a new session with copied primitive fields
	copy := &Session{
		SessionID:  session.SessionID,
		AgentID:    session.AgentID,
		TeamID:     session.TeamID,
		WorkflowID: session.WorkflowID,
		UserID:     session.UserID,
		Name:       session.Name,
		CreatedAt:  session.CreatedAt,
		UpdatedAt:  session.UpdatedAt,
	}

	// Deep copy maps
	if session.Metadata != nil {
		copy.Metadata = make(map[string]interface{}, len(session.Metadata))
		for k, v := range session.Metadata {
			copy.Metadata[k] = v
		}
	}

	if session.State != nil {
		copy.State = make(map[string]interface{}, len(session.State))
		for k, v := range session.State {
			copy.State[k] = v
		}
	}

	if session.AgentData != nil {
		copy.AgentData = make(map[string]interface{}, len(session.AgentData))
		for k, v := range session.AgentData {
			copy.AgentData[k] = v
		}
	}

	// Copy runs slice (note: runs themselves are not deep copied)
	if session.Runs != nil {
		copy.Runs = make([]*agent.RunOutput, len(session.Runs))
		copyInternal(copy.Runs, session.Runs)
	}

	// Copy summary
	if session.Summary != nil {
		copy.Summary = &SessionSummary{
			Content:     session.Summary.Content,
			RunCount:    session.Summary.RunCount,
			TotalTokens: session.Summary.TotalTokens,
			CreatedAt:   session.Summary.CreatedAt,
		}
	}

	return copy
}

// copyInternal copies elements from src to dst
func copyInternal(dst, src []*agent.RunOutput) {
	for i := range src {
		dst[i] = src[i]
	}
}
