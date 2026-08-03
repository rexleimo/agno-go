# 示例 / Examples

展示 Agno-Go 所有功能的实用示例。

## 可用示例 / Available Examples

### 1. 简单 Agent / Simple Agent

带计算器工具的基础 agent。

**位置 / Location**: `cmd/examples/simple_agent/`

**功能 / Features**:
- OpenAI GPT-4o-mini 集成
- 计算器工具包
- 基本对话

**运行 / Run**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/simple_agent/main.go
```

[查看源码 / View Source](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/simple_agent)

---

### 2. Claude Agent

Anthropic Claude 集成与工具。

**位置 / Location**: `cmd/examples/claude_agent/`

**功能 / Features**:
- Anthropic Claude 3.5 Sonnet
- HTTP 和计算器工具
- 错误处理示例

**运行 / Run**:
```bash
export ANTHROPIC_API_KEY=sk-ant-your-key
go run cmd/examples/claude_agent/main.go
```

[查看源码 / View Source](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/claude_agent)

---

### 3. Ollama Agent

使用 Ollama 支持本地模型。

**位置 / Location**: `cmd/examples/ollama_agent/`

**功能 / Features**:
- 本地 Llama 3 模型
- 注重隐私(无 API 调用)
- 文件操作工具包

**设置 / Setup**:
```bash
# 安装 Ollama / Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 拉取模型 / Pull model
ollama pull llama3

# 运行示例 / Run example
go run cmd/examples/ollama_agent/main.go
```

[查看源码 / View Source](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/ollama_agent)

---

### 4. Team 演示 / Team Demo

不同协作模式的多智能体协作。

**位置 / Location**: `cmd/examples/team_demo/`

**功能 / Features**:
- 4 种协作模式(顺序、并行、领导-跟随、共识)
- 研究员 + 作家团队
- 真实工作流

**运行 / Run**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/team_demo/main.go
```

[查看源码 / View Source](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/team_demo)

---

### 5. Workflow 演示 / Workflow Demo

带控制流原语的基于步骤的编排。

**位置 / Location**: `cmd/examples/workflow_demo/`

**功能 / Features**:
- 5 种工作流原语(Step, Condition, Loop, Parallel, Router)
- 情感分析工作流
- 条件路由

**运行 / Run**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/workflow_demo/main.go
```

[查看源码 / View Source](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/workflow_demo)

---

### 6. RAG 演示 / RAG Demo

使用 ChromaDB 的检索增强生成。

**位置 / Location**: `cmd/examples/rag_demo/`

**功能 / Features**:
- ChromaDB 向量数据库
- OpenAI embeddings
- 语义搜索
- 文档问答

**设置 / Setup**:
```bash
# 启动 ChromaDB (Docker)
docker run -d -p 8000:8000 chromadb/chroma

# 设置 API 密钥 / Set API keys
export OPENAI_API_KEY=sk-your-key

# 运行示例 / Run example
go run cmd/examples/rag_demo/main.go
```

[查看源码 / View Source](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/rag_demo)

---

### 7. Logfire 可观测性 / Logfire Observability

通过 OpenTelemetry 将推理信息与 token 统计发送到 Logfire。

**位置 / Location**: `cmd/examples/logfire_observability/`

**功能 / Features**:
- 可配置的 OTLP/HTTP 导出端点（支持 EU / US）
- 将 reasoning 内容与 token 指标写入 span 事件
- 兼容所有支持 reasoning 的模型（OpenAI o 系列、Gemini 2.5、开启 thinking 的 Claude）

**运行 / Run**:
```bash
export OPENAI_API_KEY=sk-your-key
export LOGFIRE_WRITE_TOKEN=lf_your_token
go run -tags logfire cmd/examples/logfire_observability/main.go
```

[查看源码 / View Source](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/logfire_observability)

---

## 代码片段 / Code Snippets

### 带多个工具的 Agent / Agent with Multiple Tools

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/rexleimo/agno-Go/pkg/hno/agent"
    "github.com/rexleimo/agno-Go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-Go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-Go/pkg/hno/tools/http"
    "github.com/rexleimo/agno-Go/pkg/hno/tools/toolkit"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    ag, _ := agent.New(agent.Config{
        Name:  "Smart Assistant",
        Model: model,
        Toolkits: []toolkit.Toolkit{
            calculator.New(),
            http.New(),
        },
        Instructions: "You can do math and make HTTP requests",
    })

    output, _ := ag.Run(context.Background(),
        "Calculate 15 * 23 and fetch https://api.github.com")
    fmt.Println(output.Content)
}
```

### 多智能体团队 / Multi-Agent Team

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/rexleimo/agno-Go/pkg/hno/agent"
    "github.com/rexleimo/agno-Go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-Go/pkg/hno/team"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    researcher, _ := agent.New(agent.Config{
        Name:         "Researcher",
        Model:        model,
        Instructions: "Research and gather information",
    })

    writer, _ := agent.New(agent.Config{
        Name:         "Writer",
        Model:        model,
        Instructions: "Create compelling content",
    })

    tm, _ := team.New(team.Config{
        Name:   "Content Team",
        Agents: []*agent.Agent{researcher, writer},
        Mode:   team.ModeSequential,
    })

    output, _ := tm.Run(context.Background(),
        "Write a short article about Go programming")
    fmt.Println(output.Content)
}
```

### 带条件的工作流 / Workflow with Conditions

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/rexleimo/agno-Go/pkg/hno/agent"
    "github.com/rexleimo/agno-Go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-Go/pkg/hno/workflow"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    classifier, _ := agent.New(agent.Config{
        Name:         "Classifier",
        Model:        model,
        Instructions: "Classify sentiment as positive or negative",
    })

    positiveHandler, _ := agent.New(agent.Config{
        Name:         "Positive Handler",
        Model:        model,
        Instructions: "Respond enthusiastically",
    })

    negativeHandler, _ := agent.New(agent.Config{
        Name:         "Negative Handler",
        Model:        model,
        Instructions: "Respond empathetically",
    })

    wf, _ := workflow.New(workflow.Config{
        Name: "Sentiment Workflow",
        Steps: []workflow.Primitive{
            workflow.NewStep("classify", classifier),
            workflow.NewCondition("route",
                func(ctx *workflow.ExecutionContext) bool {
                    result := ctx.GetResult("classify")
                    return strings.Contains(result.Content, "positive")
                },
                workflow.NewStep("positive", positiveHandler),
                workflow.NewStep("negative", negativeHandler),
            ),
        },
    })

    output, _ := wf.Run(context.Background(), "I love this!")
    fmt.Println(output.Content)
}
```

## 了解更多 / Learn More

- [快速开始 / Quick Start](/guide/quick-start) - 5 分钟入门
- [Agent 指南 / Agent Guide](/guide/agent) - 了解 agents
- [Team 指南 / Team Guide](/guide/team) - 多智能体协作
- [Workflow 指南 / Workflow Guide](/guide/workflow) - 编排模式
- [API 参考 / API Reference](/api/) - 完整 API 文档

## 贡献示例 / Contributing Examples

有有趣的示例?为仓库做贡献:

1. Fork 仓库
2. 在 `cmd/examples/your_example/` 中创建示例
3. 添加描述和用法的 README.md
4. 提交 pull request

[贡献指南 / Contribution Guidelines](https://github.com/rexleimo/agno-Go/blob/main/CONTRIBUTING.md)
