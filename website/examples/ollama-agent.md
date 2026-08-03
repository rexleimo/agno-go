# Ollama Agent Example

## Overview

This example demonstrates how to use local LLMs with HNO through Ollama. Ollama allows you to run powerful language models locally on your machine, providing privacy, cost savings, and offline capabilities. This is perfect for development, testing, and privacy-sensitive applications.

## What You'll Learn

- How to integrate Ollama with HNO
- How to run agents with local LLMs
- How to use tool-calling with local models
- Benefits and limitations of local models

## Prerequisites

- Go 1.21 or higher
- Ollama installed ([ollama.ai](https://ollama.ai))
- A local model pulled (e.g., llama2, mistral, codellama)

## Ollama Setup

### 1. Install Ollama

**macOS/Linux:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

**Windows:**
Download from [ollama.ai/download](https://ollama.ai/download)

### 2. Pull a Model

```bash
# Pull Llama 2 (7B parameters, ~4GB)
ollama pull llama2

# Or try other models:
ollama pull mistral      # Mistral 7B
ollama pull codellama    # Code-specialized
ollama pull llama2:13b   # Larger, more capable
```

### 3. Start Ollama Server

```bash
ollama serve
```

The server runs on `http://localhost:11434` by default.

### 4. Verify Installation

```bash
# Test the model
ollama run llama2 "Hello, how are you?"
```

## Complete Code

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/ollama"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Create Ollama model (uses local Ollama instance)
	// Make sure Ollama is running: ollama serve
	model, err := ollama.New("llama2", ollama.Config{
		BaseURL:     "http://localhost:11434",
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent with Ollama
	ag, err := agent.New(agent.Config{
		Name:         "Ollama Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are a helpful AI assistant running on Ollama. You can use calculator tools to help with math. Be concise and friendly.",
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
	output, err = ag.Run(ctx, "What is 456 multiplied by 789?")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 3: Complex calculation
	fmt.Println("=== Example 3: Complex Calculation ===")
	output, err = ag.Run(ctx, "Calculate: (100 + 50) * 2 - 75")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	fmt.Println("✅ All examples completed successfully!")
}
```

## Code Explanation

### 1. Ollama Model Configuration

```go
model, err := ollama.New("llama2", ollama.Config{
	BaseURL:     "http://localhost:11434",
	Temperature: 0.7,
	MaxTokens:   2000,
})
```

**Configuration Options:**
- **Model Name**: Must match a pulled model (e.g., "llama2", "mistral")
- **BaseURL**: Ollama server address (default: `http://localhost:11434`)
- **Temperature**: 0.0 (deterministic) to 2.0 (very creative)
- **MaxTokens**: Maximum response length

### 2. No API Key Required

Unlike OpenAI or Anthropic, Ollama runs locally:
- ✅ No API key needed
- ✅ No usage costs
- ✅ Complete privacy
- ✅ Works offline

### 3. Tool Support

Local models can use tools just like cloud models:
```go
Toolkits: []toolkit.Toolkit{calc}
```

The agent will call calculator functions when needed.

## Running the Example

### Step 1: Start Ollama
```bash
# Terminal 1
ollama serve
```

### Step 2: Run the Example
```bash
# Terminal 2
cd cmd/examples/ollama_agent
go run main.go
```

## Expected Output

```
=== Example 1: Simple Conversation ===
Agent: I'm a helpful AI assistant running on Ollama, here to assist you with questions and tasks.

=== Example 2: Calculator Tool Usage ===
Agent: Let me calculate that for you. 456 multiplied by 789 equals 359,784.

=== Example 3: Complex Calculation ===
Agent: Let me solve this step by step:
- First: 100 + 50 = 150
- Then: 150 * 2 = 300
- Finally: 300 - 75 = 225

The answer is 225.

✅ All examples completed successfully!
```

## Available Models

### General Purpose

| Model | Size | RAM | Description |
|-------|------|-----|-------------|
| llama2 | 7B | 8GB | Meta's Llama 2, general purpose |
| llama2:13b | 13B | 16GB | Larger, more capable version |
| mistral | 7B | 8GB | Mistral AI, excellent quality |
| mixtral | 47B | 32GB | Mixture of experts, very capable |

### Specialized

| Model | Use Case |
|-------|----------|
| codellama | Code generation and analysis |
| llama2-uncensored | Fewer content restrictions |
| orca-mini | Smaller, faster (3B) |
| vicuna | Conversation and chat |

### List Available Models
```bash
ollama list
```

### Pull a Specific Model
```bash
ollama pull mistral
ollama pull codellama:13b
```

## Configuration Examples

### For Speed (Small Model)
```go
ollama.Config{
	Model:       "orca-mini",
	Temperature: 0.5,
	MaxTokens:   500,
}
```

### For Quality (Large Model)
```go
ollama.Config{
	Model:       "mixtral",
	Temperature: 0.7,
	MaxTokens:   3000,
}
```

### For Code Tasks
```go
ollama.Config{
	Model:       "codellama",
	Temperature: 0.3,  // More deterministic for code
	MaxTokens:   2000,
}
```

### Custom Ollama Server
```go
ollama.Config{
	BaseURL:     "http://192.168.1.100:11434",  // Remote Ollama
	Model:       "llama2",
	Temperature: 0.7,
}
```

## Performance Considerations

### Speed Factors

1. **Model Size**: Smaller models (7B) are faster than larger ones (70B)
2. **Hardware**: GPU greatly accelerates inference
3. **Context Length**: Longer conversations slow down responses

### Typical Response Times

| Model | Hardware | Speed |
|-------|----------|-------|
| llama2 (7B) | Mac M1 | ~1-2 sec |
| mistral (7B) | Mac M1 | ~1-2 sec |
| mixtral (47B) | Mac M1 | ~5-10 sec |
| llama2 (13B) | NVIDIA 3090 | ~0.5-1 sec |

## Advantages of Local Models

### ✅ Benefits

1. **Privacy**: Data never leaves your machine
2. **Cost**: No API fees, unlimited usage
3. **Offline**: Works without internet
4. **Control**: Full control over model and data
5. **Customization**: Fine-tune models for specific tasks

### ⚠️ Limitations

1. **Quality**: Generally lower than GPT-4 or Claude Opus
2. **Speed**: Slower than cloud APIs (unless high-end GPU)
3. **Resources**: Requires RAM/VRAM (4-16GB+)
4. **Maintenance**: Need to manage models and updates

## Best Practices

### 1. Choose the Right Model

```bash
# For development/testing
ollama pull orca-mini  # Fast, 3B parameters

# For production
ollama pull mistral    # Good balance of speed/quality

# For complex tasks
ollama pull mixtral    # High quality, needs more resources
```

### 2. Optimize Instructions

Local models benefit from concise, clear instructions:

```go
// ✅ Good
Instructions: "You are a math assistant. Use calculator tools for calculations. Be concise."

// ❌ Too verbose
Instructions: "You are an extremely sophisticated mathematical assistant with deep knowledge..."
```

### 3. Monitor Resource Usage

```bash
# Check Ollama status
ollama ps

# View model info
ollama show llama2
```

### 4. Handle Errors Gracefully

```go
output, err := ag.Run(ctx, userQuery)
if err != nil {
	// Ollama might be down
	log.Printf("Ollama error: %v. Is the server running?", err)
	// Fallback to cloud model or return error
}
```

## Integration Patterns

### Hybrid Approach

Use Ollama for development, cloud for production:

```go
var model models.Model

if os.Getenv("ENV") == "production" {
	model, _ = openai.New("gpt-4o-mini", openai.Config{...})
} else {
	model, _ = ollama.New("llama2", ollama.Config{...})
}
```

### Privacy-First Applications

```go
// Use Ollama for sensitive data
sensitiveAgent, _ := agent.New(agent.Config{
	Model: ollamaModel,
	Instructions: "Handle user PII securely...",
})
```

## Troubleshooting

### Error: "connection refused"
```bash
# Check if Ollama is running
ollama serve

# Or check the process
ps aux | grep ollama
```

### Error: "model not found"
```bash
# Pull the model first
ollama pull llama2

# Verify it's available
ollama list
```

### Slow Responses
```bash
# Try a smaller model
ollama pull orca-mini

# Or check hardware acceleration
ollama show llama2 | grep -i gpu
```

### Out of Memory
```bash
# Use a smaller model
ollama pull orca-mini  # 3B instead of 7B

# Or increase swap space (Linux)
# Or close other applications
```

## Next Steps

- Compare with [OpenAI Agent](./simple-agent.md) and [Claude Agent](./claude-agent.md)
- Use local models in [Multi-Agent Teams](./team-demo.md)
- Build [Privacy-Preserving RAG](./rag-demo.md) with local embeddings
- Explore [Workflows](./workflow-demo.md) with local models

## Additional Resources

- [Ollama Documentation](https://github.com/ollama/ollama/blob/main/README.md)
- [Ollama Model Library](https://ollama.ai/library)
- [Hardware Requirements](https://github.com/ollama/ollama/blob/main/docs/gpu.md)
- [Model Comparison](https://ollama.ai/blog/model-comparison)
