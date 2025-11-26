# Quickstart: Agno-Go in 10 Minutes

This guide walks you through a minimal end-to-end flow using the **Go provider
clients** in this repository. The focus of this first Quickstart is:

1. Configure a provider (for example OpenAI).  
2. Call it from Go using `go/pkg/providers/*`.  
3. Inspect the returned message.  

If you want to call the models directly from Go code instead of via curl, you
can start with the provider client and the shared request types used in the
tests:

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

  client := openai.New("", apiKey, nil) // default endpoint

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
}
```

## Next steps

- Review [Configuration & Security](./config-and-security) to understand how to set
  provider keys, endpoints and runtime options safely.  
- Explore the [Provider Matrix](./providers/matrix) for a broader view of capabilities
  across providers and which env vars they require.  
- Advanced AgentOS HTTP flows (agents / sessions / messages) will be documented once
  the runtime surface is stabilized; for now, treat the provider clients as the main
  public entrypoint.  
