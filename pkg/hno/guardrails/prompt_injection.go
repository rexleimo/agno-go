package guardrails

import (
	"context"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// PromptInjectionGuardrail detects potential prompt injection attempts.
type PromptInjectionGuardrail struct {
	// InjectionPatterns contains patterns to check for prompt injection
	InjectionPatterns []string

	// CaseSensitive determines if pattern matching should be case-sensitive
	CaseSensitive bool
}

// DefaultInjectionPatterns returns a list of common prompt injection patterns
func DefaultInjectionPatterns() []string {
	return []string{
		"ignore previous instructions",
		"ignore your instructions",
		"you are now a",
		"forget everything above",
		"developer mode",
		"override safety",
		"disregard guidelines",
		"system prompt",
		"jailbreak",
		"act as if",
		"pretend you are",
		"roleplay as",
		"simulate being",
		"bypass restrictions",
		"ignore safeguards",
		"admin override",
		"root access",
	}
}

// NewPromptInjectionGuardrail creates a new prompt injection guardrail with default patterns.
func NewPromptInjectionGuardrail() *PromptInjectionGuardrail {
	return &PromptInjectionGuardrail{
		InjectionPatterns: DefaultInjectionPatterns(),
		CaseSensitive:     false,
	}
}

// NewPromptInjectionGuardrailWithPatterns creates a new guardrail with custom patterns.
func NewPromptInjectionGuardrailWithPatterns(patterns []string, caseSensitive bool) *PromptInjectionGuardrail {
	return &PromptInjectionGuardrail{
		InjectionPatterns: patterns,
		CaseSensitive:     caseSensitive,
	}
}

// Check validates the input for prompt injection patterns.
func (g *PromptInjectionGuardrail) Check(ctx context.Context, input *CheckInput) error {
	inputText := input.Input
	if !g.CaseSensitive {
		inputText = strings.ToLower(inputText)
	}

	for _, pattern := range g.InjectionPatterns {
		checkPattern := pattern
		if !g.CaseSensitive {
			checkPattern = strings.ToLower(pattern)
		}

		if strings.Contains(inputText, checkPattern) {
			return types.NewPromptInjectionError(
				"Potential jailbreaking or prompt injection detected",
				nil,
			)
		}
	}

	return nil
}

// Name returns the name of this guardrail.
func (g *PromptInjectionGuardrail) Name() string {
	return "PromptInjectionGuardrail"
}
