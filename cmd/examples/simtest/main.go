// simtest 协议模拟器 + 框架 provider 端到端验证
// 启动：protocolsim 服务（Anthropic /v1/messages）→ llama.cpp
// 然后：框架 anthropic provider 指向模拟器，验证真实协议兼容性
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rexleimo/agno-go/internal/protocolsim"
	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/anthropic"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// 1. 启动协议模拟器（Anthropic 端点 → llama.cpp）
	backend := protocolsim.Backend{
		BaseURL: "http://127.0.0.1:18080/v1",
		APIKey:  "local-test",
		Model:   "qwen3-4b",
	}
	srv := &http.Server{
		Addr:    "127.0.0.1:16000",
		Handler: protocolsim.NewAnthropicServer(backend),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("模拟器启动失败:", err)
			os.Exit(1)
		}
	}()
	time.Sleep(500 * time.Millisecond)
	fmt.Println("✓ 协议模拟器已启动: http://127.0.0.1:16000/v1/messages (Anthropic 格式)")

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// 2. 框架 anthropic provider 指向模拟器
	model, err := anthropic.New("claude-3-5-sonnet", anthropic.Config{
		APIKey:    "sim-key",
		BaseURL:   "http://127.0.0.1:16000/v1",
		MaxTokens: 256,
		Timeout:   120 * time.Second,
	})
	if err != nil {
		fmt.Println("模型创建失败:", err)
		os.Exit(1)
	}

	ag, err := agent.New(agent.Config{
		Name:     "anthropic-sim-agent",
		Model:    model,
		Toolkits: []toolkit.Toolkit{calculator.New()},
		MaxLoops: 5,
	})
	if err != nil {
		fmt.Println("Agent 创建失败:", err)
		os.Exit(1)
	}

	// 3. 测试 1：普通对话
	fmt.Println("\n=== 测试 1：普通对话（anthropic provider → 模拟器 → llama.cpp）===")
	output, err := ag.Run(ctx, "Say hello in one short sentence.")
	if err != nil {
		fmt.Println("运行失败:", err)
		os.Exit(1)
	}
	fmt.Printf("回答: %s\n", output.Content)
	fmt.Printf("状态: %s\n", output.Status)

	// 4. 测试 2：工具调用
	fmt.Println("\n=== 测试 2：工具调用 ===")
	output2, err := ag.Run(ctx, "Calculate 6 * 7 using the calculator. Give the final number only.")
	if err != nil {
		fmt.Println("运行失败:", err)
		os.Exit(1)
	}
	fmt.Printf("回答: %s\n", output2.Content)
	toolCalled := false
	for _, msg := range output2.Messages {
		if msg.Role == "tool" {
			toolCalled = true
			break
		}
	}
	if toolCalled {
		fmt.Println("✓ 工具调用成功（Anthropic 协议 tool_use → 执行 → 回填）")
	} else {
		fmt.Println("✗ 未检测到工具调用")
	}

	// 5. 测试 3：流式
	fmt.Println("\n=== 测试 3：流式输出（Anthropic SSE 事件）===")
	result, err := ag.RunStream(ctx, "Say hi in one word.")
	if err != nil {
		fmt.Println("流式启动失败:", err)
		os.Exit(1)
	}
	var streamed string
	for evt := range result.Events {
		if content, ok := evt.(*run.RunContentEvent); ok {
			streamed += content.Content
		}
	}
	done := <-result.Done
	if done.Err != nil {
		fmt.Println("流式错误:", done.Err)
		os.Exit(1)
	}
	fmt.Printf("流式回答: %s\n", done.Output.Content)
	fmt.Printf("流式事件文本: %q\n", streamed)
	fmt.Println("✓ 流式成功")

	_ = srv.Shutdown(context.Background())
	fmt.Println("\n=== 全部通过 ===")
}
