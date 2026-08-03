package workflow

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// TestNewSessionState tests creating a new session state
// 测试创建新的会话状态
func TestNewSessionState(t *testing.T) {
	ss := NewSessionState()
	if ss == nil {
		t.Fatal("NewSessionState() returned nil")
	}
	if ss.data == nil {
		t.Error("SessionState.data should be initialized")
	}
}

// TestSessionState_SetGet tests basic get/set operations
// 测试基本的 get/set 操作
func TestSessionState_SetGet(t *testing.T) {
	ss := NewSessionState()

	// Test string value
	ss.Set("key1", "value1")
	val, ok := ss.Get("key1")
	if !ok {
		t.Error("Get() should return true for existing key")
	}
	if val != "value1" {
		t.Errorf("Get() = %v, want 'value1'", val)
	}

	// Test int value
	ss.Set("counter", 42)
	val2, ok := ss.Get("counter")
	if !ok {
		t.Error("Get() should return true for counter")
	}
	if val2 != 42 {
		t.Errorf("Get() = %v, want 42", val2)
	}

	// Test non-existent key
	_, ok = ss.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for non-existent key")
	}
}

// TestSessionState_GetAll tests retrieving all data
// 测试获取所有数据
func TestSessionState_GetAll(t *testing.T) {
	ss := NewSessionState()
	ss.Set("key1", "value1")
	ss.Set("key2", "value2")

	all := ss.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll() returned %d items, want 2", len(all))
	}

	if all["key1"] != "value1" {
		t.Errorf("GetAll()[key1] = %v, want 'value1'", all["key1"])
	}
}

// TestSessionState_Delete tests deleting keys
// 测试删除键
func TestSessionState_Delete(t *testing.T) {
	ss := NewSessionState()
	ss.Set("key1", "value1")

	// Verify key exists
	_, ok := ss.Get("key1")
	if !ok {
		t.Error("Key should exist before delete")
	}

	// Delete key
	ss.Delete("key1")

	// Verify key is gone
	_, ok = ss.Get("key1")
	if ok {
		t.Error("Key should not exist after delete")
	}
}

// TestSessionState_Clear tests clearing all data
// 测试清空所有数据
func TestSessionState_Clear(t *testing.T) {
	ss := NewSessionState()
	ss.Set("key1", "value1")
	ss.Set("key2", "value2")

	ss.Clear()

	all := ss.GetAll()
	if len(all) != 0 {
		t.Errorf("After Clear(), GetAll() returned %d items, want 0", len(all))
	}
}

// TestSessionState_Clone tests deep cloning
// 测试深拷贝
func TestSessionState_Clone(t *testing.T) {
	original := NewSessionState()
	original.Set("key1", "value1")
	original.Set("counter", 10)

	// Clone
	cloned := original.Clone()

	// Verify cloned data matches
	val1, ok1 := cloned.Get("key1")
	if !ok1 || val1 != "value1" {
		t.Error("Cloned state should have key1=value1")
	}

	val2, ok2 := cloned.Get("counter")
	if !ok2 || val2 != 10.0 { // JSON unmarshaling converts int to float64
		t.Errorf("Cloned state counter = %v (type: %T), want 10", val2, val2)
	}

	// Modify cloned - should not affect original
	cloned.Set("key1", "modified")
	cloned.Set("new_key", "new_value")

	origVal, _ := original.Get("key1")
	if origVal != "value1" {
		t.Error("Modifying clone should not affect original")
	}

	_, exists := original.Get("new_key")
	if exists {
		t.Error("New key in clone should not appear in original")
	}
}

// TestSessionState_Merge tests merging session states
// 测试合并会话状态
func TestSessionState_Merge(t *testing.T) {
	ss1 := NewSessionState()
	ss1.Set("key1", "value1")
	ss1.Set("shared", "original")

	ss2 := NewSessionState()
	ss2.Set("key2", "value2")
	ss2.Set("shared", "modified")

	// Merge ss2 into ss1
	ss1.Merge(ss2)

	// Check merged data
	val1, _ := ss1.Get("key1")
	if val1 != "value1" {
		t.Error("Original key1 should be preserved")
	}

	val2, _ := ss1.Get("key2")
	if val2 != "value2" {
		t.Error("key2 from ss2 should be merged")
	}

	shared, _ := ss1.Get("shared")
	if shared != "modified" {
		t.Error("shared key should be overwritten with last-write-wins")
	}
}

// TestSessionState_ThreadSafety tests concurrent access
// 测试并发访问的线程安全性
func TestSessionState_ThreadSafety(t *testing.T) {
	ss := NewSessionState()
	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			ss.Set(fmt.Sprintf("key%d", val), val)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			ss.Get(fmt.Sprintf("key%d", val))
		}(i)
	}

	wg.Wait()

	// Verify final state
	all := ss.GetAll()
	if len(all) != iterations {
		t.Errorf("After concurrent writes, expected %d keys, got %d", iterations, len(all))
	}
}

// TestNewExecutionContextWithSession tests creating context with session info
// 测试创建带会话信息的上下文
func TestNewExecutionContextWithSession(t *testing.T) {
	ctx := NewExecutionContextWithSession("test input", "session-123", "user-456")

	if ctx.Input != "test input" {
		t.Errorf("Input = %v, want 'test input'", ctx.Input)
	}

	if ctx.SessionID != "session-123" {
		t.Errorf("SessionID = %v, want 'session-123'", ctx.SessionID)
	}

	if ctx.UserID != "user-456" {
		t.Errorf("UserID = %v, want 'user-456'", ctx.UserID)
	}

	if ctx.SessionState == nil {
		t.Error("SessionState should be initialized")
	}
}

// TestExecutionContext_SessionStateMethods tests session state helper methods
// 测试会话状态辅助方法
func TestExecutionContext_SessionStateMethods(t *testing.T) {
	ctx := NewExecutionContext("input")

	// SetSessionState
	ctx.SetSessionState("counter", 5)

	// GetSessionState
	val, ok := ctx.GetSessionState("counter")
	if !ok {
		t.Error("GetSessionState() should return true for existing key")
	}
	if val != 5 {
		t.Errorf("GetSessionState() = %v, want 5", val)
	}

	// Non-existent key
	_, ok = ctx.GetSessionState("nonexistent")
	if ok {
		t.Error("GetSessionState() should return false for non-existent key")
	}
}

// TestParallel_SessionStateIsolation tests that parallel branches have isolated session states
// 测试并行分支拥有隔离的会话状态
func TestParallel_SessionStateIsolation(t *testing.T) {
	// Create simple mock agents for parallel branches
	agent1 := createMockAgent("agent1", "parallel1")
	agent2 := createMockAgent("agent2", "parallel2")

	step1, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: agent1,
	})

	step2, _ := NewStep(StepConfig{
		ID:    "step2",
		Agent: agent2,
	})

	// Create parallel node
	parallel, _ := NewParallel(ParallelConfig{
		ID:    "test-parallel",
		Nodes: []Node{step1, step2},
	})

	// Create execution context with session data
	execCtx := NewExecutionContextWithSession("input", "session-123", "user-456")
	execCtx.SetSessionState("initial_key", "initial_value")

	// Execute parallel
	result, err := parallel.Execute(context.Background(), execCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify initial session state is preserved
	initialVal, ok := result.GetSessionState("initial_key")
	if !ok || initialVal != "initial_value" {
		t.Error("Initial session state should be preserved after parallel execution")
	}

	// Verify session metadata is preserved
	if result.SessionID != "session-123" {
		t.Error("SessionID should be preserved")
	}
	if result.UserID != "user-456" {
		t.Error("UserID should be preserved")
	}

	// Both branches should have executed (check output data)
	_, exists0 := result.Get("parallel_test-parallel_branch_0_output")
	if !exists0 {
		t.Error("Branch 0 should have executed")
	}

	_, exists1 := result.Get("parallel_test-parallel_branch_1_output")
	if !exists1 {
		t.Error("Branch 1 should have executed")
	}
}

// TestLoop_SessionStatePersistence tests session state persists across loop iterations
// 测试会话状态在循环迭代中持久化
func TestLoop_SessionStatePersistence(t *testing.T) {
	// Create a simple agent for loop body
	loopAgent := createMockAgent("loop-agent", "iteration executed")
	loopStep, _ := NewStep(StepConfig{
		ID:    "loop-step",
		Agent: loopAgent,
	})

	// Create loop that runs 3 times
	loop, _ := NewLoop(LoopConfig{
		ID:   "counter-loop",
		Body: loopStep,
		Condition: func(ctx *ExecutionContext, iteration int) bool {
			return iteration < 3
		},
	})

	execCtx := NewExecutionContextWithSession("start", "session-123", "user-456")
	execCtx.SetSessionState("initial_state", "preserved")

	result, err := loop.Execute(context.Background(), execCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Session state should be preserved
	initialState, ok := result.GetSessionState("initial_state")
	if !ok || initialState != "preserved" {
		t.Error("Initial session state should be preserved across loop iterations")
	}

	// Verify loop iterations count
	iterations, exists := result.Get("loop_counter-loop_iterations")
	if !exists {
		t.Fatal("Loop iterations should be stored")
	}

	if iterations != 3 {
		t.Errorf("Loop should have run 3 iterations, got %v", iterations)
	}

	// Verify SessionID and UserID are preserved
	if result.SessionID != "session-123" {
		t.Error("SessionID should be preserved")
	}
	if result.UserID != "user-456" {
		t.Error("UserID should be preserved")
	}
}

// TestCondition_SessionStatePropagation tests session state propagates through branches
// 测试会话状态通过分支传播
func TestCondition_SessionStatePropagation(t *testing.T) {
	// Create agents for true and false branches
	trueAgent := createMockAgent("true-agent", "true branch executed")
	falseAgent := createMockAgent("false-agent", "false branch executed")

	trueStep, _ := NewStep(StepConfig{
		ID:    "true-step",
		Agent: trueAgent,
	})

	falseStep, _ := NewStep(StepConfig{
		ID:    "false-step",
		Agent: falseAgent,
	})

	condition, _ := NewCondition(ConditionConfig{
		ID: "test-condition",
		Condition: func(ctx *ExecutionContext) bool {
			val, ok := ctx.GetSessionState("take_true_branch")
			if !ok {
				return false
			}
			return val.(bool)
		},
		TrueNode:  trueStep,
		FalseNode: falseStep,
	})

	// Test true branch
	execCtx1 := NewExecutionContextWithSession("input", "session-123", "user-456")
	execCtx1.SetSessionState("take_true_branch", true)
	execCtx1.SetSessionState("initial_data", "preserved")

	result1, _ := condition.Execute(context.Background(), execCtx1)

	// Verify initial session state is preserved
	initialData1, ok1 := result1.GetSessionState("initial_data")
	if !ok1 || initialData1 != "preserved" {
		t.Error("Session state should be preserved through true branch")
	}

	// Verify SessionID and UserID
	if result1.SessionID != "session-123" || result1.UserID != "user-456" {
		t.Error("Session metadata should be preserved through true branch")
	}

	// Test false branch
	execCtx2 := NewExecutionContextWithSession("input", "session-456", "user-789")
	execCtx2.SetSessionState("take_true_branch", false)
	execCtx2.SetSessionState("initial_data", "also_preserved")

	result2, _ := condition.Execute(context.Background(), execCtx2)

	// Verify initial session state is preserved
	initialData2, ok2 := result2.GetSessionState("initial_data")
	if !ok2 || initialData2 != "also_preserved" {
		t.Error("Session state should be preserved through false branch")
	}

	// Verify SessionID and UserID
	if result2.SessionID != "session-456" || result2.UserID != "user-789" {
		t.Error("Session metadata should be preserved through false branch")
	}
}

// TestRouter_SessionStatePropagation tests session state propagates through routes
// 测试会话状态通过路由传播
func TestRouter_SessionStatePropagation(t *testing.T) {
	route1Agent := createMockAgent("route1-agent", "route1 executed")
	route2Agent := createMockAgent("route2-agent", "route2 executed")

	route1, _ := NewStep(StepConfig{
		ID:    "route1",
		Agent: route1Agent,
	})

	route2, _ := NewStep(StepConfig{
		ID:    "route2",
		Agent: route2Agent,
	})

	router, _ := NewRouter(RouterConfig{
		ID: "test-router",
		Router: func(ctx *ExecutionContext) string {
			val, ok := ctx.GetSessionState("selected_route")
			if !ok {
				return "route1"
			}
			return val.(string)
		},
		Routes: map[string]Node{
			"route1": route1,
			"route2": route2,
		},
	})

	// Test route1
	execCtx1 := NewExecutionContextWithSession("input", "session-123", "user-456")
	execCtx1.SetSessionState("selected_route", "route1")
	execCtx1.SetSessionState("shared_data", "preserved")

	result1, _ := router.Execute(context.Background(), execCtx1)

	// Verify session state is preserved through route1
	sharedData1, ok1 := result1.GetSessionState("shared_data")
	if !ok1 || sharedData1 != "preserved" {
		t.Error("Session state should be preserved through route1")
	}

	// Verify SessionID and UserID
	if result1.SessionID != "session-123" || result1.UserID != "user-456" {
		t.Error("Session metadata should be preserved through route1")
	}

	// Test route2
	execCtx2 := NewExecutionContextWithSession("input", "session-789", "user-abc")
	execCtx2.SetSessionState("selected_route", "route2")
	execCtx2.SetSessionState("shared_data", "also_preserved")

	result2, _ := router.Execute(context.Background(), execCtx2)

	// Verify session state is preserved through route2
	sharedData2, ok2 := result2.GetSessionState("shared_data")
	if !ok2 || sharedData2 != "also_preserved" {
		t.Error("Session state should be preserved through route2")
	}

	// Verify SessionID and UserID
	if result2.SessionID != "session-789" || result2.UserID != "user-abc" {
		t.Error("Session metadata should be preserved through route2")
	}
}

// TestMergeParallelSessionStates tests the merging function
// 测试并行会话状态合并函数
func TestMergeParallelSessionStates(t *testing.T) {
	// Original state
	original := NewSessionState()
	original.Set("unchanged", "original_value")
	original.Set("modified", "original_value")

	// Branch 1 modifications
	branch1 := original.Clone()
	branch1.Set("modified", "branch1_value")
	branch1.Set("branch1_only", "value1")

	// Branch 2 modifications
	branch2 := original.Clone()
	branch2.Set("modified", "branch2_value") // Conflict!
	branch2.Set("branch2_only", "value2")

	// Merge
	merged := MergeParallelSessionStates(original, []*SessionState{branch1, branch2})

	// Unchanged key should remain
	unchanged, _ := merged.Get("unchanged")
	if unchanged != "original_value" {
		t.Error("Unchanged key should preserve original value")
	}

	// Branch-specific keys should be merged
	b1Val, ok1 := merged.Get("branch1_only")
	if !ok1 || b1Val != "value1" {
		t.Error("branch1_only should be merged")
	}

	b2Val, ok2 := merged.Get("branch2_only")
	if !ok2 || b2Val != "value2" {
		t.Error("branch2_only should be merged")
	}

	// Conflicting key - last-write-wins (branch2 since it's last)
	modified, _ := merged.Get("modified")
	if modified != "branch2_value" {
		t.Errorf("Conflicting key should have last-write-wins value, got '%v'", modified)
	}
}
