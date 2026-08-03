package hooks

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/guardrails"
)

// Hook represents a function that can be called before or after agent/team execution.
// Hooks can be either:
// 1. A regular function: func(ctx context.Context, input *HookInput) error
// 2. A Guardrail: implements guardrails.Guardrail interface
type Hook interface{}

// HookInput contains the data passed to hook functions.
type HookInput struct {
	// Input is the user input string
	Input string

	// Output is the agent/team output (only available for post-hooks)
	Output string

	// Messages are the conversation messages
	Messages []interface{}

	// Metadata contains additional context
	Metadata map[string]interface{}

	// AgentID is the ID of the agent/team
	AgentID string
}

// HookFunc is a function type for hooks
type HookFunc func(ctx context.Context, input *HookInput) error

// NewHookInput creates a new HookInput.
func NewHookInput(userInput string) *HookInput {
	return &HookInput{
		Input:    userInput,
		Metadata: make(map[string]interface{}),
	}
}

// WithOutput adds output to the hook input (for post-hooks).
func (hi *HookInput) WithOutput(output string) *HookInput {
	hi.Output = output
	return hi
}

// WithMessages adds messages to the hook input.
func (hi *HookInput) WithMessages(messages []interface{}) *HookInput {
	hi.Messages = messages
	return hi
}

// WithMetadata adds metadata to the hook input.
func (hi *HookInput) WithMetadata(metadata map[string]interface{}) *HookInput {
	hi.Metadata = metadata
	return hi
}

// WithAgentID adds agent ID to the hook input.
func (hi *HookInput) WithAgentID(agentID string) *HookInput {
	hi.AgentID = agentID
	return hi
}

// ExecuteHook executes a single hook, handling both function hooks and guardrail hooks.
func ExecuteHook(ctx context.Context, hook Hook, input *HookInput) error {
	// Check if it's a Guardrail
	if guardrail, ok := hook.(guardrails.Guardrail); ok {
		checkInput := &guardrails.CheckInput{
			Input:    input.Input,
			Messages: input.Messages,
			Metadata: input.Metadata,
		}
		return guardrail.Check(ctx, checkInput)
	}

	// Check if it's a HookFunc
	if hookFunc, ok := hook.(HookFunc); ok {
		return hookFunc(ctx, input)
	}

	// If it's a function with the right signature, try to call it
	if fn, ok := hook.(func(context.Context, *HookInput) error); ok {
		return fn(ctx, input)
	}

	return nil
}

// ExecuteHooks executes a list of hooks in order.
// If any hook returns an error, execution stops and the error is returned.
func ExecuteHooks(ctx context.Context, hooks []Hook, input *HookInput) error {
	for _, hook := range hooks {
		if err := ExecuteHook(ctx, hook, input); err != nil {
			return err
		}
	}
	return nil
}
