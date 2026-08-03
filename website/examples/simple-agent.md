# Simple Agent Example

## Overview

This example demonstrates the basic usage of Agno-Go to create a simple AI agent with tool-calling capabilities. The agent uses OpenAI's GPT-4o-mini model and is equipped with a calculator toolkit to perform mathematical operations.

## What You'll Learn

- How to create and configure an OpenAI model
- How to set up an agent with tools
- How to run an agent with a user query
- How to access execution metadata (loops, token usage)

## Prerequisites

- Go 1.21 or higher
- OpenAI API key

## Setup

1. Set your OpenAI API key:
```bash
export OPENAI_API_KEY=sk-your-api-key-here
```

2. Navigate to the example directory:
```bash
cd cmd/examples/simple_agent
```

## Complete Code

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create OpenAI model
	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   1000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent
	ag, err := agent.New(agent.Config{
		Name:         "Math Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are a helpful math assistant. Use the calculator tools to help users with mathematical calculations.",
		MaxLoops:     10,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Run agent
	ctx := context.Background()
	output, err := ag.Run(ctx, "What is 25 multiplied by 4, then add 15?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}

	// Print result
	fmt.Println("Agent Response:")
	fmt.Println(output.Content)
	fmt.Println("\nMetadata:")
	fmt.Printf("Loops: %v\n", output.Metadata["loops"])
	fmt.Printf("Usage: %+v\n", output.Metadata["usage"])
}
```

## Code Explanation

### 1. Model Configuration

```go
model, err := openai.New("gpt-4o-mini", openai.Config{
	APIKey:      apiKey,
	Temperature: 0.7,
	MaxTokens:   1000,
})
```

- Creates an OpenAI model instance using GPT-4o-mini
- `Temperature: 0.7` provides balanced creativity and consistency
- `MaxTokens: 1000` limits response length

### 2. Calculator Toolkit

```go
calc := calculator.New()
```

The calculator toolkit provides four functions:
- `add` - Addition of two numbers
- `subtract` - Subtraction of two numbers
- `multiply` - Multiplication of two numbers
- `divide` - Division of two numbers

### 3. Agent Configuration

```go
ag, err := agent.New(agent.Config{
	Name:         "Math Assistant",
	Model:        model,
	Toolkits:     []toolkit.Toolkit{calc},
	Instructions: "You are a helpful math assistant...",
	MaxLoops:     10,
})
```

- `Name` - Identifies the agent
- `Model` - The LLM to use for reasoning
- `Toolkits` - Array of tool collections available to the agent
- `Instructions` - System prompt that defines agent behavior
- `MaxLoops` - Maximum number of tool-calling iterations (prevents infinite loops)

### 4. Running the Agent

```go
output, err := ag.Run(ctx, "What is 25 multiplied by 4, then add 15?")
```

The agent will:
1. Analyze the user query
2. Determine it needs to use calculator tools
3. Call `multiply(25, 4)` to get 100
4. Call `add(100, 15)` to get 115
5. Return a natural language response

## Running the Example

```bash
# Option 1: Run directly
go run main.go

# Option 2: Build and run
go build -o simple_agent
./simple_agent
```

## Expected Output

```
Agent Response:
The result of 25 multiplied by 4 is 100, and when you add 15 to that, you get 115.

Metadata:
Loops: 2
Usage: map[completion_tokens:45 prompt_tokens:234 total_tokens:279]
```

## Key Concepts

### Tool Calling Loop

The `MaxLoops` parameter controls how many times the agent can call tools:

1. **Loop 1**: Agent calls `multiply(25, 4)` → receives result: 100
2. **Loop 2**: Agent calls `add(100, 15)` → receives result: 115
3. **Final**: Agent generates natural language response

Each loop represents one round of tool invocation and result processing.

### Metadata

The `output.Metadata` contains useful execution information:
- `loops` - Number of tool-calling iterations performed
- `usage` - Token consumption (prompt, completion, total)

## Next Steps

- Explore [Claude Agent Example](./claude-agent.md) for Anthropic integration
- Learn about [Team Collaboration](./team-demo.md) with multiple agents
- Try [Workflow Engine](./workflow-demo.md) for complex processes
- Build [RAG Applications](./rag-demo.md) with knowledge retrieval

## Troubleshooting

**Error: "OPENAI_API_KEY environment variable is required"**
- Make sure you've exported the API key: `export OPENAI_API_KEY=sk-...`

**Error: "model not found"**
- Check that you have access to the GPT-4o-mini model
- Try using "gpt-3.5-turbo" as an alternative

**Error: "max loops exceeded"**
- The agent hit the MaxLoops limit (10)
- Increase `MaxLoops` or simplify the query
