# 核心能力与 Go API 概览

本页介绍目前 **已经存在且可用的 Go 级 API 面**。在当前阶段，对外稳定的入口主要是：

- `go/pkg/providers/*` 下的各个模型供应商客户端  
- `internal/agent` 和 `internal/model` 中共享的数据模型  

规范里提到的 HTTP 运行时（`/agents`、`/sessions`、`/messages` 等）仍在演进中，暂时
**不建议** 作为对外稳定 API 使用，本页只做设计级别的说明。

## 1. 核心数据类型

调用模型时，你会频繁接触到以下类型：

- `agent.ModelConfig`：描述要使用的 Provider / Model 以及一些基础选项  
  （`Provider` 枚举、`ModelID`、`Stream`、`MaxTokens`、`Temperature` 等）。  
- `agent.Message`：一条消息，包含 `Role`（`user` / `assistant` / `system`）和
  `Content`（当前实现为纯文本）。  
- `model.ChatRequest`：一次对话请求：

  ```go
  type ChatRequest struct {
    Model    agent.ModelConfig `json:"model"`
    Messages []agent.Message   `json:"messages"`
    Tools    []agent.ToolCall  `json:"tools,omitempty"`
    Metadata map[string]any    `json:"metadata,omitempty"`
    Stream   bool              `json:"stream,omitempty"`
  }
  ```

- `model.ChatResponse`：一次助手回复以及用量信息：

  ```go
  type ChatResponse struct {
    Message      agent.Message `json:"message"`
    Usage        agent.Usage   `json:"usage,omitempty"`
    FinishReason string        `json:"finishReason,omitempty"`
  }
  ```

- `model.ChatStreamEvent` / `model.StreamHandler`：用于流式输出（`token` /
  `tool_call` / `end` 事件）。  
- `model.EmbeddingRequest` / `model.EmbeddingResponse`：用于 embedding 调用。  
- `model.ChatProvider` / `model.EmbeddingProvider`：各个 provider 客户端实现的接口。  

## 2. Provider 客户端（`go/pkg/providers/*`）

每个 provider 包（OpenAI、Gemini、Groq 等）都会实现 `internal/model` 中的接口。
例如 OpenAI 客户端：

- 位于 `go/pkg/providers/openai`  
- 暴露 `New(endpoint, apiKey string, missingEnv []string) *Client`  
- 实现 `model.ChatProvider` 和 `model.EmbeddingProvider`  

一个最小的非流式对话调用如下：

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

client := openai.New("", os.Getenv("OPENAI_API_KEY"), nil)

resp, err := client.Chat(ctx, model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   false,
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "请简单介绍一下 Agno-Go。"},
  },
})
if err != nil {
  log.Fatalf("chat error: %v", err)
}

fmt.Println("assistant:", resp.Message.Content)
```

如果需要流式输出，可以使用同一个客户端的 `Stream` 方法和
`model.ChatStreamEvent`：

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "请分几次输出一句问候。"},
  },
}

err := client.Stream(ctx, req, func(ev model.ChatStreamEvent) error {
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

Embedding 的调用方式与此类似，使用 `EmbeddingRequest` / `EmbeddingResponse`。
更完整的示例可以参考 `go/tests/contract` 与 `go/tests/providers`。

## 3. Router：在 Go 内部组合多个 Provider

`internal/model.Router` 提供了一个简单的路由器，可以在同一个进程内注册多个
provider 客户端：

```go
router := model.NewRouter(
  model.WithMaxConcurrency(16),
  model.WithTimeout(30*time.Second),
)

openAI := openai.New("", os.Getenv("OPENAI_API_KEY"), nil)
router.RegisterChatProvider(openAI)

// 其它 provider（Gemini、Groq 等）可以以同样方式注册

req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "Hello from router."},
  },
}

resp, err := router.Chat(ctx, req)
if err != nil {
  log.Fatalf("router chat error: %v", err)
}

fmt.Println("assistant:", resp.Message.Content)
```

Router 还提供：

- `router.Stream(ctx, req, handler)`：流式对话  
- `router.Embed(ctx, embeddingReq)`：embedding 调用  
- `router.Statuses()`：当前各 provider 的状态列表（可用于健康检查）  

这也是内部实现多 provider 组合的主要原语，你也可以在自己的服务里直接复用。

## 4. HTTP 运行时（设计说明，暂不稳定）

`specs/001-vitepress-docs/contracts/docs-site-openapi.yaml` 中描述了一个 HTTP 运行时，
包含如下端点：

- `GET /health` – 健康检查与 provider 状态  
- `POST /agents` – 创建 Agent 定义  
- `POST /agents/{agentId}/sessions` – 创建会话  
- `POST /agents/{agentId}/sessions/{sessionId}/messages` – 发送消息  

目前这套 HTTP 接口仍处于设计/实现迭代阶段：

- Go 运行时实现尚未完全稳定  
- 文档中的部分流程依赖 `go/cmd/agno` 尚未实现/对外公开的行为  

在当前阶段，用于生产代码时 **优先选择**：

- 直接通过 `go/pkg/providers/*` 调用各个模型供应商  
- 如需多 provider 组合，在你自己的服务中使用 `internal/model.Router`  

等 HTTP 运行面和契约完全稳定之后，文档会再单独给出完整的端到端示例。

## 5. 代码参考位置

- `go/pkg/providers/*` – 各供应商客户端（OpenAI、Gemini、Groq 等）  
- `go/internal/agent` – Agent / 模型配置类型、用量统计等  
- `go/internal/model` – 请求/响应类型、Router、Provider 接口  
- `go/tests/providers` – Provider 客户端的实际使用示例  
- `go/tests/contract` – 覆盖 HTTP 形状的数据模型的契约测试  

在调整自己的 Go 代码和示例时，请以以上文件为最终事实来源。

