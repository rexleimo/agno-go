package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/run"
	"github.com/rexleimo/agno-go/pkg/agno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

// streamMockModel simulates a streaming model: the first invocation streams a
// tool call, the second streams a final text answer.
// streamMockModel 模拟流式模型：第一次调用流式返回工具调用，第二次流式返回最终文本答案。
type streamMockModel struct {
	invocations int
	toolName    string
	finalText   string
	err         error
}

func (m *streamMockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	return nil, errors.New("unexpected sync invoke")
}

func (m *streamMockModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan types.ResponseChunk)
	m.invocations++

	go func() {
		defer close(ch)
		if m.invocations == 1 {
			// First round: emit a tool call in two chunks.
			// 第一轮：分两块发出一个工具调用。
			ch <- types.ResponseChunk{ToolCalls: []types.ToolCall{{
				ID:   "call_stream_1",
				Type: "function",
				Function: types.ToolCallFunction{
					Name:      m.toolName,
					Arguments: `{"a":1,"b":2}`,
				},
			}}}
			return
		}
		// Second round: stream final text in two chunks.
		// 第二轮：分两块流式返回最终文本。
		ch <- types.ResponseChunk{Content: "final "}
		ch <- types.ResponseChunk{Content: "answer"}
	}()

	return ch, nil
}

func (m *streamMockModel) GetProvider() string { return "mock" }
func (m *streamMockModel) GetID() string       { return "stream-mock" }
func (m *streamMockModel) GetName() string     { return "Stream Mock" }

// echoToolkit registers a single tool that echoes its arguments.
// echoToolkit 注册一个回显其参数的工具。
type echoToolkit struct {
	*toolkit.BaseToolkit
}

func newEchoToolkit() *echoToolkit {
	t := &echoToolkit{BaseToolkit: toolkit.NewBaseToolkit("echo")}
	t.RegisterFunction(&toolkit.Function{
		Name:        "echo_add",
		Description: "adds two numbers and returns the sum",
		Parameters: map[string]toolkit.Parameter{
			"a": {Type: "number", Description: "first number", Required: true},
			"b": {Type: "number", Description: "second number", Required: true},
		},
		Handler: func(_ context.Context, args map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{"sum": 3}, nil
		},
	})
	return t
}

func TestRunStreamExecutesToolCalls(t *testing.T) {
	model := &streamMockModel{toolName: "echo_add", finalText: "final answer"}
	ag, err := New(Config{
		Name:     "stream-tool-agent",
		Model:    model,
		Toolkits: []toolkit.Toolkit{newEchoToolkit()},
		MaxLoops: 5,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result, err := ag.RunStream(context.Background(), "compute 1+2")
	if err != nil {
		t.Fatalf("RunStream failed: %v", err)
	}

	// Collect events.
	var contents []string
	for evt := range result.Events {
		if content, ok := evt.(*run.RunContentEvent); ok {
			contents = append(contents, content.Content)
		}
	}

	done := <-result.Done
	if done.Err != nil {
		t.Fatalf("stream done error: %v", done.Err)
	}
	if done.Output == nil {
		t.Fatal("expected output, got nil")
	}
	if done.Output.Content != "final answer" {
		t.Errorf("expected 'final answer', got %q", done.Output.Content)
	}

	// The tool must have been executed: memory should contain a tool message.
	// 工具必须被执行：memory 中应包含工具消息。
	foundToolMsg := false
	for _, msg := range ag.Memory.GetMessages(ag.UserID) {
		if msg.Role == types.RoleTool && msg.ToolCallID == "call_stream_1" {
			foundToolMsg = true
			break
		}
	}
	if !foundToolMsg {
		t.Error("expected tool result message in memory after streaming tool call")
	}

	// Two model invocations must have happened.
	if model.invocations != 2 {
		t.Errorf("expected 2 model invocations, got %d", model.invocations)
	}

	// Content events should reflect the streamed final text.
	joined := ""
	for _, c := range contents {
		joined += c
	}
	if joined != "final answer" {
		t.Errorf("expected streamed content 'final answer', got %q", joined)
	}
}

func TestRunStreamCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled / 预先取消

	model := &streamMockModel{toolName: "echo_add", finalText: "final answer"}
	ag, err := New(Config{
		Name:     "stream-cancel-agent",
		Model:    model,
		Toolkits: []toolkit.Toolkit{newEchoToolkit()},
		MaxLoops: 3,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result, err := ag.RunStream(ctx, "compute")
	if err != nil {
		t.Fatalf("RunStream failed: %v", err)
	}

	for range result.Events {
	}

	done := <-result.Done
	if done.Err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	if done.Output != nil && done.Output.Status != RunStatusCancelled {
		t.Errorf("expected cancelled status, got %s", done.Output.Status)
	}
}
