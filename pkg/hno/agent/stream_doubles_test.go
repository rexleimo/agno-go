package agent

import (
	"context"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Shared streaming test doubles for the P0-stream consolidation tests. Both the
// limit/stop-loop file and the sync-vs-stream parity file build on these, so
// neither one can be deleted without the other silently losing its fixtures.
// P0-stream 收口测试共用的流式夹具：截断/停止循环用例与同步-流式对照用例都依赖它们，
// 任何一方被删除都不会让另一方静默失去夹具。

type p0sStreamModel struct {
	chunks      [][]types.ResponseChunk
	invocations int
}

func (m *p0sStreamModel) Invoke(_ context.Context, _ *models.InvokeRequest) (*types.ModelResponse, error) {
	index := m.invocations
	m.invocations++
	if index >= len(m.chunks) {
		return &types.ModelResponse{Content: "final"}, nil
	}
	resp := &types.ModelResponse{}
	for _, chunk := range m.chunks[index] {
		resp.Content += chunk.Content
		resp.ToolCalls = append(resp.ToolCalls, chunk.ToolCalls...)
	}
	return resp, nil
}

func (m *p0sStreamModel) InvokeStream(_ context.Context, _ *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	index := m.invocations
	m.invocations++
	ch := make(chan types.ResponseChunk)
	chunks := []types.ResponseChunk{{Content: "final"}}
	if index < len(m.chunks) {
		chunks = m.chunks[index]
	}
	go func() {
		defer close(ch)
		for _, chunk := range chunks {
			ch <- chunk
		}
	}()
	return ch, nil
}

func (m *p0sStreamModel) GetProvider() string { return "p0s" }
func (m *p0sStreamModel) GetID() string       { return "p0s-stream-model" }
func (m *p0sStreamModel) GetName() string     { return "P0S Stream Model" }

func p0sToolCall(id, name string) types.ToolCall {
	return types.ToolCall{
		ID:   id,
		Type: "function",
		Function: types.ToolCallFunction{
			Name:      name,
			Arguments: `{}`,
		},
	}
}

func p0sThreeCallChunks() [][]types.ResponseChunk {
	return [][]types.ResponseChunk{
		{
			{ToolCalls: []types.ToolCall{p0sToolCall("c1", "p0s_a")}},
			{ToolCalls: []types.ToolCall{p0sToolCall("c2", "p0s_b"), p0sToolCall("c3", "p0s_a")}},
		},
	}
}

func p0sToolkits(stopLoop bool) []toolkit.Toolkit {
	tk := toolkit.NewBaseToolkit("p0s-tools")
	for name, value := range map[string]string{"p0s_a": "A", "p0s_b": "B"} {
		name, value := name, value
		tk.RegisterFunction(&toolkit.Function{
			Name:        name,
			Description: "P0S test tool",
			Parameters:  map[string]toolkit.Parameter{},
			StopLoop:    stopLoop && name == "p0s_a",
			Handler: func(context.Context, map[string]interface{}) (interface{}, error) {
				return value, nil
			},
		})
	}
	return []toolkit.Toolkit{tk}
}
