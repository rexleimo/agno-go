package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestAgent_GetID(t *testing.T) {
	agent, err := New(Config{
		ID:    "test-agent-123",
		Name:  "test-agent",
		Model: &MockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if agent.GetID() != "test-agent-123" {
		t.Errorf("Expected agent ID to be 'test-agent-123', got '%s'", agent.GetID())
	}
}

func TestAgent_TempInstructions(t *testing.T) {
	agent, err := New(Config{
		Name:         "test-agent",
		Model:        &MockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
		Instructions: "original instructions",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test 1: Original instructions
	// 测试 1：原始指令
	if instr := agent.GetInstructions(); instr != "original instructions" {
		t.Errorf("Expected original instructions, got %s", instr)
	}

	// Test 2: Set temporary instructions
	// 测试 2：设置临时指令
	agent.SetTempInstructions("temporary instructions")

	if instr := agent.GetInstructions(); instr != "temporary instructions" {
		t.Errorf("Expected temporary instructions, got %s", instr)
	}

	// Test 3: Clear temporary instructions
	// 测试 3：清除临时指令
	agent.ClearTempInstructions()

	if instr := agent.GetInstructions(); instr != "original instructions" {
		t.Errorf("Expected original instructions after clear, got %s", instr)
	}
}

func TestAgent_SetInstructions(t *testing.T) {
	agent, err := New(Config{
		Name:         "test-agent",
		Model:        &MockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
		Instructions: "original instructions",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Permanently change instructions
	// 永久更改指令
	agent.SetInstructions("new permanent instructions")

	if instr := agent.GetInstructions(); instr != "new permanent instructions" {
		t.Errorf("Expected new permanent instructions, got %s", instr)
	}

	// Verify permanent change persists
	// 验证永久更改持续存在
	if agent.Instructions != "new permanent instructions" {
		t.Error("Expected Instructions field to be updated permanently")
	}
}

func TestAgent_TempInstructionsPriority(t *testing.T) {
	agent, err := New(Config{
		Name:         "test-agent",
		Model:        &MockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
		Instructions: "original instructions",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Set temporary instructions
	// 设置临时指令
	agent.SetTempInstructions("temp instructions")

	// Change permanent instructions while temp is set
	// 在临时指令设置时更改永久指令
	agent.SetInstructions("new permanent instructions")

	// Temp should still take precedence
	// 临时指令仍应优先
	if instr := agent.GetInstructions(); instr != "temp instructions" {
		t.Errorf("Expected temp instructions to take precedence, got %s", instr)
	}

	// Clear temp and verify permanent change is visible
	// 清除临时指令并验证永久更改可见
	agent.ClearTempInstructions()

	if instr := agent.GetInstructions(); instr != "new permanent instructions" {
		t.Errorf("Expected new permanent instructions after clearing temp, got %s", instr)
	}
}

func TestAgent_ConcurrentInstructionsAccess(t *testing.T) {
	// Test concurrent access to instructions is thread-safe
	// 测试对指令的并发访问是线程安全的
	agent, err := New(Config{
		Name:         "test-agent",
		Model:        &MockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
		Instructions: "original",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	// 并发写入
	for i := 0; i < iterations; i++ {
		wg.Add(3)

		go func(n int) {
			defer wg.Done()
			agent.SetTempInstructions("temp" + string(rune(n)))
		}(i)

		go func(n int) {
			defer wg.Done()
			agent.SetInstructions("perm" + string(rune(n)))
		}(i)

		// Concurrent reads
		// 并发读取
		go func() {
			defer wg.Done()
			_ = agent.GetInstructions()
		}()
	}

	wg.Wait()

	// If we get here without race condition, test passes
	// 如果到达这里没有竞态条件，测试通过
	t.Log("Concurrent access test completed successfully")
}

func TestAgent_Run_AutoClearsTempInstructions(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			// Verify system message uses temporary instructions
			// 验证系统消息使用临时指令
			if len(req.Messages) > 0 && req.Messages[0].Role == types.RoleSystem {
				if req.Messages[0].Content != "You are a helpful assistant\n\n<workflow_history_context>\n[run-1]\ninput: hello\noutput: hi\n</workflow_history_context>" {
					t.Errorf("Expected enhanced instructions in system message, got: %s", req.Messages[0].Content)
				}
			}

			return &types.ModelResponse{
				Content: "test response",
			}, nil
		},
	}

	agent, err := New(Config{
		Name:         "test-agent",
		Model:        mockModel,
		Instructions: "You are a helpful assistant",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Set temporary instructions with history
	// 设置带历史的临时指令
	historyContext := "<workflow_history_context>\n[run-1]\ninput: hello\noutput: hi\n</workflow_history_context>"
	enhancedInstructions := agent.GetInstructions() + "\n\n" + historyContext
	agent.SetTempInstructions(enhancedInstructions)

	// Verify temp instructions are set
	// 验证临时指令已设置
	if agent.GetInstructions() != enhancedInstructions {
		t.Error("Expected temp instructions to be set before Run")
	}

	// Run agent
	// 运行 agent
	_, err = agent.Run(context.Background(), "test input")
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify temp instructions are automatically cleared
	// 验证临时指令自动清除
	if agent.GetInstructions() != "You are a helpful assistant" {
		t.Errorf("Expected temp instructions to be cleared after Run, got: %s", agent.GetInstructions())
	}
}

func TestAgent_Run_WithTempInstructionsError(t *testing.T) {
	// Test that temp instructions are cleared even if Run fails
	// 测试即使 Run 失败，临时指令也会被清除
	agent, err := New(Config{
		Name:         "test-agent",
		Model:        &MockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
		Instructions: "original",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	agent.SetTempInstructions("temp instructions")

	// Run with empty input (should fail)
	// 使用空输入运行（应该失败）
	_, err = agent.Run(context.Background(), "")
	if err == nil {
		t.Error("Expected Run to fail with empty input")
	}

	// Verify temp instructions are still cleared even on error
	// 验证即使出错，临时指令仍被清除
	if agent.GetInstructions() != "original" {
		t.Error("Expected temp instructions to be cleared even on error")
	}
}

func TestAgent_UpdateSystemMessage(t *testing.T) {
	agent, err := New(Config{
		Name:         "test-agent",
		Model:        &MockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
		Instructions: "original",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	tests := []struct {
		name         string
		messages     []*types.Message
		instructions string
		wantLen      int
		wantFirst    string
	}{
		{
			name:         "empty messages",
			messages:     []*types.Message{},
			instructions: "new instructions",
			wantLen:      1,
			wantFirst:    "new instructions",
		},
		{
			name: "replace existing system message",
			messages: []*types.Message{
				types.NewSystemMessage("old system"),
				types.NewUserMessage("user msg"),
			},
			instructions: "new system",
			wantLen:      2,
			wantFirst:    "new system",
		},
		{
			name: "prepend system message when none exists",
			messages: []*types.Message{
				types.NewUserMessage("user msg"),
				types.NewAssistantMessage("assistant msg"),
			},
			instructions: "new system",
			wantLen:      3,
			wantFirst:    "new system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.updateSystemMessage(tt.messages, tt.instructions)

			if len(result) != tt.wantLen {
				t.Errorf("Expected %d messages, got %d", tt.wantLen, len(result))
			}

			if len(result) > 0 && result[0].Content != tt.wantFirst {
				t.Errorf("Expected first message content '%s', got '%s'", tt.wantFirst, result[0].Content)
			}

			if len(result) > 0 && result[0].Role != types.RoleSystem {
				t.Error("Expected first message to be system message")
			}
		})
	}
}
