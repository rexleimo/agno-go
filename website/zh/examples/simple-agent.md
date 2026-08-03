# Simple Agent 示例

## 概述

本示例演示了 Agno-Go 的基本用法,创建一个具有工具调用能力的简单 AI Agent。该 Agent 使用 OpenAI 的 GPT-4o-mini 模型,并配备了计算器工具集来执行数学运算。

## 你将学到

- 如何创建和配置 OpenAI 模型
- 如何设置带工具的 Agent
- 如何使用用户查询运行 Agent
- 如何访问执行元数据(循环次数、token 使用量)

## 前置要求

- Go 1.21 或更高版本
- OpenAI API key

## 设置

1. 设置你的 OpenAI API key:
```bash
export OPENAI_API_KEY=sk-your-api-key-here
```

2. 进入示例目录:
```bash
cd cmd/examples/simple_agent
```

## 完整代码

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create OpenAI model
	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   1000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent
	ag, err := agent.New(agent.Config{
		Name:         "Math Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are a helpful math assistant. Use the calculator tools to help users with mathematical calculations.",
		MaxLoops:     10,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Run agent
	ctx := context.Background()
	output, err := ag.Run(ctx, "What is 25 multiplied by 4, then add 15?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}

	// Print result
	fmt.Println("Agent Response:")
	fmt.Println(output.Content)
	fmt.Println("\nMetadata:")
	fmt.Printf("Loops: %v\n", output.Metadata["loops"])
	fmt.Printf("Usage: %+v\n", output.Metadata["usage"])
}
```

## 代码解释

### 1. 模型配置

```go
model, err := openai.New("gpt-4o-mini", openai.Config{
	APIKey:      apiKey,
	Temperature: 0.7,
	MaxTokens:   1000,
})
```

- 创建一个使用 GPT-4o-mini 的 OpenAI 模型实例
- `Temperature: 0.7` 提供平衡的创造性和一致性
- `MaxTokens: 1000` 限制响应长度

### 2. Calculator Toolkit

```go
calc := calculator.New()
```

计算器工具集提供四个函数:
- `add` - 两个数字相加
- `subtract` - 两个数字相减
- `multiply` - 两个数字相乘
- `divide` - 两个数字相除

### 3. Agent 配置

```go
ag, err := agent.New(agent.Config{
	Name:         "Math Assistant",
	Model:        model,
	Toolkits:     []toolkit.Toolkit{calc},
	Instructions: "You are a helpful math assistant...",
	MaxLoops:     10,
})
```

- `Name` - Agent 标识符
- `Model` - 用于推理的 LLM
- `Toolkits` - Agent 可用的工具集合数组
- `Instructions` - 定义 Agent 行为的系统提示词
- `MaxLoops` - 最大工具调用迭代次数(防止无限循环)

### 4. 运行 Agent

```go
output, err := ag.Run(ctx, "What is 25 multiplied by 4, then add 15?")
```

Agent 将:
1. 分析用户查询
2. 确定需要使用计算器工具
3. 调用 `multiply(25, 4)` 得到 100
4. 调用 `add(100, 15)` 得到 115
5. 返回自然语言响应

## 运行示例

```bash
# 方式 1: 直接运行
go run main.go

# 方式 2: 构建并运行
go build -o simple_agent
./simple_agent
```

## 预期输出

```
Agent Response:
The result of 25 multiplied by 4 is 100, and when you add 15 to that, you get 115.

Metadata:
Loops: 2
Usage: map[completion_tokens:45 prompt_tokens:234 total_tokens:279]
```

## 核心概念

### 工具调用循环

`MaxLoops` 参数控制 Agent 可以调用工具的次数:

1. **循环 1**: Agent 调用 `multiply(25, 4)` → 接收结果: 100
2. **循环 2**: Agent 调用 `add(100, 15)` → 接收结果: 115
3. **最终**: Agent 生成自然语言响应

每个循环代表一轮工具调用和结果处理。

### 元数据

`output.Metadata` 包含有用的执行信息:
- `loops` - 执行的工具调用迭代次数
- `usage` - Token 消耗量(提示词、完成、总计)

## 下一步

- 探索 [Claude Agent 示例](./claude-agent.md) 了解 Anthropic 集成
- 学习 [Team 协作](./team-demo.md) 使用多个 Agent
- 尝试 [Workflow 引擎](./workflow-demo.md) 处理复杂流程
- 构建 [RAG 应用](./rag-demo.md) 实现知识检索

## 故障排除

**错误: "OPENAI_API_KEY environment variable is required"**
- 确保你已导出 API key: `export OPENAI_API_KEY=sk-...`

**错误: "model not found"**
- 检查你是否有权访问 GPT-4o-mini 模型
- 尝试使用 "gpt-3.5-turbo" 作为替代

**错误: "max loops exceeded"**
- Agent 达到了 MaxLoops 限制 (10)
- 增加 `MaxLoops` 或简化查询
