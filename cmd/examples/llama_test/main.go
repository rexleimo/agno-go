// llama_test 真实连通性测试：openai provider + llama.cpp 本地服务
// 验证内容：
//  1. 同步工具调用循环（S1.5 验收）
//  2. 流式输出（EOF 兼容修复）
//  3. 流式 + 工具调用组合（S1.3 修复验证）
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func newAgent() (*agent.Agent, error) {
	model, err := openai.New("qwen3-4b", openai.Config{
		APIKey:  "local-test",
		BaseURL: "http://127.0.0.1:18080/v1",
		Timeout: 120 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("模型创建失败: %w", err)
	}

	return agent.New(agent.Config{
		Name:     "llama-calc-agent",
		Model:    model,
		Toolkits: []toolkit.Toolkit{calculator.New()},
		MaxLoops: 5,
	})
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// ---- 测试 1：同步工具调用循环 ----
	fmt.Println("=== 测试 1：同步工具调用（计算器）===")
	ag, err := newAgent()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	start := time.Now()
	output, err := ag.Run(ctx, "Calculate 15 * 23 using the calculator tool. Give me the final number only.")
	if err != nil {
		fmt.Println("运行失败:", err)
		os.Exit(1)
	}
	fmt.Printf("耗时: %s\n", time.Since(start).Round(time.Millisecond))
	fmt.Printf("状态: %s\n", output.Status)
	fmt.Printf("回答: %s\n", output.Content)
	fmt.Printf("轮数: %v\n", output.Metadata["loops"])

	syncTool := false
	for _, msg := range output.Messages {
		if msg.Role == "tool" {
			syncTool = true
			break
		}
	}
	if syncTool {
		fmt.Println("✓ 工具调用成功（memory 中有 tool 消息）")
	} else {
		fmt.Println("✗ 未检测到工具调用")
	}

	// ---- 测试 2：流式输出 ----
	fmt.Println()
	fmt.Println("=== 测试 2：流式输出 ===")
	ag2, err := newAgent()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	streamResult, err := ag2.RunStream(ctx, "Say hello in one short sentence.")
	if err != nil {
		fmt.Println("流式启动失败:", err)
		os.Exit(1)
	}
	for range streamResult.Events {
	}
	done := <-streamResult.Done
	if done.Err != nil {
		fmt.Println("流式错误:", done.Err)
		os.Exit(1)
	}
	fmt.Printf("流式回答: %s\n", done.Output.Content)
	fmt.Println("✓ 流式运行成功")

	// ---- 测试 3：流式 + 工具调用组合 ----
	fmt.Println()
	fmt.Println("=== 测试 3：流式 + 工具调用组合 ===")
	fmt.Println("输入: Calculate 8 / 2 using the calculator. Show the final number.")
	fmt.Println("(预期：第一轮流式返回工具调用→执行→第二轮流式返回最终答案)")
	ag3, err := newAgent()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	result3, err := ag3.RunStream(ctx, "Calculate 8 / 2 using the calculator. Show the final number.")
	if err != nil {
		fmt.Println("流式启动失败:", err)
		os.Exit(1)
	}

	var streamedText string
	eventCount := 0
	for evt := range result3.Events {
		eventCount++
		if content, ok := evt.(*run.RunContentEvent); ok {
			streamedText += content.Content
		}
	}
	done3 := <-result3.Done
	if done3.Err != nil {
		fmt.Println("流式错误:", done3.Err)
		os.Exit(1)
	}

	fmt.Printf("流式事件数: %d\n", eventCount)
	fmt.Printf("流式文本: %q\n", streamedText)
	fmt.Printf("最终回答: %s\n", done3.Output.Content)
	fmt.Printf("轮数: %v\n", done3.Output.Metadata["loops"])

	streamTool := false
	for _, msg := range done3.Output.Messages {
		if msg.Role == "tool" {
			streamTool = true
			break
		}
	}
	if streamTool {
		fmt.Println("✓ 流式路径中工具调用已执行（S1.3 修复验证通过）")
	} else {
		fmt.Println("✗ 未检测到工具调用")
		os.Exit(1)
	}
}
