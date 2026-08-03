# Memory - 对话历史

为您的 Agent 管理对话历史和上下文。

---

## 什么是 Memory?

Memory 存储对话历史,允许 Agent 在多次交互中保持上下文。Agno-Go 提供了带自动截断的内置记忆管理。

### 核心特性

- **自动历史**: 对话自动存储
- **可配置限制**: 控制记忆大小
- **消息类型**: System、User、Assistant、Tool 消息
- **手动控制**: 以编程方式清除或管理记忆

---

## 基本用法

### 默认 Memory

Agent 默认启用记忆:

```go
package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    agent, _ := agent.New(agent.Config{
        Model: model,
    })

    // First interaction
    agent.Run(context.Background(), "My name is Alice")
    // Response: "Nice to meet you, Alice!"

    // Second interaction - agent remembers
    output, _ := agent.Run(context.Background(), "What's my name?")
    fmt.Println(output.Content)
    // Response: "Your name is Alice."
}
```

---

## 配置

### 自定义 Memory 限制

设置要存储的最大消息数:

```go
import "github.com/rexleimo/agno-go/pkg/hno/memory"

customMemory := memory.New(memory.Config{
    MaxMessages: 50,  // Store up to 50 messages
})

agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: customMemory,
})
```

### 无 Memory

为无状态 Agent 禁用记忆:

```go
agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: nil,  // No conversation history
})
```

---

## Memory 操作

### 清除 Memory

重置对话历史:

```go
// Clear all history
agent.ClearMemory()

// Start fresh conversation
agent.Run(ctx, "New conversation")
```

### 获取消息历史

访问已存储的消息:

```go
messages := agent.Memory.GetMessages()
for _, msg := range messages {
    fmt.Printf("%s: %s\n", msg.Role, msg.Content)
}
```

### 添加自定义消息

手动向记忆添加消息:

```go
import "github.com/rexleimo/agno-go/pkg/hno/types"

// Add system message
agent.Memory.AddMessage(types.Message{
    Role:    types.RoleSystem,
    Content: "You are a helpful assistant.",
})

// Add user message
agent.Memory.AddMessage(types.Message{
    Role:    types.RoleUser,
    Content: "Hello!",
})
```

---

## 消息类型

### System 消息

Agent 的指令:

```go
types.Message{
    Role:    types.RoleSystem,
    Content: "You are a Python expert. Help with coding questions.",
}
```

### User 消息

用户输入:

```go
types.Message{
    Role:    types.RoleUser,
    Content: "How do I read a file in Python?",
}
```

### Assistant 消息

Agent 响应:

```go
types.Message{
    Role:    types.RoleAssistant,
    Content: "Use the open() function: open('file.txt', 'r')",
}
```

### Tool 消息

工具执行结果:

```go
types.Message{
    Role:       types.RoleTool,
    Content:    "Result: 42",
    ToolCallID: "call_123",
}
```

---

## Memory 模式

### 基于会话的 Memory

在会话之间清除记忆:

```go
func handleSession(agent *agent.Agent, sessionID string) {
    // Load session history (from database, etc.)
    loadSessionHistory(agent, sessionID)

    // Handle conversation
    output, _ := agent.Run(ctx, userInput)

    // Save session history
    saveSessionHistory(agent, sessionID)

    // Clean up
    agent.ClearMemory()
}
```

### 滑动窗口

仅保留最近的消息:

```go
memory := memory.New(memory.Config{
    MaxMessages: 20,  // Keep last 20 messages
})

// Automatically truncates older messages
agent, _ := agent.New(agent.Config{
    Memory: memory,
})
```

### 持久化 Memory

保存和恢复对话:

```go
// Save conversation
messages := agent.Memory.GetMessages()
saveToDatabase(sessionID, messages)

// Restore conversation
savedMessages := loadFromDatabase(sessionID)
for _, msg := range savedMessages {
    agent.Memory.AddMessage(msg)
}
```

---

## 高级用法

### 多 Agent Memory 共享

在 Agent 之间共享上下文:

```go
// Create shared memory
sharedMemory := memory.New(memory.Config{
    MaxMessages: 100,
})

// Both agents use same memory
agent1, _ := agent.New(agent.Config{
    Name:   "Agent1",
    Model:  model,
    Memory: sharedMemory,
})

agent2, _ := agent.New(agent.Config{
    Name:   "Agent2",
    Model:  model,
    Memory: sharedMemory,
})

// Agent1 conversation is visible to Agent2
agent1.Run(ctx, "Store this information: X=42")
output, _ := agent2.Run(ctx, "What is X?")
// Agent2 can see Agent1's conversation
```

### 条件 Memory

基于条件清除记忆:

```go
messageCount := len(agent.Memory.GetMessages())

if messageCount > 100 {
    // Keep only system message
    systemMsg := agent.Memory.GetMessages()[0]
    agent.ClearMemory()
    agent.Memory.AddMessage(systemMsg)
}
```

### Memory 检查

分析对话历史:

```go
messages := agent.Memory.GetMessages()

var userMessages, assistantMessages int
for _, msg := range messages {
    switch msg.Role {
    case types.RoleUser:
        userMessages++
    case types.RoleAssistant:
        assistantMessages++
    }
}

fmt.Printf("User messages: %d, Assistant messages: %d\n",
    userMessages, assistantMessages)
```

---

## Memory 配置

### Config 结构

```go
type Config struct {
    MaxMessages int // Maximum number of messages to store (default: 100)
}
```

### 默认行为

- 自动存储所有对话消息
- 达到限制时截断最旧消息
- 截断期间保留系统消息

---

## 最佳实践

### 1. 设置适当的限制

平衡上下文和性能:

```go
// Short conversations
memory := memory.New(memory.Config{MaxMessages: 20})

// Long conversations
memory := memory.New(memory.Config{MaxMessages: 100})

// Very long context
memory := memory.New(memory.Config{MaxMessages: 500})
```

### 2. 策略性地清除 Memory

在上下文改变时重置:

```go
// New topic
if isNewTopic(userInput) {
    agent.ClearMemory()
}

// New session
if isNewSession(sessionID) {
    agent.ClearMemory()
}
```

### 3. 监控 Memory 使用

跟踪对话长度:

```go
messages := agent.Memory.GetMessages()
if len(messages) > 80 {
    log.Printf("Warning: Approaching memory limit (%d/100)", len(messages))
}
```

### 4. 保留重要上下文

保持系统指令:

```go
// Save system message
systemMsg := agent.Memory.GetMessages()[0]

// Clear memory
agent.ClearMemory()

// Restore system message
agent.Memory.AddMessage(systemMsg)
```

---

## Memory vs Context Window

### Memory (Agno-Go)
- 由 Agno-Go 管理
- 可配置的消息限制
- 自动截断

### Context Window (LLM)
- 模型特定限制 (例如 128K Token)
- 由 LLM 提供商管理
- 超出时可能导致错误

**最佳实践**: 保持 Memory 限制低于 LLM 上下文窗口。

```go
// GPT-4o-mini: 128K tokens ≈ 100K words ≈ 400 messages
memory := memory.New(memory.Config{MaxMessages: 200})
```

---

## 故障排除

### Agent 不记得

检查记忆配置:

```go
// Bad ❌ - No memory
agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: nil,
})

// Good ✅ - With memory
agent, _ := agent.New(agent.Config{
    Model: model,
    // Memory enabled by default
})
```

### Memory 太大

减少消息限制:

```go
memory := memory.New(memory.Config{
    MaxMessages: 50,  // Smaller limit
})
```

### 丢失上下文

不要不必要地清除记忆:

```go
// Bad ❌ - Clears after each message
output, _ := agent.Run(ctx, input)
agent.ClearMemory() // DON'T DO THIS

// Good ✅ - Preserves context
output, _ := agent.Run(ctx, input)
// Memory maintained for next interaction
```

---

## 示例

### 多轮对话

```go
agent, _ := agent.New(agent.Config{Model: model})

// Turn 1
agent.Run(ctx, "I'm planning a trip to Paris")

// Turn 2
agent.Run(ctx, "What's the weather like there?")
// Agent knows "there" = Paris

// Turn 3
agent.Run(ctx, "What should I pack?")
// Agent knows about Paris and weather
```

### 会话管理

```go
type SessionManager struct {
    agents map[string]*agent.Agent
}

func (sm *SessionManager) GetAgent(sessionID string) *agent.Agent {
    if ag, exists := sm.agents[sessionID]; exists {
        return ag
    }

    // Create new agent for session
    ag, _ := agent.New(agent.Config{Model: model})
    sm.agents[sessionID] = ag
    return ag
}

func (sm *SessionManager) EndSession(sessionID string) {
    if ag, exists := sm.agents[sessionID]; exists {
        ag.ClearMemory()
        delete(sm.agents, sessionID)
    }
}
```

---

## 下一步

- 使用共享记忆构建 [Teams](/guide/team)
- 添加 [Tools](/guide/tools) 增强能力
- 使用上下文传递创建 [Workflows](/guide/workflow)
- 查看 [Memory API Reference](/api/memory) 获取详细文档

---

## 相关示例

- [Simple Agent](/examples/simple-agent) - 基础记忆使用
- [Multi-Turn Chat](/examples/chat-agent) - 对话示例
- [Session Management](/examples/session-demo) - 基于会话的记忆
