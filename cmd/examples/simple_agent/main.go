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
