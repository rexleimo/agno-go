// skillstest Skills 机制端到端验证
// 链路：Agent(Skills registry) → llama.cpp 真机
// 验证：渐进式披露（目录注入 + use_skill 工具加载技能正文）
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/skills"
)

func main() {
	// 1. 加载技能注册表（项目 skills/ 目录）
	loader := skills.NewLoader(os.DirFS("skills"))
	reg, err := skills.NewRegistry(loader, ".")
	if err != nil {
		fmt.Println("注册表创建失败:", err)
		os.Exit(1)
	}

	catalog := reg.Catalog()
	fmt.Printf("✓ 技能注册表加载: %d 个技能\n", len(catalog))
	for _, info := range catalog {
		fmt.Printf("  - %s: %s\n", info.Name, info.Description)
	}

	// 2. 创建 Agent（Skills 注入）
	model, err := openai.NewCompat("qwen3-4b", openai.CompatConfig{
		Provider:  "llama",
		APIKey:    "local-test",
		BaseURL:   "http://127.0.0.1:18080/v1",
		MaxTokens: 512,
	})
	if err != nil {
		fmt.Println("模型创建失败:", err)
		os.Exit(1)
	}

	ag, err := agent.New(agent.Config{
		Name:     "skills-agent",
		Model:    model,
		Skills:   reg,
		MaxLoops: 6,
	})
	if err != nil {
		fmt.Println("Agent 创建失败:", err)
		os.Exit(1)
	}

	// 3. 验证目录已注入系统消息
	fmt.Println("\n=== 渐进式披露第一级：目录注入 ===")
	sysFound := false
	for _, msg := range ag.Memory.GetMessages() {
		if msg.Role == "system" {
			sysFound = true
			fmt.Printf("系统消息内容（前 200 字符）:\n%s\n", truncate(msg.Content, 200))
		}
	}
	if !sysFound {
		fmt.Println("✗ 未找到系统消息")
		os.Exit(1)
	}

	// 4. use_skill 工具已注册
	fmt.Println("\n=== use_skill 工具注册 ===")
	toolFound := false
	for _, tk := range ag.Toolkits {
		if tk.Name() == "skills" {
			toolFound = true
			fmt.Println("✓ use_skill 工具已注册（toolkit: skills）")
		}
	}
	if !toolFound {
		fmt.Println("✗ use_skill 工具未注册")
		os.Exit(1)
	}

	// 5. 真机运行：模型应看到目录并调用 use_skill
	fmt.Println("\n=== 真机运行：模型使用技能（渐进式披露第二级）===")
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	output, err := ag.Run(ctx, "I need to review some Go code changes. Which skill should I use, and what does it say? Use the skill tool to load it.")
	if err != nil {
		fmt.Println("运行失败:", err)
		os.Exit(1)
	}
	fmt.Printf("回答（前 400 字符）:\n%s\n", truncate(output.Content, 400))

	// 检查模型是否真的发起了 use_skill 工具调用
	// 证据 1：assistant 消息中的 tool_calls 包含 use_skill
	fmt.Println("\n=== 证据：模型工具调用轨迹 ===")
	callFound := false
	toolResultLen := 0
	for _, msg := range output.Messages {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				if tc.Function.Name == "use_skill" {
					callFound = true
					fmt.Printf("✓ 模型发起工具调用: use_skill(%s)\n", tc.Function.Arguments)
				}
			}
		}
		if msg.Role == "tool" {
			toolResultLen += len(msg.Content)
		}
	}
	if callFound {
		fmt.Printf("✓ use_skill 被真实调用（工具结果回填 %d 字符）\n", toolResultLen)
	} else {
		fmt.Println("✗ 模型未调用 use_skill——技能机制未生效")
		os.Exit(1)
	}

	fmt.Println("\n=== 完成 ===")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
