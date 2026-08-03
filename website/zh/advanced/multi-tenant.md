---
title: 多租户支持
description: 多租户数据隔离，单个 Agent 实例服务多个用户
outline: deep
---

# 多租户支持

**多租户支持** 使 Agno-Go 能够用单个 Agent 实例为多个用户提供服务，同时确保每个用户的对话历史和会话状态完全隔离。

---

## 概述

多租户架构允许单个应用实例为多个用户（租户）提供服务，数据完全隔离：

```
                 ┌─────────────────┐
                 │  Agent Instance │
                 └────────┬────────┘
                          │
         ┌────────────────┼────────────────┐
         ▼                ▼                ▼
   ┌──────────┐     ┌──────────┐     ┌──────────┐
   │  User A  │     │  User B  │     │  User C  │
   │ Messages │     │ Messages │     │ Messages │
   └──────────┘     └──────────┘     └──────────┘
```

---

## 什么是多租户？

多租户是一种架构模式，单个应用实例为多个隔离的用户或组织提供服务。每个租户的数据与其他租户完全分离。

### 不使用多租户

```go
// ❌ 每个用户需要单独的 Agent 实例
userAgents := make(map[string]*agent.Agent)

agent1, _ := agent.New(config)  // User 1
agent2, _ := agent.New(config)  // User 2
agent3, _ := agent.New(config)  // User 3
// ... 1000+ 用户 = 1000+ Agent 实例
```

**问题：**
- 高内存占用：1000 用户 = 1000 个 Agent 实例
- 难以管理：手动管理 Agent 生命周期
- 资源浪费：每个 Agent 都有重复配置

### 使用多租户

```go
// ✅ 单个 Agent 实例服务所有用户
sharedAgent, _ := agent.New(config)

// 不同用户使用不同的 userID
output1, _ := sharedAgent.Run(ctx, "user-1 input", "user-1")
output2, _ := sharedAgent.Run(ctx, "user-2 input", "user-2")
output3, _ := sharedAgent.Run(ctx, "user-3 input", "user-3")
```

**优势：**
- ✅ 低内存占用：单个 Agent 实例
- ✅ 易于管理：统一配置和更新
- ✅ 高效资源利用：共享模型和工具

---

## 快速开始

### 1. 创建多租户 Agent

```go
package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/memory"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    // 创建模型
    model, _ := openai.New("gpt-4", openai.Config{
        APIKey: "your-api-key",
    })

    // 创建多租户 Memory
    mem := memory.NewInMemory(100)  // 自动支持多租户

    // 创建 Agent
    myAgent, _ := agent.New(&agent.Config{
        Name:         "customer-service",
        Model:        model,
        Memory:       mem,
        Instructions: "你是一个有用的客服 Agent。",
    })

    // 不同用户的对话
    ctx := context.Background()

    // User A 的对话
    myAgent.UserID = "user-a"
    output1, _ := myAgent.Run(ctx, "我叫 Alice")
    fmt.Printf("User A: %s\n", output1.Content)

    output2, _ := myAgent.Run(ctx, "我的名字是什么？")  // "你的名字是 Alice"
    fmt.Printf("User A: %s\n", output2.Content)

    // User B 的对话
    myAgent.UserID = "user-b"
    output3, _ := myAgent.Run(ctx, "我叫 Bob")
    fmt.Printf("User B: %s\n", output3.Content)

    output4, _ := myAgent.Run(ctx, "我的名字是什么？")  // "你的名字是 Bob"
    fmt.Printf("User B: %s\n", output4.Content)

    // User A 再次对话
    myAgent.UserID = "user-a"
    output5, _ := myAgent.Run(ctx, "我的名字是什么？")  // "你的名字是 Alice"
    fmt.Printf("User A: %s\n", output5.Content)
}
```

### 2. Web API 示例

```go
package main

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
)

var sharedAgent *agent.Agent

func main() {
    // 初始化 Agent
    sharedAgent, _ = agent.New(&agent.Config{
        Name:   "api-agent",
        Model:  model,
        Memory: memory.NewInMemory(100),
    })

    // 设置路由
    router := gin.Default()
    router.POST("/chat", handleChat)
    router.Run(":8080")
}

type ChatRequest struct {
    UserID  string `json:"user_id"`
    Message string `json:"message"`
}

type ChatResponse struct {
    UserID  string `json:"user_id"`
    Reply   string `json:"reply"`
}

func handleChat(c *gin.Context) {
    var req ChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 设置当前用户 ID
    sharedAgent.UserID = req.UserID

    // 执行对话
    output, err := sharedAgent.Run(context.Background(), req.Message)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, ChatResponse{
        UserID: req.UserID,
        Reply:  output.Content,
    })
}
```

**测试：**
```bash
# User A 的对话
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-a", "message": "我叫 Alice"}'

curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-a", "message": "我的名字是什么？"}'
# Response: {"user_id":"user-a","reply":"你的名字是 Alice"}

# User B 的对话
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-b", "message": "我叫 Bob"}'

curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-b", "message": "我的名字是什么？"}'
# Response: {"user_id":"user-b","reply":"你的名字是 Bob"}
```

---

## 内存管理

### Memory 接口

Memory 接口支持可选的 `userID` 参数：

```go
// pkg/hno/memory/memory.go

type Memory interface {
    // 添加消息（支持可选的 userID）
    Add(message *types.Message, userID ...string)

    // 获取消息历史（支持可选的 userID）
    GetMessages(userID ...string) []*types.Message

    // 清空特定用户的消息
    Clear(userID ...string)

    // 清空所有用户的消息
    ClearAll()

    // 获取特定用户的消息数量
    Size(userID ...string) int
}
```

### InMemory 实现

```go
type InMemory struct {
    userMessages map[string][]*types.Message  // 用户 ID → 消息列表
    maxSize      int
    mu           sync.RWMutex
}

// 默认用户 ID
const defaultUserID = "default"

// 获取用户 ID（向后兼容）
func getUserID(userID ...string) string {
    if len(userID) > 0 && userID[0] != "" {
        return userID[0]
    }
    return defaultUserID
}
```

### 使用示例

#### 基础用法

```go
mem := memory.NewInMemory(100)

// User A 的消息
mem.Add(types.NewUserMessage("来自 Alice 的问候"), "user-a")
mem.Add(types.NewAssistantMessage("你好 Alice！"), "user-a")

// User B 的消息
mem.Add(types.NewUserMessage("来自 Bob 的问候"), "user-b")
mem.Add(types.NewAssistantMessage("你好 Bob！"), "user-b")

// 获取各用户的消息
messagesA := mem.GetMessages("user-a")  // 2 条消息
messagesB := mem.GetMessages("user-b")  // 2 条消息

fmt.Printf("User A 有 %d 条消息\n", len(messagesA))  // 2
fmt.Printf("User B 有 %d 条消息\n", len(messagesB))  // 2
```

#### 向后兼容

```go
mem := memory.NewInMemory(100)

// 不指定 userID（使用默认 "default"）
mem.Add(types.NewUserMessage("你好"))
messages := mem.GetMessages()

// 等价于：
mem.Add(types.NewUserMessage("你好"), "default")
messages := mem.GetMessages("default")
```

#### 清空操作

```go
mem := memory.NewInMemory(100)

// 添加不同用户的消息
mem.Add(types.NewUserMessage("User A 消息"), "user-a")
mem.Add(types.NewUserMessage("User B 消息"), "user-b")

// 清空特定用户
mem.Clear("user-a")
fmt.Printf("User A: %d 条消息\n", mem.Size("user-a"))  // 0
fmt.Printf("User B: %d 条消息\n", mem.Size("user-b"))  // 1

// 清空所有用户
mem.ClearAll()
fmt.Printf("User A: %d 条消息\n", mem.Size("user-a"))  // 0
fmt.Printf("User B: %d 条消息\n", mem.Size("user-b"))  // 0
```

---

## Agent 集成

### Agent 配置

```go
type Agent struct {
    ID           string
    Name         string
    Model        models.Model
    Toolkits     []toolkit.Toolkit
    Memory       memory.Memory
    Instructions string
    MaxLoops     int

    UserID string  // ⭐ 新增：多租户用户 ID
}

type Config struct {
    Name         string
    Model        models.Model
    Toolkits     []toolkit.Toolkit
    Memory       memory.Memory
    Instructions string
    MaxLoops     int

    UserID string  // ⭐ 新增：多租户用户 ID
}
```

### Run 方法实现

```go
// pkg/hno/agent/agent.go

func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error) {
    // ...

    // 所有 Memory 调用都传递 UserID
    userMsg := types.NewUserMessage(input)
    a.Memory.Add(userMsg, a.UserID)  // ⭐ 传递 UserID

    // ...

    messages := a.Memory.GetMessages(a.UserID)  // ⭐ 传递 UserID

    // ...

    a.Memory.Add(types.NewAssistantMessage(content), a.UserID)  // ⭐ 传递 UserID
}
```

### 使用模式

#### 模式 1：共享 Agent + 切换 UserID

```go
agent, _ := agent.New(&agent.Config{
    Name:   "shared-agent",
    Model:  model,
    Memory: memory.NewInMemory(100),
})

// 处理 User A 的请求
agent.UserID = "user-a"
output, _ := agent.Run(ctx, "User A 消息")

// 处理 User B 的请求
agent.UserID = "user-b"
output, _ := agent.Run(ctx, "User B 消息")
```

⚠️ **注意**：此方式需要在并发环境下小心处理 UserID 切换

#### 模式 2：每个用户独立 Agent（推荐用于高并发）

```go
// 创建 Agent 工厂
func createUserAgent(userID string) (*agent.Agent, error) {
    return agent.New(&agent.Config{
        Name:   "user-agent",
        Model:  sharedModel,  // 可以共享 Model
        Memory: memory.NewInMemory(100),
        UserID: userID,  // 设置固定的 UserID
    })
}

// 使用 Agent 池
userAgents := make(map[string]*agent.Agent)

// User A
if _, exists := userAgents["user-a"]; !exists {
    userAgents["user-a"], _ = createUserAgent("user-a")
}
output, _ := userAgents["user-a"].Run(ctx, "User A 消息")

// User B
if _, exists := userAgents["user-b"]; !exists {
    userAgents["user-b"], _ = createUserAgent("user-b")
}
output, _ := userAgents["user-b"].Run(ctx, "User B 消息")
```

---

## 数据隔离保证

### 1. 内存隔离

```go
// 测试：多租户隔离
mem := memory.NewInMemory(100)

// User A 添加 10 条消息
for i := 0; i < 10; i++ {
    mem.Add(types.NewUserMessage(fmt.Sprintf("User A 消息 %d", i)), "user-a")
}

// User B 添加 5 条消息
for i := 0; i < 5; i++ {
    mem.Add(types.NewUserMessage(fmt.Sprintf("User B 消息 %d", i)), "user-b")
}

// 验证隔离
assert.Equal(t, 10, mem.Size("user-a"))  // ✅
assert.Equal(t, 5, mem.Size("user-b"))   // ✅
assert.Equal(t, 0, mem.Size("user-c"))   // ✅ 不存在的用户

messagesA := mem.GetMessages("user-a")
messagesB := mem.GetMessages("user-b")

// User A 看不到 User B 的消息
for _, msg := range messagesA {
    assert.NotContains(t, msg.Content, "User B")  // ✅
}
```

### 2. 并发安全

```go
// 测试：1000 并发请求
mem := memory.NewInMemory(100)
var wg sync.WaitGroup

// 10 个用户，每个用户 100 个并发请求
for userID := 0; userID < 10; userID++ {
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(uid, msgID int) {
            defer wg.Done()
            userIDStr := fmt.Sprintf("user-%d", uid)
            msg := types.NewUserMessage(fmt.Sprintf("消息 %d", msgID))
            mem.Add(msg, userIDStr)
        }(userID, i)
    }
}

wg.Wait()

// 验证每个用户都有正确数量的消息
for userID := 0; userID < 10; userID++ {
    userIDStr := fmt.Sprintf("user-%d", userID)
    assert.Equal(t, 100, mem.Size(userIDStr))  // ✅
}
```

---

## 最佳实践

### 1. UserID 命名规范

```go
// ✅ 推荐：使用统一的命名规范
"user-{uuid}"           // user-123e4567-e89b-12d3-a456-426614174000
"org-{org_id}-user-{id}" // org-acme-user-001
"tenant-{id}"           // tenant-12345

// ❌ 避免：使用不稳定的标识
"{ip_address}"          // IP 可能变化
"{session_id}"          // Session 会过期
```

### 2. 错误处理

```go
// 验证 UserID
func validateUserID(userID string) error {
    if userID == "" {
        return fmt.Errorf("userID 不能为空")
    }
    if len(userID) > 255 {
        return fmt.Errorf("userID 太长（最大 255 个字符）")
    }
    // 可以添加更多验证规则
    return nil
}

// 在 API 层验证
func handleChat(c *gin.Context) {
    var req ChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    if err := validateUserID(req.UserID); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // ...
}
```

### 3. 日志和监控

```go
// 记录每个请求的 UserID
logger.Info("处理请求",
    "user_id", userID,
    "input_length", len(input),
    "timestamp", time.Now(),
)

// 监控指标
metrics.RecordUserRequest(userID)
metrics.RecordMessageCount(userID, mem.Size(userID))
```

### 4. 安全考虑

```go
// 使用加密的 UserID
func encryptUserID(plainUserID string) string {
    // 使用加密算法
    return encryptedID
}

// 访问控制
func checkUserPermission(userID string, action string) bool {
    // 实现权限检查逻辑
    return hasPermission
}
```

---

## 故障排查

### 常见问题

#### 1. 用户数据混乱

**现象：** User A 看到了 User B 的消息

**原因：** UserID 未正确传递

**解决：**
```go
// ❌ 错误
agent.Run(ctx, input)  // UserID 未设置

// ✅ 正确
agent.UserID = "user-a"
agent.Run(ctx, input)
```

#### 2. 内存占用过高

**现象：** 内存持续增长

**原因：** 未清理不活跃用户的数据

**解决：**
```go
// 定期清理
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        cleanupInactiveUsers(mem, 24*time.Hour)
    }
}()
```

#### 3. 并发竞争

**现象：** 数据偶尔丢失或重复

**原因：** 共享 Agent 的 UserID 字段在并发环境下被多个 goroutine 修改

**解决：**
```go
// ❌ 错误：并发修改共享 Agent
var sharedAgent *agent.Agent
go func() { sharedAgent.UserID = "user-a"; sharedAgent.Run(ctx, input) }()
go func() { sharedAgent.UserID = "user-b"; sharedAgent.Run(ctx, input) }()

// ✅ 正确：每个用户独立 Agent
agentA := createUserAgent("user-a")
agentB := createUserAgent("user-b")
go func() { agentA.Run(ctx, input) }()
go func() { agentB.Run(ctx, input) }()
```

---

## 与其他功能的集成

### A2A 接口 + 多租户

```go
// A2A 请求包含 contextID，可以作为 userID
type Message struct {
    MessageID string `json:"messageId"`
    Role      string `json:"role"`
    AgentID   string `json:"agentId"`
    ContextID string `json:"contextId"`  // ⭐ 可以作为 userID
    Parts     []Part `json:"parts"`
}

// 映射时设置 UserID
func MapA2ARequestToRunInput(req *JSONRPC2Request) (*RunInput, error) {
    // ...
    agent.UserID = req.Params.Message.ContextID  // ⭐ 使用 contextID 作为 userID
    // ...
}
```

### Session State + 多租户

```go
// ExecutionContext 同时支持 SessionID 和 UserID
execCtx := workflow.NewExecutionContextWithSession(
    "input",
    "session-123",  // SessionID：单次会话的标识
    "user-a",       // UserID：用户的标识
)

// SessionID：用于会话状态管理
// UserID：用于多租户数据隔离
```

---

## 相关文档

- [A2A 接口](/zh/api/a2a) - Agent 间通信
- [Session State 管理](/zh/guide/session-state) - 工作流会话管理
- [Memory 指南](/zh/guide/memory) - Memory 使用指南

---

## 测试

完整的测试覆盖了以下场景：

- ✅ 多用户数据隔离
- ✅ 并发安全（1000 goroutines）
- ✅ Agent 集成测试
- ✅ Memory 容量管理
- ✅ 清空操作正确性

**测试覆盖率：** 93.1%（Memory 模块）

运行测试：
```bash
cd pkg/hno/memory
go test -v -run TestInMemory

cd pkg/hno/agent
go test -v -run TestAgent_MultiTenant
```

---

**更新时间：** 2025-01-08
