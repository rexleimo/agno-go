# Ollama Agent 示例

## 概述

本示例演示如何通过 Ollama 在 HNO 中使用本地 LLM。Ollama 允许你在本地机器上运行强大的语言模型,提供隐私保护、成本节约和离线能力。这非常适合开发、测试和对隐私敏感的应用。

## 你将学到

- 如何将 Ollama 与 HNO 集成
- 如何使用本地 LLM 运行 Agent
- 如何在本地模型中使用工具调用
- 本地模型的优势和限制

## 前置要求

- Go 1.21 或更高版本
- 已安装 Ollama ([ollama.ai](https://ollama.ai))
- 已拉取本地模型 (例如 llama2, mistral, codellama)

## Ollama 设置

### 1. 安装 Ollama

**macOS/Linux:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

**Windows:**
从 [ollama.ai/download](https://ollama.ai/download) 下载

### 2. 拉取模型

```bash
# 拉取 Llama 2 (7B 参数, ~4GB)
ollama pull llama2

# 或尝试其他模型:
ollama pull mistral      # Mistral 7B
ollama pull codellama    # 代码专用
ollama pull llama2:13b   # 更大、更强大
```

### 3. 启动 Ollama 服务器

```bash
ollama serve
```

服务器默认运行在 `http://localhost:11434`。

### 4. 验证安装

```bash
# 测试模型
ollama run llama2 "Hello, how are you?"
```

## 完整代码

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/ollama"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Create Ollama model (uses local Ollama instance)
	// Make sure Ollama is running: ollama serve
	model, err := ollama.New("llama2", ollama.Config{
		BaseURL:     "http://localhost:11434",
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent with Ollama
	ag, err := agent.New(agent.Config{
		Name:         "Ollama Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are a helpful AI assistant running on Ollama. You can use calculator tools to help with math. Be concise and friendly.",
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
	output, err = ag.Run(ctx, "What is 456 multiplied by 789?")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 3: Complex calculation
	fmt.Println("=== Example 3: Complex Calculation ===")
	output, err = ag.Run(ctx, "Calculate: (100 + 50) * 2 - 75")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	fmt.Println("✅ All examples completed successfully!")
}
```

## 代码解释

### 1. Ollama 模型配置

```go
model, err := ollama.New("llama2", ollama.Config{
	BaseURL:     "http://localhost:11434",
	Temperature: 0.7,
	MaxTokens:   2000,
})
```

**配置选项:**
- **模型名称**: 必须匹配已拉取的模型 (例如 "llama2", "mistral")
- **BaseURL**: Ollama 服务器地址 (默认: `http://localhost:11434`)
- **Temperature**: 0.0 (确定性) 到 2.0 (非常创造性)
- **MaxTokens**: 最大响应长度

### 2. 无需 API Key

与 OpenAI 或 Anthropic 不同,Ollama 在本地运行:
- ✅ 无需 API key
- ✅ 无使用成本
- ✅ 完全隐私
- ✅ 离线工作

### 3. 工具支持

本地模型可以像云模型一样使用工具:
```go
Toolkits: []toolkit.Toolkit{calc}
```

Agent 将在需要时调用计算器函数。

## 运行示例

### 步骤 1: 启动 Ollama
```bash
# 终端 1
ollama serve
```

### 步骤 2: 运行示例
```bash
# 终端 2
cd cmd/examples/ollama_agent
go run main.go
```

## 预期输出

```
=== Example 1: Simple Conversation ===
Agent: I'm a helpful AI assistant running on Ollama, here to assist you with questions and tasks.

=== Example 2: Calculator Tool Usage ===
Agent: Let me calculate that for you. 456 multiplied by 789 equals 359,784.

=== Example 3: Complex Calculation ===
Agent: Let me solve this step by step:
- First: 100 + 50 = 150
- Then: 150 * 2 = 300
- Finally: 300 - 75 = 225

The answer is 225.

✅ All examples completed successfully!
```

## 可用模型

### 通用模型

| 模型 | 大小 | 内存 | 描述 |
|-------|------|-----|-------------|
| llama2 | 7B | 8GB | Meta 的 Llama 2, 通用 |
| llama2:13b | 13B | 16GB | 更大、更强大的版本 |
| mistral | 7B | 8GB | Mistral AI, 优秀质量 |
| mixtral | 47B | 32GB | 混合专家, 非常强大 |

### 专用模型

| 模型 | 用例 |
|-------|----------|
| codellama | 代码生成和分析 |
| llama2-uncensored | 更少的内容限制 |
| orca-mini | 更小、更快 (3B) |
| vicuna | 对话和聊天 |

### 列出可用模型
```bash
ollama list
```

### 拉取特定模型
```bash
ollama pull mistral
ollama pull codellama:13b
```

## 配置示例

### 追求速度(小模型)
```go
ollama.Config{
	Model:       "orca-mini",
	Temperature: 0.5,
	MaxTokens:   500,
}
```

### 追求质量(大模型)
```go
ollama.Config{
	Model:       "mixtral",
	Temperature: 0.7,
	MaxTokens:   3000,
}
```

### 代码任务
```go
ollama.Config{
	Model:       "codellama",
	Temperature: 0.3,  // 代码更确定性
	MaxTokens:   2000,
}
```

### 自定义 Ollama 服务器
```go
ollama.Config{
	BaseURL:     "http://192.168.1.100:11434",  // 远程 Ollama
	Model:       "llama2",
	Temperature: 0.7,
}
```

## 性能考虑

### 速度因素

1. **模型大小**: 小模型 (7B) 比大模型 (70B) 更快
2. **硬件**: GPU 大大加速推理
3. **上下文长度**: 更长的对话会减慢响应速度

### 典型响应时间

| 模型 | 硬件 | 速度 |
|-------|----------|-------|
| llama2 (7B) | Mac M1 | ~1-2 秒 |
| mistral (7B) | Mac M1 | ~1-2 秒 |
| mixtral (47B) | Mac M1 | ~5-10 秒 |
| llama2 (13B) | NVIDIA 3090 | ~0.5-1 秒 |

## 本地模型的优势

### ✅ 优点

1. **隐私**: 数据永不离开你的机器
2. **成本**: 无 API 费用,无限使用
3. **离线**: 无需互联网即可工作
4. **控制**: 完全控制模型和数据
5. **定制**: 为特定任务微调模型

### ⚠️ 限制

1. **质量**: 通常低于 GPT-4 或 Claude Opus
2. **速度**: 比云 API 慢 (除非有高端 GPU)
3. **资源**: 需要 RAM/VRAM (4-16GB+)
4. **维护**: 需要管理模型和更新

## 最佳实践

### 1. 选择合适的模型

```bash
# 用于开发/测试
ollama pull orca-mini  # 快速, 3B 参数

# 用于生产
ollama pull mistral    # 速度/质量的良好平衡

# 用于复杂任务
ollama pull mixtral    # 高质量, 需要更多资源
```

### 2. 优化指令

本地模型受益于简洁、清晰的指令:

```go
// ✅ 好
Instructions: "You are a math assistant. Use calculator tools for calculations. Be concise."

// ❌ 太冗长
Instructions: "You are an extremely sophisticated mathematical assistant with deep knowledge..."
```

### 3. 监控资源使用

```bash
# 检查 Ollama 状态
ollama ps

# 查看模型信息
ollama show llama2
```

### 4. 优雅地处理错误

```go
output, err := ag.Run(ctx, userQuery)
if err != nil {
	// Ollama 可能宕机了
	log.Printf("Ollama error: %v. Is the server running?", err)
	// 回退到云模型或返回错误
}
```

## 集成模式

### 混合方法

开发使用 Ollama,生产使用云:

```go
var model models.Model

if os.Getenv("ENV") == "production" {
	model, _ = openai.New("gpt-4o-mini", openai.Config{...})
} else {
	model, _ = ollama.New("llama2", ollama.Config{...})
}
```

### 隐私优先应用

```go
// 使用 Ollama 处理敏感数据
sensitiveAgent, _ := agent.New(agent.Config{
	Model: ollamaModel,
	Instructions: "Handle user PII securely...",
})
```

## 故障排除

### 错误: "connection refused"
```bash
# 检查 Ollama 是否运行
ollama serve

# 或检查进程
ps aux | grep ollama
```

### 错误: "model not found"
```bash
# 先拉取模型
ollama pull llama2

# 验证可用性
ollama list
```

### 响应慢
```bash
# 尝试更小的模型
ollama pull orca-mini

# 或检查硬件加速
ollama show llama2 | grep -i gpu
```

### 内存不足
```bash
# 使用更小的模型
ollama pull orca-mini  # 3B 而非 7B

# 或增加交换空间 (Linux)
# 或关闭其他应用程序
```

## 下一步

- 与 [OpenAI Agent](./simple-agent.md) 和 [Claude Agent](./claude-agent.md) 比较
- 在 [多 Agent Team](./team-demo.md) 中使用本地模型
- 使用本地嵌入构建 [隐私保护 RAG](./rag-demo.md)
- 使用本地模型探索 [Workflow](./workflow-demo.md)

## 其他资源

- [Ollama 文档](https://github.com/ollama/ollama/blob/main/README.md)
- [Ollama 模型库](https://ollama.ai/library)
- [硬件要求](https://github.com/ollama/ollama/blob/main/docs/gpu.md)
- [模型比较](https://ollama.ai/blog/model-comparison)
