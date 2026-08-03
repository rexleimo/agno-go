package agent

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/memory"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// BenchmarkAgentCreation measures agent instantiation performance
func BenchmarkAgentCreation(b *testing.B) {
	model := &MockModel{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := New(Config{
			Name:  "benchmark-agent",
			Model: model,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAgentCreationWithTools measures agent instantiation with tools
// Follows Python Agno pattern: inline tool initialization for cleaner benchmarks
// BenchmarkAgentCreationWithTools 测量带工具的 agent 实例化性能
// 遵循 Python Agno 模式：内联工具初始化以实现更简洁的基准测试
func BenchmarkAgentCreationWithTools(b *testing.B) {
	model := &MockModel{}
	calc := calculator.New()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := New(Config{
			Name:     "benchmark-agent",
			Model:    model,
			Toolkits: []toolkit.Toolkit{calc},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAgentCreationWithMemory measures agent instantiation with memory
func BenchmarkAgentCreationWithMemory(b *testing.B) {
	model := &MockModel{}
	mem := memory.NewInMemory(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := New(Config{
			Name:   "benchmark-agent",
			Model:  model,
			Memory: mem,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAgentRun measures agent run performance with mock model
func BenchmarkAgentRun(b *testing.B) {
	model := &MockModel{
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				Content: "test response",
				Usage: types.Usage{
					TotalTokens: 10,
				},
			}, nil
		},
	}

	agent, _ := New(Config{
		Name:  "benchmark-agent",
		Model: model,
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := agent.Run(context.Background(), "test input")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAgentRunWithToolCalls measures agent run with tool execution
func BenchmarkAgentRunWithToolCalls(b *testing.B) {
	callCount := 0
	model := &MockModel{
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++
			if callCount%2 == 1 {
				// First call: return tool call
				return &types.ModelResponse{
					Content: "",
					ToolCalls: []types.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: types.ToolCallFunction{
								Name:      "add",
								Arguments: `{"a": 2, "b": 3}`,
							},
						},
					},
				}, nil
			}
			// Second call: return final response
			return &types.ModelResponse{
				Content: "The result is 5",
			}, nil
		},
	}

	calc := calculator.New()
	agent, _ := New(Config{
		Name:     "benchmark-agent",
		Model:    model,
		Toolkits: []toolkit.Toolkit{calc},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		callCount = 0 // Reset for each iteration
		_, err := agent.Run(context.Background(), "Calculate 2+3")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryOperations measures memory add performance
func BenchmarkMemoryOperations(b *testing.B) {
	mem := memory.NewInMemory(1000)
	msg := &types.Message{
		Role:    types.RoleUser,
		Content: "test message",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mem.Add(msg)
	}
}

// BenchmarkParallelAgentCreation measures concurrent agent creation
func BenchmarkParallelAgentCreation(b *testing.B) {
	model := &MockModel{}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := New(Config{
				Name:  "benchmark-agent",
				Model: model,
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkParallelAgentRun measures concurrent agent runs
func BenchmarkParallelAgentRun(b *testing.B) {
	model := &MockModel{
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				Content: "test response",
			}, nil
		},
	}

	agent, _ := New(Config{
		Name:  "benchmark-agent",
		Model: model,
	})

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := agent.Run(context.Background(), "test")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
