package workflow

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// mockModel for testing
type mockModel struct {
	models.BaseModel
}

func (m *mockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	return nil, nil
}

func (m *mockModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	return nil, nil
}

func TestInjectHistoryToAgent(t *testing.T) {
	mockAgent, err := agent.New(agent.Config{
		Name:         "test-agent",
		Model:        &mockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
		Instructions: "You are a helpful assistant",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	historyContext := "<workflow_history_context>\n[run-1]\ninput: hello\noutput: hi\n</workflow_history_context>"

	// Test 1: Inject history
	// 测试 1：注入历史
	original := InjectHistoryToAgent(mockAgent, historyContext)

	// Verify original instructions were returned
	// 验证返回了原始指令
	if original != "You are a helpful assistant" {
		t.Errorf("Expected original instructions to be returned, got: %s", original)
	}

	// Verify agent now has enhanced instructions
	// 验证 agent 现在有增强的指令
	enhanced := mockAgent.GetInstructions()
	if !strings.Contains(enhanced, historyContext) {
		t.Error("Expected history context in enhanced instructions")
	}

	if !strings.Contains(enhanced, "You are a helpful assistant") {
		t.Error("Expected original instructions in enhanced instructions")
	}

	// Clear and verify restoration
	// 清除并验证恢复
	mockAgent.ClearTempInstructions()
	if mockAgent.GetInstructions() != original {
		t.Error("Expected instructions to be restored after clear")
	}
}

func TestInjectHistoryToAgent_NilAgent(t *testing.T) {
	// Test with nil agent - should not panic
	// 使用 nil agent 测试 - 不应 panic
	result := InjectHistoryToAgent(nil, "some context")
	if result != "" {
		t.Error("Expected empty string when agent is nil")
	}
}

func TestInjectHistoryToAgent_EmptyHistory(t *testing.T) {
	mockAgent, err := agent.New(agent.Config{
		Name:         "test-agent",
		Model:        &mockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
		Instructions: "You are a helpful assistant",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test with empty history - should not modify agent
	// 使用空历史测试 - 不应修改 agent
	original := InjectHistoryToAgent(mockAgent, "")

	if original != "" {
		t.Error("Expected empty string when history is empty")
	}

	// Instructions should remain unchanged
	// 指令应保持不变
	if mockAgent.GetInstructions() != "You are a helpful assistant" {
		t.Error("Expected instructions to remain unchanged with empty history")
	}
}

func TestBuildEnhancedInstructions(t *testing.T) {
	tests := []struct {
		name           string
		original       string
		historyContext string
		wantContains   []string
	}{
		{
			name:           "both original and history",
			original:       "You are a helpful assistant",
			historyContext: "<workflow_history_context>\ntest\n</workflow_history_context>",
			wantContains:   []string{"You are a helpful assistant", "<workflow_history_context>", "test"},
		},
		{
			name:           "only history",
			original:       "",
			historyContext: "<workflow_history_context>\ntest\n</workflow_history_context>",
			wantContains:   []string{"<workflow_history_context>", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEnhancedInstructions(tt.original, tt.historyContext)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Expected result to contain '%s', got: %s", want, result)
				}
			}
		})
	}
}

func TestRestoreAgentInstructions(t *testing.T) {
	mockAgent, err := agent.New(agent.Config{
		Name:         "test-agent",
		Model:        &mockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
		Instructions: "original",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Set temp instructions
	// 设置临时指令
	mockAgent.SetTempInstructions("temp instructions")

	// Restore
	// 恢复
	RestoreAgentInstructions(mockAgent)

	// Verify temp instructions are cleared
	// 验证临时指令已清除
	if mockAgent.GetInstructions() != "original" {
		t.Error("Expected instructions to be restored")
	}
}

func TestRestoreAgentInstructions_NilAgent(t *testing.T) {
	// Should not panic with nil agent
	// nil agent 不应 panic
	RestoreAgentInstructions(nil)
}

func TestFormatHistoryForAgent(t *testing.T) {
	history := []HistoryEntry{
		{
			Input:     "hello",
			Output:    "hi there",
			Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			Input:     "how are you",
			Output:    "I'm good",
			Timestamp: time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC),
		},
	}

	// Test 1: Default format
	// 测试 1：默认格式
	formatted := FormatHistoryForAgent(history, nil)

	if !strings.Contains(formatted, "<workflow_history_context>") {
		t.Error("Expected context header")
	}

	if !strings.Contains(formatted, "</workflow_history_context>") {
		t.Error("Expected context footer")
	}

	if !strings.Contains(formatted, "[run-1]") {
		t.Error("Expected run-1")
	}

	if !strings.Contains(formatted, "[run-2]") {
		t.Error("Expected run-2")
	}

	if !strings.Contains(formatted, "input: hello") {
		t.Error("Expected input: hello")
	}

	if !strings.Contains(formatted, "output: hi there") {
		t.Error("Expected output: hi there")
	}

	// Timestamp should not be included by default
	// 默认不应包含时间戳
	if strings.Contains(formatted, "2024-01-01") {
		t.Error("Expected no timestamp in default format")
	}
}

func TestFormatHistoryForAgent_CustomOptions(t *testing.T) {
	history := []HistoryEntry{
		{
			Input:     "hello",
			Output:    "hi there",
			Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	// Custom format with timestamps, no output
	// 自定义格式：带时间戳，无输出
	options := &HistoryFormatOptions{
		Header:           "# History",
		Footer:           "# End",
		IncludeInput:     true,
		IncludeOutput:    false,
		IncludeTimestamp: true,
		InputLabel:       "User",
		OutputLabel:      "Assistant",
	}

	formatted := FormatHistoryForAgent(history, options)

	if !strings.Contains(formatted, "# History") {
		t.Error("Expected custom header '# History'")
	}

	if !strings.Contains(formatted, "# End") {
		t.Error("Expected custom footer '# End'")
	}

	if !strings.Contains(formatted, "2024-01-01 12:00:00") {
		t.Error("Expected timestamp to be included")
	}

	if !strings.Contains(formatted, "User: hello") {
		t.Error("Expected custom input label 'User'")
	}

	if strings.Contains(formatted, "output:") || strings.Contains(formatted, "Assistant:") {
		t.Error("Expected output to be excluded")
	}
}

func TestFormatHistoryForAgent_EmptyHistory(t *testing.T) {
	formatted := FormatHistoryForAgent([]HistoryEntry{}, nil)

	if formatted != "" {
		t.Error("Expected empty string for empty history")
	}
}

func TestFormatHistoryForAgent_InputOutputControl(t *testing.T) {
	history := []HistoryEntry{
		{
			Input:     "test input",
			Output:    "test output",
			Timestamp: time.Now(),
		},
	}

	// Test: Include input only
	// 测试：仅包含输入
	options := &HistoryFormatOptions{
		Header:        "# History",
		Footer:        "# End",
		IncludeInput:  true,
		IncludeOutput: false,
		InputLabel:    "input",
		OutputLabel:   "output",
	}

	formatted := FormatHistoryForAgent(history, options)

	if !strings.Contains(formatted, "input: test input") {
		t.Error("Expected input to be included")
	}

	if strings.Contains(formatted, "output: test output") {
		t.Error("Expected output to be excluded")
	}

	// Test: Include output only
	// 测试：仅包含输出
	options.IncludeInput = false
	options.IncludeOutput = true

	formatted = FormatHistoryForAgent(history, options)

	if strings.Contains(formatted, "input: test input") {
		t.Error("Expected input to be excluded")
	}

	if !strings.Contains(formatted, "output: test output") {
		t.Error("Expected output to be included")
	}
}

func TestDefaultHistoryFormatOptions(t *testing.T) {
	options := DefaultHistoryFormatOptions()

	if options.Header != "<workflow_history_context>" {
		t.Error("Expected default header to be '<workflow_history_context>'")
	}

	if options.Footer != "</workflow_history_context>" {
		t.Error("Expected default footer to be '</workflow_history_context>'")
	}

	if !options.IncludeInput {
		t.Error("Expected IncludeInput to be true by default")
	}

	if !options.IncludeOutput {
		t.Error("Expected IncludeOutput to be true by default")
	}

	if options.IncludeTimestamp {
		t.Error("Expected IncludeTimestamp to be false by default")
	}

	if options.InputLabel != "input" {
		t.Errorf("Expected InputLabel to be 'input', got '%s'", options.InputLabel)
	}

	if options.OutputLabel != "output" {
		t.Errorf("Expected OutputLabel to be 'output', got '%s'", options.OutputLabel)
	}
}

func TestFormatHistoryForAgent_MultipleRuns(t *testing.T) {
	// Test with multiple history entries
	// 测试多个历史条目
	history := []HistoryEntry{
		{Input: "input1", Output: "output1", Timestamp: time.Now()},
		{Input: "input2", Output: "output2", Timestamp: time.Now()},
		{Input: "input3", Output: "output3", Timestamp: time.Now()},
	}

	formatted := FormatHistoryForAgent(history, nil)

	// Verify all runs are numbered correctly
	// 验证所有运行编号正确
	if !strings.Contains(formatted, "[run-1]") ||
		!strings.Contains(formatted, "[run-2]") ||
		!strings.Contains(formatted, "[run-3]") {
		t.Error("Expected all runs to be numbered correctly")
	}

	// Verify all inputs and outputs are present
	// 验证所有输入和输出都存在
	for i := 1; i <= 3; i++ {
		inputStr := "input" + string(rune('0'+i))
		outputStr := "output" + string(rune('0'+i))

		if !strings.Contains(formatted, inputStr) {
			t.Errorf("Expected to find %s", inputStr)
		}

		if !strings.Contains(formatted, outputStr) {
			t.Errorf("Expected to find %s", outputStr)
		}
	}
}
