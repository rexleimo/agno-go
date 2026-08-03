# Workflow Session State Management

Workflow 会话状态管理

## 概述 / Overview

**中文**:
SessionState 提供了跨工作流步骤的会话级状态管理，特别解决了并行步骤执行时的竞态条件问题。

**English**:
SessionState provides session-level state management across workflow steps, specifically solving race condition issues during parallel step execution.

## 核心功能 / Core Features

- ✅ **线程安全** / Thread-safe - 使用 sync.RWMutex 保护并发访问
- ✅ **深拷贝** / Deep copy - 为并行分支创建独立副本
- ✅ **智能合并** / Smart merging - 仅应用实际变更
- ✅ **会话隔离** / Session isolation - 支持 SessionID 和 UserID

## 快速开始 / Quick Start

### 1. 创建带会话状态的上下文 / Create Context with Session State

```go
// 基础上下文 / Basic context
ctx := workflow.NewExecutionContext("input text")

// 带会话信息的上下文 / Context with session info
ctx := workflow.NewExecutionContextWithSession(
    "input text",
    "session-123",
    "user-456",
)
```

### 2. 使用会话状态 / Use Session State

```go
// 设置值 / Set value
ctx.SetSessionState("counter", 1)
ctx.SetSessionState("user_name", "Alice")

// 获取值 / Get value
if val, ok := ctx.GetSessionState("counter"); ok {
    counter := val.(int)
    fmt.Println("Counter:", counter)
}

// 直接访问 / Direct access
ctx.SessionState.Set("key", "value")
value, exists := ctx.SessionState.Get("key")
```

### 3. 并行步骤中的会话状态 / Session State in Parallel Steps

```go
// 创建并行节点 / Create parallel node
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
    ID: "parallel-processing",
    Nodes: []workflow.Node{
        step1, // Each step operates on its own SessionState clone
        step2, // 每个步骤操作自己的 SessionState 克隆
        step3,
    },
})

// 执行后自动合并状态 / States are auto-merged after execution
result, _ := parallel.Execute(ctx, execCtx)
```

## 问题解决 / Problem Solved

### Python 版本的问题 / Issue in Python Version

**问题 / Problem**:
并行步骤共享同一个 `session_state` 字典，导致竞态条件。

Parallel steps sharing the same `session_state` dict cause race conditions.

```python
# Python - 问题代码 / Problem code
def execute_parallel(steps, session_state):
    # All steps share the same dict!
    # 所有步骤共享同一个字典！
    for step in steps:
        await step.run(session_state)  # ❌ Race condition
```

### Go 版本的解决方案 / Solution in Go Version

**解决方案 / Solution**:
为每个并行分支创建独立的 SessionState 克隆。

Create independent SessionState clone for each parallel branch.

```go
// Go - 解决方案 / Solution
sessionStateCopies := make([]*SessionState, len(nodes))
for i := range nodes {
    if execCtx.SessionState != nil {
        sessionStateCopies[i] = execCtx.SessionState.Clone() // ✅ Independent copy
    }
}

// 执行后合并 / Merge after execution
execCtx.SessionState = MergeParallelSessionStates(
    originalSessionState,
    modifiedSessionStates,
)
```

## 高级用法 / Advanced Usage

### 克隆会话状态 / Clone Session State

```go
// 深拷贝会话状态 / Deep copy session state
cloned := sessionState.Clone()

// 修改克隆不影响原始 / Modifying clone doesn't affect original
cloned.Set("key", "new value")
```

### 合并多个状态 / Merge Multiple States

```go
// 合并另一个状态 / Merge another state
sessionState.Merge(anotherState)

// 合并并行分支状态 / Merge parallel branch states
merged := workflow.MergeParallelSessionStates(
    originalState,
    []SessionState{branch1State, branch2State, branch3State},
)
```

### 获取所有数据 / Get All Data

```go
// 获取所有数据的副本 / Get copy of all data
allData := sessionState.GetAll()

// 转换为普通 map / Convert to plain map
dataMap := sessionState.ToMap()
```

## 并发安全保证 / Concurrency Safety Guarantees

### 读写锁机制 / RWMutex Mechanism

```go
type SessionState struct {
    mu   sync.RWMutex              // 读写锁 / Read-write lock
    data map[string]interface{}
}

// 读操作使用读锁 / Read operations use read lock
func (ss *SessionState) Get(key string) (interface{}, bool) {
    ss.mu.RLock()                  // ✅ Multiple readers allowed
    defer ss.mu.RUnlock()
    // ...
}

// 写操作使用写锁 / Write operations use write lock
func (ss *SessionState) Set(key string, value interface{}) {
    ss.mu.Lock()                   // ✅ Exclusive write access
    defer ss.mu.Unlock()
    // ...
}
```

## 完整示例 / Complete Example

```go
package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/workflow"
)

func main() {
    // 创建工作流上下文 / Create workflow context
    execCtx := workflow.NewExecutionContextWithSession(
        "Process data",
        "session-abc-123",
        "user-xyz-789",
    )

    // 初始化会话状态 / Initialize session state
    execCtx.SetSessionState("total_processed", 0)
    execCtx.SetSessionState("errors", []string{})

    // 创建并行处理步骤 / Create parallel processing steps
    parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
        ID: "data-processing",
        Nodes: []workflow.Node{
            // Step 1: Process batch 1
            step1,
            // Step 2: Process batch 2
            step2,
            // Step 3: Process batch 3
            step3,
        },
    })

    // 执行并行步骤 / Execute parallel steps
    result, err := parallel.Execute(context.Background(), execCtx)
    if err != nil {
        panic(err)
    }

    // 检查合并后的状态 / Check merged state
    if total, ok := result.GetSessionState("total_processed"); ok {
        fmt.Printf("Total processed: %v\n", total)
    }

    if errors, ok := result.GetSessionState("errors"); ok {
        fmt.Printf("Errors: %v\n", errors)
    }
}
```

## 最佳实践 / Best Practices

### 1. 使用描述性键名 / Use Descriptive Key Names

```go
// ✅ 好 / Good
ctx.SetSessionState("user_authentication_token", token)
ctx.SetSessionState("workflow_start_time", time.Now())

// ❌ 不好 / Bad
ctx.SetSessionState("t", token)
ctx.SetSessionState("x", time.Now())
```

### 2. 检查值是否存在 / Check Value Existence

```go
// ✅ 好 / Good
if val, ok := ctx.GetSessionState("key"); ok {
    // Use val
}

// ❌ 不好 / Bad
val := ctx.GetSessionState("key")  // Doesn't compile
```

### 3. 避免存储大对象 / Avoid Storing Large Objects

```go
// ✅ 好 / Good - 存储引用或ID / Store reference or ID
ctx.SetSessionState("document_id", docID)

// ❌ 不好 / Bad - 存储大对象 / Store large object
ctx.SetSessionState("full_document", largeDocument) // Will be deep-copied!
```

### 4. 并行步骤中避免删除共享键 / Avoid Deleting Shared Keys in Parallel

```go
// ⚠️ 小心 / Careful - 并行步骤中删除键 / Deleting keys in parallel
// 合并时可能导致不可预测的结果 / May lead to unpredictable results during merge
ctx.SessionState.Delete("shared_key")
```

## 性能考虑 / Performance Considerations

- **克隆开销** / Cloning overhead: O(n) where n = number of keys
- **合并开销** / Merging overhead: O(m*n) where m = branches, n = keys
- **深拷贝** / Deep copy: Uses JSON serialization (可以优化 / can be optimized)
- **推荐** / Recommendation: 限制会话状态大小 / Limit session state size

## API 参考 / API Reference

### SessionState Methods

| 方法 / Method | 描述 / Description |
|--------------|-------------------|
| `Set(key, value)` | 设置值 / Set value |
| `Get(key)` | 获取值 / Get value |
| `GetAll()` | 获取所有数据副本 / Get all data copy |
| `Delete(key)` | 删除键 / Delete key |
| `Clear()` | 清空所有数据 / Clear all data |
| `Clone()` | 深拷贝 / Deep copy |
| `Merge(other)` | 合并另一个状态 / Merge another state |
| `ToMap()` | 转换为普通map / Convert to plain map |

### ExecutionContext Methods

| 方法 / Method | 描述 / Description |
|--------------|-------------------|
| `SetSessionState(key, value)` | 设置会话状态值 / Set session state value |
| `GetSessionState(key)` | 获取会话状态值 / Get session state value |
| `Set(key, value)` | 设置上下文数据 / Set context data |
| `Get(key)` | 获取上下文数据 / Get context data |

## 与 Python 版本的兼容性 / Compatibility with Python Version

此实现与 Python Agno v2.1.2 的会话状态管理兼容，修复了相同的并发问题。

This implementation is compatible with Python Agno v2.1.2 session state management, fixing the same concurrency issues.

## 许可证 / License

Apache License 2.0
