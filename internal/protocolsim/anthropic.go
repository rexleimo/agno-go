// Package protocolsim provides local protocol simulation servers that expose
// Anthropic Messages API and Gemini generateContent endpoints in front of any
// OpenAI-compatible backend (e.g. llama.cpp). This lets framework providers be
// integration-tested against real protocol formats without cloud credentials.
//
// protocolsim 包提供本地协议模拟服务：在任意 OpenAI 兼容后端（如 llama.cpp）
// 之前暴露 Anthropic Messages API 与 Gemini generateContent 端点，
// 使框架 provider 无需云端凭据即可针对真实协议格式做集成测试。
package protocolsim

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Backend is an OpenAI-compatible chat completions endpoint.
// Backend 是 OpenAI 兼容的 chat completions 端点。
type Backend struct {
	BaseURL string // e.g. http://127.0.0.1:18080/v1
	APIKey  string
	Model   string // model name sent to the backend
}

func (b Backend) chatURL() string {
	return strings.TrimRight(b.BaseURL, "/") + "/chat/completions"
}

// ---------- OpenAI wire types (backend side) ----------

type oaiMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content"`
	ToolCalls  []oaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

type oaiToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function oaiFunctionCall `json:"function"`
}

type oaiFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type oaiTool struct {
	Type     string            `json:"type"`
	Function oaiFunctionSchema `json:"function"`
}

type oaiFunctionSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type oaiChatRequest struct {
	Model       string       `json:"model"`
	Messages    []oaiMessage `json:"messages"`
	Tools       []oaiTool    `json:"tools,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Temperature *float64     `json:"temperature,omitempty"`
	Stream      bool         `json:"stream,omitempty"`
}

type oaiChatResponse struct {
	Choices []struct {
		Message oaiMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// ---------- Anthropic wire types (front-end side) ----------

type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature *float64           `json:"temperature,omitempty"`
	Tools       []anthropicTool    `json:"tools,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// NewAnthropicServer returns an HTTP handler that serves the Anthropic
// Messages API (POST /v1/messages) in front of the given OpenAI backend.
// NewAnthropicServer 返回一个 HTTP handler，在给定 OpenAI 后端之前
// 提供 Anthropic Messages API（POST /v1/messages）。
func NewAnthropicServer(backend Backend) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		handleAnthropicMessages(w, r, backend)
	})
	return mux
}

func handleAnthropicMessages(w http.ResponseWriter, r *http.Request, backend Backend) {
	var req anthropicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body: "+err.Error())
		return
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 4096
	}
	if req.Stream {
		handleAnthropicStream(w, r, req, backend)
		return
	}
	handleAnthropicSync(w, r, req, backend)
}

// buildOpenAIChat converts an Anthropic request into an OpenAI chat request.
// buildOpenAIChat 将 Anthropic 请求转换为 OpenAI chat 请求。
func buildOpenAIChat(req anthropicRequest, backend Backend, stream bool) *oaiChatRequest {
	oai := &oaiChatRequest{
		Model:     backend.Model,
		MaxTokens: req.MaxTokens,
		Stream:    stream,
	}
	if req.Temperature != nil {
		oai.Temperature = req.Temperature
	}
	if req.System != "" {
		oai.Messages = append(oai.Messages, oaiMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		content := string(m.Content)
		// Content may be a plain string or an array of blocks; extract text.
		// content 可能是普通字符串或块数组；提取文本部分。
		content = extractAnthropicText(content)
		switch m.Role {
		case "user", "assistant":
			oai.Messages = append(oai.Messages, oaiMessage{Role: m.Role, Content: content})
		case "tool":
			// Anthropic tool results arrive as user messages with tool_result blocks.
			// Anthropic 工具结果以带 tool_result 块的 user 消息到达。
			oai.Messages = append(oai.Messages, oaiMessage{Role: "tool", Content: content})
		}
	}
	for _, t := range req.Tools {
		oai.Tools = append(oai.Tools, oaiTool{
			Type: "function",
			Function: oaiFunctionSchema{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return oai
}

// extractAnthropicText handles both "string" and block-array content.
// extractAnthropicText 同时处理字符串与块数组两种 content。
func extractAnthropicText(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "[") {
		return trimmed
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(trimmed), &blocks); err != nil {
		return trimmed
	}
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}

func handleAnthropicSync(w http.ResponseWriter, r *http.Request, req anthropicRequest, backend Backend) {
	oaiReq := buildOpenAIChat(req, backend, false)
	resp, err := callBackend(r.Context(), backend, oaiReq)
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", err.Error())
		return
	}

	// Convert OpenAI response to Anthropic Messages format.
	// 将 OpenAI 响应转换为 Anthropic Messages 格式。
	content := []map[string]interface{}{}
	stopReason := "end_turn"
	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		if msg.Content != "" {
			content = append(content, map[string]interface{}{"type": "text", "text": msg.Content})
		}
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				var input map[string]interface{}
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": input,
				})
			}
			stopReason = "tool_use"
		}
	}

	payload := map[string]interface{}{
		"id":            "msg_sim_" + strconv.FormatInt(timeNowUnix(), 10),
		"type":          "message",
		"role":          "assistant",
		"model":         req.Model,
		"content":       content,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": map[string]interface{}{
			"input_tokens":  resp.Usage.PromptTokens,
			"output_tokens": resp.Usage.CompletionTokens,
		},
	}
	writeJSON(w, http.StatusOK, payload)
}

func handleAnthropicStream(w http.ResponseWriter, r *http.Request, req anthropicRequest, backend Backend) {
	oaiReq := buildOpenAIChat(req, backend, true)
	resp, err := callBackendRaw(r.Context(), backend, oaiReq)
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		writeAnthropicError(w, resp.StatusCode, "api_error", string(body))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "streaming unsupported")
		return
	}

	// Anthropic SSE sequence:
	// message_start -> content_block_start -> content_block_delta* -> content_block_stop
	// -> message_delta -> message_stop
	// Anthropic SSE 序列：
	// message_start -> content_block_start -> content_block_delta* -> content_block_stop
	// -> message_delta -> message_stop
	writeSSE(w, flusher, "message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":   "msg_sim_" + strconv.FormatInt(timeNowUnix(), 10),
			"type": "message", "role": "assistant",
			"content": []interface{}{}, "model": req.Model,
			"stop_reason": nil, "stop_sequence": nil,
			"usage": map[string]interface{}{"input_tokens": 0, "output_tokens": 0},
		},
	})
	writeSSE(w, flusher, "ping", map[string]interface{}{"type": "ping"})
	writeSSE(w, flusher, "content_block_start", map[string]interface{}{
		"type": "content_block_start", "index": 0,
		"content_block": map[string]interface{}{"type": "text", "text": ""},
	})

	outputTokens := 0
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta
		// Reasoning models (e.g. Qwen3) emit reasoning_content deltas with
		// content == null; forward both so the SSE stream is never empty.
		// 推理模型（如 Qwen3）发出 reasoning_content 增量且 content 为 null；
		// 同时转发两者，确保 SSE 流不为空。
		text := delta.Content
		if text == "" && delta.ReasoningContent != "" {
			text = delta.ReasoningContent
		}
		if text != "" {
			writeSSE(w, flusher, "content_block_delta", map[string]interface{}{
				"type": "content_block_delta", "index": 0,
				"delta": map[string]interface{}{"type": "text_delta", "text": text},
			})
			outputTokens++
		}
	}
	writeSSE(w, flusher, "content_block_stop", map[string]interface{}{"type": "content_block_stop", "index": 0})
	writeSSE(w, flusher, "message_delta", map[string]interface{}{
		"type":  "message_delta",
		"delta": map[string]interface{}{"stop_reason": "end_turn", "stop_sequence": nil},
		"usage": map[string]interface{}{"output_tokens": outputTokens},
	})
	writeSSE(w, flusher, "message_stop", map[string]interface{}{"type": "message_stop"})
}

func callBackend(ctx context.Context, backend Backend, oaiReq *oaiChatRequest) (*oaiChatResponse, error) {
	resp, err := callBackendRaw(ctx, backend, oaiReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("backend returned %d: %s", resp.StatusCode, string(body))
	}
	var out oaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func callBackendRaw(ctx context.Context, backend Backend, oaiReq *oaiChatRequest) (*http.Response, error) {
	body, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, backend.chatURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if backend.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+backend.APIKey)
	}
	return http.DefaultClient.Do(req)
}

func writeAnthropicError(w http.ResponseWriter, status int, typ, msg string) {
	writeJSON(w, status, map[string]interface{}{
		"type": "error",
		"error": map[string]interface{}{
			"type":    typ,
			"message": msg,
		},
	})
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload map[string]interface{}) {
	data, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	flusher.Flush()
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func timeNowUnix() int64 {
	return time.Now().Unix()
}
