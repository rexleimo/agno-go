# Team - 多 Agent 协作

使用 4 种协作模式构建强大的多 Agent 系统。

---

## 什么是 Team?

**Team** 是一组 Agent 协同工作以解决复杂任务的集合。不同的协作模式支持各种协作模式。

### 核心特性

- **4 种协作模式**: Sequential(顺序)、Parallel(并行)、Leader-Follower(领导-跟随)、Consensus(共识)
- **动态成员**: 运行时添加/移除 Agent
- **灵活配置**: 每种模式的行为可定制
- **类型安全**: 完整的 Go 类型检查

---

## 创建 Team

### 基础示例

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/team"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    // Create model
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    // Create team members
    researcher, _ := agent.New(agent.Config{
        Name:         "Researcher",
        Model:        model,
        Instructions: "You are a research expert.",
    })

    writer, _ := agent.New(agent.Config{
        Name:         "Writer",
        Model:        model,
        Instructions: "You are a technical writer.",
    })

    // Create team
    t, err := team.New(team.Config{
        Name:   "Content Team",
        Agents: []*agent.Agent{researcher, writer},
        Mode:   team.ModeSequential,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Run team
    output, _ := t.Run(context.Background(), "Write about AI")
    fmt.Println(output.Content)
}
```

---

## 协作模式

### 1. Sequential 模式

Agent 依次执行,输出传递给下一个 Agent。

```go
t, _ := team.New(team.Config{
    Name:   "Pipeline",
    Agents: []*agent.Agent{agent1, agent2, agent3},
    Mode:   team.ModeSequential,
})
```

**使用场景:**
- 内容管道 (研究 → 写作 → 编辑)
- 数据处理工作流
- 多步推理

**工作原理:**
1. Agent 1 处理输入 → 输出 A
2. Agent 2 处理输出 A → 输出 B
3. Agent 3 处理输出 B → 最终输出

---

### 2. Parallel 模式

所有 Agent 同时执行,结果合并。

```go
t, _ := team.New(team.Config{
    Name:   "Multi-Perspective",
    Agents: []*agent.Agent{techAgent, bizAgent, ethicsAgent},
    Mode:   team.ModeParallel,
})
```

**使用场景:**
- 多视角分析
- 并行数据处理
- 生成多样化意见

**工作原理:**
1. 所有 Agent 接收相同输入
2. 并发执行 (Go goroutine)
3. 结果合并为单一输出

---

### 3. Leader-Follower 模式

领导者将任务委派给跟随者并综合结果。

```go
t, _ := team.New(team.Config{
    Name:   "Project Team",
    Leader: leaderAgent,
    Agents: []*agent.Agent{follower1, follower2},
    Mode:   team.ModeLeaderFollower,
})
```

**使用场景:**
- 任务委派
- 层级化工作流
- 专家咨询

**工作原理:**
1. 领导者分析任务并创建子任务
2. 委派给适当的跟随者
3. 综合跟随者的输出为最终结果

---

### 4. Consensus 模式

Agent 讨论直到达成一致。

```go
t, _ := team.New(team.Config{
    Name:      "Decision Team",
    Agents:    []*agent.Agent{optimist, realist, critic},
    Mode:      team.ModeConsensus,
    MaxRounds: 3,  // Maximum discussion rounds
})
```

**使用场景:**
- 决策制定
- 质量保证
- 辩论和改进

**工作原理:**
1. 所有 Agent 提供初始意见
2. Agent 审查其他人的意见
3. 迭代直到达成共识或达到最大轮次
4. 最终共识输出

---

## 配置

### Config 结构

```go
type Config struct {
    // Required
    Agents []*agent.Agent  // Team 成员

    // Optional
    Name      string              // Team 名称 (默认: "Team")
    Mode      CoordinationMode    // 协作模式 (默认: Sequential)
    Leader    *agent.Agent        // 领导者 (用于 LeaderFollower 模式)
    MaxRounds int                 // 最大轮次 (用于 Consensus 模式, 默认: 3)
}
```

### 协作模式

```go
const (
    ModeSequential     CoordinationMode = "sequential"
    ModeParallel       CoordinationMode = "parallel"
    ModeLeaderFollower CoordinationMode = "leader-follower"
    ModeConsensus      CoordinationMode = "consensus"
)
```

---

## API 参考

### team.New

创建新的 Team 实例。

**签名:**
```go
func New(config Config) (*Team, error)
```

**返回:**
- `*Team`: 创建的 Team 实例
- `error`: 如果 Agent 列表为空或配置无效则返回错误

---

### Team.Run

使用输入执行 Team。

**签名:**
```go
func (t *Team) Run(ctx context.Context, input string) (*RunOutput, error)
```

**参数:**
- `ctx`: 用于取消/超时的 Context
- `input`: 用户输入字符串

**返回:**
```go
type RunOutput struct {
    Content      string                 // Team 最终输出
    AgentOutputs []AgentOutput          // 各个 Agent 的输出
    Metadata     map[string]interface{} // 附加元数据
}
```

---

### Team.AddAgent / RemoveAgent

动态管理 Team 成员。

**签名:**
```go
func (t *Team) AddAgent(ag *agent.Agent)
func (t *Team) RemoveAgent(name string) error
func (t *Team) GetAgents() []*agent.Agent
```

**示例:**
```go
// Add new agent
t.AddAgent(newAgent)

// Remove agent by name
err := t.RemoveAgent("OldAgent")

// Get all agents
agents := t.GetAgents()
```

---

## 完整示例

### Sequential Team 示例

使用 研究 → 分析 → 写作 的内容创建管道。

```go
func createContentPipeline(apiKey string) {
    model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

    researcher, _ := agent.New(agent.Config{
        Name:         "Researcher",
        Model:        model,
        Instructions: "Research the topic and provide key facts.",
    })

    analyst, _ := agent.New(agent.Config{
        Name:         "Analyst",
        Model:        model,
        Instructions: "Analyze research findings and extract insights.",
    })

    writer, _ := agent.New(agent.Config{
        Name:         "Writer",
        Model:        model,
        Instructions: "Write a concise summary based on insights.",
    })

    t, _ := team.New(team.Config{
        Name:   "Content Pipeline",
        Agents: []*agent.Agent{researcher, analyst, writer},
        Mode:   team.ModeSequential,
    })

    output, _ := t.Run(context.Background(),
        "Write about the benefits of AI in healthcare")

    fmt.Println(output.Content)
}
```

### Parallel Team 示例

并发执行的多视角分析。

```go
func multiPerspectiveAnalysis(apiKey string) {
    model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

    techAgent, _ := agent.New(agent.Config{
        Name:         "Tech Specialist",
        Model:        model,
        Instructions: "Focus on technical aspects.",
    })

    bizAgent, _ := agent.New(agent.Config{
        Name:         "Business Specialist",
        Model:        model,
        Instructions: "Focus on business implications.",
    })

    ethicsAgent, _ := agent.New(agent.Config{
        Name:         "Ethics Specialist",
        Model:        model,
        Instructions: "Focus on ethical considerations.",
    })

    t, _ := team.New(team.Config{
        Name:   "Analysis Team",
        Agents: []*agent.Agent{techAgent, bizAgent, ethicsAgent},
        Mode:   team.ModeParallel,
    })

    output, _ := t.Run(context.Background(),
        "Evaluate the impact of autonomous vehicles")

    fmt.Println(output.Content)
}
```

---

## 最佳实践

### 1. 选择正确的模式

- **Sequential**: 当输出依赖于前面的步骤时使用
- **Parallel**: 当视角相互独立时使用
- **Leader-Follower**: 当需要任务委派时使用
- **Consensus**: 当质量和一致性至关重要时使用

### 2. Agent 专业化

给每个 Agent 明确、具体的指令:

```go
// Good ✅
Instructions: "You are a Python expert. Focus on code quality."

// Bad ❌
Instructions: "You help with coding."
```

### 3. 错误处理

始终处理 Team 操作的错误:

```go
output, err := t.Run(ctx, input)
if err != nil {
    log.Printf("Team execution failed: %v", err)
    return
}
```

### 4. Context 管理

使用 Context 进行超时和取消:

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

output, err := t.Run(ctx, input)
```

---

## 性能考虑

### 并行执行

Parallel 模式使用 Go goroutine 实现真正的并发:

```go
// 3 agents execute simultaneously
t, _ := team.New(team.Config{
    Agents: []*agent.Agent{a1, a2, a3},
    Mode:   team.ModeParallel,
})

// Total time ≈ slowest agent (not sum of all)
```

### 内存使用

每个 Agent 维护自己的记忆。对于大型 Team:

```go
// Clear memory after each run
output, _ := t.Run(ctx, input)
for _, ag := range t.GetAgents() {
    ag.ClearMemory()
}
```

---

## 下一步

- 了解 [Workflow](/guide/workflow) 的基于步骤的编排
- 探索 [Models](/guide/models) 的不同 LLM 提供商
- 添加 [Tools](/guide/tools) 增强 Agent 能力
- 查看 [Team API Reference](/api/team) 获取详细的 API 文档

---

## 相关示例

- [Team Demo](/examples/team-demo) - 完整工作示例
- [Leader-Follower Pattern](/examples/team-demo#leader-follower)
- [Consensus Decision Making](/examples/team-demo#consensus)
