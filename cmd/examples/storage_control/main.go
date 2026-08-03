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

// 本示例演示 Agent 的存储控制功能
// 可以控制是否存储工具消息和历史消息

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// 创建 OpenAI 模型
	model, err := openai.New("gpt-4", openai.Config{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// 示例 1: 默认行为 - 存储所有消息
	fmt.Println("========== 示例 1: 默认行为（存储所有消息）==========")
	agent1, err := agent.New(agent.Config{
		Name:         "calculator-agent-default",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calculator.New()},
		Instructions: "你是一个数学助手，可以执行基本的数学运算。",
		// StoreToolMessages 和 StoreHistoryMessages 默认为 true
	})
	if err != nil {
		log.Fatalf("Failed to create agent1: %v", err)
	}

	output1, err := agent1.Run(context.Background(), "计算 15 * 8")
	if err != nil {
		log.Fatalf("Agent1 run failed: %v", err)
	}

	fmt.Printf("输出: %s\n", output1.Content)
	fmt.Printf("消息数量: %d\n", len(output1.Messages))
	fmt.Println()

	// 示例 2: 不存储工具消息
	fmt.Println("========== 示例 2: 不存储工具消息 ==========")
	storeToolMessages := false
	agent2, err := agent.New(agent.Config{
		Name:              "calculator-agent-no-tool-msgs",
		Model:             model,
		Toolkits:          []toolkit.Toolkit{calculator.New()},
		Instructions:      "你是一个数学助手，可以执行基本的数学运算。",
		StoreToolMessages: &storeToolMessages, // 不存储工具消息
	})
	if err != nil {
		log.Fatalf("Failed to create agent2: %v", err)
	}

	output2, err := agent2.Run(context.Background(), "计算 25 + 17")
	if err != nil {
		log.Fatalf("Agent2 run failed: %v", err)
	}

	fmt.Printf("输出: %s\n", output2.Content)
	fmt.Printf("消息数量: %d (工具消息已被过滤)\n", len(output2.Messages))
	fmt.Println()

	// 示例 3: 不存储历史消息
	fmt.Println("========== 示例 3: 不存储历史消息 ==========")
	storeHistoryMessages := false
	agent3, err := agent.New(agent.Config{
		Name:                 "calculator-agent-no-history",
		Model:                model,
		Toolkits:             []toolkit.Toolkit{calculator.New()},
		Instructions:         "你是一个数学助手，可以执行基本的数学运算。",
		StoreHistoryMessages: &storeHistoryMessages, // 不存储历史消息
	})
	if err != nil {
		log.Fatalf("Failed to create agent3: %v", err)
	}

	// 第一次运行
	_, err = agent3.Run(context.Background(), "计算 10 + 5")
	if err != nil {
		log.Fatalf("Agent3 first run failed: %v", err)
	}

	// 第二次运行 - 历史消息不会被存储
	output3, err := agent3.Run(context.Background(), "再计算 20 * 3")
	if err != nil {
		log.Fatalf("Agent3 second run failed: %v", err)
	}

	fmt.Printf("输出: %s\n", output3.Content)
	fmt.Printf("消息数量: %d (只包含当前 Run 的消息，不包含历史)\n", len(output3.Messages))
	fmt.Println()

	// 示例 4: 同时不存储工具消息和历史消息
	fmt.Println("========== 示例 4: 最小存储（不存储工具消息和历史消息）==========")
	storeToolMessages4 := false
	storeHistoryMessages4 := false
	agent4, err := agent.New(agent.Config{
		Name:                 "calculator-agent-minimal",
		Model:                model,
		Toolkits:             []toolkit.Toolkit{calculator.New()},
		Instructions:         "你是一个数学助手，可以执行基本的数学运算。",
		StoreToolMessages:    &storeToolMessages4,    // 不存储工具消息
		StoreHistoryMessages: &storeHistoryMessages4, // 不存储历史消息
	})
	if err != nil {
		log.Fatalf("Failed to create agent4: %v", err)
	}

	// 多次运行
	_, _ = agent4.Run(context.Background(), "计算 5 + 3")
	output4, err := agent4.Run(context.Background(), "计算 100 / 4")
	if err != nil {
		log.Fatalf("Agent4 run failed: %v", err)
	}

	fmt.Printf("输出: %s\n", output4.Content)
	fmt.Printf("消息数量: %d (最小存储，只有最终的用户消息和助手响应)\n", len(output4.Messages))
	fmt.Println()

	fmt.Println("========== 存储控制示例完成 ==========")
	fmt.Println("\n关键点:")
	fmt.Println("1. StoreToolMessages=false: 过滤工具调用和工具响应消息")
	fmt.Println("2. StoreHistoryMessages=false: 只保留当前 Run 的消息，不保留历史")
	fmt.Println("3. 两者结合可以实现最小化存储，适用于无状态或隐私敏感场景")
	fmt.Println("4. 默认情况下（nil 或 true），所有消息都会被存储")
}
