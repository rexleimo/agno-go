package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/workflow"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	ctx := context.Background()

	fmt.Println("=== Demo 1: Sequential Workflow ===")
	runSequentialWorkflow(ctx, apiKey)

	fmt.Println("\n=== Demo 2: Conditional Workflow ===")
	runConditionalWorkflow(ctx, apiKey)

	fmt.Println("\n=== Demo 3: Loop Workflow ===")
	runLoopWorkflow(ctx, apiKey)

	fmt.Println("\n=== Demo 4: Parallel Workflow ===")
	runParallelWorkflow(ctx, apiKey)

	fmt.Println("\n=== Demo 5: Complex Workflow with Router ===")
	runComplexWorkflow(ctx, apiKey)
}

func createAgent(id string, apiKey string, instructions string, tools ...toolkit.Toolkit) *agent.Agent {
	model, _ := openai.New("gpt-4o-mini", openai.Config{
		APIKey: apiKey,
	})

	ag, _ := agent.New(agent.Config{
		ID:           id,
		Name:         id,
		Model:        model,
		Instructions: instructions,
		Toolkits:     tools,
	})

	return ag
}

func runSequentialWorkflow(ctx context.Context, apiKey string) {
	// Create agents for pipeline
	researcher := createAgent("researcher", apiKey, "You are a researcher. Gather facts about the topic.")
	analyzer := createAgent("analyzer", apiKey, "You are an analyst. Analyze the facts and draw conclusions.")
	writer := createAgent("writer", apiKey, "You are a writer. Write a concise summary.")

	// Create steps
	step1, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "research",
		Agent: researcher,
	})

	step2, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "analyze",
		Agent: analyzer,
	})

	step3, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "write",
		Agent: writer,
	})

	// Create workflow
	wf, _ := workflow.New(workflow.Config{
		Name:  "Content Pipeline",
		Steps: []workflow.Node{step1, step2, step3},
	})

	result, err := wf.Run(ctx, "The impact of renewable energy on climate change", "")
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}

	fmt.Printf("Final Output: %s\n", result.Output)
}

func runConditionalWorkflow(ctx context.Context, apiKey string) {
	classifier := createAgent("classifier", apiKey, "Classify the sentiment as positive or negative. Respond with just 'positive' or 'negative'.")
	positiveHandler := createAgent("positive", apiKey, "You handle positive feedback. Thank the user warmly.")
	negativeHandler := createAgent("negative", apiKey, "You handle negative feedback. Apologize and offer help.")

	// Classification step
	classifyStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "classify",
		Agent: classifier,
	})

	// Positive branch
	positiveStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "positive-response",
		Agent: positiveHandler,
	})

	// Negative branch
	negativeStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "negative-response",
		Agent: negativeHandler,
	})

	// Conditional node
	condition, _ := workflow.NewCondition(workflow.ConditionConfig{
		ID: "sentiment-branch",
		Condition: func(ctx *workflow.ExecutionContext) bool {
			return strings.Contains(strings.ToLower(ctx.Output), "positive")
		},
		TrueNode:  positiveStep,
		FalseNode: negativeStep,
	})

	// Create workflow
	wf, _ := workflow.New(workflow.Config{
		Name:  "Sentiment Handler",
		Steps: []workflow.Node{classifyStep, condition},
	})

	result, err := wf.Run(ctx, "Your product is amazing! I love it!", "")
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}

	fmt.Printf("Response: %s\n", result.Output)
}

func runLoopWorkflow(ctx context.Context, apiKey string) {
	refiner := createAgent("refiner", apiKey, "Refine and improve the given text. Make it more concise.")

	// Loop body
	refineStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "refine",
		Agent: refiner,
	})

	// Loop 3 times for iterative refinement
	loop, _ := workflow.NewLoop(workflow.LoopConfig{
		ID:   "refinement-loop",
		Body: refineStep,
		Condition: func(ctx *workflow.ExecutionContext, iteration int) bool {
			return iteration < 3
		},
	})

	wf, _ := workflow.New(workflow.Config{
		Name:  "Iterative Refinement",
		Steps: []workflow.Node{loop},
	})

	result, err := wf.Run(ctx, "AI is a technology that enables machines to perform tasks that typically require human intelligence, such as learning, reasoning, problem-solving, and understanding natural language.", "")
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}

	iterations, _ := result.Get("loop_refinement-loop_iterations")
	fmt.Printf("Refined after %v iterations: %s\n", iterations, result.Output)
}

func runParallelWorkflow(ctx context.Context, apiKey string) {
	techAgent := createAgent("tech", apiKey, "Analyze technical aspects in 1-2 sentences.")
	bizAgent := createAgent("biz", apiKey, "Analyze business aspects in 1-2 sentences.")
	ethicsAgent := createAgent("ethics", apiKey, "Analyze ethical aspects in 1-2 sentences.")

	techStep, _ := workflow.NewStep(workflow.StepConfig{ID: "tech-analysis", Agent: techAgent})
	bizStep, _ := workflow.NewStep(workflow.StepConfig{ID: "biz-analysis", Agent: bizAgent})
	ethicsStep, _ := workflow.NewStep(workflow.StepConfig{ID: "ethics-analysis", Agent: ethicsAgent})

	parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
		ID:    "multi-perspective",
		Nodes: []workflow.Node{techStep, bizStep, ethicsStep},
	})

	wf, _ := workflow.New(workflow.Config{
		Name:  "Parallel Analysis",
		Steps: []workflow.Node{parallel},
	})

	result, err := wf.Run(ctx, "The use of facial recognition technology in public spaces", "")
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}

	fmt.Println("Analysis Results:")
	tech, _ := result.Get("parallel_multi-perspective_branch_0_output")
	biz, _ := result.Get("parallel_multi-perspective_branch_1_output")
	ethics, _ := result.Get("parallel_multi-perspective_branch_2_output")

	fmt.Printf("  Tech: %v\n", tech)
	fmt.Printf("  Business: %v\n", biz)
	fmt.Printf("  Ethics: %v\n", ethics)
}

func runComplexWorkflow(ctx context.Context, apiKey string) {
	// Router determines task type
	router := createAgent("router", apiKey, "Determine if this is a 'calculation' or 'general' task. Respond with just the word 'calculation' or 'general'.")

	// Calculation route
	calcAgent := createAgent("calculator", apiKey, "You perform calculations.", calculator.New())

	// General route
	generalAgent := createAgent("general", apiKey, "You handle general questions.")

	// Create steps
	routerStep, _ := workflow.NewStep(workflow.StepConfig{ID: "router", Agent: router})
	calcStep, _ := workflow.NewStep(workflow.StepConfig{ID: "calc-task", Agent: calcAgent})
	generalStep, _ := workflow.NewStep(workflow.StepConfig{ID: "general-task", Agent: generalAgent})

	// Router node
	routerNode, _ := workflow.NewRouter(workflow.RouterConfig{
		ID: "task-router",
		Router: func(ctx *workflow.ExecutionContext) string {
			if strings.Contains(strings.ToLower(ctx.Output), "calculation") {
				return "calc"
			}
			return "general"
		},
		Routes: map[string]workflow.Node{
			"calc":    calcStep,
			"general": generalStep,
		},
	})

	wf, _ := workflow.New(workflow.Config{
		Name:  "Smart Router",
		Steps: []workflow.Node{routerStep, routerNode},
	})

	// Test with calculation
	result1, _ := wf.Run(ctx, "What is 25 * 4 + 100?", "")
	fmt.Printf("Calculation result: %s\n", result1.Output)

	// Test with general question
	result2, _ := wf.Run(ctx, "What is the capital of France?", "")
	fmt.Printf("General result: %s\n", result2.Output)
}
