# 快速开始

在不到 5 分钟内开始使用 HNO!

## 前置要求

- Go 1.21 或更高版本
- OpenAI API 密钥 (或 Anthropic/Ollama)
- 对 AI Agent 的基本了解

## 安装

### 方式 1: 使用 Go Get

```bash
go get github.com/rexleimo/HNO
```

### 方式 2: 克隆仓库

```bash
git clone https://github.com/rexleimo/HNO.git
cd HNO
go mod download
```

## 您的第一个 Agent

### 1. 简单 Agent (无工具)

创建文件 `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)

func main() {
    // Get API key from environment
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY environment variable is required")
    }

    // Create OpenAI model
    model, err := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })
    if err != nil {
        log.Fatalf("Failed to create model: %v", err)
    }

    // Create agent
    ag, err := agent.New(agent.Config{
        Name:         "Assistant",
        Model:        model,
        Instructions: "You are a helpful assistant.",
    })
    if err != nil {
        log.Fatalf("Failed to create agent: %v", err)
    }

    // Run agent
    output, err := ag.Run(context.Background(), "What is the capital of France?")
    if err != nil {
        log.Fatalf("Agent run failed: %v", err)
    }

    fmt.Println("Agent:", output.Content)
}
```

**运行:**

```bash
export OPENAI_API_KEY=sk-your-key-here
go run main.go
```

**预期输出:**

```
Agent: The capital of France is Paris.
```

### 2. 带工具的 Agent

添加计算器工具:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
    "github.com/rexleimo/HNO/pkg/hno/tools/calculator"
    "github.com/rexleimo/HNO/pkg/hno/tools/toolkit"
)

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY required")
    }

    // Create model
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })

    // Create agent WITH tools
    ag, _ := agent.New(agent.Config{
        Name:  "Calculator Agent",
        Model: model,
        Toolkits: []toolkit.Toolkit{
            calculator.New(),
        },
        Instructions: "You are a math assistant. Use the calculator tools for calculations.",
    })

    // Ask a math question
    output, _ := ag.Run(context.Background(), "What is 123 * 456 + 789?")

    fmt.Println("Question: What is 123 * 456 + 789?")
    fmt.Println("Agent:", output.Content)
}
```

**运行:**

```bash
go run main.go
```

**预期输出:**

```
Question: What is 123 * 456 + 789?
Agent: The result is 56,877
```

### 3. 多轮对话

添加记忆功能进行对话:

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY required")
    }

    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })

    ag, _ := agent.New(agent.Config{
        Name:         "Chat Assistant",
        Model:        model,
        Instructions: "You are a friendly chatbot. Remember context from previous messages.",
    })

    fmt.Println("Chat Assistant (type 'quit' to exit)")
    fmt.Println("=====================================")

    scanner := bufio.NewScanner(os.Stdin)
    for {
        fmt.Print("\nYou: ")
        if !scanner.Scan() {
            break
        }

        input := strings.TrimSpace(scanner.Text())
        if input == "quit" || input == "exit" {
            fmt.Println("Goodbye!")
            break
        }

        if input == "" {
            continue
        }

        // Run agent (memory is automatically maintained)
        output, err := ag.Run(context.Background(), input)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }

        fmt.Printf("Agent: %s\n", output.Content)
    }
}
```

**示例对话:**

```
You: My name is Alice
Agent: Nice to meet you, Alice! How can I help you today?

You: What's my name?
Agent: Your name is Alice!

You: quit
Goodbye!
```

## 使用 AgentOS (HTTP 服务器)

### 1. 启动服务器

#### 使用 Docker Compose (推荐)

```bash
# Copy environment template
cp .env.example .env

# Edit .env and add your API key
nano .env  # Add: OPENAI_API_KEY=sk-your-key

# Start server
docker-compose up -d

# Check health
curl http://localhost:8080/health
```

#### 使用 Go (原生)

```bash
# Build server
go build -o agentos cmd/server/main.go

# Run server
export OPENAI_API_KEY=sk-your-key
./agentos
```

### 2. 使用 API

#### 健康检查

```bash
curl http://localhost:8080/health
```

**响应:**
```json
{
  "status": "healthy",
  "service": "agentos",
  "time": 1704067200
}
```

#### 运行 Agent

```bash
curl -X POST http://localhost:8080/api/v1/agents/assistant/run \
  -H "Content-Type: application/json" \
  -d '{
    "input": "What is 2+2?"
  }'
```

**响应:**
```json
{
  "content": "2 + 2 equals 4.",
  "metadata": {
    "agent_id": "assistant"
  }
}
```

查看 [AgentOS API Reference](/api/agentos) 获取完整的 API 文档。

## 下一步

### 了解更多

- [Core Concepts](/guide/agent) - 理解 Agent、Team、Workflow
- [Tools Guide](/guide/tools) - 了解内置和自定义工具
- [Models Guide](/guide/models) - 多模型支持
- [Advanced Topics](/advanced/) - 架构、性能、部署

### 尝试示例

所有示例都在 `cmd/examples/` 目录中:

```bash
# Simple agent with calculator
go run cmd/examples/simple_agent/main.go

# Anthropic Claude
go run cmd/examples/claude_agent/main.go

# Local models with Ollama
go run cmd/examples/ollama_agent/main.go

# Multi-agent team
go run cmd/examples/team_demo/main.go

# Workflow engine
go run cmd/examples/workflow_demo/main.go

# RAG with ChromaDB
go run cmd/examples/rag_demo/main.go
```

查看 [Examples](/examples/) 获取每个示例的详细文档。

## 故障排除

### 常见问题

**1. "OPENAI_API_KEY not set"**

```bash
export OPENAI_API_KEY=sk-your-key-here
```

**2. "Module not found"**

```bash
go mod download
go mod tidy
```

**3. "Port 8080 already in use"**

在 `.env` 或配置中更改端口:
```bash
AGENTOS_ADDRESS=:9090
```

**4. "Context deadline exceeded"**

增加超时时间:
```bash
export REQUEST_TIMEOUT=60
```

### 获取调试日志

```bash
export AGENTOS_DEBUG=true
export LOG_LEVEL=debug
```

## 快速参考

### 常用导入

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
    "github.com/rexleimo/HNO/pkg/hno/tools/calculator"
    "github.com/rexleimo/HNO/pkg/hno/team"
    "github.com/rexleimo/HNO/pkg/hno/workflow"
    "github.com/rexleimo/HNO/pkg/agentos"
)
```

### Agent 创建模板

```go
model, err := openai.New("gpt-4o-mini", openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

ag, err := agent.New(agent.Config{
    Name:         "Agent Name",
    Model:        model,
    Toolkits:     []toolkit.Toolkit{/* tools */},
    Instructions: "System instructions",
    MaxLoops:     10,
})

output, err := ag.Run(context.Background(), "input")
```

## 下一步: 核心概念

了解三个核心抽象:

- [Agent](/guide/agent) - 自主 AI Agent
- [Team](/guide/team) - 多 Agent 协作
- [Workflow](/guide/workflow) - 基于步骤的编排
