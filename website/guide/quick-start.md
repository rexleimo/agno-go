# Quick Start

Get started with HNO in less than 5 minutes!

## Prerequisites

- Go 1.21 or later
- OpenAI API key (or Anthropic/Ollama)
- Basic understanding of AI agents

## Installation

### Option 1: Using Go Get

```bash
go get github.com/rexleimo/HNO
```

### Option 2: Clone Repository

```bash
git clone https://github.com/rexleimo/HNO.git
cd HNO
go mod download
```

## Your First Agent

### 1. Simple Agent (No Tools)

Create a file `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)

func main() {
    // Get API key from environment
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY environment variable is required")
    }

    // Create OpenAI model
    model, err := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })
    if err != nil {
        log.Fatalf("Failed to create model: %v", err)
    }

    // Create agent
    ag, err := agent.New(agent.Config{
        Name:         "Assistant",
        Model:        model,
        Instructions: "You are a helpful assistant.",
    })
    if err != nil {
        log.Fatalf("Failed to create agent: %v", err)
    }

    // Run agent
    output, err := ag.Run(context.Background(), "What is the capital of France?")
    if err != nil {
        log.Fatalf("Agent run failed: %v", err)
    }

    fmt.Println("Agent:", output.Content)
}
```

**Run it:**

```bash
export OPENAI_API_KEY=sk-your-key-here
go run main.go
```

**Expected output:**

```
Agent: The capital of France is Paris.
```

### 2. Agent with Tools

Add calculator tools:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
    "github.com/rexleimo/HNO/pkg/hno/tools/calculator"
    "github.com/rexleimo/HNO/pkg/hno/tools/toolkit"
)

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY required")
    }

    // Create model
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })

    // Create agent WITH tools
    ag, _ := agent.New(agent.Config{
        Name:  "Calculator Agent",
        Model: model,
        Toolkits: []toolkit.Toolkit{
            calculator.New(),
        },
        Instructions: "You are a math assistant. Use the calculator tools for calculations.",
    })

    // Ask a math question
    output, _ := ag.Run(context.Background(), "What is 123 * 456 + 789?")

    fmt.Println("Question: What is 123 * 456 + 789?")
    fmt.Println("Agent:", output.Content)
}
```

**Run it:**

```bash
go run main.go
```

**Expected output:**

```
Question: What is 123 * 456 + 789?
Agent: The result is 56,877
```

### 3. Multi-Turn Conversation

Add memory for conversation:

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY required")
    }

    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })

    ag, _ := agent.New(agent.Config{
        Name:         "Chat Assistant",
        Model:        model,
        Instructions: "You are a friendly chatbot. Remember context from previous messages.",
    })

    fmt.Println("Chat Assistant (type 'quit' to exit)")
    fmt.Println("=====================================")

    scanner := bufio.NewScanner(os.Stdin)
    for {
        fmt.Print("\nYou: ")
        if !scanner.Scan() {
            break
        }

        input := strings.TrimSpace(scanner.Text())
        if input == "quit" || input == "exit" {
            fmt.Println("Goodbye!")
            break
        }

        if input == "" {
            continue
        }

        // Run agent (memory is automatically maintained)
        output, err := ag.Run(context.Background(), input)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }

        fmt.Printf("Agent: %s\n", output.Content)
    }
}
```

**Example conversation:**

```
You: My name is Alice
Agent: Nice to meet you, Alice! How can I help you today?

You: What's my name?
Agent: Your name is Alice!

You: quit
Goodbye!
```

## Using AgentOS (HTTP Server)

### 1. Start the Server

#### Using Docker Compose (Recommended)

```bash
# Copy environment template
cp .env.example .env

# Edit .env and add your API key
nano .env  # Add: OPENAI_API_KEY=sk-your-key

# Start server
docker-compose up -d

# Check health
curl http://localhost:8080/health
```

#### Using Go (Native)

```bash
# Build server
go build -o agentos cmd/server/main.go

# Run server
export OPENAI_API_KEY=sk-your-key
./agentos
```

### 2. Use the API

#### Health Check

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "agentos",
  "time": 1704067200
}
```

#### Run Agent

```bash
curl -X POST http://localhost:8080/api/v1/agents/assistant/run \
  -H "Content-Type: application/json" \
  -d '{
    "input": "What is 2+2?"
  }'
```

**Response:**
```json
{
  "content": "2 + 2 equals 4.",
  "metadata": {
    "agent_id": "assistant"
  }
}
```

See the [AgentOS API Reference](/api/agentos) for complete API documentation.

## Next Steps

### Learn More

- [Core Concepts](/guide/agent) - Understand Agent, Team, Workflow
- [Tools Guide](/guide/tools) - Learn about built-in and custom tools
- [Models Guide](/guide/models) - Multi-model support
- [Advanced Topics](/advanced/) - Architecture, performance, deployment

### Try Examples

All examples are in the `cmd/examples/` directory:

```bash
# Simple agent with calculator
go run cmd/examples/simple_agent/main.go

# Anthropic Claude
go run cmd/examples/claude_agent/main.go

# Local models with Ollama
go run cmd/examples/ollama_agent/main.go

# Multi-agent team
go run cmd/examples/team_demo/main.go

# Workflow engine
go run cmd/examples/workflow_demo/main.go

# RAG with ChromaDB
go run cmd/examples/rag_demo/main.go
```

See [Examples](/examples/) for detailed documentation on each example.

## Troubleshooting

### Common Issues

**1. "OPENAI_API_KEY not set"**

```bash
export OPENAI_API_KEY=sk-your-key-here
```

**2. "Module not found"**

```bash
go mod download
go mod tidy
```

**3. "Port 8080 already in use"**

Change port in `.env` or configuration:
```bash
AGENTOS_ADDRESS=:9090
```

**4. "Context deadline exceeded"**

Increase timeout:
```bash
export REQUEST_TIMEOUT=60
```

### Getting Debug Logs

```bash
export AGENTOS_DEBUG=true
export LOG_LEVEL=debug
```

## Quick Reference

### Common Imports

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
    "github.com/rexleimo/HNO/pkg/hno/tools/calculator"
    "github.com/rexleimo/HNO/pkg/hno/team"
    "github.com/rexleimo/HNO/pkg/hno/workflow"
    "github.com/rexleimo/HNO/pkg/agentos"
)
```

### Agent Creation Template

```go
model, err := openai.New("gpt-4o-mini", openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

ag, err := agent.New(agent.Config{
    Name:         "Agent Name",
    Model:        model,
    Toolkits:     []toolkit.Toolkit{/* tools */},
    Instructions: "System instructions",
    MaxLoops:     10,
})

output, err := ag.Run(context.Background(), "input")
```

## Next: Core Concepts

Learn about the three core abstractions:

- [Agent](/guide/agent) - Autonomous AI agents
- [Team](/guide/team) - Multi-agent collaboration
- [Workflow](/guide/workflow) - Step-based orchestration
