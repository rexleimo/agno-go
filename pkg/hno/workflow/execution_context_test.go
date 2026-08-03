package workflow

import (
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// TestExecutionContext_HistoryMethods 测试历史方法
// TestExecutionContext_HistoryMethods tests history methods
func TestExecutionContext_HistoryMethods(t *testing.T) {
	execCtx := NewExecutionContext("test input")

	// 初始状态
	// Initial state
	if execCtx.HasHistory() {
		t.Error("expected no history initially")
	}

	if execCtx.GetHistoryCount() != 0 {
		t.Error("expected history count to be 0")
	}

	// 添加历史
	// Add history
	history := []HistoryEntry{
		{Input: "input-1", Output: "output-1", Timestamp: time.Now()},
		{Input: "input-2", Output: "output-2", Timestamp: time.Now()},
	}

	execCtx.SetWorkflowHistory(history)

	// 验证
	// Verify
	if !execCtx.HasHistory() {
		t.Error("expected to have history")
	}

	if execCtx.GetHistoryCount() != 2 {
		t.Errorf("expected history count 2, got %d", execCtx.GetHistoryCount())
	}

	// 获取最后一个条目
	// Get last entry
	last := execCtx.GetLastHistoryEntry()
	if last == nil || last.Input != "input-2" {
		t.Error("expected last entry to be input-2")
	}

	// 验证获取历史
	// Verify get history
	retrieved := execCtx.GetWorkflowHistory()
	if len(retrieved) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(retrieved))
	}

	if retrieved[0].Input != "input-1" {
		t.Errorf("expected first entry to be input-1, got %s", retrieved[0].Input)
	}
}

// TestExecutionContext_GetHistoryInput 测试历史输入获取
// TestExecutionContext_GetHistoryInput tests history input retrieval
func TestExecutionContext_GetHistoryInput(t *testing.T) {
	execCtx := NewExecutionContext("test")

	history := []HistoryEntry{
		{Input: "input-0", Output: "output-0"},
		{Input: "input-1", Output: "output-1"},
		{Input: "input-2", Output: "output-2"},
	}
	execCtx.SetWorkflowHistory(history)

	// 测试正索引
	// Test positive indices
	if input := execCtx.GetHistoryInput(0); input != "input-0" {
		t.Errorf("expected input-0, got %s", input)
	}

	if input := execCtx.GetHistoryInput(1); input != "input-1" {
		t.Errorf("expected input-1, got %s", input)
	}

	// 测试负索引
	// Test negative indices
	if input := execCtx.GetHistoryInput(-1); input != "input-2" {
		t.Errorf("expected input-2 (last), got %s", input)
	}

	if input := execCtx.GetHistoryInput(-2); input != "input-1" {
		t.Errorf("expected input-1 (second last), got %s", input)
	}

	// 测试越界
	// Test out of bounds
	if input := execCtx.GetHistoryInput(999); input != "" {
		t.Error("expected empty string for out of bounds")
	}

	if input := execCtx.GetHistoryInput(-999); input != "" {
		t.Error("expected empty string for negative out of bounds")
	}
}

// TestExecutionContext_GetHistoryOutput 测试历史输出获取
// TestExecutionContext_GetHistoryOutput tests history output retrieval
func TestExecutionContext_GetHistoryOutput(t *testing.T) {
	execCtx := NewExecutionContext("test")

	history := []HistoryEntry{
		{Input: "input-0", Output: "output-0"},
		{Input: "input-1", Output: "output-1"},
		{Input: "input-2", Output: "output-2"},
	}
	execCtx.SetWorkflowHistory(history)

	// 测试正索引
	// Test positive indices
	if output := execCtx.GetHistoryOutput(0); output != "output-0" {
		t.Errorf("expected output-0, got %s", output)
	}

	if output := execCtx.GetHistoryOutput(1); output != "output-1" {
		t.Errorf("expected output-1, got %s", output)
	}

	// 测试负索引
	// Test negative indices
	if output := execCtx.GetHistoryOutput(-1); output != "output-2" {
		t.Errorf("expected output-2 (last), got %s", output)
	}

	if output := execCtx.GetHistoryOutput(-2); output != "output-1" {
		t.Errorf("expected output-1 (second last), got %s", output)
	}

	// 测试越界
	// Test out of bounds
	if output := execCtx.GetHistoryOutput(999); output != "" {
		t.Error("expected empty string for out of bounds")
	}

	if output := execCtx.GetHistoryOutput(-999); output != "" {
		t.Error("expected empty string for negative out of bounds")
	}
}

// TestExecutionContext_MessageManagement 测试消息管理
// TestExecutionContext_MessageManagement tests message management
func TestExecutionContext_MessageManagement(t *testing.T) {
	execCtx := NewExecutionContext("test")

	// 初始无消息
	// Initially no messages
	messages := execCtx.GetMessages()
	if len(messages) != 0 {
		t.Error("expected no messages initially")
	}

	// 添加消息
	// Add message
	msg1 := types.NewUserMessage("hello")
	execCtx.AddMessage(msg1)

	messages = execCtx.GetMessages()
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "hello" {
		t.Errorf("expected 'hello', got '%s'", messages[0].Content)
	}

	// 批量添加
	// Add multiple messages
	msgs := []*types.Message{
		types.NewAssistantMessage("hi"),
		types.NewUserMessage("how are you"),
	}
	execCtx.AddMessages(msgs)

	messages = execCtx.GetMessages()
	if len(messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(messages))
	}

	// 验证消息顺序
	// Verify message order
	if messages[0].Content != "hello" {
		t.Error("first message should be 'hello'")
	}
	if messages[1].Content != "hi" {
		t.Error("second message should be 'hi'")
	}
	if messages[2].Content != "how are you" {
		t.Error("third message should be 'how are you'")
	}

	// 清空
	// Clear
	execCtx.ClearMessages()
	messages = execCtx.GetMessages()
	if len(messages) != 0 {
		t.Error("expected no messages after clear")
	}
}

// TestExecutionContext_HistoryContext 测试历史上下文
// TestExecutionContext_HistoryContext tests history context
func TestExecutionContext_HistoryContext(t *testing.T) {
	execCtx := NewExecutionContext("test")

	// 初始为空
	// Initially empty
	if execCtx.GetHistoryContext() != "" {
		t.Error("expected empty history context initially")
	}

	// 设置历史上下文
	// Set history context
	context := "<workflow_history_context>test</workflow_history_context>"
	execCtx.SetHistoryContext(context)

	retrieved := execCtx.GetHistoryContext()
	if retrieved != context {
		t.Error("expected history context to match")
	}

	// 设置为空
	// Set to empty
	execCtx.SetHistoryContext("")
	if execCtx.GetHistoryContext() != "" {
		t.Error("expected empty history context after clearing")
	}
}

// TestExecutionContext_EmptyHistory 测试空历史
// TestExecutionContext_EmptyHistory tests empty history
func TestExecutionContext_EmptyHistory(t *testing.T) {
	execCtx := NewExecutionContext("test")

	// GetLastHistoryEntry 应该返回 nil
	// GetLastHistoryEntry should return nil
	if entry := execCtx.GetLastHistoryEntry(); entry != nil {
		t.Error("expected nil for empty history")
	}

	// GetHistoryInput 应该返回空字符串
	// GetHistoryInput should return empty string
	if input := execCtx.GetHistoryInput(0); input != "" {
		t.Error("expected empty string for empty history")
	}

	if input := execCtx.GetHistoryInput(-1); input != "" {
		t.Error("expected empty string for empty history with negative index")
	}

	// GetHistoryOutput 应该返回空字符串
	// GetHistoryOutput should return empty string
	if output := execCtx.GetHistoryOutput(0); output != "" {
		t.Error("expected empty string for empty history")
	}

	if output := execCtx.GetHistoryOutput(-1); output != "" {
		t.Error("expected empty string for empty history with negative index")
	}
}

// TestExecutionContext_NewExecutionContextWithSession 测试带 session 的上下文创建
// TestExecutionContext_NewExecutionContextWithSession tests context creation with session
func TestExecutionContext_NewExecutionContextWithSession(t *testing.T) {
	sessionID := "test-session-123"
	userID := "user-456"
	input := "test input"

	execCtx := NewExecutionContextWithSession(input, sessionID, userID)

	if execCtx.Input != input {
		t.Errorf("expected input '%s', got '%s'", input, execCtx.Input)
	}

	if execCtx.SessionID != sessionID {
		t.Errorf("expected sessionID '%s', got '%s'", sessionID, execCtx.SessionID)
	}

	if execCtx.UserID != userID {
		t.Errorf("expected userID '%s', got '%s'", userID, execCtx.UserID)
	}

	// 验证初始化的字段
	// Verify initialized fields
	if execCtx.Data == nil {
		t.Error("expected Data to be initialized")
	}

	// Metadata is lazily initialized (nil until first write).
	// Metadata 为懒初始化（首次写入前为 nil）。
	if execCtx.Metadata != nil {
		t.Error("expected Metadata to be nil (lazy initialization)")
	}

	if execCtx.SessionState == nil {
		t.Error("expected SessionState to be initialized")
	}

	if execCtx.WorkflowHistory == nil {
		t.Error("expected WorkflowHistory to be initialized")
	}

	if len(execCtx.WorkflowHistory) != 0 {
		t.Error("expected WorkflowHistory to be empty initially")
	}
}

// TestExecutionContext_RunContextMetadata 测试 Run Context 元数据挂载
// TestExecutionContext_RunContextMetadata verifies SetRunContextMetadata behaviour
func TestExecutionContext_RunContextMetadata(t *testing.T) {
	execCtx := NewExecutionContext("test")

	if execCtx.Metadata != nil {
		t.Fatal("expected Metadata to be nil before setting run context")
	}

	rcMeta := map[string]interface{}{
		"run_id":      "run-123",
		"session_id":  "sess-456",
		"workflow_id": "wf-789",
	}
	execCtx.SetRunContextMetadata(rcMeta)

	if execCtx.Metadata == nil {
		t.Fatal("expected Metadata to be initialised after setting run context")
	}

	raw, ok := execCtx.Metadata["run_context"]
	if !ok {
		t.Fatalf("expected run_context key in Metadata, got %#v", execCtx.Metadata)
	}

	stored, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("run_context has wrong type: %#v", raw)
	}
	if stored["run_id"] != "run-123" || stored["session_id"] != "sess-456" || stored["workflow_id"] != "wf-789" {
		t.Fatalf("unexpected run_context payload: %#v", stored)
	}
}

// TestExecutionContext_DataAndMetadata 测试数据和元数据操作
// TestExecutionContext_DataAndMetadata tests data and metadata operations
func TestExecutionContext_DataAndMetadata(t *testing.T) {
	execCtx := NewExecutionContext("test")

	// 测试 Data 操作
	// Test Data operations
	execCtx.Set("key1", "value1")
	execCtx.Set("key2", 42)

	val1, ok1 := execCtx.Get("key1")
	if !ok1 || val1 != "value1" {
		t.Error("expected to get key1 value")
	}

	val2, ok2 := execCtx.Get("key2")
	if !ok2 || val2 != 42 {
		t.Error("expected to get key2 value")
	}

	// 测试不存在的键
	// Test non-existent key
	_, ok := execCtx.Get("nonexistent")
	if ok {
		t.Error("expected false for non-existent key")
	}
}

// TestExecutionContext_SessionState 测试会话状态
// TestExecutionContext_SessionState tests session state
func TestExecutionContext_SessionState(t *testing.T) {
	execCtx := NewExecutionContext("test")

	// 设置会话状态
	// Set session state
	execCtx.SetSessionState("state1", "value1")
	execCtx.SetSessionState("state2", map[string]interface{}{"nested": "data"})

	// 获取会话状态
	// Get session state
	val1, ok1 := execCtx.GetSessionState("state1")
	if !ok1 || val1 != "value1" {
		t.Error("expected to get state1 value")
	}

	val2, ok2 := execCtx.GetSessionState("state2")
	if !ok2 {
		t.Error("expected to get state2 value")
	}

	nested, ok := val2.(map[string]interface{})
	if !ok {
		t.Error("expected state2 to be a map")
	}

	if nested["nested"] != "data" {
		t.Error("expected nested data to match")
	}

	// 测试不存在的键
	// Test non-existent key
	_, ok = execCtx.GetSessionState("nonexistent")
	if ok {
		t.Error("expected false for non-existent session state key")
	}
}

// TestExecutionContext_HistoryWithTimestamps 测试带时间戳的历史
// TestExecutionContext_HistoryWithTimestamps tests history with timestamps
func TestExecutionContext_HistoryWithTimestamps(t *testing.T) {
	execCtx := NewExecutionContext("test")

	now := time.Now()
	history := []HistoryEntry{
		{Input: "input-1", Output: "output-1", Timestamp: now.Add(-2 * time.Hour)},
		{Input: "input-2", Output: "output-2", Timestamp: now.Add(-1 * time.Hour)},
		{Input: "input-3", Output: "output-3", Timestamp: now},
	}

	execCtx.SetWorkflowHistory(history)

	// 验证时间戳被正确存储
	// Verify timestamps are stored correctly
	retrieved := execCtx.GetWorkflowHistory()
	if len(retrieved) != 3 {
		t.Errorf("expected 3 history entries, got %d", len(retrieved))
	}

	// 验证最后一个条目的时间戳是最近的
	// Verify last entry has the most recent timestamp
	last := execCtx.GetLastHistoryEntry()
	if last == nil {
		t.Fatal("expected last entry to exist")
	}

	if !last.Timestamp.Equal(now) {
		t.Error("expected last entry to have most recent timestamp")
	}

	// 验证时间戳顺序
	// Verify timestamp order
	if !retrieved[0].Timestamp.Before(retrieved[1].Timestamp) {
		t.Error("expected timestamps to be in ascending order")
	}

	if !retrieved[1].Timestamp.Before(retrieved[2].Timestamp) {
		t.Error("expected timestamps to be in ascending order")
	}
}
