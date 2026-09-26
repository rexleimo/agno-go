package agent

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// RED test cycle assumes本周期will add:
// 1. Config.ToolCallLimit (does not exist)
// 2. RunOutput.StopReason (does not exist)
// 3. toolkit.Function.StopLoop bool (does not exist)
//
// Current compile failures are **expected** and constitute valid RED.

// fakeModel is scripted model for P0B RED tests.
type fakeModel struct {
	responses []*types.ModelResponse
	index     int
}

func (m *fakeModel) Invoke(_ context.Context, _ *models.InvokeRequest) (*types.ModelResponse, error) {
	if m.index >= len(m.responses) {
		return &types.ModelResponse{Content: "fallback"}, nil
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

func (m *fakeModel) InvokeStream(context.Context, *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	return nil, nil
}

func (m *fakeModel) GetProvider() string { return "fake" }
func (m *fakeModel) GetID() string       { return "fake-model" }
func (m *fakeModel) GetName() string     { return "Fake Model" }

// fakeToolkit returns toolkit with configurable tool behavior.
func fakeToolkit(stopLoop bool) *toolkit.BaseToolkit {
	tk := toolkit.NewBaseToolkit("fake")
	tk.RegisterFunction(&toolkit.Function{
		Name:        "tool1",
		Description: "fake tool 1",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "result1", nil
		},
	})
	tk.RegisterFunction(&toolkit.Function{
		Name:        "tool2",
		Description: "fake tool 2",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "result2", nil
		},
	})
	if stopLoop {
		tk.RegisterFunction(&toolkit.Function{
			Name:        "stopTool",
			Description: "tool that requests stop",
			Parameters:  map[string]toolkit.Parameter{},
			StopLoop:    true,
			Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
				return "stopped", nil
			},
		})
	}
	return tk
}

func toolCall(id, name string) types.ToolCall {
	return types.ToolCall{
		ID:   id,
		Type: "function",
		Function: types.ToolCallFunction{
			Name:      name,
			Arguments: `{}`,
		},
	}
}

// TestP0B_ToolCallLimitTruncatesBatch_SingleRound tests limit truncation in single round.
func TestP0B_ToolCallLimitTruncatesBatch_SingleRound(t *testing.T) {
	model := &fakeModel{
		responses: []*types.ModelResponse{
			{
				ToolCalls: []types.ToolCall{
					toolCall("call1", "tool1"),
					toolCall("call2", "tool2"),
					toolCall("call3", "tool1"),
				},
			},
		},
	}

	agent, err := New(Config{
		Name:          "test-agent",
		Model:         model,
		Toolkits:      []toolkit.Toolkit{fakeToolkit(false)},
		ToolCallLimit: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	output, err := agent.Run(context.Background(), "test input")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have 3 tool messages: 2 executed + 1 skipped
	toolMessages := 0
	var skippedMessage *types.Message
	for _, msg := range output.Messages {
		if msg.Role == types.RoleTool {
			toolMessages++
			if msg.Content == "tool call limit reached; call not executed" {
				skippedMessage = msg
			}
		}
	}

	if toolMessages != 3 {
		t.Errorf("Expected 3 tool messages (2 executed + 1 skipped), got %d", toolMessages)
	}

	if skippedMessage == nil {
		t.Error("Expected skipped tool message not found")
	}

	if skippedMessage != nil && skippedMessage.ToolCallID != "call3" {
		t.Errorf("Expected skipped message for call3, got ID %s", skippedMessage.ToolCallID)
	}

	if output.StopReason != "limit_reached" {
		t.Errorf("Expected StopReason 'limit_reached', got %q", output.StopReason)
	}
}

// TestP0B_ToolCallLimitTruncatesBatch_MultiRound tests limit exhaustion across rounds.
func TestP0B_ToolCallLimitTruncatesBatch_MultiRound(t *testing.T) {
	model := &fakeModel{
		responses: []*types.ModelResponse{
			{
				ToolCalls: []types.ToolCall{
					toolCall("call1", "tool1"),
				},
			},
			{
				ToolCalls: []types.ToolCall{
					toolCall("call2", "tool2"),
				},
			},
			{
				ToolCalls: []types.ToolCall{
					toolCall("call3", "tool1"),
				},
			},
		},
	}

	agent, err := New(Config{
		Name:          "test-agent",
		Model:         model,
		Toolkits:      []toolkit.Toolkit{fakeToolkit(false)},
		ToolCallLimit: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	output, err := agent.Run(context.Background(), "test input")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should have exactly 2 tool messages (limit reached before third)
	toolMessages := 0
	for _, msg := range output.Messages {
		if msg.Role == types.RoleTool {
			toolMessages++
		}
	}

	if toolMessages != 2 {
		t.Errorf("Expected exactly 2 tool messages (limit exhausted), got %d", toolMessages)
	}

	if output.StopReason != "limit_reached" {
		t.Errorf("Expected StopReason 'limit_reached', got %q", output.StopReason)
	}

	// Model should have been called exactly 3 times (initial + 2 rounds)
	if model.index != 3 {
		t.Errorf("Expected model called 3 times, got %d", model.index)
	}
}

// TestP0B_StopLoopHaltsBeforeNextModelCall tests stopLoop tool behavior.
func TestP0B_StopLoopHaltsBeforeNextModelCall(t *testing.T) {
	model := &fakeModel{
		responses: []*types.ModelResponse{
			{
				ToolCalls: []types.ToolCall{
					toolCall("call1", "stopTool"),
				},
			},
			{
				Content: "should not reach this",
			},
		},
	}

	agent, err := New(Config{
		Name:     "test-agent",
		Model:    model,
		Toolkits: []toolkit.Toolkit{fakeToolkit(true)},
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	output, err := agent.Run(context.Background(), "test input")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if output.StopReason != "stop_after_tool_call" {
		t.Errorf("Expected StopReason 'stop_after_tool_call', got %q", output.StopReason)
	}

	// Model should be called only once (stopLoop prevents second call)
	if model.index != 1 {
		t.Errorf("Expected model called once (stopLoop active), got %d times", model.index)
	}
}

// TestP0B_StopReasonObservable tests StopReason field presence in RunOutput.
func TestP0B_StopReasonObservable(t *testing.T) {
	model := &fakeModel{
		responses: []*types.ModelResponse{
			{
				Content: "final answer",
			},
		},
	}

	agent, err := New(Config{
		Name:  "test-agent",
		Model: model,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	output, err := agent.Run(context.Background(), "test input")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if output.StopReason != "no_tool_calls" {
		t.Errorf("Expected StopReason 'no_tool_calls' (final answer), got %q", output.StopReason)
	}
}
