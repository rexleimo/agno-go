package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// TestWorkflow_WithHistory 测试启用历史的工作流
// TestWorkflow_WithHistory tests workflow with history enabled
func TestWorkflow_WithHistory(t *testing.T) {
	// 创建带历史的 workflow
	// Create workflow with history
	storage := NewMemoryStorage(0)

	model := &MockModel{
		BaseModel: models.BaseModel{ID: "history-model", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test-response",
				Content: "response: " + req.Messages[len(req.Messages)-1].Content,
				Model:   "history-model",
			}, nil
		},
	}

	ag, _ := agent.New(agent.Config{
		ID:    "history-agent",
		Name:  "History Agent",
		Model: model,
	})

	step, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: ag,
	})

	workflow, err := New(Config{
		ID:                "test-workflow",
		Name:              "Test Workflow",
		Steps:             []Node{step},
		EnableHistory:     true,
		HistoryStore:      storage,
		NumHistoryRuns:    3,
		AddHistoryToSteps: true,
	})

	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	ctx := context.Background()
	sessionID := "test-session-1"

	// 第一次运行
	// First run
	result1, err := workflow.Run(ctx, "input-1", sessionID)
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	if result1.Output != "response: input-1" {
		t.Errorf("expected 'response: input-1', got '%s'", result1.Output)
	}

	// 第二次运行（应该能看到历史）
	// Second run (should see history)
	result2, err := workflow.Run(ctx, "input-2", sessionID)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	// 验证历史被加载
	// Verify history is loaded
	history, ok := result2.GetSessionState("workflow_history")
	if !ok {
		t.Error("expected workflow_history in session state")
	}

	historyEntries, ok := history.([]HistoryEntry)
	if !ok {
		t.Error("expected workflow_history to be []HistoryEntry")
	}

	if len(historyEntries) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(historyEntries))
	}

	// 验证历史上下文
	// Verify history context
	historyContext, ok := result2.GetSessionState("workflow_history_context")
	if !ok {
		t.Error("expected workflow_history_context in session state")
	}

	if historyContext == "" {
		t.Error("history context should not be empty")
	}

	// 验证 session 中的运行记录
	// Verify run records in session
	session, err := storage.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if session.CountRuns() != 2 {
		t.Errorf("expected 2 runs in session, got %d", session.CountRuns())
	}

	// 验证第一次运行是成功的
	// Verify first run is successful
	runs := session.GetRuns()
	if runs[0].Status != RunStatusCompleted {
		t.Errorf("expected first run to be completed, got %s", runs[0].Status)
	}

	// 验证 run_context 元数据已写入 WorkflowRun
	// Verify run_context metadata is attached to workflow runs
	rcRaw, ok := runs[0].Metadata["run_context"]
	if !ok {
		t.Fatalf("expected run_context metadata on first run, got %#v", runs[0].Metadata)
	}
	rcMeta, ok := rcRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("run_context metadata has wrong type: %#v", rcRaw)
	}
	if rcMeta["workflow_id"] != workflow.ID {
		t.Fatalf("expected workflow_id %q in run_context, got %#v", workflow.ID, rcMeta["workflow_id"])
	}
	if rcMeta["session_id"] != sessionID {
		t.Fatalf("expected session_id %q in run_context, got %#v", sessionID, rcMeta["session_id"])
	}
}

// TestWorkflow_WithoutHistory 测试未启用历史的工作流
// TestWorkflow_WithoutHistory tests workflow without history enabled
func TestWorkflow_WithoutHistory(t *testing.T) {
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "no-history-model", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test-response",
				Content: "no history response",
				Model:   "no-history-model",
			}, nil
		},
	}

	ag, _ := agent.New(agent.Config{
		ID:    "no-history-agent",
		Name:  "No History Agent",
		Model: model,
	})

	step, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: ag,
	})

	workflow, err := New(Config{
		ID:            "test-workflow-no-history",
		Name:          "Test Workflow No History",
		Steps:         []Node{step},
		EnableHistory: false,
	})

	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	ctx := context.Background()

	// 运行不应该失败
	// Run should not fail
	result, err := workflow.Run(ctx, "input", "session-1")
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// 不应该有历史上下文
	// Should not have history context
	_, ok := result.GetSessionState("workflow_history")
	if ok {
		t.Error("expected no workflow_history in session state when history is disabled")
	}
}

// TestWorkflow_HistoryLoadError 测试历史加载失败的情况
// TestWorkflow_HistoryLoadError tests history load failure
func TestWorkflow_HistoryLoadError(t *testing.T) {
	// 使用一个会失败的存储
	// Use a storage that will fail
	storage := &FailingStorage{}

	model := &MockModel{
		BaseModel: models.BaseModel{ID: "error-model", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test-response",
				Content: "response",
				Model:   "error-model",
			}, nil
		},
	}

	ag, _ := agent.New(agent.Config{
		ID:    "error-agent",
		Name:  "Error Agent",
		Model: model,
	})

	step, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: ag,
	})

	workflow, err := New(Config{
		ID:                "test-workflow-error",
		Name:              "Test Workflow Error",
		Steps:             []Node{step},
		EnableHistory:     true,
		HistoryStore:      storage,
		NumHistoryRuns:    3,
		AddHistoryToSteps: true,
	})

	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	ctx := context.Background()

	// 运行应该成功（历史加载失败不应该阻止执行）
	// Run should succeed (history load failure should not block execution)
	result, err := workflow.Run(ctx, "input", "session-1")
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.Output != "response" {
		t.Errorf("expected 'response', got '%s'", result.Output)
	}
}

// TestWorkflow_MultipleSessionsHistory 测试多个 session 的历史隔离
// TestWorkflow_MultipleSessionsHistory tests history isolation between sessions
func TestWorkflow_MultipleSessionsHistory(t *testing.T) {
	storage := NewMemoryStorage(0)

	model := &MockModel{
		BaseModel: models.BaseModel{ID: "multi-model", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			msg := req.Messages[len(req.Messages)-1].Content
			return &types.ModelResponse{
				ID:      "test-response",
				Content: "response: " + msg,
				Model:   "multi-model",
			}, nil
		},
	}

	ag, _ := agent.New(agent.Config{
		ID:    "multi-agent",
		Name:  "Multi Agent",
		Model: model,
	})

	step, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: ag,
	})

	workflow, err := New(Config{
		ID:                "test-workflow-multi",
		Name:              "Test Workflow Multi",
		Steps:             []Node{step},
		EnableHistory:     true,
		HistoryStore:      storage,
		NumHistoryRuns:    3,
		AddHistoryToSteps: true,
	})

	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	ctx := context.Background()

	// Session 1: 运行 2 次
	// Session 1: Run twice
	session1 := "session-1"
	_, err = workflow.Run(ctx, "session1-input1", session1)
	if err != nil {
		t.Fatalf("session1 run1 failed: %v", err)
	}

	_, err = workflow.Run(ctx, "session1-input2", session1)
	if err != nil {
		t.Fatalf("session1 run2 failed: %v", err)
	}

	// Session 2: 运行 1 次
	// Session 2: Run once
	session2 := "session-2"
	_, err = workflow.Run(ctx, "session2-input1", session2)
	if err != nil {
		t.Fatalf("session2 run1 failed: %v", err)
	}

	// 验证 session 1 有 2 次运行
	// Verify session 1 has 2 runs
	sess1, err := storage.GetSession(ctx, session1)
	if err != nil {
		t.Fatalf("failed to get session1: %v", err)
	}

	if sess1.CountRuns() != 2 {
		t.Errorf("session1 expected 2 runs, got %d", sess1.CountRuns())
	}

	// 验证 session 2 有 1 次运行
	// Verify session 2 has 1 run
	sess2, err := storage.GetSession(ctx, session2)
	if err != nil {
		t.Fatalf("failed to get session2: %v", err)
	}

	if sess2.CountRuns() != 1 {
		t.Errorf("session2 expected 1 run, got %d", sess2.CountRuns())
	}

	// 验证历史隔离
	// Verify history isolation
	history1 := sess1.GetHistory(10)
	history2 := sess2.GetHistory(10)

	if len(history1) != 2 {
		t.Errorf("session1 expected 2 history entries, got %d", len(history1))
	}

	if len(history2) != 1 {
		t.Errorf("session2 expected 1 history entry, got %d", len(history2))
	}

	// 验证历史内容不同
	// Verify history content is different
	if history1[0].Input == history2[0].Input {
		t.Error("session histories should be different")
	}
}

// TestWorkflow_SessionIDGeneration 测试 sessionID 自动生成
// TestWorkflow_SessionIDGeneration tests automatic sessionID generation
func TestWorkflow_SessionIDGeneration(t *testing.T) {
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "gen-model", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test-response",
				Content: "generated",
				Model:   "gen-model",
			}, nil
		},
	}

	ag, _ := agent.New(agent.Config{
		ID:    "gen-agent",
		Name:  "Gen Agent",
		Model: model,
	})

	step, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: ag,
	})

	workflow, err := New(Config{
		ID:            "test-workflow-gen",
		Name:          "Test Workflow Gen",
		Steps:         []Node{step},
		EnableHistory: false,
	})

	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	ctx := context.Background()

	// 不提供 sessionID，应该自动生成
	// Don't provide sessionID, should auto-generate
	result, err := workflow.Run(ctx, "input", "")
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// 验证 sessionID 被设置
	// Verify sessionID is set
	if result.SessionID == "" {
		t.Error("expected sessionID to be generated")
	}

	// 验证 sessionID 格式
	// Verify sessionID format
	if len(result.SessionID) < 10 {
		t.Error("generated sessionID too short")
	}
}

// FailingStorage is a mock storage that always fails
// FailingStorage 是一个总是失败的模拟存储
type FailingStorage struct{}

func (f *FailingStorage) CreateSession(ctx context.Context, sessionID, workflowID, userID string) (*WorkflowSession, error) {
	return nil, ErrSessionNotFound
}

func (f *FailingStorage) GetSession(ctx context.Context, sessionID string) (*WorkflowSession, error) {
	return nil, ErrSessionNotFound
}

func (f *FailingStorage) UpdateSession(ctx context.Context, session *WorkflowSession) error {
	return ErrSessionNotFound
}

func (f *FailingStorage) DeleteSession(ctx context.Context, sessionID string) error {
	return ErrSessionNotFound
}

func (f *FailingStorage) ListSessions(ctx context.Context, workflowID string, limit, offset int) ([]*WorkflowSession, error) {
	return nil, ErrSessionNotFound
}

func (f *FailingStorage) ListUserSessions(ctx context.Context, userID string, limit, offset int) ([]*WorkflowSession, error) {
	return nil, ErrSessionNotFound
}

func (f *FailingStorage) Clear(ctx context.Context, olderThan time.Duration) (int, error) {
	return 0, ErrSessionNotFound
}

func (f *FailingStorage) Close() error {
	return nil
}
