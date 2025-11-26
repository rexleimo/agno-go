
# Advanced Guide: Multi-provider Routing (Go-first)

This guide shows how to route requests across multiple model providers **inside
your own Go service**, using the existing provider clients and the
`internal/model.Router`. It deliberately avoids relying on the unfinished HTTP
runtime; all examples here are pure Go.

## 1. When to use multi-provider routing

Typical scenarios:

- Use one provider for general-purpose chat, and another for latency- or
  cost-sensitive workloads.  
- Fall back to a different provider when the primary one is unavailable.  
- Experiment with new models while keeping your application’s integration
  surface stable.  

In all of these cases, the core idea is the same: **compose multiple
`ChatProvider` implementations behind a single API**.

## 2. Core building blocks

The main types involved are:

- `go/pkg/providers/*` – provider clients (OpenAI, Gemini, Groq, …) implementing
  `model.ChatProvider` / `model.EmbeddingProvider`.  
- `internal/model.Router` – a dispatcher that knows how to route `ChatRequest`
  and `EmbeddingRequest` to the registered providers.  
- `agent.ModelConfig` – determines which provider/model to use for a given
  request.  

## 3. Example: router with OpenAI and Gemini

The following example shows a router that can talk to OpenAI and Gemini. It
does not start any HTTP server; it’s just a Go service that you can embed in
your own code.

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

  // Prefer OpenAI; fall back to Gemini if needed.
  providers := []agent.Provider{agent.ProviderOpenAI, agent.ProviderGemini}

  var lastErr error
  for _, prov := range providers {
    req := model.ChatRequest{
      Model: agent.ModelConfig{
        Provider: prov,
        ModelID:  "gpt-4o-mini", // or a Gemini model id when prov == ProviderGemini
        Stream:   false,
      },
      Messages: []agent.Message{
        {Role: agent.RoleUser, Content: "Recommend a cheap, fast model for a small internal tool."},
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

This pattern keeps provider-specific details (model IDs, keys, endpoints) in
your configuration, while your application logic just chooses which
`agent.Provider` to try first.

## 4. Streaming variants

The same router can be used for streaming:

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "Stream a short greeting."},
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

You can apply the same “try primary, then fallback” logic as in the non-streaming
example by constructing different `ChatRequest` values per provider.

## 5. Configuration considerations

When building multi-provider routing around the router:

- Keep API keys and endpoints in `.env` / `config/default.yaml`, not hard-coded
  in your application code.  
- Use the [Provider Matrix](../providers/matrix) to decide which providers to
  enable and which models to use.  
- Treat the router as the *only* component that knows about specific providers;
  the rest of your code should only depend on `agent.ModelConfig` and
  `model.ChatRequest`.  

## 6. Relation to the HTTP runtime

The HTTP runtime design in the specs follows the same concepts (model config,
providers, routing), but its implementation is not yet a stable public surface.

Until that stabilizes, you can:

- Build production flows directly on top of `go/pkg/providers/*` and
  `internal/model.Router` as shown above.  
- Use the HTTP contracts in `specs/001-go-agno-rewrite/contracts` only as a
  reference for data shapes, not as a guaranteed external API.  

Once the HTTP layer is ready, these routing patterns will naturally carry over
to the runtime’s agents/sessions/messages endpoints.

