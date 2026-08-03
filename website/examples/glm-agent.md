# GLM Agent Example

This example demonstrates how to use Agno-Go with GLM (智谱AI), China's leading domestic LLM platform.

## Overview

GLM (Zhipu AI) is an advanced language model developed by Tsinghua University's Knowledge Engineering Group. It offers:

- **Optimized for Chinese**: Excellent performance on Chinese language tasks
- **GLM-4**: Main conversational model with 128K context
- **GLM-4V**: Vision-enabled multimodal capabilities
- **GLM-3-Turbo**: Fast and cost-effective variant

## Prerequisites

1. **Go 1.21+** installed
2. **GLM API Key** from https://open.bigmodel.cn/

## Getting Your API Key

1. Visit https://open.bigmodel.cn/
2. Sign up or log in
3. Navigate to API Keys section
4. Create a new API key

The API key format is: `{key_id}.{key_secret}`

## Installation

```bash
go get github.com/rexleimo/agno-go
```

## Environment Setup

Create a `.env` file or export the environment variable:

```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
```

## Basic Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
)

func main() {
    // Create GLM model
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatalf("Failed to create GLM model: %v", err)
    }

    // Create agent
    agent, err := agent.New(agent.Config{
        Name:         "GLM Assistant",
        Model:        model,
        Instructions: "You are a helpful AI assistant.",
    })
    if err != nil {
        log.Fatalf("Failed to create agent: %v", err)
    }

    // Run agent
    output, err := agent.Run(context.Background(), "Hello! Tell me about yourself.")
    if err != nil {
        log.Fatalf("Agent run failed: %v", err)
    }

    fmt.Println(output.Content)
}
```

## Example with Tools

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
    // Create GLM model
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create agent with calculator tools
    agent, err := agent.New(agent.Config{
        Name:         "GLM Calculator Agent",
        Model:        model,
        Toolkits:     []toolkit.Toolkit{calculator.New()},
        Instructions: "You are a helpful AI assistant that can perform calculations.",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Test calculation
    output, err := agent.Run(context.Background(), "What is 123 * 456?")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Result: %s\n", output.Content)
}
```

## Chinese Language Example

GLM excels at Chinese language tasks:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
)

func main() {
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
    })
    if err != nil {
        log.Fatal(err)
    }

    agent, err := agent.New(agent.Config{
        Name:         "中文助手",
        Model:        model,
        Instructions: "你是一个有用的中文AI助手。",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Ask in Chinese
    output, err := agent.Run(context.Background(), "请用中文介绍一下人工智能的发展历史。")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output.Content)
}
```

## Running the Example

1. Clone the repository:
```bash
git clone https://github.com/rexleimo/agno-go.git
cd agno-Go
```

2. Set your API key:
```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
```

3. Run the example:
```bash
go run cmd/examples/glm_agent/main.go
```

## Configuration Options

```go
glm.Config{
    APIKey:      string  // Required: {key_id}.{key_secret} format
    BaseURL:     string  // Optional: Custom API endpoint
    Temperature: float64 // Optional: 0.0-1.0 (default: 0.7)
    MaxTokens:   int     // Optional: Max response tokens
    TopP:        float64 // Optional: Top-p sampling parameter
    DoSample:    bool    // Optional: Enable sampling
}
```

## Authentication

GLM uses JWT (JSON Web Token) authentication:

- API key is split into `key_id` and `key_secret`
- JWT token is generated using HMAC-SHA256 signing
- Token is valid for 7 days
- Automatically handled by the SDK

## Supported Models

| Model | Context | Best For |
|-------|---------|----------|
| `glm-4` | 128K | General conversation, Chinese language |
| `glm-4v` | 128K | Vision tasks, multimodal |
| `glm-3-turbo` | 128K | Fast responses, cost-effective |

## Common Issues

### Invalid API Key Format

**Problem**: `API key must be in format {key_id}.{key_secret}`

**Solution**: Ensure your API key contains a dot (.) separator between key_id and key_secret.

### Authentication Failed

**Problem**: `GLM API error: Invalid API key`

**Solution**:
- Verify your API key is correct
- Check if the API key is active at https://open.bigmodel.cn/
- Ensure no extra spaces in the environment variable

### Rate Limiting

**Problem**: `GLM API error: Rate limit exceeded`

**Solution**:
- Implement retry logic with exponential backoff
- Reduce request frequency
- Upgrade your API plan if needed

## Next Steps

- Learn about [Models](/guide/models) for more LLM options
- Add more [Tools](/guide/tools) to enhance capabilities
- Build [Teams](/guide/team) with multiple agents
- Explore [Workflows](/guide/workflow) for complex processes

## Related Examples

- [Simple Agent](/examples/simple-agent) - OpenAI example
- [Claude Agent](/examples/claude-agent) - Anthropic example
- [Team Demo](/examples/team-demo) - Multi-agent collaboration

## Resources

- [GLM Official Website](https://www.bigmodel.cn/)
- [GLM API Documentation](https://open.bigmodel.cn/dev/api)
- [Agno-Go Repository](https://github.com/rexleimo/agno-go)
