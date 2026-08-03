---
title: 会话状态管理
description: Workflow 执行过程中的线程安全状态管理,支持智能并行分支合并
outline: deep
---

# 会话状态管理

**会话状态管理** 提供了 Workflow 执行过程中的状态管理能力,支持跨步骤的数据共享、并发安全的状态访问,以及并行分支的智能合并。

## 为什么需要 Session State?

在复杂的 Workflow 中,步骤之间需要共享数据:

```
Step1: 获取用户信息
  ↓
  保存到 SessionState: {"user_id": "123", "name": "Alice"}
  ↓
Step2: 根据用户信息查询订单
  ↓
  从 SessionState 读取: user_id = "123"
  保存到 SessionState: {"orders": [...]}
  ↓
Step3: 生成报告
  ↓
  从 SessionState 读取: user_id, name, orders
```

没有会话状态,你需要通过步骤输出传递数据,这会创建紧密耦合和复杂性。会话状态充当所有步骤可访问的共享内存空间。

## 核心特性

1. **线程安全**: 使用 `sync.RWMutex` 保护并发访问
2. **深拷贝**: 并行分支获得独立的状态副本
3. **智能合并**: 并行执行后自动合并状态变更
4. **类型灵活**: 支持任意 `interface{}` 类型数据

## 快速开始

### 基础用法

```go
package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/workflow"
)

func main() {
    // 创建带会话状态的执行上下文
    execCtx := workflow.NewExecutionContextWithSession(
        "initial input",
        "session-123",  // sessionID
        "user-456",     // userID
    )

    // 设置会话状态
    execCtx.SetSessionState("user_name", "Alice")
    execCtx.SetSessionState("user_age", 30)
    execCtx.SetSessionState("preferences", map[string]string{
        "language": "zh-CN",
        "theme":    "dark",
    })

    // 读取会话状态
    if name, ok := execCtx.GetSessionState("user_name"); ok {
        fmt.Printf("User Name: %s\n", name)
    }

    if age, ok := execCtx.GetSessionState("user_age"); ok {
        fmt.Printf("User Age: %d\n", age)
    }
}
```

### 在 Workflow 中使用

```go
// 创建 Workflow
wf := workflow.NewWorkflow("user-workflow")

// Step 1: 获取用户信息
step1 := workflow.NewStep("get-user", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    // 模拟获取用户信息
    userInfo := map[string]interface{}{
        "id":    "user-123",
        "name":  "Alice",
        "email": "alice@example.com",
    }

    // 保存到 SessionState
    execCtx.SetSessionState("user_info", userInfo)

    return execCtx, nil
})

// Step 2: 获取用户订单
step2 := workflow.NewStep("get-orders", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    // 从 SessionState 读取用户信息
    userInfoRaw, ok := execCtx.GetSessionState("user_info")
    if !ok {
        return execCtx, fmt.Errorf("user_info not found in session state")
    }

    userInfo := userInfoRaw.(map[string]interface{})
    userID := userInfo["id"].(string)

    // 模拟获取订单
    orders := []string{"order-1", "order-2", "order-3"}
    execCtx.SetSessionState("orders", orders)

    fmt.Printf("Got %d orders for user %s\n", len(orders), userID)

    return execCtx, nil
})

// 链接步骤
step1.Then(step2)
wf.AddStep(step1)

// 执行 Workflow
execCtx := workflow.NewExecutionContextWithSession("", "session-123", "user-456")
result, err := wf.Execute(context.Background(), execCtx)
if err != nil {
    panic(err)
}

// 检查最终状态
orders, _ := result.GetSessionState("orders")
fmt.Printf("Final orders: %v\n", orders)
```

## 并行执行与状态合并

### 问题场景

在并行执行时,多个分支可能同时修改 SessionState:

```
              ┌─→ Branch A: Set("key1", "value_A")
Parallel Step ├─→ Branch B: Set("key2", "value_B")
              └─→ Branch C: Set("key1", "value_C")  // ⚠️ 冲突!
```

这在传统实现中会产生竞态条件。如何处理冲突的写入?

### 解决方案

Agno-Go 使用 **深拷贝 + 最终写入优先** 策略:

1. 每个并行分支获得独立的 SessionState 副本
2. 分支独立执行,互不干扰
3. 执行完成后,按顺序合并所有变更
4. 如果有冲突,后执行的分支覆盖先执行的分支

```go
// pkg/hno/workflow/parallel.go

func (p *Parallel) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
    // 1. 为每个分支创建独立的 SessionState 副本
    sessionStateCopies := make([]*SessionState, len(p.Nodes))
    for i := range p.Nodes {
        if execCtx.SessionState != nil {
            sessionStateCopies[i] = execCtx.SessionState.Clone()  // 深拷贝
        } else {
            sessionStateCopies[i] = NewSessionState()
        }
    }

    // 2. 并行执行各分支
    // ... (goroutines execution)

    // 3. 合并所有分支的状态变更
    execCtx.SessionState = MergeParallelSessionStates(
        originalSessionState,
        modifiedSessionStates,
    )

    return execCtx, nil
}
```

### 合并策略

```go
// pkg/hno/workflow/session_state.go

func MergeParallelSessionStates(original *SessionState, modified []*SessionState) *SessionState {
    merged := NewSessionState()

    // 1. 复制原始状态
    if original != nil {
        for k, v := range original.data {
            merged.data[k] = v
        }
    }

    // 2. 按顺序合并各分支的变更
    for _, modState := range modified {
        if modState == nil {
            continue
        }
        for k, v := range modState.data {
            merged.data[k] = v  // 最终写入优先
        }
    }

    return merged
}
```

### 示例

```go
// 并行执行 3 个分支
parallel := workflow.NewParallel()

// Branch A: 设置 counter = 1
branchA := workflow.NewStep("branch-a", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 1)
    execCtx.SetSessionState("branch_a_result", "done")
    return execCtx, nil
})

// Branch B: 设置 counter = 2
branchB := workflow.NewStep("branch-b", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 2)
    execCtx.SetSessionState("branch_b_result", "done")
    return execCtx, nil
})

// Branch C: 设置 counter = 3
branchC := workflow.NewStep("branch-c", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 3)
    execCtx.SetSessionState("branch_c_result", "done")
    return execCtx, nil
})

parallel.AddNode(branchA)
parallel.AddNode(branchB)
parallel.AddNode(branchC)

// 执行并行步骤
execCtx := workflow.NewExecutionContextWithSession("", "session-123", "user-456")
result, _ := parallel.Execute(context.Background(), execCtx)

// 检查合并结果
counter, _ := result.GetSessionState("counter")
fmt.Printf("Counter: %v\n", counter)  // 输出可能是 1, 2, 或 3 (取决于执行顺序)

branchAResult, _ := result.GetSessionState("branch_a_result")
branchBResult, _ := result.GetSessionState("branch_b_result")
branchCResult, _ := result.GetSessionState("branch_c_result")
fmt.Printf("All branches completed: %v, %v, %v\n", branchAResult, branchBResult, branchCResult)
// 输出: All branches completed: done, done, done
```

## API 参考

### SessionState 类型

```go
type SessionState struct {
    mu   sync.RWMutex
    data map[string]interface{}
}
```

#### 方法

##### NewSessionState()

```go
func NewSessionState() *SessionState
```

创建新的 SessionState 实例。

##### Set(key string, value interface{})

```go
func (ss *SessionState) Set(key string, value interface{})
```

设置键值对 (线程安全)。

**参数**:
- `key`: 要设置的键
- `value`: 任意类型的值 (interface{})

##### Get(key string) (interface{}, bool)

```go
func (ss *SessionState) Get(key string) (interface{}, bool)
```

获取键对应的值 (线程安全)。

**返回**:
- `value`: 键关联的值
- `exists`: 键是否存在

**示例**:
```go
if value, ok := sessionState.Get("user_id"); ok {
    userID := value.(string)
    fmt.Printf("User ID: %s\n", userID)
}
```

##### Clone() *SessionState

```go
func (ss *SessionState) Clone() *SessionState
```

深拷贝 SessionState (使用 JSON 序列化)。

**注意**: 不可序列化的类型会回退到浅拷贝。

##### GetAll() map[string]interface{}

```go
func (ss *SessionState) GetAll() map[string]interface{}
```

获取所有键值对的副本 (线程安全)。

##### MergeFrom(other *SessionState)

```go
func (ss *SessionState) MergeFrom(other *SessionState)
```

从另一个 SessionState 合并数据。

##### Clear()

```go
func (ss *SessionState) Clear()
```

清空所有数据。

---

### ExecutionContext 扩展

```go
type ExecutionContext struct {
    // 现有字段
    Input    string
    Output   string
    Data     map[string]interface{}
    Metadata map[string]interface{}

    // 新增字段
    SessionState *SessionState `json:"session_state,omitempty"`
    SessionID    string        `json:"session_id,omitempty"`
    UserID       string        `json:"user_id,omitempty"`
}
```

#### 新增方法

##### NewExecutionContextWithSession()

```go
func NewExecutionContextWithSession(input, sessionID, userID string) *ExecutionContext
```

创建带会话状态的执行上下文。

**参数**:
- `input`: 初始输入字符串
- `sessionID`: 会话标识符
- `userID`: 用户标识符

##### SetSessionState(key string, value interface{})

```go
func (ec *ExecutionContext) SetSessionState(key string, value interface{})
```

设置会话状态 (便捷方法)。

##### GetSessionState(key string) (interface{}, bool)

```go
func (ec *ExecutionContext) GetSessionState(key string) (interface{}, bool)
```

获取会话状态 (便捷方法)。

## 高级用法

### 类型安全的访问

```go
// 使用类型断言确保类型安全
func getUserInfo(execCtx *workflow.ExecutionContext) (map[string]interface{}, error) {
    raw, ok := execCtx.GetSessionState("user_info")
    if !ok {
        return nil, fmt.Errorf("user_info not found")
    }

    userInfo, ok := raw.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("user_info has invalid type")
    }

    return userInfo, nil
}
```

### 结构化数据存储

```go
type UserProfile struct {
    ID    string
    Name  string
    Email string
    Age   int
}

// 存储结构体
profile := UserProfile{
    ID:    "user-123",
    Name:  "Alice",
    Email: "alice@example.com",
    Age:   30,
}
execCtx.SetSessionState("user_profile", profile)

// 读取结构体
raw, _ := execCtx.GetSessionState("user_profile")
profile := raw.(UserProfile)
fmt.Printf("User: %s (%s)\n", profile.Name, profile.Email)
```

### 嵌套数据访问

```go
// 存储嵌套数据
execCtx.SetSessionState("config", map[string]interface{}{
    "database": map[string]interface{}{
        "host": "localhost",
        "port": 5432,
    },
    "cache": map[string]interface{}{
        "enabled": true,
        "ttl":     300,
    },
})

// 访问嵌套数据
configRaw, _ := execCtx.GetSessionState("config")
config := configRaw.(map[string]interface{})
db := config["database"].(map[string]interface{})
fmt.Printf("Database: %s:%d\n", db["host"], db["port"])
```

### 会话隔离

```go
// 不同 session 使用不同的 ExecutionContext

// Session A
sessionA := workflow.NewExecutionContextWithSession("input-a", "session-a", "user-1")
sessionA.SetSessionState("session_name", "Session A")

// Session B
sessionB := workflow.NewExecutionContextWithSession("input-b", "session-b", "user-2")
sessionB.SetSessionState("session_name", "Session B")

// Session A 和 Session B 的状态完全隔离
nameA, _ := sessionA.GetSessionState("session_name")  // "Session A"
nameB, _ := sessionB.GetSessionState("session_name")  // "Session B"
```

## 最佳实践

### 1. 避免存储大型数据

```go
// ❌ 不推荐
execCtx.SetSessionState("all_users", []User{ /* 10000+ users */ })

// ✅ 推荐
execCtx.SetSessionState("user_ids", []string{"id1", "id2", "id3"})
```

**原因**: `Clone()` 使用 JSON 序列化,大型数据结构会降低性能。

### 2. 使用引用而非复制

```go
// ❌ 不推荐 - 修改不会影响 SessionState
data, _ := execCtx.GetSessionState("config")
config := data.(map[string]interface{})
config["new_key"] = "new_value"  // 丢失!

// ✅ 推荐 - 读取,修改,然后重新设置
configRaw, _ := execCtx.GetSessionState("config")
config := configRaw.(map[string]interface{})
config["new_key"] = "new_value"
execCtx.SetSessionState("config", config)  // 保存
```

### 3. 合理使用并行分支

```go
// ✅ 并行分支独立处理不同数据
// Branch A: 处理用户数据
// Branch B: 处理订单数据
// Branch C: 处理日志数据

// ⚠️ 避免并行分支修改同一个键
// (除非你理解最终写入优先策略)
```

**最佳实践**: 让每个并行分支处理不同的键:

```go
// Branch A
execCtx.SetSessionState("branch_a_counter", 1)

// Branch B
execCtx.SetSessionState("branch_b_counter", 2)

// Branch C
execCtx.SetSessionState("branch_c_counter", 3)
```

## 性能考虑

### 深拷贝性能

`Clone()` 使用 JSON 序列化实现深拷贝:

```go
// 高效的场景
execCtx.SetSessionState("counter", 42)           // 简单类型
execCtx.SetSessionState("name", "Alice")         // 字符串
execCtx.SetSessionState("enabled", true)         // 布尔值

// 低效的场景 (避免频繁克隆)
execCtx.SetSessionState("large_data", hugeSlice) // 大型数据结构
```

**基准测试结果**:
- 小值 (< 1KB): 每次克隆约 1-5 μs
- 中值 (1-10 KB): 每次克隆约 10-50 μs
- 大值 (> 10 KB): 每次克隆 > 50 μs

## 故障排查

### 常见问题

#### 1. SessionState 为 nil

**现象**:
```go
panic: runtime error: invalid memory address or nil pointer dereference
```

**原因**: 未初始化 SessionState

**解决**:
```go
// ❌ 错误
execCtx := &workflow.ExecutionContext{}
execCtx.SetSessionState("key", "value")  // panic!

// ✅ 正确
execCtx := workflow.NewExecutionContextWithSession("", "session-id", "user-id")
execCtx.SetSessionState("key", "value")  // OK
```

#### 2. 类型断言失败

**现象**:
```go
panic: interface conversion: interface {} is string, not int
```

**原因**: 类型不匹配

**解决**:
```go
// ❌ 错误
execCtx.SetSessionState("age", "30")  // 存储字符串
age := execCtx.GetSessionState("age").(int)  // 尝试读取为 int - panic!

// ✅ 正确
execCtx.SetSessionState("age", 30)  // 存储 int
raw, ok := execCtx.GetSessionState("age")
if !ok {
    // 键不存在
}
age, ok := raw.(int)
if !ok {
    // 类型不匹配
}
```

#### 3. 并行分支状态丢失

**现象**: 并行分支设置的状态在合并后丢失

**原因**: 没有理解最终写入优先策略

**解决**:
```go
// 避免并行分支修改同一个键
// 使用不同的键名

// ✅ 推荐
// Branch A
execCtx.SetSessionState("branch_a_counter", 1)

// Branch B
execCtx.SetSessionState("branch_b_counter", 2)

// Branch C
execCtx.SetSessionState("branch_c_counter", 3)
```

## 测试

完整的测试覆盖了以下场景:

- ✅ 基础 Get/Set 操作
- ✅ 深拷贝 (Clone)
- ✅ 状态合并 (Merge)
- ✅ 并发安全 (1000 goroutines)
- ✅ Workflow 集成测试
- ✅ 并行分支隔离
- ✅ 多租户隔离

**测试覆盖率**: 543 行测试代码

运行测试:
```bash
cd pkg/hno/workflow
go test -v -run TestSessionState
```

## 相关文档

- [Workflow 指南](/zh/guide/workflow) - Workflow 引擎使用
- [Team 指南](/zh/guide/team) - 多智能体协作
- [Memory 管理](/zh/guide/memory) - 对话记忆

---

**更新时间**: 2025-01-XX
