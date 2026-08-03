package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/deepseek"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		log.Fatal("DEEPSEEK_API_KEY environment variable is required")
	}

	// Create DeepSeek model (deepseek-chat for general tasks)
	model, err := deepseek.New("deepseek-chat", deepseek.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		log.Fatalf("Failed to create DeepSeek model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent with DeepSeek
	ag, err := agent.New(agent.Config{
		Name:         "DeepSeekAssistant",
		Model:        model,
		Instructions: "You are a helpful AI assistant powered by DeepSeek. You can perform calculations and answer questions.",
		Toolkits:     []toolkit.Toolkit{calc},
		MaxLoops:     5,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	fmt.Println("DeepSeek Agent initialized successfully!")
	fmt.Println("Model:", model.GetName())
	fmt.Println("Provider:", model.GetProvider())
	fmt.Println()

	ctx := context.Background()

	// Example 1: Simple question
	fmt.Println("=== Example 1: Simple Question ===")
	output, err := ag.Run(ctx, "What is the capital of Japan?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Q: What is the capital of Japan?\n")
	fmt.Printf("A: %s\n\n", output.Content)

	// Example 2: Math calculation
	fmt.Println("=== Example 2: Math Calculation ===")
	output, err = ag.Run(ctx, "What is 234 multiplied by 567?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Q: What is 234 multiplied by 567?\n")
	fmt.Printf("A: %s\n\n", output.Content)

	// Example 3: Multi-step reasoning
	fmt.Println("=== Example 3: Multi-step Reasoning ===")
	output, err = ag.Run(ctx, "If I have 500 dollars and spend 25% on rent, then save 30% of what's left, how much do I have remaining?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Q: If I have 500 dollars and spend 25%% on rent, then save 30%% of what's left, how much do I have remaining?\n")
	fmt.Printf("A: %s\n\n", output.Content)

	// Display metadata
	fmt.Println("=== Run Metadata ===")
	fmt.Printf("Loops: %v\n", output.Metadata["loops"])
	fmt.Printf("Usage: %+v\n", output.Metadata["usage"])
}
