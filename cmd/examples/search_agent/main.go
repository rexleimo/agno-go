package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/tools/search"
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

	// Create search toolkit
	searchTool := search.New(search.Config{
		MaxResults: 3,
	})

	// Create agent with search capability
	ag, err := agent.New(agent.Config{
		Name:     "Search Assistant",
		Model:    model,
		Toolkits: []toolkit.Toolkit{searchTool},
		Instructions: `You are a helpful search assistant. When asked a question, use the search tool to find relevant information on the web.
Always provide the sources (URLs) for your information.
Summarize the key findings from the search results.`,
		MaxLoops: 5,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Example searches
	queries := []string{
		"What are the latest developments in Go programming language?",
		"What is the difference between goroutines and threads?",
		"How to implement AI agents in Go?",
	}

	fmt.Println("🔍 Search Agent Demo")
	fmt.Println("=" + repeat("=", 60))
	fmt.Println()

	for i, query := range queries {
		fmt.Printf("Query %d: %s\n", i+1, query)
		fmt.Println(repeat("-", 60))

		output, err := ag.Run(context.Background(), query)
		if err != nil {
			log.Printf("Error running agent: %v", err)
			continue
		}

		fmt.Println("Answer:")
		fmt.Println(output.Content)
		fmt.Println()

		// Display metrics
		fmt.Printf("📊 Metrics:\n")
		fmt.Printf("   - Messages exchanged: %d\n", len(output.Messages))
		fmt.Println()
		fmt.Println(repeat("=", 60))
		fmt.Println()
	}

	// Interactive mode
	fmt.Println("💬 Interactive Search Mode (type 'exit' to quit)")
	fmt.Println(repeat("=", 60))
	fmt.Println()

	for {
		fmt.Print("Your question: ")
		var question string
		fmt.Scanln(&question)

		if question == "exit" || question == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		if question == "" {
			continue
		}

		output, err := ag.Run(context.Background(), question)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		fmt.Println("\nAnswer:")
		fmt.Println(output.Content)
		fmt.Printf("\n(Messages: %d)\n", len(output.Messages))
		fmt.Println()
	}
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
