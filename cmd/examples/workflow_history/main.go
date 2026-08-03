package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/workflow"
)

// main 演示 Workflow History 功能的真实使用场景
// main demonstrates real-world usage of Workflow History feature
func main() {
	// 检查 API Key
	// Check API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("错误: 请设置 OPENAI_API_KEY 环境变量\nError: Please set OPENAI_API_KEY environment variable")
	}

	fmt.Println("=== Workflow History 集成示例 ===")
	fmt.Println("=== Workflow History Integration Example ===")
	fmt.Println()

	// 创建 OpenAI 模型
	// Create OpenAI model
	model, err := openai.New("gpt-4", openai.Config{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("创建模型失败: %v\nFailed to create model: %v", err, err)
	}

	// 创建 agent
	// Create agent
	chatAgent, err := agent.New(agent.Config{
		ID:           "chatbot",
		Name:         "ChatBot",
		Model:        model,
		Instructions: "You are a helpful assistant with excellent memory. Remember previous conversations and refer to them when relevant. Be concise but friendly.",
	})
	if err != nil {
		log.Fatalf("创建 agent 失败: %v\nFailed to create agent: %v", err, err)
	}

	// 创建 workflow step
	// Create workflow step
	chatStep, err := workflow.NewStep(workflow.StepConfig{
		ID:    "chat",
		Name:  "Chat Step",
		Agent: chatAgent,
	})
	if err != nil {
		log.Fatalf("创建 step 失败: %v\nFailed to create step: %v", err, err)
	}

	// 创建带历史的 workflow
	// Create workflow with history enabled
	storage := workflow.NewMemoryStorage(100)
	wf, err := workflow.New(workflow.Config{
		ID:                "chat-workflow",
		Name:              "Conversational Chat",
		EnableHistory:     true,
		HistoryStore:      storage,
		NumHistoryRuns:    5, // 记住最近 5 轮对话
		AddHistoryToSteps: true,
		Steps:             []workflow.Node{chatStep},
	})
	if err != nil {
		log.Fatalf("创建 workflow 失败: %v\nFailed to create workflow: %v", err, err)
	}

	ctx := context.Background()
	sessionID := "user-session-123"

	// 多轮对话场景
	// Multi-turn conversation scenarios
	conversations := []string{
		"Hello, my name is Alice and I love programming in Go",
		"What's my name?",
		"What programming language do I like?",
		"Can you remind me what we talked about?",
	}

	// 执行对话
	// Execute conversations
	for i, input := range conversations {
		fmt.Printf("\n=== 第 %d 轮对话 / Round %d ===\n", i+1, i+1)
		fmt.Printf("用户 / User: %s\n", input)

		result, err := wf.Run(ctx, input, sessionID)
		if err != nil {
			log.Fatalf("工作流运行失败: %v\nWorkflow run failed: %v", err, err)
		}

		fmt.Printf("助手 / Assistant: %s\n", result.Output)

		// 显示历史信息
		// Display history info
		if result.HasHistory() {
			fmt.Printf("(历史记录: %d 条之前的对话)\n", result.GetHistoryCount())
			fmt.Printf("(History: %d previous conversations)\n", result.GetHistoryCount())
		} else {
			fmt.Println("(无历史记录)")
			fmt.Println("(No history)")
		}
	}

	// 显示完整的 session 统计
	// Display complete session statistics
	fmt.Println("\n=== Session 统计信息 / Session Statistics ===")
	session, err := storage.GetSession(ctx, sessionID)
	if err != nil {
		log.Fatalf("获取 session 失败: %v\nFailed to get session: %v", err, err)
	}

	fmt.Printf("总运行次数 / Total runs: %d\n", session.CountRuns())
	fmt.Printf("成功运行次数 / Successful runs: %d\n", session.CountSuccessfulRuns())
	fmt.Printf("失败运行次数 / Failed runs: %d\n", session.CountFailedRuns())

	// 显示历史详情
	// Display history details
	fmt.Println("\n=== 历史记录详情 / History Details ===")
	history := session.GetHistory(10) // 获取最近 10 条
	for i, entry := range history {
		fmt.Printf("\n历史 %d / History %d:\n", i+1, i+1)
		fmt.Printf("  输入 / Input: %s\n", entry.Input)
		fmt.Printf("  输出 / Output: %s\n", truncateString(entry.Output, 80))
		fmt.Printf("  时间 / Time: %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
	}

	fmt.Println("\n=== 示例完成 / Example Completed ===")
}

// truncateString 截断字符串以便显示
// truncateString truncates string for display
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
