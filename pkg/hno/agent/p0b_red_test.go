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

	// 逐条校验身份与内容：前两条是真实执行结果，第三条才是被跳过的。
	// 只断言条数无法区分「正确执行了 2 个」与「一个都没执行」。
	// Verify identity and content per message: two real results, one skipped.
	gotContent := map[string]string{}
	for _, m := range output.Messages {
		if m.Role == types.RoleTool {
			gotContent[m.ToolCallID] = m.Content
		}
	}
	wantContent := map[string]string{
		// toolkit.FormatResult 对 handler 返回值做 json.Marshal，字符串会带上引号；
		// call3 的跳过消息由 runner 直接构造，不经格式化。
		// FormatResult JSON-marshals handler returns, so strings arrive quoted;
		// call3's skip message is built by runner and is not formatted.
		"call1": "\"result1\"",
		"call2": "\"result2\"",
		"call3": "tool call limit reached; call not executed",
	}
	for _, id := range []string{"call1", "call2", "call3"} {
		if gotContent[id] != wantContent[id] {
			t.Errorf("call %s: expected %q, got %q", id, wantContent[id], gotContent[id])
		}
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

	// The limit stops the loop before call3 runs, but call3 was still requested
	// by the model, so it must come back paired with a tool message.
	// 上限在 call3 执行前终止了循环，但 call3 仍是模型请求过的调用，
	// 因此它必须带回一条配对的 tool 消息。
	toolMessages := 0
	for _, msg := range output.Messages {
		if msg.Role == types.RoleTool {
			toolMessages++
		}
	}

	if toolMessages != 3 {
		t.Errorf("Expected exactly 3 tool messages (2 executed + 1 paired skip), got %d", toolMessages)
	}

	if output.StopReason != "limit_reached" {
		t.Errorf("Expected StopReason 'limit_reached', got %q", output.StopReason)
	}

	// Model should have been called exactly 3 times (initial + 2 rounds)
	if model.index != 3 {
		t.Errorf("Expected model called 3 times, got %d", model.index)
	}

	// 跨轮消耗掉的两条调用是 call1、call2，第三条必须是被截断后的配对文案。
	// 只数条数会漏掉「数量对、身份错」的情况。
	// The two calls consumed across rounds must be call1 and call2, and call3 must
	// carry the paired truncation message. A bare count misses "right size, wrong identity".
	gotRound := map[string]string{}
	for _, m := range output.Messages {
		if m.Role == types.RoleTool {
			gotRound[m.ToolCallID] = m.Content
		}
	}
	for id, w := range map[string]string{
		"call1": "\"result1\"",
		"call2": "\"result2\"",
		"call3": "tool call limit reached; call not executed",
	} {
		if gotRound[id] != w {
			t.Errorf("call %s: expected %q, got %q", id, w, gotRound[id])
		}
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

	// 工具结果必须保留在消息中，不能因 stopLoop 而丢弃。
	// 此前该场景下「终止循环但丢弃工具输出」的实现可以完全通过测试。
	// The tool result must survive in the message stream; stopLoop must not discard it.
	// stopTool 的结果同样经 FormatResult 序列化，故为带引号的 "\"stopped\""。
	// stopTool's result is also JSON-marshalled, hence the quoted form.
	var keptResult bool
	for _, m := range output.Messages {
		if m.Role == types.RoleTool && m.ToolCallID == "call1" && m.Content == "\"stopped\"" {
			keptResult = true
		}
	}
	if !keptResult {
		t.Error("stopTool result message should be preserved in output.Messages")
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
