
# Core Features & API Overview

This page gives a high-level overview of the **Go-level API surface that exists
today** in Agno-Go. For this first iteration, the stable public entry point is:

- The provider clients under `go/pkg/providers/*`  
- The shared types in `internal/agent` and `internal/model`  

The HTTP runtime (agents / sessions / messages) described in the specs is still
under active development and shouldn’t be treated as a stable public API yet.

## 1. Shared Go types

The main types you will interact with when calling models from Go are:

- `agent.ModelConfig` – describes which provider/model to use and basic options
  (provider enum, `ModelID`, `Stream`, `MaxTokens`, `Temperature`, …).  
- `agent.Message` – a single message with `Role` (`user`, `assistant`, `system`)
  and `Content` (plain text in the current implementation).  
- `model.ChatRequest` – a chat completion request:

  ```go
  type ChatRequest struct {
    Model    agent.ModelConfig `json:"model"`
    Messages []agent.Message   `json:"messages"`
    Tools    []agent.ToolCall  `json:"tools,omitempty"`
    Metadata map[string]any    `json:"metadata,omitempty"`
    Stream   bool              `json:"stream,omitempty"`
  }
  ```

- `model.ChatResponse` – a single assistant turn and usage data:

  ```go
  type ChatResponse struct {
    Message      agent.Message `json:"message"`
    Usage        agent.Usage   `json:"usage,omitempty"`
    FinishReason string        `json:"finishReason,omitempty"`
  }
  ```

- `model.ChatStreamEvent` / `model.StreamHandler` – used when you want token-level
  streaming (`token` / `tool_call` / `end` events).  
- `model.EmbeddingRequest` / `model.EmbeddingResponse` – used for embeddings.  
- `model.ChatProvider` / `model.EmbeddingProvider` – interfaces implemented by
  the provider clients in `go/pkg/providers/*`.  

## 2. Provider clients (`go/pkg/providers/*`)

Each provider package (OpenAI, Gemini, Groq, etc.) implements the shared
interfaces from `internal/model`. For example, the OpenAI client:

- Lives in `go/pkg/providers/openai`  
- Exposes `New(endpoint, apiKey string, missingEnv []string) *Client`  
- Implements `model.ChatProvider` and `model.EmbeddingProvider`  

A minimal non-streaming chat call looks like this:

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
    {Role: agent.RoleUser, Content: "Introduce Agno-Go briefly."},
  },
})
if err != nil {
  log.Fatalf("chat error: %v", err)
}

fmt.Println("assistant:", resp.Message.Content)
```

For streaming output, the same client exposes `Stream` and uses
`model.ChatStreamEvent`:

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "Say hello in a few short tokens."},
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

Embeddings follow the same pattern via `EmbeddingRequest`/`EmbeddingResponse`.
See the contract tests under `go/tests/contract` for concrete examples.

## 3. Router: composing multiple providers

The `internal/model.Router` type lets you plug multiple provider clients into a
single dispatcher:

```go
router := model.NewRouter(
  model.WithMaxConcurrency(16),
  model.WithTimeout(30*time.Second),
)

openAI := openai.New("", os.Getenv("OPENAI_API_KEY"), nil)
router.RegisterChatProvider(openAI)

// You can register additional providers (Gemini, Groq, …) the same way.

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

The router also exposes:

- `router.Stream(ctx, req, handler)` – streaming chat  
- `router.Embed(ctx, embeddingReq)` – embeddings  
- `router.Statuses()` – provider status list for health checks  

This is the main composition primitive used internally; you can also embed it in
your own services for multi-provider routing.

## 4. HTTP runtime (design notes, not stable)

The specs and OpenAPI document under
`specs/001-vitepress-docs/contracts/docs-site-openapi.yaml` describe an HTTP
runtime with endpoints such as:

- `GET /health` – health and provider status  
- `POST /agents` – create an agent definition  
- `POST /agents/{agentId}/sessions` – create a session  
- `POST /agents/{agentId}/sessions/{sessionId}/messages` – send a message  

At the moment, this HTTP surface is **not** considered a stable public API:

- The Go runtime implementation is still evolving.  
- Some flows in the design docs reference behavior that has not yet been fully
  implemented in `go/cmd/agno`.  

For production code today, prefer:

- Calling providers directly via `go/pkg/providers/*`  
- Optionally composing them with `internal/model.Router` in your own service  

The documentation will be updated to treat the HTTP runtime as a first-class
surface once the implementation and contracts have stabilized.

## 5. Where to look in the repo

- `go/pkg/providers/*` – provider clients (OpenAI, Gemini, Groq, …)  
- `go/internal/agent` – shared agent/model configuration types, usage tracking  
- `go/internal/model` – request/response types, router, provider interfaces  
- `go/tests/providers` – concrete usage examples for provider clients  
- `go/tests/contract` – contract tests that exercise the HTTP-shaped data model  

Use these files as the ground truth when adapting the examples above to your
own Go applications.

