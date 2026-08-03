# Claude Agent Example

## Overview

This example demonstrates how to use Anthropic's Claude model with HNO. Claude is known for its thoughtful, detailed responses and strong reasoning capabilities. This example shows multiple use cases including simple conversation, calculator tool usage, complex calculations, and mathematical reasoning.

## What You'll Learn

- How to integrate Anthropic Claude with HNO
- How to configure Claude models (Opus, Sonnet, Haiku)
- How to use Claude with tool-calling capabilities
- Best practices for Claude instructions

## Prerequisites

- Go 1.21 or higher
- Anthropic API key (get one at [console.anthropic.com](https://console.anthropic.com))

## Setup

1. Set your Anthropic API key:
```bash
export ANTHROPIC_API_KEY=sk-ant-your-api-key-here
```

2. Navigate to the example directory:
```bash
cd cmd/examples/claude_agent
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
	"github.com/rexleimo/agno-go/pkg/hno/models/anthropic"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Create Anthropic Claude model
	model, err := anthropic.New("claude-3-opus-20240229", anthropic.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent with Claude
	ag, err := agent.New(agent.Config{
		Name:         "Claude Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are Claude, a helpful AI assistant created by Anthropic. Use the calculator tools to help users with mathematical calculations. Be precise and explain your reasoning.",
		MaxLoops:     10,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Example 1: Simple conversation
	fmt.Println("=== Example 1: Simple Conversation ===")
	ctx := context.Background()
	output, err := ag.Run(ctx, "Introduce yourself in one sentence.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 2: Using calculator tools
	fmt.Println("=== Example 2: Calculator Tool Usage ===")
	output, err = ag.Run(ctx, "What is 156 multiplied by 23, then subtract 100?")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 3: Complex calculation
	fmt.Println("=== Example 3: Complex Calculation ===")
	output, err = ag.Run(ctx, "Calculate the following: (45 + 67) * 3 - 89. Show your work step by step.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 4: Mathematical reasoning
	fmt.Println("=== Example 4: Mathematical Reasoning ===")
	output, err = ag.Run(ctx, "If I have $500 and spend $123, then earn $250, how much money do I have? Use the calculator to verify.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	fmt.Println("✅ All examples completed successfully!")
}
```

## Code Explanation

### 1. Claude Model Configuration

```go
model, err := anthropic.New("claude-3-opus-20240229", anthropic.Config{
	APIKey:      apiKey,
	Temperature: 0.7,
	MaxTokens:   2000,
})
```

**Available Claude Models:**
- `claude-3-opus-20240229` - Most capable, best for complex tasks
- `claude-3-sonnet-20240229` - Balanced performance and speed
- `claude-3-haiku-20240307` - Fastest, best for simple tasks

**Configuration Options:**
- `Temperature: 0.7` - Balanced creativity (0.0 = deterministic, 1.0 = creative)
- `MaxTokens: 2000` - Maximum response length

### 2. Claude-Specific Instructions

```go
Instructions: "You are Claude, a helpful AI assistant created by Anthropic.
Use the calculator tools to help users with mathematical calculations.
Be precise and explain your reasoning."
```

Claude responds well to:
- Clear identity and purpose
- Explicit instructions about tool usage
- Emphasis on reasoning and explanation

### 3. Example Scenarios

#### Example 1: Simple Conversation
Tests basic conversational ability without tools.

#### Example 2: Calculator Tool Usage
```
Query: "What is 156 multiplied by 23, then subtract 100?"
Expected Flow:
1. multiply(156, 23) → 3588
2. subtract(3588, 100) → 3488
```

#### Example 3: Complex Calculation
```
Query: "Calculate: (45 + 67) * 3 - 89. Show your work step by step."
Expected Flow:
1. add(45, 67) → 112
2. multiply(112, 3) → 336
3. subtract(336, 89) → 247
Claude also explains each step
```

#### Example 4: Mathematical Reasoning
Tests Claude's ability to:
- Break down word problems
- Choose appropriate tools
- Provide clear explanations

## Running the Example

```bash
# Option 1: Run directly
go run main.go

# Option 2: Build and run
go build -o claude_agent
./claude_agent
```

## Expected Output

```
=== Example 1: Simple Conversation ===
Agent: I'm Claude, an AI assistant created by Anthropic to be helpful, harmless, and honest.

=== Example 2: Calculator Tool Usage ===
Agent: Let me calculate that for you. First, 156 multiplied by 23 equals 3,588. Then, subtracting 100 from 3,588 gives us 3,488.

=== Example 3: Complex Calculation ===
Agent: I'll solve this step by step:
1. First, calculate the parentheses: 45 + 67 = 112
2. Then multiply: 112 * 3 = 336
3. Finally subtract: 336 - 89 = 247

The final answer is 247.

=== Example 4: Mathematical Reasoning ===
Agent: Let me help you track your money:
- Starting amount: $500
- After spending $123: $500 - $123 = $377
- After earning $250: $377 + $250 = $627

You have $627 in total.

✅ All examples completed successfully!
```

## Claude vs OpenAI

### When to Use Claude

**Best for:**
- Complex reasoning tasks
- Detailed explanations
- Safety-critical applications
- Thoughtful, nuanced responses

**Characteristics:**
- More verbose and explanatory
- Strong ethical reasoning
- Excellent at following complex instructions
- Better at admitting uncertainty

### When to Use OpenAI

**Best for:**
- Fast responses
- Code generation
- Creative writing
- Function calling at scale

## Model Selection Guide

| Model | Speed | Capability | Cost | Use Case |
|-------|-------|------------|------|----------|
| Claude 3 Opus | Slow | Highest | High | Complex analysis, research |
| Claude 3 Sonnet | Medium | High | Medium | General purpose, balanced |
| Claude 3 Haiku | Fast | Good | Low | Simple tasks, high volume |

## Configuration Tips

### For Deterministic Output
```go
anthropic.Config{
	Temperature: 0.0,
	MaxTokens:   1000,
}
```

### For Creative Tasks
```go
anthropic.Config{
	Temperature: 1.0,
	MaxTokens:   3000,
}
```

### For Production (Balanced)
```go
anthropic.Config{
	Temperature: 0.7,
	MaxTokens:   2000,
}
```

## Best Practices

1. **Clear Instructions**: Claude responds well to detailed, structured prompts
2. **Reasoning Requests**: Ask Claude to "explain" or "show work" for better results
3. **Safety**: Claude is more cautious - frame sensitive queries appropriately
4. **Context**: Claude has a 200K token context window - use it for long documents

## Next Steps

- Compare with [OpenAI Simple Agent](./simple-agent.md)
- Try [Ollama for Local Models](./ollama-agent.md)
- Build [Multi-Agent Teams](./team-demo.md)
- Explore [RAG with Claude](./rag-demo.md)

## Troubleshooting

**Error: "ANTHROPIC_API_KEY environment variable is required"**
- Set your API key: `export ANTHROPIC_API_KEY=sk-ant-...`

**Error: "model not found"**
- Verify model name matches exactly: `claude-3-opus-20240229`
- Check your API tier has access to the model

**Slow responses with Opus**
- Consider using Sonnet for faster responses
- Reduce MaxTokens if you don't need long outputs

**Rate limit errors**
- Anthropic has different rate limits per tier
- Implement retry logic with exponential backoff
- Consider using Haiku for high-volume tasks
