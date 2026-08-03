package workflow

import (
	"context"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// TestStep_HistoryInheritance 测试 step 继承 workflow 的历史设置
// TestStep_HistoryInheritance tests that step inherits workflow history settings
func TestStep_HistoryInheritance(t *testing.T) {
	storage := NewMemoryStorage(0)

	// 创建带历史的 workflow
	// Create workflow with history enabled
	ag := createMockAgent("test-agent", "mock response")

	step, err := NewStep(StepConfig{
		ID:    "step1",
		Name:  "Test Step",
		Agent: ag,
		// 不设置 AddHistoryToStep 和 NumHistoryRuns,应该继承 workflow 配置
		// Don't set AddHistoryToStep and NumHistoryRuns, should inherit from workflow
	})
	if err != nil {
		t.Fatalf("Failed to create step: %v", err)
	}

	wf, err := New(Config{
		ID:                "test-workflow",
		Name:              "Test Workflow",
		Steps:             []Node{step},
		EnableHistory:     true,
		HistoryStore:      storage,
		NumHistoryRuns:    2,
		AddHistoryToSteps: true, // 全局启用历史
	})
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	ctx := context.Background()
	sessionID := "test-session"

	// 运行第一次 (无历史)
	// First run (no history)
	_, err = wf.Run(ctx, "first run", sessionID)
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	// 运行第二次 (应该包含第一次的历史)
	// Second run (should include first run history)
	_, err = wf.Run(ctx, "second run", sessionID)
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	// 验证历史已被加载
	// Verify history was loaded
	session, err := storage.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if session.CountRuns() != 2 {
		t.Errorf("Expected 2 runs, got %d", session.CountRuns())
	}
}

// TestStep_HistoryOverride_Disable 测试 step 覆盖 workflow 设置禁用历史
// TestStep_HistoryOverride_Disable tests step overriding workflow settings to disable history
func TestStep_HistoryOverride_Disable(t *testing.T) {
	storage := NewMemoryStorage(0)

	disableHistory := false
	ag := createMockAgent("test-agent", "mock response")

	step, _ := NewStep(StepConfig{
		ID:               "step1",
		Agent:            ag,
		AddHistoryToStep: &disableHistory, // 覆盖为禁用
	})

	wf, _ := New(Config{
		ID:                "test-workflow",
		Steps:             []Node{step},
		EnableHistory:     true,
		HistoryStore:      storage,
		AddHistoryToSteps: true, // Workflow 启用,但 Step 禁用
	})

	ctx := context.Background()
	sessionID := "test-session-override"

	// 运行两次
	// Run twice
	wf.Run(ctx, "first run", sessionID)
	wf.Run(ctx, "second run", sessionID)

	// Step 应该没有使用历史(因为被覆盖为 false)
	// Step should not use history (overridden to false)
	// 这个通过检查 step 的内部逻辑验证
	// This is verified through step's internal logic
	if step.addHistoryToStep == nil || *step.addHistoryToStep != false {
		t.Error("Step should have history disabled")
	}
}

// TestStep_HistoryOverride_Enable 测试 step 覆盖 workflow 设置启用历史
// TestStep_HistoryOverride_Enable tests step overriding workflow settings to enable history
func TestStep_HistoryOverride_Enable(t *testing.T) {
	storage := NewMemoryStorage(0)

	enableHistory := true
	ag := createMockAgent("test-agent", "mock response")

	step, _ := NewStep(StepConfig{
		ID:               "step1",
		Agent:            ag,
		AddHistoryToStep: &enableHistory, // 覆盖为启用
	})

	wf, _ := New(Config{
		ID:                "test-workflow",
		Steps:             []Node{step},
		EnableHistory:     true,
		HistoryStore:      storage,
		AddHistoryToSteps: false, // Workflow 禁用,但 Step 启用
	})

	ctx := context.Background()
	sessionID := "test-session-enable"

	wf.Run(ctx, "first run", sessionID)
	wf.Run(ctx, "second run", sessionID)

	if step.addHistoryToStep == nil || *step.addHistoryToStep != true {
		t.Error("Step should have history enabled")
	}
}

// TestStep_CustomHistoryCount 测试自定义历史数量
// TestStep_CustomHistoryCount tests custom history count
func TestStep_CustomHistoryCount(t *testing.T) {
	storage := NewMemoryStorage(0)

	customCount := 5
	ag := createMockAgent("test-agent", "mock response")

	step, _ := NewStep(StepConfig{
		ID:             "step1",
		Agent:          ag,
		NumHistoryRuns: &customCount, // 自定义为 5
	})

	wf, _ := New(Config{
		ID:                "test-workflow",
		Steps:             []Node{step},
		EnableHistory:     true,
		HistoryStore:      storage,
		NumHistoryRuns:    3, // Workflow 默认 3
		AddHistoryToSteps: true,
	})

	ctx := context.Background()
	sessionID := "test-session-count"

	for i := 0; i < 10; i++ {
		wf.Run(ctx, "run", sessionID)
	}

	if step.numHistoryRuns == nil || *step.numHistoryRuns != 5 {
		t.Error("Step should use custom history count of 5")
	}
}

// TestStep_HistoryInjectionFormat 测试历史注入格式
// TestStep_HistoryInjectionFormat tests history injection format
func TestStep_HistoryInjectionFormat(t *testing.T) {
	storage := NewMemoryStorage(0)

	var capturedSystemMessage string
	var capturedUserMessage string
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			// 捕获传给模型的系统消息和用户消息
			// Capture system message and user message passed to model
			if len(req.Messages) > 0 {
				// 第一条消息通常是系统消息
				// First message is usually system message
				if req.Messages[0].Role == types.RoleSystem {
					capturedSystemMessage = req.Messages[0].Content
				}
				// 最后一条消息是用户消息
				// Last message is user message
				capturedUserMessage = req.Messages[len(req.Messages)-1].Content
			}
			return &types.ModelResponse{
				ID:      "test",
				Content: "response",
				Model:   "test",
			}, nil
		},
	}

	ag, _ := agent.New(agent.Config{
		ID:    "test-agent",
		Name:  "Test Agent",
		Model: model,
	})

	step, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: ag,
	})

	wf, _ := New(Config{
		ID:                "test-workflow",
		Steps:             []Node{step},
		EnableHistory:     true,
		HistoryStore:      storage,
		NumHistoryRuns:    2,
		AddHistoryToSteps: true,
	})

	ctx := context.Background()
	sessionID := "test-session-format"

	// 运行两次
	// Run twice
	wf.Run(ctx, "first input", sessionID)
	wf.Run(ctx, "second input", sessionID)

	// 第二次运行应该在系统消息中包含历史上下文（S008 新行为）
	// Second run should include history context in system message (S008 new behavior)
	if !strings.Contains(capturedSystemMessage, "<workflow_history_context>") {
		t.Errorf("System message should contain workflow_history_context tag, got: %s", capturedSystemMessage)
	}

	// 用户消息应该只包含实际输入（不再包含历史前缀）
	// User message should only contain actual input (no history prefix anymore)
	if capturedUserMessage != "second input" {
		t.Errorf("User message should be 'second input', got: %s", capturedUserMessage)
	}

	// 历史上下文应该包含第一次运行的输入和输出
	// History context should include first run's input and output
	if !strings.Contains(capturedSystemMessage, "first input") {
		t.Error("System message should contain history from first run")
	}
}

// TestStep_NoHistoryWhenDisabled 测试禁用历史时不注入
// TestStep_NoHistoryWhenDisabled tests no history injection when disabled
func TestStep_NoHistoryWhenDisabled(t *testing.T) {
	storage := NewMemoryStorage(0)

	var capturedInput string
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			if len(req.Messages) > 0 {
				capturedInput = req.Messages[len(req.Messages)-1].Content
			}
			return &types.ModelResponse{
				ID:      "test",
				Content: "response",
				Model:   "test",
			}, nil
		},
	}

	ag, _ := agent.New(agent.Config{
		ID:    "test-agent",
		Name:  "Test Agent",
		Model: model,
	})

	step, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: ag,
	})

	wf, _ := New(Config{
		ID:                "test-workflow",
		Steps:             []Node{step},
		EnableHistory:     true,
		HistoryStore:      storage,
		AddHistoryToSteps: false, // 禁用历史注入
	})

	ctx := context.Background()
	sessionID := "test-session-disabled"

	wf.Run(ctx, "first input", sessionID)
	wf.Run(ctx, "second input", sessionID)

	// 不应该包含历史上下文
	// Should not contain history context
	if strings.Contains(capturedInput, "<workflow_history_context>") {
		t.Error("Input should not contain workflow_history_context when history is disabled")
	}
}

// TestStep_ShouldAddHistory 测试 shouldAddHistory 方法
// TestStep_ShouldAddHistory tests shouldAddHistory method
func TestStep_ShouldAddHistory(t *testing.T) {
	tests := []struct {
		name           string
		stepConfig     *bool
		workflowConfig *WorkflowHistoryConfig
		expectedResult bool
	}{
		{
			name:       "Step nil, Workflow true",
			stepConfig: nil,
			workflowConfig: &WorkflowHistoryConfig{
				AddHistoryToSteps: true,
			},
			expectedResult: true,
		},
		{
			name:       "Step nil, Workflow false",
			stepConfig: nil,
			workflowConfig: &WorkflowHistoryConfig{
				AddHistoryToSteps: false,
			},
			expectedResult: false,
		},
		{
			name:       "Step true, Workflow false",
			stepConfig: boolPtr(true),
			workflowConfig: &WorkflowHistoryConfig{
				AddHistoryToSteps: false,
			},
			expectedResult: true, // Step 覆盖
		},
		{
			name:       "Step false, Workflow true",
			stepConfig: boolPtr(false),
			workflowConfig: &WorkflowHistoryConfig{
				AddHistoryToSteps: true,
			},
			expectedResult: false, // Step 覆盖
		},
		{
			name:           "Step nil, Workflow nil",
			stepConfig:     nil,
			workflowConfig: nil,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{
				addHistoryToStep: tt.stepConfig,
			}

			result := step.shouldAddHistory(tt.workflowConfig)
			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

// TestStep_GetHistoryRunCount 测试 getHistoryRunCount 方法
// TestStep_GetHistoryRunCount tests getHistoryRunCount method
func TestStep_GetHistoryRunCount(t *testing.T) {
	tests := []struct {
		name           string
		stepConfig     *int
		workflowConfig *WorkflowHistoryConfig
		expectedResult int
	}{
		{
			name:       "Step nil, Workflow 5",
			stepConfig: nil,
			workflowConfig: &WorkflowHistoryConfig{
				NumHistoryRuns: 5,
			},
			expectedResult: 5,
		},
		{
			name:       "Step 10, Workflow 5",
			stepConfig: intPtr(10),
			workflowConfig: &WorkflowHistoryConfig{
				NumHistoryRuns: 5,
			},
			expectedResult: 10, // Step 覆盖
		},
		{
			name:           "Step nil, Workflow nil",
			stepConfig:     nil,
			workflowConfig: nil,
			expectedResult: 3, // 默认值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{
				numHistoryRuns: tt.stepConfig,
			}

			result := step.getHistoryRunCount(tt.workflowConfig)
			if result != tt.expectedResult {
				t.Errorf("Expected %d, got %d", tt.expectedResult, result)
			}
		})
	}
}

// BenchmarkStep_WithHistory 测试带历史的性能
// BenchmarkStep_WithHistory benchmarks performance with history
func BenchmarkStep_WithHistory(b *testing.B) {
	storage := NewMemoryStorage(0)

	ag := createMockAgent("bench-agent", "mock response")

	step, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: ag,
	})

	wf, _ := New(Config{
		ID:                "bench-workflow",
		Steps:             []Node{step},
		EnableHistory:     true,
		HistoryStore:      storage,
		NumHistoryRuns:    3,
		AddHistoryToSteps: true,
	})

	ctx := context.Background()

	// 预热: 创建一些历史
	// Warm-up: Create some history
	for i := 0; i < 5; i++ {
		wf.Run(ctx, "warmup", "bench-session")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = wf.Run(ctx, "benchmark input", "bench-session")
	}
}

// BenchmarkStep_WithoutHistory 测试无历史的基线性能
// BenchmarkStep_WithoutHistory benchmarks baseline performance without history
func BenchmarkStep_WithoutHistory(b *testing.B) {
	storage := NewMemoryStorage(0)

	ag := createMockAgent("bench-agent", "mock response")

	step, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: ag,
	})

	wf, _ := New(Config{
		ID:                "bench-workflow",
		Steps:             []Node{step},
		EnableHistory:     true,
		HistoryStore:      storage,
		AddHistoryToSteps: false, // 禁用历史
	})

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = wf.Run(ctx, "benchmark input", "bench-session")
	}
}

// 辅助函数
// Helper functions

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}
