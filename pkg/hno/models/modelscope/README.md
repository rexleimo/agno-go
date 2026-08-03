# ModelScope Provider

ModelScope (魔搭社区) API integration for Agno-Go, providing access to Alibaba Cloud's Qwen and other Chinese models.

## Features

- ✅ OpenAI-compatible API format via DashScope
- ✅ Synchronous API calls (Invoke)
- ✅ Streaming responses (InvokeStream)
- ✅ Function calling / Tool use
- ✅ Excellent Chinese language support
- ✅ Comprehensive error handling
- ✅ 78.9% test coverage

## Supported Models

ModelScope provides access to various models through Alibaba Cloud's DashScope service:

- **qwen-turbo** - Fast and cost-effective
- **qwen-plus** - Balanced performance and quality
- **qwen-max** - Maximum capability
- **qwen-long** - Extended context window
- And many more Chinese-optimized models

## Installation

```bash
go get github.com/rexleimo/agno-go
```

## Configuration

```go
import "github.com/rexleimo/agno-go/pkg/hno/models/modelscope"

model, err := modelscope.New("qwen-plus", modelscope.Config{
    APIKey:      "your-dashscope-api-key", // Required
    BaseURL:     "custom-url",             // Optional
    Temperature: 0.7,                      // Optional, 0.0-1.0
    MaxTokens:   2048,                     // Optional
})
```

### Environment Variables

```bash
export DASHSCOPE_API_KEY="your-api-key-here"
```

Get your API key from: https://dashscope.console.aliyun.com/

## Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rexleimo/agno-go/pkg/hno/models"
    "github.com/rexleimo/agno-go/pkg/hno/models/modelscope"
    "github.com/rexleimo/agno-go/pkg/hno/types"
)

func main() {
    // Create model
    model, err := modelscope.New("qwen-plus", modelscope.Config{
        APIKey:      "your-api-key",
        Temperature: 0.7,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create request
    req := &models.InvokeRequest{
        Messages: []*types.Message{
            {Role: types.RoleUser, Content: "介绍一下北京"},
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
        {Role: types.RoleUser, Content: "写一首关于AI的诗"},
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

### With Agent

See the complete example: [`cmd/examples/modelscope_agent/main.go`](../../../../cmd/examples/modelscope_agent/main.go)

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/modelscope"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
)

// Create model
model, err := modelscope.New("qwen-plus", modelscope.Config{
    APIKey: os.Getenv("DASHSCOPE_API_KEY"),
})

// Create agent
ag, err := agent.New(agent.Config{
    Name:         "智能助手",
    Model:        model,
    Instructions: "你是一个有帮助的AI助手",
    Toolkits:     []toolkit.Toolkit{calculator.New()},
})

// Run agent
output, err := ag.Run(context.Background(), "计算 123 * 456")
fmt.Println(output.Content)
```

## API Reference

### Config

```go
type Config struct {
    APIKey      string  // Required: DashScope API key
    BaseURL     string  // Optional: Custom API endpoint
    Temperature float64 // Optional: 0.0-1.0
    MaxTokens   int     // Optional: Max tokens to generate
}
```

### Methods

#### New

```go
func New(modelID string, config Config) (*ModelScope, error)
```

Creates a new ModelScope model instance.

#### Invoke

```go
func (m *ModelScope) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error)
```

Calls the ModelScope API synchronously.

#### InvokeStream

```go
func (m *ModelScope) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error)
```

Calls the ModelScope API with streaming response.

## Implementation Details

### DashScope Integration

ModelScope uses Alibaba Cloud's DashScope service with OpenAI-compatible API:

```go
clientConfig := openai.DefaultConfig(config.APIKey)
clientConfig.BaseURL = "https://api-inference.modelscope.cn/v1"
```

This provides:
- ✅ Access to Qwen and other Chinese models
- ✅ OpenAI-compatible interface
- ✅ Excellent Chinese language understanding
- ✅ Integration with Alibaba Cloud ecosystem

## Testing

```bash
# Run tests
go test -v ./pkg/hno/models/modelscope/

# With coverage
go test -v -cover ./pkg/hno/models/modelscope/

# Generate coverage report
go test -coverprofile=coverage.out ./pkg/hno/models/modelscope/
go tool cover -html=coverage.out
```

## Performance

- Average latency: ~500ms for simple queries
- Streaming: First token in ~200ms
- Excellent Chinese language processing
- Supports concurrent requests

## Model Comparison

| Model | Context | Speed | Quality | Use Case |
|-------|---------|-------|---------|----------|
| qwen-turbo | 8K | Fast | Good | General tasks |
| qwen-plus | 32K | Medium | Better | Most applications |
| qwen-max | 32K | Slower | Best | Complex reasoning |
| qwen-long | 1M | Medium | Good | Long documents |

## Pricing

Competitive pricing through Alibaba Cloud DashScope:
- Free tier available for testing
- Pay-as-you-go pricing
- Volume discounts available

Visit: https://help.aliyun.com/zh/dashscope/developer-reference/tongyi-thousand-questions-metering-and-billing

## Limitations

1. Requires Alibaba Cloud account
2. API access may have geographic restrictions
3. Some models optimized primarily for Chinese
4. Rate limits based on account tier

## Resources

- [ModelScope Platform](https://modelscope.cn/)
- [DashScope Documentation](https://help.aliyun.com/zh/dashscope/)
- [API Reference](https://help.aliyun.com/zh/dashscope/developer-reference/api-details)
- [Model List](https://help.aliyun.com/zh/dashscope/developer-reference/model-square)

## License

MIT License - see LICENSE file for details
