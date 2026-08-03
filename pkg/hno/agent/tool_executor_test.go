package agent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// testToolkit builds a toolkit with a few functions for executor tests.
// testToolkit 构建含若干函数的 toolkit 供执行器测试。
func testToolkit() *toolkit.BaseToolkit {
	tk := toolkit.NewBaseToolkit("test")
	tk.RegisterFunction(&toolkit.Function{
		Name:        "echo",
		Description: "echoes the input",
		Parameters: map[string]toolkit.Parameter{
			"text": {Type: "string", Required: true},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return args["text"], nil
		},
	})
	tk.RegisterFunction(&toolkit.Function{
		Name:        "slow",
		Description: "sleeps then returns",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			time.Sleep(50 * time.Millisecond)
			return "done", nil
		},
	})
	tk.RegisterFunction(&toolkit.Function{
		Name:        "boom",
		Description: "always errors",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("kaboom")
		},
	})
	return tk
}

// newTestAgent builds an agent with a toolkit and recorder memory.
// newTestAgent 构建带 toolkit 和记录内存的 agent。
func newTestAgent() (*Agent, *memRecorder) {
	a := &Agent{
		Toolkits: []toolkit.Toolkit{testToolkit()},
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	return a, newMemRecorder()
}

func TestExecuteToolCalls_O1LookupAndOrder(t *testing.T) {
	a, mem := newTestAgent()
	index := a.buildFunctionIndex()
	if len(index) != 3 {
		t.Fatalf("index size = %d, want 3", len(index))
	}
	if _, ok := index["echo"]; !ok {
		t.Fatal("echo missing from index")
	}

	// Results must be appended in call order.
	// 结果必须按调用顺序追加。
	a.Memory = mem
	err := a.executeToolCalls(context.Background(), []types.ToolCall{
		{ID: "c1", Function: types.ToolCallFunction{Name: "echo", Arguments: `{"text":"first"}`}},
		{ID: "c2", Function: types.ToolCallFunction{Name: "echo", Arguments: `{"text":"second"}`}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(mem.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(mem.messages))
	}
	if mem.messages[0].Content != `"first"` {
		t.Errorf("msg[0] = %q, want \"first\"", mem.messages[0].Content)
	}
	if mem.messages[1].Content != `"second"` {
		t.Errorf("msg[1] = %q, want \"second\"", mem.messages[1].Content)
	}
}

func TestExecuteToolCalls_Concurrent(t *testing.T) {
	a, mem := newTestAgent()
	a.Memory = mem

	start := time.Now()
	err := a.executeToolCalls(context.Background(), []types.ToolCall{
		{ID: "s1", Function: types.ToolCallFunction{Name: "slow"}},
		{ID: "s2", Function: types.ToolCallFunction{Name: "slow"}},
		{ID: "s3", Function: types.ToolCallFunction{Name: "slow"}},
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	// Three 50ms calls in parallel should take well under 150ms.
	// 三个 50ms 调用并发执行应远小于 150ms。
	if elapsed > 120*time.Millisecond {
		t.Errorf("elapsed = %v, expected concurrent execution", elapsed)
	}
}

func TestExecuteToolCalls_Errors(t *testing.T) {
	a, mem := newTestAgent()
	a.Memory = mem

	err := a.executeToolCalls(context.Background(), []types.ToolCall{
		{ID: "m1", Function: types.ToolCallFunction{Name: "missing"}},
		{ID: "m2", Function: types.ToolCallFunction{Name: "echo", Arguments: `{}`}},            // missing required param
		{ID: "m3", Function: types.ToolCallFunction{Name: "echo", Arguments: `not json`}},      // bad JSON
		{ID: "m4", Function: types.ToolCallFunction{Name: "boom", Arguments: `{}`}},            // handler error
		{ID: "m5", Function: types.ToolCallFunction{Name: "echo", Arguments: `{"text":"ok"}`}}, // success
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(mem.messages) != 5 {
		t.Fatalf("messages = %d, want 5 (errors become tool messages, never dropped)", len(mem.messages))
	}
	// Every failed call must surface an error message, not silently vanish.
	// 每个失败调用都必须产生错误消息，不能静默消失。
	for _, m := range mem.messages {
		if m.Content == "" {
			t.Errorf("empty tool message for call %s", m.ToolCallID)
		}
	}
}

func TestExecuteToolCalls_Empty(t *testing.T) {
	a := &Agent{}
	if err := a.executeToolCalls(context.Background(), nil); err != nil {
		t.Errorf("empty calls should be a no-op, got %v", err)
	}
}

// memRecorder is a minimal memory that records messages for assertions.
// memRecorder 是记录消息以便断言的最小 memory 实现。
type memRecorder struct {
	messages []*types.Message
}

func newMemRecorder() *memRecorder { return &memRecorder{} }

func (m *memRecorder) Add(msg *types.Message, userIDs ...string) {
	m.messages = append(m.messages, msg)
}
func (m *memRecorder) GetMessages(userIDs ...string) []*types.Message { return m.messages }
func (m *memRecorder) Clear(userIDs ...string)                        {}
func (m *memRecorder) Size(userIDs ...string) int                     { return len(m.messages) }
func (m *memRecorder) Search(query string, limit int, userIDs ...string) ([]*types.Message, error) {
	return nil, nil
}
