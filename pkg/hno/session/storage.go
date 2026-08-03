package session

import (
	"context"
	"errors"
)

var (
	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = errors.New("session not found")

	// ErrInvalidSessionID is returned when a session ID is invalid
	ErrInvalidSessionID = errors.New("invalid session ID")
)

// Storage defines the interface for session storage
type Storage interface {
	// Create creates a new session
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by ID
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Update updates an existing session
	Update(ctx context.Context, session *Session) error

	// Delete deletes a session by ID
	Delete(ctx context.Context, sessionID string) error

	// List lists all sessions (with optional filters)
	List(ctx context.Context, filters map[string]interface{}) ([]*Session, error)

	// ListByAgent lists all sessions for a specific agent
	ListByAgent(ctx context.Context, agentID string) ([]*Session, error)

	// ListByUser lists all sessions for a specific user
	ListByUser(ctx context.Context, userID string) ([]*Session, error)

	// Close closes the storage connection
	Close() error
}

// ensureContext surfaces context cancellation or deadline errors before proceeding
func ensureContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
