# Claude Agent 示例

## 概述

本示例演示如何在 HNO 中使用 Anthropic 的 Claude 模型。Claude 以其深思熟虑、详细的响应和强大的推理能力而闻名。本示例展示了多个用例,包括简单对话、计算器工具使用、复杂计算和数学推理。

## 你将学到

- 如何将 Anthropic Claude 与 HNO 集成
- 如何配置 Claude 模型 (Opus, Sonnet, Haiku)
- 如何使用 Claude 的工具调用能力
- Claude 指令的最佳实践

## 前置要求

- Go 1.21 或更高版本
- Anthropic API key (在 [console.anthropic.com](https://console.anthropic.com) 获取)

## 设置

1. 设置你的 Anthropic API key:
```bash
export ANTHROPIC_API_KEY=sk-ant-your-api-key-here
```

2. 进入示例目录:
```bash
cd cmd/examples/claude_agent
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
	"github.com/rexleimo/agno-go/pkg/hno/models/anthropic"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Create Anthropic Claude model
	model, err := anthropic.New("claude-3-opus-20240229", anthropic.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent with Claude
	ag, err := agent.New(agent.Config{
		Name:         "Claude Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are Claude, a helpful AI assistant created by Anthropic. Use the calculator tools to help users with mathematical calculations. Be precise and explain your reasoning.",
		MaxLoops:     10,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Example 1: Simple conversation
	fmt.Println("=== Example 1: Simple Conversation ===")
	ctx := context.Background()
	output, err := ag.Run(ctx, "Introduce yourself in one sentence.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 2: Using calculator tools
	fmt.Println("=== Example 2: Calculator Tool Usage ===")
	output, err = ag.Run(ctx, "What is 156 multiplied by 23, then subtract 100?")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 3: Complex calculation
	fmt.Println("=== Example 3: Complex Calculation ===")
	output, err = ag.Run(ctx, "Calculate the following: (45 + 67) * 3 - 89. Show your work step by step.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 4: Mathematical reasoning
	fmt.Println("=== Example 4: Mathematical Reasoning ===")
	output, err = ag.Run(ctx, "If I have $500 and spend $123, then earn $250, how much money do I have? Use the calculator to verify.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	fmt.Println("✅ All examples completed successfully!")
}
```

## 代码解释

### 1. Claude 模型配置

```go
model, err := anthropic.New("claude-3-opus-20240229", anthropic.Config{
	APIKey:      apiKey,
	Temperature: 0.7,
	MaxTokens:   2000,
})
```

**可用的 Claude 模型:**
- `claude-3-opus-20240229` - 最强大,适合复杂任务
- `claude-3-sonnet-20240229` - 性能和速度平衡
- `claude-3-haiku-20240307` - 最快,适合简单任务

**配置选项:**
- `Temperature: 0.7` - 平衡的创造性 (0.0 = 确定性, 1.0 = 创造性)
- `MaxTokens: 2000` - 最大响应长度

### 2. Claude 专属指令

```go
Instructions: "You are Claude, a helpful AI assistant created by Anthropic.
Use the calculator tools to help users with mathematical calculations.
Be precise and explain your reasoning."
```

Claude 对以下内容响应良好:
- 清晰的身份和目的
- 关于工具使用的明确指令
- 强调推理和解释

### 3. 示例场景

#### 示例 1: 简单对话
测试基本对话能力,不使用工具。

#### 示例 2: Calculator 工具使用
```
查询: "What is 156 multiplied by 23, then subtract 100?"
预期流程:
1. multiply(156, 23) → 3588
2. subtract(3588, 100) → 3488
```

#### 示例 3: 复杂计算
```
查询: "Calculate: (45 + 67) * 3 - 89. Show your work step by step."
预期流程:
1. add(45, 67) → 112
2. multiply(112, 3) → 336
3. subtract(336, 89) → 247
Claude 还会解释每一步
```

#### 示例 4: 数学推理
测试 Claude 的能力:
- 分解文字问题
- 选择合适的工具
- 提供清晰的解释

## 运行示例

```bash
# 方式 1: 直接运行
go run main.go

# 方式 2: 构建并运行
go build -o claude_agent
./claude_agent
```

## 预期输出

```
=== Example 1: Simple Conversation ===
Agent: I'm Claude, an AI assistant created by Anthropic to be helpful, harmless, and honest.

=== Example 2: Calculator Tool Usage ===
Agent: Let me calculate that for you. First, 156 multiplied by 23 equals 3,588. Then, subtracting 100 from 3,588 gives us 3,488.

=== Example 3: Complex Calculation ===
Agent: I'll solve this step by step:
1. First, calculate the parentheses: 45 + 67 = 112
2. Then multiply: 112 * 3 = 336
3. Finally subtract: 336 - 89 = 247

The final answer is 247.

=== Example 4: Mathematical Reasoning ===
Agent: Let me help you track your money:
- Starting amount: $500
- After spending $123: $500 - $123 = $377
- After earning $250: $377 + $250 = $627

You have $627 in total.

✅ All examples completed successfully!
```

## Claude vs OpenAI

### 何时使用 Claude

**最适合:**
- 复杂推理任务
- 详细解释
- 安全关键应用
- 深思熟虑、细致入微的响应

**特点:**
- 更冗长和解释性
- 强大的伦理推理
- 擅长遵循复杂指令
- 更善于承认不确定性

### 何时使用 OpenAI

**最适合:**
- 快速响应
- 代码生成
- 创意写作
- 大规模函数调用

## 模型选择指南

| 模型 | 速度 | 能力 | 成本 | 用例 |
|-------|-------|------------|------|----------|
| Claude 3 Opus | 慢 | 最高 | 高 | 复杂分析、研究 |
| Claude 3 Sonnet | 中等 | 高 | 中等 | 通用目的、平衡 |
| Claude 3 Haiku | 快 | 良好 | 低 | 简单任务、高并发 |

## 配置技巧

### 确定性输出
```go
anthropic.Config{
	Temperature: 0.0,
	MaxTokens:   1000,
}
```

### 创造性任务
```go
anthropic.Config{
	Temperature: 1.0,
	MaxTokens:   3000,
}
```

### 生产环境(平衡)
```go
anthropic.Config{
	Temperature: 0.7,
	MaxTokens:   2000,
}
```

## 最佳实践

1. **清晰的指令**: Claude 对详细、结构化的提示词响应良好
2. **推理请求**: 要求 Claude "解释" 或 "展示工作过程" 以获得更好的结果
3. **安全性**: Claude 更加谨慎 - 适当地框架敏感查询
4. **上下文**: Claude 有 200K token 的上下文窗口 - 可用于长文档

## 下一步

- 与 [OpenAI Simple Agent](./simple-agent.md) 比较
- 尝试 [Ollama 本地模型](./ollama-agent.md)
- 构建 [多 Agent Team](./team-demo.md)
- 探索 [Claude RAG](./rag-demo.md)

## 故障排除

**错误: "ANTHROPIC_API_KEY environment variable is required"**
- 设置你的 API key: `export ANTHROPIC_API_KEY=sk-ant-...`

**错误: "model not found"**
- 验证模型名称完全匹配: `claude-3-opus-20240229`
- 检查你的 API 层级是否有权访问该模型

**Opus 响应慢**
- 考虑使用 Sonnet 以获得更快响应
- 如果不需要长输出,减少 MaxTokens

**速率限制错误**
- Anthropic 的不同层级有不同的速率限制
- 实现带指数退避的重试逻辑
- 考虑为高并发任务使用 Haiku
