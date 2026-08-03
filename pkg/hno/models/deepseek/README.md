# DeepSeek Provider

DeepSeek API integration for Agno-Go, providing access to DeepSeek-V3 and DeepSeek-R1 models.

## Features

- ✅ OpenAI-compatible API format
- ✅ Synchronous API calls (Invoke)
- ✅ Streaming responses (InvokeStream)
- ✅ Function calling / Tool use
- ✅ High-quality Chinese and English support
- ✅ Comprehensive error handling
- ✅ 81.6% test coverage

## Supported Models

- **deepseek-chat** - DeepSeek-V3, optimized for general tasks
  - Trained on 15 trillion tokens
  - Excellent reasoning and coding abilities
  - Cost-effective pricing

- **deepseek-reasoner** - DeepSeek-R1, specialized for complex reasoning
  - Advanced reasoning model
  - Excellent for math, coding, and logic problems
  - Extended context window

## Installation

```bash
go get github.com/rexleimo/agno-go
```

## Configuration

```go
import "github.com/rexleimo/agno-go/pkg/hno/models/deepseek"

model, err := deepseek.New("deepseek-chat", deepseek.Config{
    APIKey:      "your-api-key",           // Required
    BaseURL:     "custom-url",             // Optional, defaults to official API
    Temperature: 0.7,                      // Optional, 0.0-1.0
    MaxTokens:   2048,                     // Optional, max output tokens
})
```

### Environment Variables

```bash
export DEEPSEEK_API_KEY="your-api-key-here"
```

Get your API key from: https://platform.deepseek.com/api_keys

## Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rexleimo/agno-go/pkg/hno/models"
    "github.com/rexleimo/agno-go/pkg/hno/models/deepseek"
    "github.com/rexleimo/agno-go/pkg/hno/types"
)

func main() {
    // Create model
    model, err := deepseek.New("deepseek-chat", deepseek.Config{
        APIKey:      "your-api-key",
        Temperature: 0.7,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create request
    req := &models.InvokeRequest{
        Messages: []*types.Message{
            {Role: types.RoleUser, Content: "What is the capital of China?"},
        },
    }

    // Call model
    resp, err := model.Invoke(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Content)
}
```

### Streaming

```go
req := &models.InvokeRequest{
    Messages: []*types.Message{
        {Role: types.RoleUser, Content: "Write a short poem about AI"},
    },
}

chunks, err := model.InvokeStream(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

for chunk := range chunks {
    if chunk.Error != nil {
        log.Printf("Error: %v", chunk.Error)
        break
    }

    fmt.Print(chunk.Content)

    if chunk.Done {
        break
    }
}
```

### With System Instructions

```go
req := &models.InvokeRequest{
    Messages: []*types.Message{
        {
            Role:    types.RoleSystem,
            Content: "You are a helpful programming assistant specialized in Go.",
        },
        {
            Role:    types.RoleUser,
            Content: "Explain goroutines",
        },
    },
}

resp, err := model.Invoke(context.Background(), req)
```

### Function Calling

```go
req := &models.InvokeRequest{
    Messages: []*types.Message{
        {Role: types.RoleUser, Content: "What's 15 multiplied by 8?"},
    },
    Tools: []models.ToolDefinition{
        {
            Type: "function",
            Function: models.FunctionSchema{
                Name:        "multiply",
                Description: "Multiply two numbers",
                Parameters: map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "a": map[string]interface{}{
                            "type":        "number",
                            "description": "First number",
                        },
                        "b": map[string]interface{}{
                            "type":        "number",
                            "description": "Second number",
                        },
                    },
                    "required": []string{"a", "b"},
                },
            },
        },
    },
}

resp, err := model.Invoke(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

// Check for tool calls
if len(resp.ToolCalls) > 0 {
    for _, tc := range resp.ToolCalls {
        fmt.Printf("Function: %s\n", tc.Function.Name)
        fmt.Printf("Arguments: %s\n", tc.Function.Arguments)
    }
}
```

### With Agent

See the complete example: [`cmd/examples/deepseek_agent/main.go`](../../../../cmd/examples/deepseek_agent/main.go)

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/deepseek"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
)

// Create model
model, err := deepseek.New("deepseek-chat", deepseek.Config{
    APIKey: os.Getenv("DEEPSEEK_API_KEY"),
})

// Create agent
ag, err := agent.New(agent.Config{
    Name:     "DeepSeekAssistant",
    Model:    model,
    Toolkits: []toolkit.Toolkit{calculator.New()},
})

// Run agent
output, err := ag.Run(context.Background(), "What is 234 * 567?")
fmt.Println(output.Content)
```

## API Reference

### Config

```go
type Config struct {
    APIKey      string  // Required: DeepSeek API key
    BaseURL     string  // Optional: Custom API endpoint
    Temperature float64 // Optional: 0.0-1.0, controls randomness
    MaxTokens   int     // Optional: Maximum tokens to generate
}
```

### Methods

#### New

```go
func New(modelID string, config Config) (*DeepSeek, error)
```

Creates a new DeepSeek model instance.

**Parameters:**
- `modelID`: Model identifier ("deepseek-chat" or "deepseek-reasoner")
- `config`: Configuration options

**Returns:**
- `*DeepSeek`: Initialized model instance
- `error`: Configuration or validation errors

#### Invoke

```go
func (d *DeepSeek) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error)
```

Calls the DeepSeek API synchronously.

#### InvokeStream

```go
func (d *DeepSeek) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error)
```

Calls the DeepSeek API with streaming response.

## Implementation Details

### OpenAI Compatibility

DeepSeek uses the OpenAI-compatible API format, allowing the use of the official `go-openai` SDK with a custom base URL:

```go
clientConfig := openai.DefaultConfig(config.APIKey)
clientConfig.BaseURL = "https://api.deepseek.com/v1"
```

This approach provides:
- ✅ Code reuse and maintainability
- ✅ Automatic updates when OpenAI SDK improves
- ✅ Consistent interface with OpenAI models
- ✅ Battle-tested streaming implementation

### Performance Optimizations

- **Context Caching**: Automatically reduces input token costs by ~40%
- **Efficient Streaming**: Goroutine-based async processing
- **Connection Pooling**: Reuses HTTP connections

## Error Handling

```go
resp, err := model.Invoke(ctx, req)
if err != nil {
    switch {
    case types.IsInvalidConfigError(err):
        // Handle configuration errors
    case types.IsAPIError(err):
        // Handle API errors (network, auth, rate limits)
    default:
        // Handle other errors
    }
}
```

## Testing

```bash
# Run tests
go test -v ./pkg/hno/models/deepseek/

# With coverage
go test -v -cover ./pkg/hno/models/deepseek/

# Generate coverage report
go test -coverprofile=coverage.out ./pkg/hno/models/deepseek/
go tool cover -html=coverage.out
```

## Performance

- Average latency: ~400ms for simple queries
- Streaming: First token in ~150ms
- Supports concurrent requests
- Context caching for repeated content

## Model Comparison

| Model | Best For | Context | Cost |
|-------|----------|---------|------|
| deepseek-chat | General tasks, conversations | 64K | Low |
| deepseek-reasoner | Complex reasoning, math, code | 64K | Medium |

## Pricing

DeepSeek offers competitive pricing:
- **Input tokens**: $0.14 per million tokens (cached: $0.014)
- **Output tokens**: $0.28 per million tokens

Significantly cheaper than GPT-4 while maintaining comparable quality.

## Limitations

1. Context caching requires repeated content (>1K tokens)
2. Some features may differ slightly from OpenAI
3. Rate limiting applies based on subscription tier
4. No image input support (yet)

## Resources

- [DeepSeek Platform](https://platform.deepseek.com/)
- [API Documentation](https://api-docs.deepseek.com/)
- [Pricing](https://api-docs.deepseek.com/quick_start/pricing)
- [Model Comparison](https://platform.deepseek.com/models)

## License

MIT License - see LICENSE file for details
