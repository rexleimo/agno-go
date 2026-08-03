package protocolsim

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ---------- OpenAI Responses wire types (front-end side) ----------

type responsesRequest struct {
	Model           string           `json:"model"`
	Input           []responsesInput `json:"input"`
	Instructions    string           `json:"instructions,omitempty"`
	Tools           []responsesTool  `json:"tools,omitempty"`
	MaxOutputTokens int              `json:"max_output_tokens,omitempty"`
	Temperature     *float64         `json:"temperature,omitempty"`
	Stream          bool             `json:"stream,omitempty"`
}

type responsesInput struct {
	Role    string          `json:"role,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	// Function-call result item (role=tool)
	CallID string `json:"call_id,omitempty"`
	Name   string `json:"name,omitempty"`
	// Output text (assistant previous turns)
	Output json.RawMessage `json:"output,omitempty"`
}

type responsesTool struct {
	Type        string                 `json:"type"` // "function"
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// NewResponsesServer returns an HTTP handler serving the OpenAI Responses API
// (POST /v1/responses) in front of the given OpenAI backend.
// NewResponsesServer 返回一个 HTTP handler，在给定 OpenAI 后端之前
// 提供 OpenAI Responses API（POST /v1/responses）。
func NewResponsesServer(backend Backend) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/responses", func(w http.ResponseWriter, r *http.Request) {
		handleResponses(w, r, backend)
	})
	return mux
}

func handleResponses(w http.ResponseWriter, r *http.Request, backend Backend) {
	var req responsesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResponsesError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if req.Stream {
		handleResponsesStream(w, r, req, backend)
		return
	}
	handleResponsesSync(w, r, req, backend)
}

// buildResponsesOpenAIChat converts a Responses request into an OpenAI chat request.
// buildResponsesOpenAIChat 将 Responses 请求转换为 OpenAI chat 请求。
func buildResponsesOpenAIChat(req responsesRequest, backend Backend, stream bool) *oaiChatRequest {
	oai := &oaiChatRequest{
		Model:     backend.Model,
		MaxTokens: req.MaxOutputTokens,
		Stream:    stream,
	}
	if oai.MaxTokens <= 0 {
		oai.MaxTokens = 4096
	}
	if req.Temperature != nil {
		oai.Temperature = req.Temperature
	}
	if req.Instructions != "" {
		oai.Messages = append(oai.Messages, oaiMessage{Role: "system", Content: req.Instructions})
	}

	for _, item := range req.Input {
		switch item.Role {
		case "user", "system":
			msg := oaiMessage{Role: item.Role, Content: extractResponsesText(item.Content)}
			oai.Messages = append(oai.Messages, msg)
		case "assistant":
			msg := oaiMessage{Role: "assistant"}
			if len(item.Output) > 0 {
				msg.Content = extractResponsesOutputText(item.Output)
			} else {
				msg.Content = extractResponsesText(item.Content)
			}
			oai.Messages = append(oai.Messages, msg)
		case "tool":
			msg := oaiMessage{
				Role:       "tool",
				ToolCallID: item.CallID,
				Content:    extractResponsesText(item.Content),
			}
			oai.Messages = append(oai.Messages, msg)
		case "function_call_output":
			msg := oaiMessage{
				Role:       "tool",
				ToolCallID: item.CallID,
				Content:    extractResponsesText(item.Content),
			}
			oai.Messages = append(oai.Messages, msg)
		}
	}

	for _, t := range req.Tools {
		if t.Type != "function" {
			continue
		}
		oai.Tools = append(oai.Tools, oaiTool{
			Type: "function",
			Function: oaiFunctionSchema{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return oai
}

// extractResponsesText extracts text from either a plain string or an array of
// input_text/output_text parts.
// extractResponsesText 从普通字符串或 input_text/output_text 部分数组中提取文本。
func extractResponsesText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "[") {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
		return trimmed
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return trimmed
	}
	var sb strings.Builder
	for _, p := range parts {
		if p.Text != "" {
			sb.WriteString(p.Text)
		}
	}
	return sb.String()
}

// extractResponsesOutputText extracts text from an assistant output item array.
// extractResponsesOutputText 从助手输出项数组中提取文本。
func extractResponsesOutputText(raw json.RawMessage) string {
	var outputs []struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &outputs); err != nil {
		return extractResponsesText(raw)
	}
	var sb strings.Builder
	for _, o := range outputs {
		if o.Type == "message" {
			sb.WriteString(extractResponsesText(o.Content))
		}
	}
	return sb.String()
}

func handleResponsesSync(w http.ResponseWriter, r *http.Request, req responsesRequest, backend Backend) {
	oaiReq := buildResponsesOpenAIChat(req, backend, false)
	resp, err := callBackend(r.Context(), backend, oaiReq)
	if err != nil {
		writeResponsesError(w, http.StatusBadGateway, err.Error())
		return
	}

	// Convert OpenAI chat response to Responses API format.
	// 将 OpenAI chat 响应转换为 Responses API 格式。
	output := []map[string]interface{}{}
	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		if msg.Content != "" {
			output = append(output, map[string]interface{}{
				"type": "message",
				"id":   "msg_" + strconv.FormatInt(time.Now().UnixNano(), 10),
				"role": "assistant",
				"content": []map[string]interface{}{
					{"type": "output_text", "text": msg.Content},
				},
			})
		}
		for _, tc := range msg.ToolCalls {
			output = append(output, map[string]interface{}{
				"type":      "function_call",
				"id":        "fc_" + tc.ID,
				"call_id":   tc.ID,
				"name":      tc.Function.Name,
				"arguments": tc.Function.Arguments,
			})
		}
	}

	payload := map[string]interface{}{
		"id":     "resp_" + strconv.FormatInt(time.Now().UnixNano(), 10),
		"object": "response",
		"status": "completed",
		"model":  req.Model,
		"output": output,
		"usage": map[string]interface{}{
			"input_tokens":  resp.Usage.PromptTokens,
			"output_tokens": resp.Usage.CompletionTokens,
			"total_tokens":  resp.Usage.PromptTokens + resp.Usage.CompletionTokens,
		},
	}
	writeJSON(w, http.StatusOK, payload)
}

func handleResponsesStream(w http.ResponseWriter, r *http.Request, req responsesRequest, backend Backend) {
	oaiReq := buildResponsesOpenAIChat(req, backend, true)
	resp, err := callBackendRaw(r.Context(), backend, oaiReq)
	if err != nil {
		writeResponsesError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		writeResponsesError(w, resp.StatusCode, string(body))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeResponsesError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	respID := "resp_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	itemID := "msg_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	writeResponsesEvent(w, flusher, map[string]interface{}{
		"type": "response.created", "response": map[string]interface{}{
			"id": respID, "object": "response", "status": "in_progress", "model": req.Model,
		},
	})
	writeResponsesEvent(w, flusher, map[string]interface{}{
		"type": "response.output_item.added", "output_index": 0, "item": map[string]interface{}{
			"id": itemID, "type": "message", "role": "assistant",
			"content": []interface{}{},
		},
	})
	writeResponsesEvent(w, flusher, map[string]interface{}{
		"type": "response.content_part.added", "item_id": itemID, "output_index": 0, "content_index": 0,
		"part": map[string]interface{}{"type": "output_text", "text": ""},
	})

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
		text := delta.Content
		if text == "" && delta.ReasoningContent != "" {
			text = delta.ReasoningContent
		}
		if text == "" {
			continue
		}
		writeResponsesEvent(w, flusher, map[string]interface{}{
			"type": "response.output_text.delta", "item_id": itemID, "output_index": 0, "content_index": 0,
			"delta": text,
		})
	}

	writeResponsesEvent(w, flusher, map[string]interface{}{
		"type": "response.output_text.done", "item_id": itemID, "output_index": 0, "content_index": 0,
		"text": "",
	})
	writeResponsesEvent(w, flusher, map[string]interface{}{
		"type": "response.content_part.done", "item_id": itemID, "output_index": 0, "content_index": 0,
		"part": map[string]interface{}{"type": "output_text", "text": ""},
	})
	writeResponsesEvent(w, flusher, map[string]interface{}{
		"type": "response.output_item.done", "output_index": 0,
		"item": map[string]interface{}{"id": itemID, "type": "message", "role": "assistant", "content": []interface{}{}},
	})
	writeResponsesEvent(w, flusher, map[string]interface{}{
		"type": "response.completed", "response": map[string]interface{}{
			"id": respID, "object": "response", "status": "completed", "model": req.Model,
		},
	})
}

func writeResponsesEvent(w http.ResponseWriter, flusher http.Flusher, payload map[string]interface{}) {
	data, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", payload["type"], data)
	flusher.Flush()
}

func writeResponsesError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"type":    "invalid_request_error",
			"message": msg,
		},
	})
}
