package tool

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
)

// Registry keeps a simple registry of tool handlers keyed by name.
type Registry struct {
	handlers map[string]Handler
}

// Handler executes a tool call and returns result.
type Handler func(ctx context.Context, call agent.ToolCall) (string, error)

// NewRegistry builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// Register adds or replaces a handler.
func (r *Registry) Register(name string, h Handler) {
	if r.handlers == nil {
		r.handlers = make(map[string]Handler)
	}
	r.handlers[name] = h
}

// Invoke executes a tool call with basic timeout and status filling.
func (r *Registry) Invoke(ctx context.Context, call agent.ToolCall, timeout time.Duration) agent.ToolCallResult {
	result := agent.ToolCallResult{
		ToolCallID:  call.ToolCallID,
		Status:      agent.ToolStatusError,
		CompletedAt: time.Now().UTC(),
	}
	h, ok := r.handlers[call.Name]
	if !ok {
		result.Error = "tool not registered"
		return result
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	out, err := h(callCtx, call)
	result.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Status = agent.ToolStatusSuccess
	result.Output = out
	return result
}

// GuardrailHook is a simple hook to block PII/prompt injection; here we only simulate flagging by keyword.
func GuardrailHook(content string) error {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "password") || strings.Contains(lower, "ssn") {
		return errors.New("guardrail: sensitive content detected")
	}
	return nil
}
