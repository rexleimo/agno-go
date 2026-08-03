package guardrails

import (
	"context"
)

// Guardrail defines the interface for all guardrail implementations.
// Guardrails are used to validate and protect agent/team inputs and outputs.
type Guardrail interface {
	// Check performs synchronous guardrail validation.
	// It should return an error if the check fails.
	Check(ctx context.Context, input *CheckInput) error

	// Name returns the name of this guardrail for logging and debugging.
	Name() string
}

// CheckInput contains the data to be validated by a guardrail.
type CheckInput struct {
	// Input is the raw input string to validate
	Input string

	// Messages are the conversation messages (optional)
	Messages []interface{}

	// Metadata contains additional context for validation
	Metadata map[string]interface{}
}

// NewCheckInput creates a new CheckInput with the given input string.
func NewCheckInput(input string) *CheckInput {
	return &CheckInput{
		Input:    input,
		Metadata: make(map[string]interface{}),
	}
}

// WithMessages adds messages to the check input.
func (ci *CheckInput) WithMessages(messages []interface{}) *CheckInput {
	ci.Messages = messages
	return ci
}

// WithMetadata adds metadata to the check input.
func (ci *CheckInput) WithMetadata(metadata map[string]interface{}) *CheckInput {
	ci.Metadata = metadata
	return ci
}
