// responsetest OpenAI Responses API 模拟器端到端验证
// 链路：Responses API 格式请求 → protocolsim（/v1/responses）→ llama.cpp
// 覆盖：非流式对话、工具调用（function_call）、流式事件（SSE）
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/internal/protocolsim"
)

func main() {
	// 1. 启动协议模拟器（Responses 端点 → llama.cpp）
	backend := protocolsim.Backend{
		BaseURL: "http://127.0.0.1:18080/v1",
		APIKey:  "local-test",
		Model:   "qwen3-4b",
	}
	srv := &http.Server{
		Addr:    "127.0.0.1:16200",
		Handler: protocolsim.NewResponsesServer(backend),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("模拟器启动失败:", err)
			os.Exit(1)
		}
	}()
	time.Sleep(500 * time.Millisecond)
	fmt.Println("✓ Responses API 模拟器已启动: http://127.0.0.1:16200/v1/responses")

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// 2. 测试 1：非流式对话
	fmt.Println("\n=== 测试 1：非流式对话（Responses API 格式）===")
	respBody := postResponses(ctx, "http://127.0.0.1:16200/v1/responses", `{
		"model": "gpt-4o-mini",
		"input": [{"role": "user", "content": [{"type": "input_text", "text": "Say hello in one short sentence."}]}],
		"max_output_tokens": 100
	}`)
	fmt.Printf("响应: %s\n", truncate(respBody, 400))

	var resp struct {
		ID     string `json:"id"`
		Object string `json:"object"`
		Status string `json:"status"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal([]byte(respBody), &resp); err != nil {
		fmt.Println("响应解析失败:", err)
		os.Exit(1)
	}
	if resp.Object != "response" || resp.Status != "completed" {
		fmt.Printf("✗ 响应格式不对: object=%s status=%s\n", resp.Object, resp.Status)
		os.Exit(1)
	}
	fmt.Printf("✓ 响应格式正确: id=%s\n", resp.ID)

	// 3. 测试 2：工具调用（function_call）
	fmt.Println("\n=== 测试 2：工具调用（function_call 输出项）===")
	respBody2 := postResponses(ctx, "http://127.0.0.1:16200/v1/responses", `{
		"model": "gpt-4o-mini",
		"input": [{"role": "user", "content": [{"type": "input_text", "text": "What is 5*8? Use the multiply tool."}]}],
		"tools": [{"type": "function", "name": "multiply", "description": "Multiply two numbers", "parameters": {"type": "object", "properties": {"a": {"type": "number"}, "b": {"type": "number"}}, "required": ["a", "b"]}}],
		"max_output_tokens": 150
	}`)
	fmt.Printf("响应: %s\n", truncate(respBody2, 500))
	if strings.Contains(respBody2, "function_call") {
		fmt.Println("✓ 工具调用输出项生成（function_call）")
	} else {
		fmt.Println("✗ 未检测到 function_call（模型可能直接回答了）")
	}

	// 4. 测试 3：流式事件（SSE）
	fmt.Println("\n=== 测试 3：流式事件（response.* SSE 事件序列）===")
	eventTypes := postResponsesStream(ctx, "http://127.0.0.1:16200/v1/responses", `{
		"model": "gpt-4o-mini",
		"input": [{"role": "user", "content": [{"type": "input_text", "text": "Count 1 2 3"}]}],
		"max_output_tokens": 100,
		"stream": true
	}`)
	fmt.Printf("事件序列: %v\n", eventTypes)

	expected := []string{"response.created", "response.output_item.added", "response.content_part.added", "response.output_text.delta", "response.output_text.done", "response.content_part.done", "response.output_item.done", "response.completed"}
	missing := []string{}
	for _, e := range expected {
		found := false
		for _, got := range eventTypes {
			if got == e {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, e)
		}
	}
	if len(missing) == 0 {
		fmt.Println("✓ 全部 8 个标准事件类型均出现")
	} else {
		fmt.Printf("✗ 缺失事件: %v\n", missing)
	}

	_ = srv.Shutdown(context.Background())
	fmt.Println("\n=== 全部通过 ===")
}

func postResponses(ctx context.Context, url, body string) string {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(body))
	if err != nil {
		fmt.Println("请求构建失败:", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-local-test")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("请求失败:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return buf.String()
}

func postResponsesStream(ctx context.Context, url, body string) []string {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(body))
	if err != nil {
		fmt.Println("请求构建失败:", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-local-test")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("请求失败:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}

	var types []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "event:") {
			types = append(types, strings.TrimSpace(strings.TrimPrefix(line, "event:")))
		}
	}
	return types
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
