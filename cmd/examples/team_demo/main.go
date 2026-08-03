package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/team"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	ctx := context.Background()

	// Demo 1: Sequential Team
	fmt.Println("=== Demo 1: Sequential Team ===")
	runSequentialDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 2: Parallel Team ===")
	runParallelDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 3: Leader-Follower Team ===")
	runLeaderFollowerDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 4: Consensus Team ===")
	runConsensusDemo(ctx, apiKey)
}

func runSequentialDemo(ctx context.Context, apiKey string) {
	// Create model
	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create 3 agents for sequential processing
	researcher, _ := agent.New(agent.Config{
		Name:         "Researcher",
		Model:        model,
		Instructions: "You are a research expert. Analyze the topic and provide key facts.",
	})

	analyst, _ := agent.New(agent.Config{
		Name:         "Analyst",
		Model:        model,
		Instructions: "You are an analyst. Take research findings and extract insights.",
	})

	writer, _ := agent.New(agent.Config{
		Name:         "Writer",
		Model:        model,
		Instructions: "You are a writer. Take insights and write a concise summary.",
	})

	// Create sequential team
	t, err := team.New(team.Config{
		Name:   "Content Pipeline",
		Agents: []*agent.Agent{researcher, analyst, writer},
		Mode:   team.ModeSequential,
	})
	if err != nil {
		log.Fatalf("Failed to create team: %v", err)
	}

	// Run team
	output, err := t.Run(ctx, "Analyze the benefits of AI in healthcare")
	if err != nil {
		log.Fatalf("Team run failed: %v", err)
	}

	fmt.Printf("Final Output: %s\n", output.Content)
	fmt.Printf("Agents involved: %d\n", len(output.AgentOutputs))
}

func runParallelDemo(ctx context.Context, apiKey string) {
	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create agents with different specializations
	techAgent, _ := agent.New(agent.Config{
		Name:         "Tech Specialist",
		Model:        model,
		Instructions: "You are a technology expert. Focus on technical aspects.",
	})

	bizAgent, _ := agent.New(agent.Config{
		Name:         "Business Specialist",
		Model:        model,
		Instructions: "You are a business expert. Focus on business implications.",
	})

	ethicsAgent, _ := agent.New(agent.Config{
		Name:         "Ethics Specialist",
		Model:        model,
		Instructions: "You are an ethics expert. Focus on ethical considerations.",
	})

	// Create parallel team
	t, err := team.New(team.Config{
		Name:   "Multi-Perspective Analysis",
		Agents: []*agent.Agent{techAgent, bizAgent, ethicsAgent},
		Mode:   team.ModeParallel,
	})
	if err != nil {
		log.Fatalf("Failed to create team: %v", err)
	}

	output, err := t.Run(ctx, "Evaluate the impact of autonomous vehicles")
	if err != nil {
		log.Fatalf("Team run failed: %v", err)
	}

	fmt.Printf("Combined Analysis:\n%s\n", output.Content)
}

func runLeaderFollowerDemo(ctx context.Context, apiKey string) {
	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create leader
	leader, _ := agent.New(agent.Config{
		Name:         "Team Leader",
		Model:        model,
		Instructions: "You are a team leader. Delegate tasks and synthesize results.",
	})

	// Create followers with tools
	calcAgent, _ := agent.New(agent.Config{
		Name:         "Calculator Agent",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calculator.New()},
		Instructions: "You perform calculations as requested.",
	})

	dataAgent, _ := agent.New(agent.Config{
		Name:         "Data Agent",
		Model:        model,
		Instructions: "You analyze and present data.",
	})

	// Create leader-follower team
	t, err := team.New(team.Config{
		Name:   "Project Team",
		Leader: leader,
		Agents: []*agent.Agent{calcAgent, dataAgent},
		Mode:   team.ModeLeaderFollower,
	})
	if err != nil {
		log.Fatalf("Failed to create team: %v", err)
	}

	output, err := t.Run(ctx, "Calculate the ROI for a $100,000 investment with 15% annual return over 5 years")
	if err != nil {
		log.Fatalf("Team run failed: %v", err)
	}

	fmt.Printf("Leader's Final Report: %s\n", output.Content)
}

func runConsensusDemo(ctx context.Context, apiKey string) {
	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create agents with different perspectives
	optimist, _ := agent.New(agent.Config{
		Name:         "Optimist",
		Model:        model,
		Instructions: "You are optimistic and focus on opportunities.",
	})

	realist, _ := agent.New(agent.Config{
		Name:         "Realist",
		Model:        model,
		Instructions: "You are realistic and balanced in your views.",
	})

	critic, _ := agent.New(agent.Config{
		Name:         "Critic",
		Model:        model,
		Instructions: "You are critical and focus on potential problems.",
	})

	// Create consensus team
	t, err := team.New(team.Config{
		Name:      "Decision Team",
		Agents:    []*agent.Agent{optimist, realist, critic},
		Mode:      team.ModeConsensus,
		MaxRounds: 2,
	})
	if err != nil {
		log.Fatalf("Failed to create team: %v", err)
	}

	output, err := t.Run(ctx, "Should we invest in renewable energy for our company?")
	if err != nil {
		log.Fatalf("Team run failed: %v", err)
	}

	fmt.Printf("Consensus Result: %s\n", output.Content)
	fmt.Printf("Total discussion rounds: %v\n", output.Metadata["rounds"])
}
