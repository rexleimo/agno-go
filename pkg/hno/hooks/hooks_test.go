package hooks

import (
	"context"
	"errors"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/guardrails"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// MockGuardrail is a mock guardrail for testing
type MockGuardrail struct {
	name      string
	shouldErr bool
}

func (m *MockGuardrail) Check(ctx context.Context, input *guardrails.CheckInput) error {
	if m.shouldErr {
		return types.NewPromptInjectionError("mock error", nil)
	}
	return nil
}

func (m *MockGuardrail) Name() string {
	return m.name
}

func TestExecuteHook_WithGuardrail(t *testing.T) {
	tests := []struct {
		name      string
		guardrail *MockGuardrail
		input     string
		wantError bool
	}{
		{
			name:      "guardrail passes",
			guardrail: &MockGuardrail{name: "test", shouldErr: false},
			input:     "test input",
			wantError: false,
		},
		{
			name:      "guardrail fails",
			guardrail: &MockGuardrail{name: "test", shouldErr: true},
			input:     "test input",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hookInput := NewHookInput(tt.input)
			err := ExecuteHook(context.Background(), tt.guardrail, hookInput)

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestExecuteHook_WithFunction(t *testing.T) {
	var capturedInput string

	hookFunc := func(ctx context.Context, input *HookInput) error {
		capturedInput = input.Input
		if input.Input == "error" {
			return errors.New("hook error")
		}
		return nil
	}

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "function hook passes",
			input:     "test input",
			wantError: false,
		},
		{
			name:      "function hook fails",
			input:     "error",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hookInput := NewHookInput(tt.input)
			err := ExecuteHook(context.Background(), hookFunc, hookInput)

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if !tt.wantError && capturedInput != tt.input {
				t.Errorf("expected captured input %q, got %q", tt.input, capturedInput)
			}
		})
	}
}

func TestExecuteHooks(t *testing.T) {
	executionOrder := []string{}

	hook1 := func(ctx context.Context, input *HookInput) error {
		executionOrder = append(executionOrder, "hook1")
		return nil
	}

	hook2 := func(ctx context.Context, input *HookInput) error {
		executionOrder = append(executionOrder, "hook2")
		return nil
	}

	hook3 := func(ctx context.Context, input *HookInput) error {
		executionOrder = append(executionOrder, "hook3")
		return errors.New("hook3 error")
	}

	t.Run("all hooks execute successfully", func(t *testing.T) {
		executionOrder = []string{}
		hooks := []Hook{hook1, hook2}
		hookInput := NewHookInput("test")

		err := ExecuteHooks(context.Background(), hooks, hookInput)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if len(executionOrder) != 2 {
			t.Errorf("expected 2 hooks executed, got %d", len(executionOrder))
		}

		if executionOrder[0] != "hook1" || executionOrder[1] != "hook2" {
			t.Errorf("unexpected execution order: %v", executionOrder)
		}
	})

	t.Run("execution stops on error", func(t *testing.T) {
		executionOrder = []string{}
		hooks := []Hook{hook1, hook3, hook2}
		hookInput := NewHookInput("test")

		err := ExecuteHooks(context.Background(), hooks, hookInput)

		if err == nil {
			t.Error("expected error, got nil")
		}

		if len(executionOrder) != 2 {
			t.Errorf("expected 2 hooks executed before error, got %d", len(executionOrder))
		}

		if executionOrder[0] != "hook1" || executionOrder[1] != "hook3" {
			t.Errorf("unexpected execution order: %v", executionOrder)
		}
	})
}

func TestHookInput_Builders(t *testing.T) {
	input := NewHookInput("user input").
		WithOutput("agent output").
		WithAgentID("agent-123").
		WithMessages([]interface{}{"msg1", "msg2"}).
		WithMetadata(map[string]interface{}{"key": "value"})

	if input.Input != "user input" {
		t.Errorf("expected input %q, got %q", "user input", input.Input)
	}

	if input.Output != "agent output" {
		t.Errorf("expected output %q, got %q", "agent output", input.Output)
	}

	if input.AgentID != "agent-123" {
		t.Errorf("expected agent ID %q, got %q", "agent-123", input.AgentID)
	}

	if len(input.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(input.Messages))
	}

	if val, ok := input.Metadata["key"]; !ok || val != "value" {
		t.Errorf("expected metadata[key] = %q, got %v", "value", val)
	}
}

func TestExecuteHook_MixedHooks(t *testing.T) {
	guardrail := &MockGuardrail{name: "test-guardrail", shouldErr: false}
	functionHook := func(ctx context.Context, input *HookInput) error {
		return nil
	}

	hooks := []Hook{
		guardrail,
		functionHook,
	}

	hookInput := NewHookInput("test input")
	err := ExecuteHooks(context.Background(), hooks, hookInput)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
