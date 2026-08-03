# Team 协作示例

## 概述

本示例演示 Agno-Go 的多 Agent Team 协作能力。Team 允许多个 Agent 使用不同的协调模式协同工作:Sequential、Parallel、Leader-Follower 和 Consensus。每种模式适合不同类型的任务和协作模式。

## 你将学到

- 如何创建多 Agent Team
- 四种 Team 协调模式及何时使用每种模式
- Agent 如何共享上下文并在彼此的工作基础上构建
- 如何访问单个 Agent 的输出

## 前置要求

- Go 1.21 或更高版本
- OpenAI API key

## 设置

```bash
export OPENAI_API_KEY=sk-your-api-key-here
cd cmd/examples/team_demo
```

## 完整代码

完整示例包含 4 个演示 - 详见下面的代码解释。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/team"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	ctx := context.Background()

	// Demo 1: Sequential Team
	fmt.Println("=== Demo 1: Sequential Team ===")
	runSequentialDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 2: Parallel Team ===")
	runParallelDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 3: Leader-Follower Team ===")
	runLeaderFollowerDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 4: Consensus Team ===")
	runConsensusDemo(ctx, apiKey)
}
```

## Team 协调模式

### 1. Sequential 模式

Agent 依次工作,每个 Agent 在前一个 Agent 的输出基础上构建。

```go
func runSequentialDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create 3 agents for sequential processing
	researcher, _ := agent.New(agent.Config{
		Name:         "Researcher",
		Model:        model,
		Instructions: "You are a research expert. Analyze the topic and provide key facts.",
	})

	analyst, _ := agent.New(agent.Config{
		Name:         "Analyst",
		Model:        model,
		Instructions: "You are an analyst. Take research findings and extract insights.",
	})

	writer, _ := agent.New(agent.Config{
		Name:         "Writer",
		Model:        model,
		Instructions: "You are a writer. Take insights and write a concise summary.",
	})

	// Create sequential team
	t, _ := team.New(team.Config{
		Name:   "Content Pipeline",
		Agents: []*agent.Agent{researcher, analyst, writer},
		Mode:   team.ModeSequential,
	})

	// Run team
	output, _ := t.Run(ctx, "Analyze the benefits of AI in healthcare")

	fmt.Printf("Final Output: %s\n", output.Content)
	fmt.Printf("Agents involved: %d\n", len(output.AgentOutputs))
}
```

**流程:**
1. **Researcher** 分析主题 → 产生研究结果
2. **Analyst** 接收结果 → 提取洞察
3. **Writer** 接收洞察 → 撰写最终摘要

**用例:**
- 内容创建管道
- 数据处理工作流
- 多阶段分析任务

### 2. Parallel 模式

所有 Agent 同时处理相同的输入,合并它们的输出。

```go
func runParallelDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create agents with different specializations
	techAgent, _ := agent.New(agent.Config{
		Name:         "Tech Specialist",
		Model:        model,
		Instructions: "You are a technology expert. Focus on technical aspects.",
	})

	bizAgent, _ := agent.New(agent.Config{
		Name:         "Business Specialist",
		Model:        model,
		Instructions: "You are a business expert. Focus on business implications.",
	})

	ethicsAgent, _ := agent.New(agent.Config{
		Name:         "Ethics Specialist",
		Model:        model,
		Instructions: "You are an ethics expert. Focus on ethical considerations.",
	})

	// Create parallel team
	t, _ := team.New(team.Config{
		Name:   "Multi-Perspective Analysis",
		Agents: []*agent.Agent{techAgent, bizAgent, ethicsAgent},
		Mode:   team.ModeParallel,
	})

	output, _ := t.Run(ctx, "Evaluate the impact of autonomous vehicles")
	fmt.Printf("Combined Analysis:\n%s\n", output.Content)
}
```

**流程:**
1. 所有 Agent 同时接收相同输入
2. 每个 Agent 提供其视角
3. 输出被合并为综合分析

**用例:**
- 多视角分析
- 头脑风暴会议
- 独立评估
- 并行数据处理

### 3. Leader-Follower 模式

领导者 Agent 将任务委派给跟随者 Agent 并综合结果。

```go
func runLeaderFollowerDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create leader
	leader, _ := agent.New(agent.Config{
		Name:         "Team Leader",
		Model:        model,
		Instructions: "You are a team leader. Delegate tasks and synthesize results.",
	})

	// Create followers with tools
	calcAgent, _ := agent.New(agent.Config{
		Name:         "Calculator Agent",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calculator.New()},
		Instructions: "You perform calculations as requested.",
	})

	dataAgent, _ := agent.New(agent.Config{
		Name:         "Data Agent",
		Model:        model,
		Instructions: "You analyze and present data.",
	})

	// Create leader-follower team
	t, _ := team.New(team.Config{
		Name:   "Project Team",
		Leader: leader,
		Agents: []*agent.Agent{calcAgent, dataAgent},
		Mode:   team.ModeLeaderFollower,
	})

	output, _ := t.Run(ctx, "Calculate the ROI for a $100,000 investment with 15% annual return over 5 years")
	fmt.Printf("Leader's Final Report: %s\n", output.Content)
}
```

**流程:**
1. **Leader** 分析任务并委派给跟随者
2. **Followers** 执行分配的子任务
3. **Leader** 综合结果并提供最终输出

**用例:**
- 复杂任务分解
- 层级工作流
- 项目管理场景
- 专用工具使用

### 4. Consensus 模式

Agent 讨论直到达成一致或达到最大轮数。

```go
func runConsensusDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create agents with different perspectives
	optimist, _ := agent.New(agent.Config{
		Name:         "Optimist",
		Model:        model,
		Instructions: "You are optimistic and focus on opportunities.",
	})

	realist, _ := agent.New(agent.Config{
		Name:         "Realist",
		Model:        model,
		Instructions: "You are realistic and balanced in your views.",
	})

	critic, _ := agent.New(agent.Config{
		Name:         "Critic",
		Model:        model,
		Instructions: "You are critical and focus on potential problems.",
	})

	// Create consensus team
	t, _ := team.New(team.Config{
		Name:      "Decision Team",
		Agents:    []*agent.Agent{optimist, realist, critic},
		Mode:      team.ModeConsensus,
		MaxRounds: 2,
	})

	output, _ := t.Run(ctx, "Should we invest in renewable energy for our company?")

	fmt.Printf("Consensus Result: %s\n", output.Content)
	fmt.Printf("Total discussion rounds: %v\n", output.Metadata["rounds"])
}
```

**流程:**
1. **第 1 轮**: 每个 Agent 提供初始观点
2. **第 2 轮**: Agent 看到其他人的观点并改进立场
3. **最终**: 系统综合共识或最终立场

**用例:**
- 决策制定
- 辩论模拟
- 多视角分析
- 风险评估

## Team 配置

### 基本配置

```go
team.Config{
	Name:   "My Team",           // Team 标识符
	Agents: []*agent.Agent{...}, // Team 成员
	Mode:   team.ModeSequential, // 协调模式
}
```

### 高级配置

```go
team.Config{
	Name:      "Decision Team",
	Leader:    leaderAgent,      // 用于 Leader-Follower 模式
	Agents:    followerAgents,   // Team 成员
	Mode:      team.ModeConsensus,
	MaxRounds: 3,                // 用于 Consensus 模式
}
```

## 访问结果

### Team 输出

```go
output, err := t.Run(ctx, "Your query here")

// 最终结果
fmt.Println(output.Content)

// 单个 Agent 输出
for _, agentOut := range output.AgentOutputs {
	fmt.Printf("%s: %s\n", agentOut.AgentName, agentOut.Content)
}

// 元数据
fmt.Printf("Rounds: %v\n", output.Metadata["rounds"])
```

### 单个 Agent 输出

```go
// 访问特定 Agent 的贡献
if len(output.AgentOutputs) > 0 {
	firstAgent := output.AgentOutputs[0]
	fmt.Printf("Agent: %s\n", firstAgent.AgentName)
	fmt.Printf("Output: %s\n", firstAgent.Content)
}
```

## 运行示例

```bash
go run main.go
```

## 预期输出

```
=== Demo 1: Sequential Team ===
Final Output: AI in healthcare offers significant benefits including improved diagnostic accuracy through machine learning, personalized treatment plans, reduced administrative burden, and enhanced patient monitoring through IoT devices.
Agents involved: 3

=== Demo 2: Parallel Team ===
Combined Analysis:
Technical: Autonomous vehicles use advanced sensors, AI algorithms, and real-time processing...
Business: Market disruption, new revenue models, infrastructure investment needs...
Ethics: Privacy concerns, liability questions, job displacement, safety standards...

=== Demo 3: Leader-Follower Team ===
Leader's Final Report: Based on calculations, a $100,000 investment at 15% annual return over 5 years yields $201,136, representing a 101% ROI.

=== Demo 4: Consensus Team ===
Consensus Result: After thorough discussion, the team recommends investing in renewable energy with careful planning for upfront costs and long-term savings.
Total discussion rounds: 2
```

## 模式比较

| 模式 | 何时使用 | Agent 数量 | 通信模式 |
|------|-------------|-------------|----------------------|
| **Sequential** | 管道任务、有序步骤 | 2-10 | 线性: A → B → C |
| **Parallel** | 独立任务、多视角 | 2-20 | 广播: 所有获得相同输入 |
| **Leader-Follower** | 复杂委派、层级 | 1 个领导者 + 1-10 个跟随者 | 星型: 领导者协调 |
| **Consensus** | 决策制定、辩论 | 2-5 | 轮流讨论 |

## 最佳实践

### 1. 选择合适的模式

```go
// Sequential: 当顺序很重要时
team.ModeSequential  // 研究 → 分析 → 写作

// Parallel: 当需要多个视角时
team.ModeParallel    // 技术 + 商业 + 法律分析

// Leader-Follower: 当需要委派时
team.ModeLeaderFollower  // 复杂任务分解

// Consensus: 当需要达成一致时
team.ModeConsensus   // 决策制定、辩论
```

### 2. 设计清晰的 Agent 角色

```go
// ✅ 好: 具体、独特的角色
researcher := "You are a research expert. Focus on facts and data."
analyst := "You are an analyst. Extract insights from research."

// ❌ 坏: 重叠、模糊的角色
agent1 := "You are helpful."
agent2 := "You are smart."
```

### 3. 优化 Agent 数量

- **Sequential**: 2-5 个 Agent (更多 = 更长的管道)
- **Parallel**: 2-10 个 Agent (更多 = 更丰富的分析)
- **Leader-Follower**: 1 个领导者 + 2-5 个跟随者
- **Consensus**: 2-4 个 Agent (更多 = 更难收敛)

### 4. 处理错误

```go
output, err := team.Run(ctx, query)
if err != nil {
	log.Printf("Team execution failed: %v", err)
	// 回退逻辑
}
```

## 高级模式

### 混合工具使用

```go
// 一些 Agent 有工具,其他没有
calcAgent := agent.New(agent.Config{
	Toolkits: []toolkit.Toolkit{calculator.New()},
})

analysisAgent := agent.New(agent.Config{
	// 没有工具,纯推理
})
```

### 动态 Team 组成

```go
var agents []*agent.Agent

if needsCalculation {
	agents = append(agents, calcAgent)
}
if needsWebSearch {
	agents = append(agents, searchAgent)
}

team, _ := team.New(team.Config{Agents: agents, Mode: team.ModeParallel})
```

### 嵌套 Team

```go
// 创建子 Team
researchTeam := team.New(team.Config{...})
analysisTeam := team.New(team.Config{...})

// 将一个 Team 的输出用作另一个的输入
researchOutput, _ := researchTeam.Run(ctx, query)
finalOutput, _ := analysisTeam.Run(ctx, researchOutput.Content)
```

## 性能考虑

### Sequential 模式
- **延迟**: 所有 Agent 时间之和 (最慢)
- **成本**: 所有 Agent 成本之和
- **最适合**: 当顺序很关键时

### Parallel 模式
- **延迟**: Agent 时间的最大值 (更快)
- **成本**: 所有 Agent 成本之和
- **最适合**: 当速度很重要时

### Leader-Follower 模式
- **延迟**: 领导者 + 跟随者 (中等)
- **成本**: 领导者 + 跟随者成本
- **最适合**: 复杂任务委派

### Consensus 模式
- **延迟**: 轮数 × Agent 时间 (最慢)
- **成本**: 轮数 × Agent 数量
- **最适合**: 当共识很关键时

## 下一步

- 从 [Simple Agent](./simple-agent.md) 基础开始
- 探索 [Workflow 引擎](./workflow-demo.md) 进行受控执行
- 使用 Team 协作构建 [RAG 系统](./rag-demo.md)
- 尝试不同的 [模型提供商](./claude-agent.md)

## 故障排除

**Agent 协作不有效:**
- 审查 Agent 指令的清晰度
- 检查模式是否适合任务
- 确保 Agent 有独特的角色

**Sequential Team 太慢:**
- 减少 Agent 数量
- 使用更小/更快的模型
- 考虑 Parallel 模式

**Consensus 不收敛:**
- 增加 MaxRounds
- 简化决策
- 减少 Agent 数量
- 调整 Agent 指令

**Leader 未正确委派:**
- 明确领导者的委派指令
- 确保跟随者有合适的工具
- 检查跟随者指令是否清晰
