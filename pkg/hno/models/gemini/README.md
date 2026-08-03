# Gemini Provider

Google Gemini API integration for Agno-Go.

## Features

- ✅ Synchronous API calls (Invoke)
- ✅ Server-Sent Events (SSE) streaming (InvokeStream)
- ✅ Function calling / Tool use
- ✅ System instructions
- ✅ Multi-turn conversations
- ✅ Comprehensive error handling
- ✅ 100% test coverage

## Supported Models

- `gemini-pro` - Optimized for text-based tasks
- `gemini-pro-vision` - Supports both text and image inputs
- `gemini-ultra` - Most capable model for complex tasks

## Installation

```bash
go get github.com/rexleimo/agno-go
```

## Configuration

```go
import "github.com/rexleimo/agno-go/pkg/hno/models/gemini"

model, err := gemini.New("gemini-pro", gemini.Config{
    APIKey:      "your-api-key",           // Required
    BaseURL:     "custom-url",             // Optional, defaults to Google's API
    Temperature: 0.7,                      // Optional, 0.0-1.0
    MaxTokens:   2048,                     // Optional, max output tokens
})
```

### Environment Variables

```bash
export GEMINI_API_KEY="your-api-key-here"
```

Get your API key from: https://makersuite.google.com/app/apikey

## Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rexleimo/agno-go/pkg/hno/models"
    "github.com/rexleimo/agno-go/pkg/hno/models/gemini"
    "github.com/rexleimo/agno-go/pkg/hno/types"
)

func main() {
    // Create model
    model, err := gemini.New("gemini-pro", gemini.Config{
        APIKey:      "your-api-key",
        Temperature: 0.7,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create request
    req := &models.InvokeRequest{
        Messages: []*types.Message{
            {Role: types.RoleUser, Content: "What is the capital of France?"},
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
        {Role: types.RoleUser, Content: "Write a short story"},
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
            Content: "You are a helpful assistant that speaks like a pirate.",
        },
        {
            Role:    types.RoleUser,
            Content: "Tell me about the weather",
        },
    },
}

resp, err := model.Invoke(context.Background(), req)
```

### Function Calling

```go
req := &models.InvokeRequest{
    Messages: []*types.Message{
        {Role: types.RoleUser, Content: "What's the weather in San Francisco?"},
    },
    Tools: []models.ToolDefinition{
        {
            Type: "function",
            Function: models.FunctionSchema{
                Name:        "get_weather",
                Description: "Get the current weather in a location",
                Parameters: map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "location": map[string]interface{}{
                            "type":        "string",
                            "description": "The city and state, e.g. San Francisco, CA",
                        },
                        "unit": map[string]interface{}{
                            "type": "string",
                            "enum": []string{"celsius", "fahrenheit"},
                        },
                    },
                    "required": []string{"location"},
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

See the complete example: [`cmd/examples/gemini_agent/main.go`](../../../../cmd/examples/gemini_agent/main.go)

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/gemini"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
)

// Create model
model, err := gemini.New("gemini-pro", gemini.Config{
    APIKey: os.Getenv("GEMINI_API_KEY"),
})

// Create agent
ag, err := agent.New(agent.Config{
    Name:     "GeminiAssistant",
    Model:    model,
    Toolkits: []toolkit.Toolkit{calculator.New()},
})

// Run agent
output, err := ag.Run(context.Background(), "What is 25 times 4?")
fmt.Println(output.Content)
```

## API Reference

### Config

```go
type Config struct {
    APIKey      string  // Required: Google AI API key
    BaseURL     string  // Optional: Custom API endpoint
    Temperature float64 // Optional: 0.0-1.0, controls randomness
    MaxTokens   int     // Optional: Maximum tokens to generate
}
```

### Methods

#### New

```go
func New(modelID string, config Config) (*Gemini, error)
```

Creates a new Gemini model instance.

**Parameters:**
- `modelID`: Model identifier (e.g., "gemini-pro")
- `config`: Configuration options

**Returns:**
- `*Gemini`: Initialized model instance
- `error`: Configuration or validation errors

#### Invoke

```go
func (g *Gemini) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error)
```

Calls the Gemini API synchronously.

#### InvokeStream

```go
func (g *Gemini) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error)
```

Calls the Gemini API with streaming response using Server-Sent Events.

## Implementation Details

### Message Conversion

Gemini uses a specific message format:
- `user` role for user messages
- `model` role for assistant messages (not "assistant")
- `function` role for tool results
- System instructions are separate from conversation messages

### Tool Calling

Gemini's function calling format differs from OpenAI:
- Uses `functionCall` instead of `tool_calls`
- Arguments are passed as objects, not JSON strings
- Function responses use the `function` role

### Streaming

The implementation uses a custom SSE decoder to parse Server-Sent Events:
- Handles `data:` prefixed lines
- Supports multi-line events with `\n\n` separator
- Gracefully handles connection errors

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
go test -v ./pkg/hno/models/gemini/

# With coverage
go test -v -cover ./pkg/hno/models/gemini/

# Generate coverage report
go test -coverprofile=coverage.out ./pkg/hno/models/gemini/
go tool cover -html=coverage.out
```

## Performance

- Average latency: ~500ms for simple queries
- Streaming: First token in ~200ms
- Supports concurrent requests
- Automatic retry with exponential backoff (TODO)

## Limitations

1. Image inputs not yet supported (coming soon)
2. Safety settings not configurable yet
3. No built-in caching mechanism
4. Rate limiting handled by API, not client

## Resources

- [Google AI Studio](https://makersuite.google.com/)
- [Gemini API Documentation](https://ai.google.dev/docs)
- [API Reference](https://ai.google.dev/api/rest)
- [Pricing](https://ai.google.dev/pricing)

## License

MIT License - see LICENSE file for details
