# 快速开始：在 Go 中调用 Agno-Go Provider

本指南演示如何在 **Go 代码** 中直接调用本仓库内置的模型供应商客户端（`go/pkg/providers/*`）。当前的第一个版本只关注：

1. 配置一个模型供应商（例如 OpenAI）  
2. 在 Go 中通过 `go/pkg/providers/openai` 发起一次对话请求  
3. 打印并检查返回的回答  

> 说明：目前 AgentOS HTTP 运行时（`/agents`、`/sessions`、`/messages` 等端点）仍在收敛中，本快速开始 **不依赖** 这些尚未稳定的接口，只使用已经在测试中验证过的 Go provider 客户端。

## 环境准备

1. 安装 Go 1.25.1
2. 在项目根目录复制环境变量模板并填入 OpenAI Key：

```bash
cd <your-project-root>
cp .env.example .env
```

在 `.env` 中设置：

```bash
OPENAI_API_KEY=你的-openai-key
```

## 最小示例：使用 OpenAI Chat

下面的代码片段与仓库中的测试模式保持一致，直接复用：

- `github.com/rexleimo/agno-go/internal/agent`  
- `github.com/rexleimo/agno-go/internal/model`  
- `github.com/rexleimo/agno-go/pkg/providers/openai`  

```go
package main

import (
  "context"
  "fmt"
  "log"
  "os"
  "time"

  "github.com/rexleimo/agno-go/internal/agent"
  "github.com/rexleimo/agno-go/internal/model"
  "github.com/rexleimo/agno-go/pkg/providers/openai"
)

func main() {
  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()

  apiKey := os.Getenv("OPENAI_API_KEY")
  if apiKey == "" {
    log.Fatal("OPENAI_API_KEY not set")
  }

  // 使用默认 OpenAI 公共端点；如需代理可在 .env 中配置 OPENAI_ENDPOINT
  client := openai.New("", apiKey, nil)

  resp, err := client.Chat(ctx, model.ChatRequest{
    Model: agent.ModelConfig{
      Provider: agent.ProviderOpenAI,
      ModelID:  "gpt-4o-mini",
      Stream:   false,
    },
    Messages: []agent.Message{
      {Role: agent.RoleUser, Content: "请用一两句话介绍一下 Agno-Go。"},
    },
  })
  if err != nil {
    log.Fatalf("chat error: %v", err)
  }

  fmt.Println("assistant:", resp.Message.Content)
}
```

把上面的代码保存为：

```bash
<your-project-root>/examples/openai_quickstart/main.go
```

然后在项目根目录运行：

```bash
cd <your-project-root>
go run ./examples/openai_quickstart
```

你应该能看到类似如下输出（具体内容视模型而定）：

```text
assistant: Agno-Go 是一个用 Go 实现的 AgentOS...
```

## 下一步

- 查看 [配置与安全实践](./config-and-security)，了解如何安全地配置各模型供应商的 Key、端点与运行参数  
- 查看 [模型供应商矩阵](./providers/matrix)，了解不同 Provider 支持的 Chat / Embedding / Streaming 能力以及需要的环境变量  
- HTTP 形式的 AgentOS 运行时（agents / sessions / messages）目前仍在打磨中，等运行面稳定后才会在文档中公开示例；在此之前，请把 `go/pkg/providers/*` 视为主要的公共入口  

