package run

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type contextKey string

const (
	ctxKeyRunContext contextKey = "hno.run_context"
)

// RunContext carries correlation identifiers that allow multiple agents or
// teams to participate in the same logical execution.
type RunContext struct {
	RunID       string                 `json:"run_id,omitempty"`
	ParentRunID string                 `json:"parent_run_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	WorkflowID  string                 `json:"workflow_id,omitempty"`
	TeamID      string                 `json:"team_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	mu          sync.RWMutex
}

// NewContext constructs an empty run context.
func NewContext() *RunContext {
	return &RunContext{}
}

// Clone performs a shallow copy of the run context including metadata map.
func (rc *RunContext) Clone() *RunContext {
	if rc == nil {
		return nil
	}
	copyCtx := *rc
	if rc.Metadata != nil {
		copyCtx.Metadata = make(map[string]interface{}, len(rc.Metadata))
		for k, v := range rc.Metadata {
			copyCtx.Metadata[k] = v
		}
	}
	return &copyCtx
}

// EnsureRunID initialises the run identifier when absent and returns it.
func (rc *RunContext) EnsureRunID() string {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if rc.RunID == "" {
		rc.RunID = "run-" + uuid.New().String()
	}
	return rc.RunID
}

// WithContext stores the run context in the parent context.
func WithContext(ctx context.Context, rc *RunContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if rc == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyRunContext, rc)
}

// FromContext extracts the run context from the provided context.
func FromContext(ctx context.Context) (*RunContext, bool) {
	if ctx == nil {
		return nil, false
	}
	rc, ok := ctx.Value(ctxKeyRunContext).(*RunContext)
	return rc, ok && rc != nil
}
