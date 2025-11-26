
# 高级指南：多模型供应商路由（Go 优先）

本指南介绍如何在 **你自己的 Go 服务内部** 使用现有的 provider 客户端和
`internal/model.Router` 在多个模型供应商之间进行路由。示例全部基于 Go 代码，
**不依赖** 尚未完成的 HTTP 运行时。

## 1. 适用场景

典型场景包括：

- 通用对话使用一个 Provider，低延迟/低成本场景使用另一个 Provider  
- 主 Provider 故障或限流时自动切换到备用 Provider  
- 在保持应用集成面稳定的前提下，对新模型做 A/B 或灰度实验  

核心思路是：**在一个进程里组合多个 `ChatProvider` 实现**，对上层代码只暴露统一的
请求形状。

## 2. 核心构件

- `go/pkg/providers/*` – 各个模型供应商的 Go 客户端，实现
  `model.ChatProvider` / `model.EmbeddingProvider`  
- `internal/model.Router` – 负责把 `ChatRequest` / `EmbeddingRequest` 路由到
  已注册的 provider  
- `agent.ModelConfig` – 决定当前请求要走哪个 Provider / Model  

## 3. 示例：同时接入 OpenAI 与 Gemini

下面是一个只存在于 Go 进程里的路由器示例，它可以同时调用 OpenAI 和 Gemini。

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
  "github.com/rexleimo/agno-go/pkg/providers/gemini"
  "github.com/rexleimo/agno-go/pkg/providers/openai"
)

func main() {
  ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
  defer cancel()

  openaiKey := os.Getenv("OPENAI_API_KEY")
  geminiKey := os.Getenv("GEMINI_API_KEY")
  if openaiKey == "" && geminiKey == "" {
    log.Fatal("at least one of OPENAI_API_KEY or GEMINI_API_KEY must be set")
  }

  router := model.NewRouter(
    model.WithMaxConcurrency(16),
    model.WithTimeout(30*time.Second),
  )

  if openaiKey != "" {
    router.RegisterChatProvider(openai.New("", openaiKey, nil))
  }
  if geminiKey != "" {
    router.RegisterChatProvider(gemini.New("", geminiKey, nil))
  }

  // 优先尝试 OpenAI，不可用时回退到 Gemini。
  providers := []agent.Provider{agent.ProviderOpenAI, agent.ProviderGemini}

  var lastErr error
  for _, prov := range providers {
    req := model.ChatRequest{
      Model: agent.ModelConfig{
        Provider: prov,
        ModelID:  "gpt-4o-mini", // 当 prov 为 Gemini 时换成合适的 Gemini 模型 ID
        Stream:   false,
      },
      Messages: []agent.Message{
        {Role: agent.RoleUser, Content: "针对一个内部小工具，推荐性价比高的模型并说明理由。"},
      },
    }

    resp, err := router.Chat(ctx, req)
    if err != nil {
      lastErr = err
      log.Printf("provider %s failed: %v", prov, err)
      continue
    }

    fmt.Printf("provider=%s reply=%s\n", prov, resp.Message.Content)
    return
  }

  log.Fatalf("all providers failed, last error: %v", lastErr)
}
```

这样，provider 具体的模型 ID、Key、端点都留在配置和环境变量里，上层业务只需要关心
“先试哪个 Provider、再试哪个 Provider”。

## 4. 流式变体

同一个 Router 也可以用于流式输出：

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "请以流式方式输出一句简短问候。"},
  },
}

err := router.Stream(ctx, req, func(ev model.ChatStreamEvent) error {
  if ev.Type == "token" {
    fmt.Print(ev.Delta)
  }
  if ev.Done {
    fmt.Println()
  }
  return nil
})
if err != nil {
  log.Fatalf("stream error: %v", err)
}
```

需要回退时，可以像非流式示例那样，对不同 Provider 构造不同的 `ChatRequest` 并逐个尝试。

## 5. 配置建议

在 Go 侧做多 Provider 路由时建议：

- 把 Key 和端点放在 `.env` / `config/default.yaml` 中，而不是硬编码在应用代码里  
- 使用 [模型供应商矩阵](../providers/matrix) 选择要启用的 Provider 及模型  
- 让 Router 成为唯一知道具体 Provider 细节的地方，其它代码只依赖
  `agent.ModelConfig` 和 `model.ChatRequest`  

## 6. 与 HTTP 运行时的关系

规范中的 HTTP 运行时设计（agents / sessions / messages）沿用了相同的概念
（模型配置、多 Provider 路由），但实现尚未稳定。

在此之前，你可以：

- 按本文所示，直接在 Go 服务中使用 `go/pkg/providers/*` 与
  `internal/model.Router` 组合出自己的路由逻辑  
- 把 `specs/001-go-agno-rewrite/contracts` 中的 HTTP 契约当作数据形状参考，
  而不是“已经上线的外部 API”  

当 HTTP 层稳定之后，这些路由模式会自然映射到运行时公开的 `agents/sessions/messages`
接口上。

