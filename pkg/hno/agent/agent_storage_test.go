package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// ========== 工具消息存储测试 ==========

// TestStoreToolMessagesEnabledByDefault 测试工具消息默认被存储
func TestStoreToolMessagesEnabledByDefault(t *testing.T) {
	callCount := 0
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++
			if callCount == 1 {
				// 第一次调用: 返回工具调用
				return &types.ModelResponse{
					ID:    "test-1",
					Model: "test",
					ToolCalls: []types.ToolCall{
						{
							ID:   "call_add_1",
							Type: "function",
							Function: types.ToolCallFunction{
								Name:      "add",
								Arguments: `{"a": 10, "b": 5}`,
							},
						},
					},
				}, nil
			}
			// 第二次调用: 返回最终答案
			return &types.ModelResponse{
				ID:      "test-2",
				Content: "The result is 15",
				Model:   "test",
			}, nil
		},
	}

	// 创建 Agent，不设置 StoreToolMessages（应该默认为 true）
	agent, err := New(Config{
		Name:     "test-agent",
		Model:    model,
		Toolkits: []toolkit.Toolkit{calculator.New()},
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 默认值应该是 true
	if !agent.storeToolMessages {
		t.Error("storeToolMessages should be true by default")
	}

	// 运行 agent
	output, err := agent.Run(context.Background(), "What is 10 + 5?")
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// 验证输出中包含工具消息
	hasToolMessages := false
	hasToolCalls := false
	for _, msg := range output.Messages {
		if msg.Role == types.RoleTool {
			hasToolMessages = true
		}
		if len(msg.ToolCalls) > 0 {
			hasToolCalls = true
		}
	}

	if !hasToolMessages {
		t.Error("Output should contain tool messages when storeToolMessages is true")
	}
	if !hasToolCalls {
		t.Error("Output should contain tool calls when storeToolMessages is true")
	}
}

// TestStoreToolMessagesDisabled 测试禁用工具消息存储
func TestStoreToolMessagesDisabled(t *testing.T) {
	callCount := 0
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++
			if callCount == 1 {
				// 第一次调用: 返回工具调用
				return &types.ModelResponse{
					ID:    "test-1",
					Model: "test",
					ToolCalls: []types.ToolCall{
						{
							ID:   "call_multiply_1",
							Type: "function",
							Function: types.ToolCallFunction{
								Name:      "multiply",
								Arguments: `{"a": 7, "b": 8}`,
							},
						},
					},
				}, nil
			}
			// 第二次调用: 返回最终答案
			return &types.ModelResponse{
				ID:      "test-2",
				Content: "The result is 56",
				Model:   "test",
			}, nil
		},
	}

	// 禁用工具消息存储
	storeToolMessages := false
	agent, err := New(Config{
		Name:              "test-agent",
		Model:             model,
		Toolkits:          []toolkit.Toolkit{calculator.New()},
		StoreToolMessages: &storeToolMessages,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 验证配置
	if agent.storeToolMessages {
		t.Error("storeToolMessages should be false")
	}

	// 运行 agent
	output, err := agent.Run(context.Background(), "What is 7 * 8?")
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// 验证输出中不包含工具消息
	for _, msg := range output.Messages {
		if msg.Role == types.RoleTool {
			t.Error("Output should NOT contain tool messages when storeToolMessages is false")
		}
		if len(msg.ToolCalls) > 0 {
			t.Error("Output should NOT contain tool calls when storeToolMessages is false")
		}
		if msg.ToolCallID != "" {
			t.Error("Output should NOT contain tool call IDs when storeToolMessages is false")
		}
	}
}

// TestToolMessagesFilteredCorrectly 测试工具消息被正确过滤
func TestToolMessagesFilteredCorrectly(t *testing.T) {
	callCount := 0
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++
			if callCount == 1 {
				return &types.ModelResponse{
					ID:    "test-1",
					Model: "test",
					ToolCalls: []types.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: types.ToolCallFunction{
								Name:      "add",
								Arguments: `{"a": 3, "b": 4}`,
							},
						},
					},
				}, nil
			}
			return &types.ModelResponse{
				ID:      "test-2",
				Content: "Sum is 7",
				Model:   "test",
			}, nil
		},
	}

	storeToolMessages := false
	agent, err := New(Config{
		Name:              "test-agent",
		Model:             model,
		Toolkits:          []toolkit.Toolkit{calculator.New()},
		StoreToolMessages: &storeToolMessages,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	output, err := agent.Run(context.Background(), "Calculate 3 + 4")
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// 应该至少有 user 和 assistant 消息
	if len(output.Messages) < 2 {
		t.Errorf("Expected at least 2 messages (user + assistant), got %d", len(output.Messages))
	}

	// 验证 user 和 assistant 消息存在
	hasUser := false
	hasAssistant := false
	for _, msg := range output.Messages {
		if msg.Role == types.RoleUser {
			hasUser = true
		}
		if msg.Role == types.RoleAssistant {
			hasAssistant = true
		}
	}

	if !hasUser {
		t.Error("Output should contain user message")
	}
	if !hasAssistant {
		t.Error("Output should contain assistant message")
	}
}

// ========== 历史消息存储测试 ==========

// TestStoreHistoryMessagesEnabledByDefault 测试历史消息默认被存储
func TestStoreHistoryMessagesEnabledByDefault(t *testing.T) {
	callCount := 0
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++
			return &types.ModelResponse{
				ID:      fmt.Sprintf("test-%d", callCount),
				Content: fmt.Sprintf("Response %d", callCount),
				Model:   "test",
			}, nil
		},
	}

	agent, err := New(Config{
		Name:  "test-agent",
		Model: model,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 默认值应该是 true
	if !agent.storeHistoryMessages {
		t.Error("storeHistoryMessages should be true by default")
	}

	// 第一次运行
	output1, err := agent.Run(context.Background(), "First message")
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	// 第二次运行
	output2, err := agent.Run(context.Background(), "Second message")
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	// 第二次运行的输出应该包含历史消息（来自第一次运行）
	// 预期: user1, assistant1, user2, assistant2 = 4条消息
	if len(output2.Messages) < 4 {
		t.Errorf("Second run should include history messages, expected at least 4 messages, got %d", len(output2.Messages))
	}

	// 第一次运行的输出应该只有2条消息
	if len(output1.Messages) != 2 {
		t.Errorf("First run should have 2 messages, got %d", len(output1.Messages))
	}
}

// TestStoreHistoryMessagesDisabled 测试禁用历史消息存储
func TestStoreHistoryMessagesDisabled(t *testing.T) {
	callCount := 0
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++
			return &types.ModelResponse{
				ID:      fmt.Sprintf("test-%d", callCount),
				Content: fmt.Sprintf("Response %d", callCount),
				Model:   "test",
			}, nil
		},
	}

	// 禁用历史消息存储
	storeHistoryMessages := false
	agent, err := New(Config{
		Name:                 "test-agent",
		Model:                model,
		StoreHistoryMessages: &storeHistoryMessages,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 验证配置
	if agent.storeHistoryMessages {
		t.Error("storeHistoryMessages should be false")
	}

	// 第一次运行
	output1, err := agent.Run(context.Background(), "First message")
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	// 第二次运行
	output2, err := agent.Run(context.Background(), "Second message")
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	// 两次运行的输出都应该只包含当前的消息，不包含历史
	if len(output1.Messages) != 2 {
		t.Errorf("First run should have 2 messages (user + assistant), got %d", len(output1.Messages))
	}
	if len(output2.Messages) != 2 {
		t.Errorf("Second run should have 2 messages (user + assistant), got %d (history should not be stored)", len(output2.Messages))
	}

	// 但是 Memory 中应该有所有消息（历史仍然被使用，只是不存储到输出）
	memorySize := agent.Memory.Size(agent.UserID)
	if memorySize != 4 {
		t.Errorf("Memory should have 4 messages (2 runs * 2 messages), got %d", memorySize)
	}
}

// TestHistoryMessagesFilteredCorrectly 测试历史消息被正确过滤
func TestHistoryMessagesFilteredCorrectly(t *testing.T) {
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test",
				Content: "OK",
				Model:   "test",
			}, nil
		},
	}

	storeHistoryMessages := false
	agent, err := New(Config{
		Name:                 "test-agent",
		Model:                model,
		StoreHistoryMessages: &storeHistoryMessages,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 多次运行
	for i := 1; i <= 3; i++ {
		output, err := agent.Run(context.Background(), fmt.Sprintf("Message %d", i))
		if err != nil {
			t.Fatalf("Run %d failed: %v", i, err)
		}

		// 每次输出应该只有2条消息
		if len(output.Messages) != 2 {
			t.Errorf("Run %d: expected 2 messages, got %d", i, len(output.Messages))
		}

		// 验证这两条消息是当前运行的
		if output.Messages[0].Role != types.RoleUser {
			t.Errorf("Run %d: first message should be user", i)
		}
		if output.Messages[1].Role != types.RoleAssistant {
			t.Errorf("Run %d: second message should be assistant", i)
		}
	}
}

// ========== 组合选项测试 ==========

// TestBothStorageOptionsDisabled 测试同时禁用工具和历史消息存储
func TestBothStorageOptionsDisabled(t *testing.T) {
	callCount := 0
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++
			if callCount == 1 || callCount == 3 {
				// 返回工具调用
				return &types.ModelResponse{
					ID:    fmt.Sprintf("test-%d", callCount),
					Model: "test",
					ToolCalls: []types.ToolCall{
						{
							ID:   fmt.Sprintf("call_%d", callCount),
							Type: "function",
							Function: types.ToolCallFunction{
								Name:      "add",
								Arguments: `{"a": 1, "b": 1}`,
							},
						},
					},
				}, nil
			}
			// 返回最终答案
			return &types.ModelResponse{
				ID:      fmt.Sprintf("test-%d", callCount),
				Content: "Done",
				Model:   "test",
			}, nil
		},
	}

	// 禁用所有存储
	storeToolMessages := false
	storeHistoryMessages := false
	agent, err := New(Config{
		Name:                 "test-agent",
		Model:                model,
		Toolkits:             []toolkit.Toolkit{calculator.New()},
		StoreToolMessages:    &storeToolMessages,
		StoreHistoryMessages: &storeHistoryMessages,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 第一次运行（带工具调用）
	output1, err := agent.Run(context.Background(), "Calculate 1 + 1")
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	// 第二次运行（带工具调用）
	output2, err := agent.Run(context.Background(), "Calculate again")
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	// 两次运行都应该只有当前Run的消息，没有历史，没有工具消息
	// 每次Run带工具调用会产生: user + assistant(tool calls) + tool + assistant(final)
	// 过滤Tool后: user + assistant(cleared) + assistant(final) = 3条
	for i, output := range []*RunOutput{output1, output2} {
		expectedCount := 3 // user + assistant(cleared) + assistant(final)
		if len(output.Messages) != expectedCount {
			t.Errorf("Run %d: expected %d messages, got %d", i+1, expectedCount, len(output.Messages))
		}

		// 验证没有工具消息
		for _, msg := range output.Messages {
			if msg.Role == types.RoleTool {
				t.Errorf("Run %d: should NOT have tool messages", i+1)
			}
			if len(msg.ToolCalls) > 0 {
				t.Errorf("Run %d: should NOT have tool calls", i+1)
			}
		}
	}
}

// TestSelectiveStorage 测试选择性存储
func TestSelectiveStorage(t *testing.T) {
	callCount := 0
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++
			if callCount == 1 || callCount == 3 {
				return &types.ModelResponse{
					ID:    fmt.Sprintf("test-%d", callCount),
					Model: "test",
					ToolCalls: []types.ToolCall{
						{
							ID:   fmt.Sprintf("call_%d", callCount),
							Type: "function",
							Function: types.ToolCallFunction{
								Name:      "add",
								Arguments: `{"a": 2, "b": 3}`,
							},
						},
					},
				}, nil
			}
			return &types.ModelResponse{
				ID:      fmt.Sprintf("test-%d", callCount),
				Content: "Result is 5",
				Model:   "test",
			}, nil
		},
	}

	// 禁用工具消息，保留历史消息
	storeToolMessages := false
	agent, err := New(Config{
		Name:              "test-agent",
		Model:             model,
		Toolkits:          []toolkit.Toolkit{calculator.New()},
		StoreToolMessages: &storeToolMessages,
		// StoreHistoryMessages 未设置，默认为 true
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 第一次运行
	output1, err := agent.Run(context.Background(), "First calculation")
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	// 第二次运行
	output2, err := agent.Run(context.Background(), "Second calculation")
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	// 第一次输出: 3条消息（user + assistant(cleared) + assistant(final)）
	// 因为工具调用会产生: user + assistant(tool calls) + tool + assistant(final)
	// 过滤Tool后: user + assistant(cleared) + assistant(final) = 3条
	if len(output1.Messages) != 3 {
		t.Errorf("First run: expected 3 messages, got %d", len(output1.Messages))
	}

	// 第二次输出: 应该包含历史（6条消息 = 3条历史 + 3条新消息），但没有工具消息
	if len(output2.Messages) != 6 {
		t.Errorf("Second run: expected 6 messages (3 history + 3 new), got %d", len(output2.Messages))
	}

	// 验证没有工具消息
	for _, msg := range output2.Messages {
		if msg.Role == types.RoleTool {
			t.Error("Should NOT have tool messages")
		}
		if len(msg.ToolCalls) > 0 {
			t.Error("Should NOT have tool calls")
		}
	}
}

// ========== 边界情况测试 ==========

// TestNoToolsUsed 测试未使用工具时的行为
func TestNoToolsUsed(t *testing.T) {
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test",
				Content: "Just a simple response",
				Model:   "test",
			}, nil
		},
	}

	storeToolMessages := false
	agent, err := New(Config{
		Name:              "test-agent",
		Model:             model,
		Toolkits:          []toolkit.Toolkit{calculator.New()},
		StoreToolMessages: &storeToolMessages,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	output, err := agent.Run(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// 应该正常工作，即使没有工具被调用
	if len(output.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(output.Messages))
	}
}

// TestNoHistoryAvailable 测试首次运行时无历史记录的情况
func TestNoHistoryAvailable(t *testing.T) {
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test",
				Content: "First response",
				Model:   "test",
			}, nil
		},
	}

	storeHistoryMessages := false
	agent, err := New(Config{
		Name:                 "test-agent",
		Model:                model,
		StoreHistoryMessages: &storeHistoryMessages,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 首次运行，无历史记录
	output, err := agent.Run(context.Background(), "First message")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// 应该正常工作
	if len(output.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(output.Messages))
	}
}

// TestEmptyMessages 测试空消息列表的处理
func TestEmptyMessages(t *testing.T) {
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test",
				Content: "Response",
				Model:   "test",
			}, nil
		},
	}

	storeToolMessages := false
	storeHistoryMessages := false
	agent, err := New(Config{
		Name:                 "test-agent",
		Model:                model,
		StoreToolMessages:    &storeToolMessages,
		StoreHistoryMessages: &storeHistoryMessages,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	output, err := agent.Run(context.Background(), "Test")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// 应该正常处理
	if output == nil {
		t.Error("Output should not be nil")
	}
}

// TestMultipleRunsSameAgent 测试同一个 agent 的多次运行
func TestMultipleRunsSameAgent(t *testing.T) {
	callCount := 0
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++
			if callCount%2 == 1 {
				// 奇数次调用: 返回工具调用
				return &types.ModelResponse{
					ID:    fmt.Sprintf("test-%d", callCount),
					Model: "test",
					ToolCalls: []types.ToolCall{
						{
							ID:   fmt.Sprintf("call_%d", callCount),
							Type: "function",
							Function: types.ToolCallFunction{
								Name:      "add",
								Arguments: fmt.Sprintf(`{"a": %d, "b": %d}`, callCount, callCount),
							},
						},
					},
				}, nil
			}
			// 偶数次调用: 返回最终答案
			return &types.ModelResponse{
				ID:      fmt.Sprintf("test-%d", callCount),
				Content: fmt.Sprintf("Result %d", callCount),
				Model:   "test",
			}, nil
		},
	}

	storeToolMessages := false
	storeHistoryMessages := false
	agent, err := New(Config{
		Name:                 "test-agent",
		Model:                model,
		Toolkits:             []toolkit.Toolkit{calculator.New()},
		StoreToolMessages:    &storeToolMessages,
		StoreHistoryMessages: &storeHistoryMessages,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 多次运行
	for i := 1; i <= 3; i++ {
		output, err := agent.Run(context.Background(), fmt.Sprintf("Run %d", i))
		if err != nil {
			t.Fatalf("Run %d failed: %v", i, err)
		}

		// 每次都应该只有当前Run的3条消息（user + assistant(cleared) + assistant(final)）
		// 因为工具调用会产生: user + assistant(tool calls) + tool + assistant(final)
		// 过滤Tool和History后: user + assistant(cleared) + assistant(final) = 3条
		if len(output.Messages) != 3 {
			t.Errorf("Run %d: expected 3 messages, got %d", i, len(output.Messages))
		}

		// 验证没有工具消息和历史消息
		for _, msg := range output.Messages {
			if msg.Role == types.RoleTool {
				t.Errorf("Run %d: should NOT have tool messages", i)
			}
			if len(msg.ToolCalls) > 0 {
				t.Errorf("Run %d: should NOT have tool calls", i)
			}
		}
	}
}

// ========== 并发测试 ==========

// TestConcurrentRunsWithStorageControl 测试并发运行时的存储控制
func TestConcurrentRunsWithStorageControl(t *testing.T) {
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test",
				Content: "Concurrent response",
				Model:   "test",
			}, nil
		},
	}

	// 并发运行 - 为每个goroutine创建独立的agent实例，避免内存共享问题
	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	storeHistoryMessages := false

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// 每个goroutine创建自己的agent实例
			agent, err := New(Config{
				Name:                 fmt.Sprintf("test-agent-%d", id),
				Model:                model,
				StoreHistoryMessages: &storeHistoryMessages,
			})
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: failed to create agent: %w", id, err)
				return
			}

			output, err := agent.Run(context.Background(), fmt.Sprintf("Message %d", id))
			if err != nil {
				errors <- fmt.Errorf("goroutine %d failed: %w", id, err)
				return
			}

			// 每次应该只有2条消息（user + assistant，无工具调用）
			if len(output.Messages) != 2 {
				errors <- fmt.Errorf("goroutine %d: expected 2 messages, got %d", id, len(output.Messages))
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查错误
	for err := range errors {
		t.Error(err)
	}
}
