package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/modelscope"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		log.Fatal("DASHSCOPE_API_KEY environment variable is required")
	}

	// Create ModelScope model (using Qwen models from Alibaba Cloud)
	// Popular models: qwen-plus, qwen-turbo, qwen-max
	model, err := modelscope.New("qwen-plus", modelscope.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		log.Fatalf("Failed to create ModelScope model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent with ModelScope
	ag, err := agent.New(agent.Config{
		Name:         "ModelScopeAssistant",
		Model:        model,
		Instructions: "你是一个有帮助的AI助手，由阿里云魔搭社区提供支持。你可以进行计算和回答问题。",
		Toolkits:     []toolkit.Toolkit{calc},
		MaxLoops:     5,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	fmt.Println("ModelScope Agent initialized successfully!")
	fmt.Println("Model:", model.GetName())
	fmt.Println("Provider:", model.GetProvider())
	fmt.Println()

	ctx := context.Background()

	// Example 1: Simple question (Chinese)
	fmt.Println("=== 示例 1: 简单问答 ===")
	output, err := ag.Run(ctx, "中国的首都是哪里？")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("问: 中国的首都是哪里？\n")
	fmt.Printf("答: %s\n\n", output.Content)

	// Example 2: Math calculation
	fmt.Println("=== 示例 2: 数学计算 ===")
	output, err = ag.Run(ctx, "计算 789 乘以 456 等于多少？")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("问: 计算 789 乘以 456 等于多少？\n")
	fmt.Printf("答: %s\n\n", output.Content)

	// Example 3: Multi-step reasoning (Chinese)
	fmt.Println("=== 示例 3: 多步推理 ===")
	output, err = ag.Run(ctx, "如果我有1000元，先花掉30%买书，然后把剩下的40%存起来，我还剩多少钱？")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
	fmt.Printf("问: 如果我有1000元，先花掉30%%买书，然后把剩下的40%%存起来，我还剩多少钱？\n")
	fmt.Printf("答: %s\n\n", output.Content)

	// Display metadata
	fmt.Println("=== 运行元数据 ===")
	fmt.Printf("循环次数: %v\n", output.Metadata["loops"])
	fmt.Printf("Token使用: %+v\n", output.Metadata["usage"])
}
