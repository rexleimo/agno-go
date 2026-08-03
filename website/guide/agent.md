# Agent

An **Agent** is an autonomous AI entity that can use tools, maintain conversation context, and execute tasks independently.

## Overview

```go
import "github.com/rexleimo/agno-Go/pkg/hno/agent"

agent, err := agent.New(agent.Config{
    Name:         "My Agent",
    Model:        model,
    Toolkits:     []toolkit.Toolkit{calculator.New()},
    Instructions: "You are a helpful assistant",
    MaxLoops:     10,
})

output, err := agent.Run(context.Background(), "What is 2+2?")
```

## Configuration

### Config Structure

```go
type Config struct {
    Name         string            // Agent name
    Model        models.Model      // LLM model
    Toolkits     []toolkit.Toolkit // Available tools
    Memory       memory.Memory     // Conversation memory
    Instructions string            // System instructions
    MaxLoops     int               // Max tool call loops (default: 10)
    UserID       string            // Optional tenant identifier
    PreHooks     []hooks.Hook      // Pre-execution hooks
    PostHooks    []hooks.Hook      // Post-execution hooks
    Logger       *slog.Logger      // Custom logger (optional)
    EnableCache  bool              // Enable response caching
    CacheProvider cache.Provider   // Custom cache provider (optional)
    CacheTTL     time.Duration     // Cache TTL (default: 5m)
    StoreToolMessages   *bool      // Include tool messages in RunOutput (default: true)
    StoreHistoryMessages *bool     // Include memory messages in RunOutput (default: true)
}
```

### Parameters

- **Name** (required): Human-readable agent identifier
- **Model** (required): LLM model instance (OpenAI, Claude, etc.)
- **Toolkits** (optional): List of available tools
- **Memory** (optional): Defaults to in-memory storage with 100 message limit
- **Instructions** (optional): System prompt/persona
- **MaxLoops** (optional): Prevent infinite tool call loops (default: 10)
- **UserID** (optional): Associate runs with a tenant or end-user
- **PreHooks** (optional): Validation hooks before execution
- **PostHooks** (optional): Validation hooks after execution
- **Logger** (optional): Custom logger for structured output
- **EnableCache** (optional): Deduplicate identical model calls
- **CacheProvider** (optional): Supply custom cache backend (defaults to in-memory LRU)
- **CacheTTL** (optional): Override cache expiration (default 5 minutes)
- **StoreToolMessages** (optional): Filter tool call transcripts from output
- **StoreHistoryMessages** (optional): Filter historical memory messages from output

## Basic Usage

### Simple Agent

```go
package main

import (
    "context"
    "fmt"
    "github.com/rexleimo/agno-Go/pkg/hno/agent"
    "github.com/rexleimo/agno-Go/pkg/hno/models/openai"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    ag, _ := agent.New(agent.Config{
        Name:         "Assistant",
        Model:        model,
        Instructions: "You are a helpful assistant",
    })

    output, _ := ag.Run(context.Background(), "Hello!")
    fmt.Println(output.Content)
}
```

### Agent with Tools

```go
import (
    "github.com/rexleimo/agno-Go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-Go/pkg/hno/tools/http"
)

ag, _ := agent.New(agent.Config{
    Name:  "Smart Assistant",
    Model: model,
    Toolkits: []toolkit.Toolkit{
        calculator.New(),
        http.New(),
    },
    Instructions: "You can do math and make HTTP requests",
})

output, _ := ag.Run(ctx, "Calculate 15 * 23 and fetch https://api.github.com")
```

## Advanced Features

### Custom Memory

```go
import "github.com/rexleimo/agno-Go/pkg/hno/memory"

// Create memory with custom limit
mem := memory.NewInMemory(50) // Keep last 50 messages

ag, _ := agent.New(agent.Config{
    Memory: mem,
    // ... other config
})
```

### Hooks & Guardrails

Validate inputs and outputs with hooks:

```go
import "github.com/rexleimo/agno-Go/pkg/hno/guardrails"

// Built-in prompt injection guard
promptGuard := guardrails.NewPromptInjectionGuardrail()

// Custom validation hook
customHook := func(ctx context.Context, input *hooks.HookInput) error {
    if len(input.Input) > 1000 {
        return fmt.Errorf("input too long")
    }
    return nil
}

ag, _ := agent.New(agent.Config{
    PreHooks:  []hooks.Hook{customHook, promptGuard},
    PostHooks: []hooks.Hook{outputValidator},
    // ... other config
})
```

### Context and Timeouts

```go
import "time"

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := ag.Run(ctx, "Complex task...")
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        fmt.Println("Timeout!")
    }
}
```

### Response Caching (v1.2.6)

Enable deterministic responses to reuse cached model outputs:

```go
ag, _ := agent.New(agent.Config{
    Model:       model,
    EnableCache: true,
    CacheTTL:    2 * time.Minute,
})

first, _ := ag.Run(ctx, "Summarise REST vs gRPC")
second, _ := ag.Run(ctx, "Summarise REST vs gRPC")

if cached, _ := second.Metadata["cache_hit"].(bool); cached {
    // Handle cached response
}
```

Provide a custom `cache.Provider` when you want Redis or shared storage; otherwise an in-memory LRU is used.

## Run Output

The `Run` method returns `*RunOutput`:

```go
type RunOutput struct {
    Content  string                 // Agent's response
    Messages []types.Message        // Full message history
    Metadata map[string]interface{} // Additional data
}
```

Example:

```go
output, err := ag.Run(ctx, "Tell me a joke")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Response:", output.Content)
fmt.Println("Messages:", len(output.Messages))
fmt.Println("Metadata:", output.Metadata)
```

## Memory Management

### Clear Memory

```go
// Clear all conversation history
ag.ClearMemory()
```

### Access Memory

```go
// Get current messages
messages := ag.GetMemory().GetMessages()
fmt.Println("History:", len(messages))
```

## Error Handling

```go
output, err := ag.Run(ctx, input)
if err != nil {
    switch {
    case errors.Is(err, types.ErrInvalidInput):
        // Handle invalid input
    case errors.Is(err, types.ErrRateLimit):
        // Handle rate limit
    case errors.Is(err, context.DeadlineExceeded):
        // Handle timeout
    default:
        // Handle other errors
    }
}
```

## Best Practices

### 1. Always Use Context

```go
ctx := context.Background()
output, err := ag.Run(ctx, input)
```

### 2. Set Appropriate MaxLoops

```go
// For simple tasks
MaxLoops: 5

// For complex reasoning
MaxLoops: 15
```

### 3. Provide Clear Instructions

```go
Instructions: `You are a customer support agent.
- Be polite and professional
- Use tools to look up information
- If unsure, ask for clarification`
```

### 4. Use Type-Safe Tool Configurations

```go
calc := calculator.New()
httpClient := http.New(http.Config{
    Timeout: 10 * time.Second,
})

ag, _ := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{calc, httpClient},
})
```

## Performance Considerations

- **Agent Creation**: ~180ns average
- **Memory Footprint**: ~1.2KB per agent
- **Concurrent Agents**: Fully thread-safe, use goroutines freely

```go
// Concurrent agents
for i := 0; i < 100; i++ {
    go func(id int) {
        ag, _ := agent.New(config)
        output, _ := ag.Run(ctx, input)
        fmt.Printf("Agent %d: %s\n", id, output.Content)
    }(i)
}
```

## Examples

See working examples:

- [Simple Agent](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/simple_agent)
- [Claude Agent](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/claude_agent)
- [Ollama Agent](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/ollama_agent)
- [Agent with Guardrails](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/agent_with_guardrails)

## API Reference

For complete API documentation, see [Agent API Reference](/api/agent).

## Next Steps

- [Team](/guide/team) - Multi-agent collaboration
- [Workflow](/guide/workflow) - Step-based orchestration
- [Tools](/guide/tools) - Built-in and custom tools
- [Models](/guide/models) - LLM provider configuration
