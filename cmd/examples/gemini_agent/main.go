package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/gemini"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	// Create Gemini model
	model, err := gemini.New("gemini-pro", gemini.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   2048,
	})
	if err != nil {
		log.Fatalf("Failed to create Gemini model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent with Gemini
	ag, err := agent.New(agent.Config{
		Name:         "GeminiAssistant",
		Model:        model,
		Instructions: "You are a helpful AI assistant powered by Google Gemini. You can perform calculations and answer questions.",
		Toolkits:     []toolkit.Toolkit{calc},
		MaxLoops:     5,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	fmt.Println("Gemini Agent initialized successfully!")
	fmt.Println("Model:", model.GetName())
	fmt.Println("Provider:", model.GetProvider())
	fmt.Println()

	ctx := context.Background()

	// Example 1: Simple question
	fmt.Println("=== Example 1: Simple Question ===")
	output, err := ag.Run(ctx, "What is the capital of France?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Q: What is the capital of France?\n")
	fmt.Printf("A: %s\n\n", output.Content)

	// Example 2: Math calculation
	fmt.Println("=== Example 2: Math Calculation ===")
	output, err = ag.Run(ctx, "What is 156 multiplied by 47?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Q: What is 156 multiplied by 47?\n")
	fmt.Printf("A: %s\n\n", output.Content)

	// Example 3: Multi-step reasoning
	fmt.Println("=== Example 3: Multi-step Reasoning ===")
	output, err = ag.Run(ctx, "If I have 100 dollars and spend 35% on food, then save 40% of what's left, how much do I have remaining?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("Q: If I have 100 dollars and spend 35%% on food, then save 40%% of what's left, how much do I have remaining?\n")
	fmt.Printf("A: %s\n\n", output.Content)

	// Display metadata
	fmt.Println("=== Run Metadata ===")
	fmt.Printf("Loops: %v\n", output.Metadata["loops"])
	fmt.Printf("Usage: %+v\n", output.Metadata["usage"])
}
