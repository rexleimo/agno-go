package protocolsim

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ---------- Gemini wire types (front-end side) ----------

type geminiRequest struct {
	Contents          []geminiContent  `json:"contents"`
	SystemInstruction *geminiContent   `json:"systemInstruction,omitempty"`
	Tools             []geminiTool     `json:"tools,omitempty"`
	GenerationConfig  *geminiGenConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string              `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResp `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

type geminiFunctionResp struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type geminiGenConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// NewGeminiServer returns an HTTP handler that serves the Gemini generateContent
// API (POST /v1beta/models/{model}:generateContent and the streaming variant)
// in front of the given OpenAI backend.
// NewGeminiServer 返回一个 HTTP handler，在给定 OpenAI 后端之前提供
// Gemini generateContent API（POST /v1beta/models/{model}:generateContent 及流式变体）。
func NewGeminiServer(backend Backend) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1beta/models/", func(w http.ResponseWriter, r *http.Request) {
		handleGemini(w, r, backend)
	})
	return mux
}

func handleGemini(w http.ResponseWriter, r *http.Request, backend Backend) {
	var req geminiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeGeminiError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	// Streaming endpoint: /v1beta/models/{model}:streamGenerateContent?alt=sse
	// 流式端点：/v1beta/models/{model}:streamGenerateContent?alt=sse
	if strings.Contains(r.URL.Path, ":streamGenerateContent") {
		handleGeminiStream(w, r, req, backend)
		return
	}
	handleGeminiSync(w, r, req, backend)
}

// buildGeminiOpenAIChat converts a Gemini request into an OpenAI chat request.
// buildGeminiOpenAIChat 将 Gemini 请求转换为 OpenAI chat 请求。
func buildGeminiOpenAIChat(req geminiRequest, backend Backend, stream bool) *oaiChatRequest {
	oai := &oaiChatRequest{
		Model:  backend.Model,
		Stream: stream,
	}
	if req.GenerationConfig != nil {
		oai.MaxTokens = req.GenerationConfig.MaxOutputTokens
		if req.GenerationConfig.Temperature > 0 {
			t := req.GenerationConfig.Temperature
			oai.Temperature = &t
		}
	}
	if oai.MaxTokens <= 0 {
		oai.MaxTokens = 4096
	}

	if req.SystemInstruction != nil {
		sysText := ""
		for _, p := range req.SystemInstruction.Parts {
			sysText += p.Text
		}
		if sysText != "" {
			oai.Messages = append(oai.Messages, oaiMessage{Role: "system", Content: sysText})
		}
	}

	for _, c := range req.Contents {
		role := c.Role
		switch role {
		case "user", "model":
			// Gemini uses "model" role; OpenAI uses "assistant".
			// Gemini 使用 "model" 角色；OpenAI 使用 "assistant"。
			if role == "model" {
				role = "assistant"
			}
		case "function":
			role = "tool"
		}
		msg := oaiMessage{Role: role}
		var textParts []string
		var toolCalls []oaiToolCall
		var toolCallID string
		for _, p := range c.Parts {
			if p.Text != "" {
				textParts = append(textParts, p.Text)
			}
			if p.FunctionCall != nil {
				argsJSON, _ := json.Marshal(p.FunctionCall.Args)
				toolCalls = append(toolCalls, oaiToolCall{
					ID:   "call_" + p.FunctionCall.Name,
					Type: "function",
					Function: oaiFunctionCall{
						Name:      p.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
			if p.FunctionResponse != nil {
				respJSON, _ := json.Marshal(p.FunctionResponse.Response)
				msg.ToolCallID = "call_" + p.FunctionResponse.Name
				textParts = append(textParts, string(respJSON))
			}
		}
		msg.Content = strings.Join(textParts, "")
		msg.ToolCalls = toolCalls
		_ = toolCallID
		oai.Messages = append(oai.Messages, msg)
	}

	for _, t := range req.Tools {
		for _, fd := range t.FunctionDeclarations {
			oai.Tools = append(oai.Tools, oaiTool{
				Type: "function",
				Function: oaiFunctionSchema{
					Name:        fd.Name,
					Description: fd.Description,
					Parameters:  fd.Parameters,
				},
			})
		}
	}
	return oai
}

func handleGeminiSync(w http.ResponseWriter, r *http.Request, req geminiRequest, backend Backend) {
	oaiReq := buildGeminiOpenAIChat(req, backend, false)
	resp, err := callBackend(r.Context(), backend, oaiReq)
	if err != nil {
		writeGeminiError(w, http.StatusBadGateway, err.Error())
		return
	}

	var parts []geminiPart
	finishReason := "STOP"
	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		if msg.Content != "" {
			parts = append(parts, geminiPart{Text: msg.Content})
		}
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				var args map[string]interface{}
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				parts = append(parts, geminiPart{
					FunctionCall: &geminiFunctionCall{Name: tc.Function.Name, Args: args},
				})
			}
			finishReason = "TOOL_CALL"
		}
	}

	payload := geminiResponse{
		Candidates: []geminiCandidate{{
			Content:      geminiContent{Role: "model", Parts: parts},
			FinishReason: finishReason,
			Index:        0,
		}},
		UsageMetadata: geminiUsage{
			PromptTokenCount:     resp.Usage.PromptTokens,
			CandidatesTokenCount: resp.Usage.CompletionTokens,
			TotalTokenCount:      resp.Usage.PromptTokens + resp.Usage.CompletionTokens,
		},
	}
	writeJSON(w, http.StatusOK, payload)
}

func handleGeminiStream(w http.ResponseWriter, r *http.Request, req geminiRequest, backend Backend) {
	oaiReq := buildGeminiOpenAIChat(req, backend, true)
	resp, err := callBackendRaw(r.Context(), backend, oaiReq)
	if err != nil {
		writeGeminiError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		writeGeminiError(w, resp.StatusCode, string(body))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeGeminiError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	// Gemini streaming emits one SSE data: per chunk, each carrying a partial
	// Candidate. Emit each text delta as its own candidate part.
	// Gemini 流式每个分块发一条 SSE data:，每条携带部分 Candidate。
	// 将每个文本增量作为独立 candidate part 发出。
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta
		// Reasoning models (e.g. Qwen3) emit reasoning_content deltas with
		// content == null. Forward both fields so the stream is never empty.
		// 推理模型（如 Qwen3）发出 reasoning_content 增量且 content 为 null。
		// 同时转发两个字段，确保流不会为空。
		text := delta.Content
		if text == "" && delta.ReasoningContent != "" {
			text = delta.ReasoningContent
		}
		if text == "" {
			continue
		}
		payload := geminiResponse{
			Candidates: []geminiCandidate{{
				Content: geminiContent{
					Role:  "model",
					Parts: []geminiPart{{Text: text}},
				},
				FinishReason: "",
				Index:        0,
			}},
		}
		dataJSON, _ := json.Marshal(payload)
		fmt.Fprintf(w, "data: %s\n\n", dataJSON)
		flusher.Flush()
	}

	// Terminal chunk with finish reason.
	// 带完成原因的终止分块。
	final := geminiResponse{
		Candidates: []geminiCandidate{{
			Content:      geminiContent{Role: "model", Parts: []geminiPart{}},
			FinishReason: "STOP",
			Index:        0,
		}},
	}
	finalJSON, _ := json.Marshal(final)
	fmt.Fprintf(w, "data: %s\n\n", finalJSON)
	flusher.Flush()
}

// ---------- Gemini wire types (response side) ----------

type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata geminiUsage       `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
	Index        int           `json:"index"`
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

func writeGeminiError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    status,
			"message": msg,
			"status":  http.StatusText(status),
		},
	})
}
