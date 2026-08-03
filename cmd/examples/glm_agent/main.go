package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/glm"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	// 从环境变量获取 API key
	apiKey := os.Getenv("ZHIPUAI_API_KEY")
	if apiKey == "" {
		log.Fatal("ZHIPUAI_API_KEY environment variable is required\n" +
			"环境变量 ZHIPUAI_API_KEY 是必需的")
	}

	// Create GLM model
	// 创建 GLM 模型
	model, err := glm.New("glm-4", glm.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   1024,
	})
	if err != nil {
		log.Fatalf("Failed to create GLM model: %v\n创建 GLM 模型失败: %v", err, err)
	}

	// Create calculator toolkit
	// 创建计算器工具集
	calc := calculator.New()

	// Create agent with GLM model
	// 使用 GLM 模型创建 agent
	glmAgent, err := agent.New(agent.Config{
		Name:         "GLM Calculator Agent",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are a helpful AI assistant powered by GLM-4. You can perform calculations using your calculator tools.\n你是一个由 GLM-4 驱动的有用的 AI 助手。你可以使用计算器工具执行计算。",
		MaxLoops:     5,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v\n创建 agent 失败: %v", err, err)
	}

	// Example 1: Simple conversation
	// 示例 1: 简单对话
	fmt.Println("=== Example 1: Simple Conversation ===")
	fmt.Println("=== 示例 1: 简单对话 ===")

	output, err := glmAgent.Run(context.Background(), "你好！请用中文介绍一下你自己。")
	if err != nil {
		log.Fatalf("Agent run failed: %v\nAgent 运行失败: %v", err, err)
	}

	fmt.Printf("Assistant: %s\n\n", output.Content)

	// Example 2: Using calculator tools
	// 示例 2: 使用计算器工具
	fmt.Println("=== Example 2: Calculator Tools ===")
	fmt.Println("=== 示例 2: 计算器工具 ===")

	output, err = glmAgent.Run(context.Background(), "请计算 (123 + 456) * 789 的结果")
	if err != nil {
		log.Fatalf("Agent run failed: %v\nAgent 运行失败: %v", err, err)
	}

	fmt.Printf("Assistant: %s\n\n", output.Content)

	// Example 3: Multi-step calculation
	// 示例 3: 多步骤计算
	fmt.Println("=== Example 3: Multi-step Calculation ===")
	fmt.Println("=== 示例 3: 多步骤计算 ===")

	output, err = glmAgent.Run(context.Background(), "如果我每天跑步5公里，一周跑5天，那么一个月（按30天计算）我总共跑了多少公里？请一步步计算。")
	if err != nil {
		log.Fatalf("Agent run failed: %v\nAgent 运行失败: %v", err, err)
	}

	fmt.Printf("Assistant: %s\n\n", output.Content)

	fmt.Println("=== All examples completed successfully! ===")
	fmt.Println("=== 所有示例成功完成！===")
}
