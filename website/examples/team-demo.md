# Team Collaboration Example

## Overview

This example demonstrates Agno-Go's multi-agent team collaboration capabilities. Teams allow multiple agents to work together using different coordination modes: Sequential, Parallel, Leader-Follower, and Consensus. Each mode is suited for different types of tasks and collaboration patterns.

## What You'll Learn

- How to create multi-agent teams
- Four team coordination modes and when to use each
- How agents share context and build on each other's work
- How to access individual agent outputs

## Prerequisites

- Go 1.21 or higher
- OpenAI API key

## Setup

```bash
export OPENAI_API_KEY=sk-your-api-key-here
cd cmd/examples/team_demo
```

## Complete Code

The full example includes 4 demos - see the code explanation below for details.

```go
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
```

## Team Coordination Modes

### 1. Sequential Mode

Agents work one after another, each building on the previous agent's output.

```go
func runSequentialDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

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
	t, _ := team.New(team.Config{
		Name:   "Content Pipeline",
		Agents: []*agent.Agent{researcher, analyst, writer},
		Mode:   team.ModeSequential,
	})

	// Run team
	output, _ := t.Run(ctx, "Analyze the benefits of AI in healthcare")

	fmt.Printf("Final Output: %s\n", output.Content)
	fmt.Printf("Agents involved: %d\n", len(output.AgentOutputs))
}
```

**Flow:**
1. **Researcher** analyzes topic → produces research findings
2. **Analyst** receives findings → extracts insights
3. **Writer** receives insights → writes final summary

**Use Cases:**
- Content creation pipelines
- Data processing workflows
- Multi-stage analysis tasks

### 2. Parallel Mode

All agents work simultaneously on the same input, combining their outputs.

```go
func runParallelDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

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
	t, _ := team.New(team.Config{
		Name:   "Multi-Perspective Analysis",
		Agents: []*agent.Agent{techAgent, bizAgent, ethicsAgent},
		Mode:   team.ModeParallel,
	})

	output, _ := t.Run(ctx, "Evaluate the impact of autonomous vehicles")
	fmt.Printf("Combined Analysis:\n%s\n", output.Content)
}
```

**Flow:**
1. All agents receive the same input simultaneously
2. Each agent provides their perspective
3. Outputs are combined into comprehensive analysis

**Use Cases:**
- Multi-perspective analysis
- Brainstorming sessions
- Independent evaluations
- Parallel data processing

### 3. Leader-Follower Mode

A leader agent delegates tasks to follower agents and synthesizes results.

```go
func runLeaderFollowerDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

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
	t, _ := team.New(team.Config{
		Name:   "Project Team",
		Leader: leader,
		Agents: []*agent.Agent{calcAgent, dataAgent},
		Mode:   team.ModeLeaderFollower,
	})

	output, _ := t.Run(ctx, "Calculate the ROI for a $100,000 investment with 15% annual return over 5 years")
	fmt.Printf("Leader's Final Report: %s\n", output.Content)
}
```

**Flow:**
1. **Leader** analyzes task and delegates to followers
2. **Followers** execute assigned subtasks
3. **Leader** synthesizes results and provides final output

**Use Cases:**
- Complex task decomposition
- Hierarchical workflows
- Project management scenarios
- Specialized tool usage

### 4. Consensus Mode

Agents discuss until reaching agreement or max rounds.

```go
func runConsensusDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

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
	t, _ := team.New(team.Config{
		Name:      "Decision Team",
		Agents:    []*agent.Agent{optimist, realist, critic},
		Mode:      team.ModeConsensus,
		MaxRounds: 2,
	})

	output, _ := t.Run(ctx, "Should we invest in renewable energy for our company?")

	fmt.Printf("Consensus Result: %s\n", output.Content)
	fmt.Printf("Total discussion rounds: %v\n", output.Metadata["rounds"])
}
```

**Flow:**
1. **Round 1**: Each agent provides initial perspective
2. **Round 2**: Agents see others' views and refine position
3. **Final**: System synthesizes consensus or final positions

**Use Cases:**
- Decision making
- Debate simulation
- Multi-viewpoint analysis
- Risk assessment

## Team Configuration

### Basic Configuration

```go
team.Config{
	Name:   "My Team",           // Team identifier
	Agents: []*agent.Agent{...}, // Team members
	Mode:   team.ModeSequential, // Coordination mode
}
```

### Advanced Configuration

```go
team.Config{
	Name:      "Decision Team",
	Leader:    leaderAgent,      // For Leader-Follower mode
	Agents:    followerAgents,   // Team members
	Mode:      team.ModeConsensus,
	MaxRounds: 3,                // For Consensus mode
}
```

## Accessing Results

### Team Output

```go
output, err := t.Run(ctx, "Your query here")

// Final result
fmt.Println(output.Content)

// Individual agent outputs
for _, agentOut := range output.AgentOutputs {
	fmt.Printf("%s: %s\n", agentOut.AgentName, agentOut.Content)
}

// Metadata
fmt.Printf("Rounds: %v\n", output.Metadata["rounds"])
```

### Individual Agent Outputs

```go
// Access specific agent's contribution
if len(output.AgentOutputs) > 0 {
	firstAgent := output.AgentOutputs[0]
	fmt.Printf("Agent: %s\n", firstAgent.AgentName)
	fmt.Printf("Output: %s\n", firstAgent.Content)
}
```

## Running the Example

```bash
go run main.go
```

## Expected Output

```
=== Demo 1: Sequential Team ===
Final Output: AI in healthcare offers significant benefits including improved diagnostic accuracy through machine learning, personalized treatment plans, reduced administrative burden, and enhanced patient monitoring through IoT devices.
Agents involved: 3

=== Demo 2: Parallel Team ===
Combined Analysis:
Technical: Autonomous vehicles use advanced sensors, AI algorithms, and real-time processing...
Business: Market disruption, new revenue models, infrastructure investment needs...
Ethics: Privacy concerns, liability questions, job displacement, safety standards...

=== Demo 3: Leader-Follower Team ===
Leader's Final Report: Based on calculations, a $100,000 investment at 15% annual return over 5 years yields $201,136, representing a 101% ROI.

=== Demo 4: Consensus Team ===
Consensus Result: After thorough discussion, the team recommends investing in renewable energy with careful planning for upfront costs and long-term savings.
Total discussion rounds: 2
```

## Mode Comparison

| Mode | When to Use | Agent Count | Communication Pattern |
|------|-------------|-------------|----------------------|
| **Sequential** | Pipeline tasks, ordered steps | 2-10 | Linear: A → B → C |
| **Parallel** | Independent tasks, multiple views | 2-20 | Broadcast: All get same input |
| **Leader-Follower** | Complex delegation, hierarchy | 1 leader + 1-10 followers | Hub-spoke: Leader coordinates |
| **Consensus** | Decision making, debate | 2-5 | Round-robin discussion |

## Best Practices

### 1. Choose the Right Mode

```go
// Sequential: When order matters
team.ModeSequential  // Research → Analysis → Writing

// Parallel: When you need multiple perspectives
team.ModeParallel    // Tech + Business + Legal analysis

// Leader-Follower: When delegation is needed
team.ModeLeaderFollower  // Complex task breakdown

// Consensus: When agreement is needed
team.ModeConsensus   // Decision making, debate
```

### 2. Design Clear Agent Roles

```go
// ✅ Good: Specific, distinct roles
researcher := "You are a research expert. Focus on facts and data."
analyst := "You are an analyst. Extract insights from research."

// ❌ Bad: Overlapping, vague roles
agent1 := "You are helpful."
agent2 := "You are smart."
```

### 3. Optimize Agent Count

- **Sequential**: 2-5 agents (more = longer pipeline)
- **Parallel**: 2-10 agents (more = richer analysis)
- **Leader-Follower**: 1 leader + 2-5 followers
- **Consensus**: 2-4 agents (more = harder to converge)

### 4. Handle Errors

```go
output, err := team.Run(ctx, query)
if err != nil {
	log.Printf("Team execution failed: %v", err)
	// Fallback logic
}
```

## Advanced Patterns

### Mixed Tool Usage

```go
// Some agents have tools, others don't
calcAgent := agent.New(agent.Config{
	Toolkits: []toolkit.Toolkit{calculator.New()},
})

analysisAgent := agent.New(agent.Config{
	// No tools, pure reasoning
})
```

### Dynamic Team Composition

```go
var agents []*agent.Agent

if needsCalculation {
	agents = append(agents, calcAgent)
}
if needsWebSearch {
	agents = append(agents, searchAgent)
}

team, _ := team.New(team.Config{Agents: agents, Mode: team.ModeParallel})
```

### Nested Teams

```go
// Create sub-teams
researchTeam := team.New(team.Config{...})
analysisTeam := team.New(team.Config{...})

// Use outputs from one team as input to another
researchOutput, _ := researchTeam.Run(ctx, query)
finalOutput, _ := analysisTeam.Run(ctx, researchOutput.Content)
```

## Performance Considerations

### Sequential Mode
- **Latency**: Sum of all agent times (slowest)
- **Cost**: Sum of all agent costs
- **Best for**: When order is critical

### Parallel Mode
- **Latency**: Max of agent times (faster)
- **Cost**: Sum of all agent costs
- **Best for**: When speed matters

### Leader-Follower Mode
- **Latency**: Leader + followers (moderate)
- **Cost**: Leader + follower costs
- **Best for**: Complex task delegation

### Consensus Mode
- **Latency**: Rounds × agent time (slowest)
- **Cost**: Rounds × agent count
- **Best for**: When consensus is critical

## Next Steps

- Start with [Simple Agent](./simple-agent.md) basics
- Explore [Workflow Engine](./workflow-demo.md) for controlled execution
- Build [RAG Systems](./rag-demo.md) with team collaboration
- Try different [Model Providers](./claude-agent.md)

## Troubleshooting

**Agents not collaborating effectively:**
- Review agent instructions for clarity
- Check if the mode fits the task
- Ensure agents have distinct roles

**Sequential team too slow:**
- Reduce number of agents
- Use smaller/faster models
- Consider Parallel mode

**Consensus not converging:**
- Increase MaxRounds
- Simplify the decision
- Reduce agent count
- Adjust agent instructions

**Leader not delegating properly:**
- Clarify leader's delegation instructions
- Ensure followers have appropriate tools
- Check follower instructions are clear
