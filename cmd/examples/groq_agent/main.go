package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/groq"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	// 从环境变量获取 API key
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		log.Fatal("GROQ_API_KEY environment variable is required\n" +
			"Get your API key from: https://console.groq.com/keys")
	}

	// Create Groq model with LLaMA 3.1 8B (ultra-fast inference)
	// 使用 LLaMA 3.1 8B 创建 Groq 模型 (超快速推理)
	model, err := groq.New(groq.ModelLlama38B, groq.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   1024,
	})
	if err != nil {
		log.Fatalf("Failed to create Groq model: %v", err)
	}

	// Print model information
	// 打印模型信息
	if info, found := groq.GetModelInfo(groq.ModelLlama38B); found {
		fmt.Printf("Using: %s (%s)\n", info.Name, info.Developer)
		fmt.Printf("Context Window: %d tokens\n", info.ContextWindow)
		fmt.Printf("Description: %s\n\n", info.Description)
	}

	// Create calculator toolkit
	// 创建计算器工具集
	calc := calculator.New()

	// Create agent with Groq model
	// 使用 Groq 模型创建 agent
	groqAgent, err := agent.New(agent.Config{
		Name:         "Groq Calculator Agent",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are a helpful AI assistant powered by Groq's ultra-fast LLaMA 3.1 inference. You can perform calculations using your calculator tools and provide quick, accurate responses.",
		MaxLoops:     5,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Example 1: Simple conversation
	// 示例 1: 简单对话
	fmt.Println("=== Example 1: Simple Conversation ===")
	fmt.Println("Question: What makes Groq's inference so fast?")
	fmt.Println()

	output, err := groqAgent.Run(context.Background(), "What makes Groq's inference so fast? Give a brief answer.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}

	fmt.Printf("Assistant: %s\n\n", output.Content)

	// Example 2: Using calculator tools
	// 示例 2: 使用计算器工具
	fmt.Println("=== Example 2: Calculator Tools ===")
	fmt.Println("Question: Calculate (1234 + 5678) * 9")
	fmt.Println()

	output, err = groqAgent.Run(context.Background(), "Calculate (1234 + 5678) * 9")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}

	fmt.Printf("Assistant: %s\n\n", output.Content)

	// Example 3: Multi-step calculation
	// 示例 3: 多步骤计算
	fmt.Println("=== Example 3: Multi-step Calculation ===")
	fmt.Println("Question: If I have 15 apples and give away 3 each to 4 friends, how many apples do I have left?")
	fmt.Println()

	output, err = groqAgent.Run(context.Background(), "If I have 15 apples and give away 3 each to 4 friends, how many apples do I have left? Calculate step by step.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}

	fmt.Printf("Assistant: %s\n\n", output.Content)

	// Example 4: Complex calculation
	// 示例 4: 复杂计算
	fmt.Println("=== Example 4: Complex Calculation ===")
	fmt.Println("Question: What is the result of ((100 / 4) + 25) * 3 - 10?")
	fmt.Println()

	output, err = groqAgent.Run(context.Background(), "What is the result of ((100 / 4) + 25) * 3 - 10? Show me the calculation steps.")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}

	fmt.Printf("Assistant: %s\n\n", output.Content)

	fmt.Println("=== All examples completed successfully! ===")
	fmt.Println("\nNote: Groq provides ultra-fast inference with speeds up to 10x faster than traditional cloud LLM providers!")
}
