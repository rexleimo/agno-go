package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

func (a *Anthropic) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	claudeReq := a.buildClaudeRequest(req)
	claudeReq.Stream = true

	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, types.NewAPIError("failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.config.BaseURL+"/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, types.NewAPIError("failed to create HTTP request", err)
	}

	a.setHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, types.NewAPIError("failed to call Anthropic API", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, types.NewAPIError(fmt.Sprintf("API error: %s", string(body)), nil)
	}

	chunks := make(chan types.ResponseChunk)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		// Anthropic streams use SSE framing (event:/data: lines), not raw
		// JSON lines. Each "data:" payload is a StreamEvent.
		// Anthropic 流式使用 SSE 帧（event:/data: 行），而非原始 JSON 行。
		// 每个 "data:" 负载都是一个 StreamEvent。
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}

			var event StreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				chunks <- types.ResponseChunk{
					Done:  true,
					Error: fmt.Errorf("failed to decode stream event: %w", err),
				}
				return
			}

			chunk := a.convertStreamEvent(&event)
			select {
			case chunks <- chunk:
				if chunk.Done {
					return
				}
			case <-ctx.Done():
				chunks <- types.ResponseChunk{
					Done:  true,
					Error: ctx.Err(),
				}
				return
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			chunks <- types.ResponseChunk{
				Done:  true,
				Error: fmt.Errorf("stream read error: %w", err),
			}
		}
	}()

	return chunks, nil
}

func (a *Anthropic) convertStreamEvent(event *StreamEvent) types.ResponseChunk {
	chunk := types.ResponseChunk{}

	switch event.Type {
	case "content_block_delta":
		if event.Delta.Type == "text_delta" {
			chunk.Content = event.Delta.Text
		}
	case "message_stop":
		chunk.Done = true
	case "error":
		chunk.Done = true
		chunk.Error = fmt.Errorf("stream error: %s", event.Error.Message)
	}

	return chunk
}

// StreamEvent represents a streaming event
type StreamEvent struct {
	Type  string      `json:"type"`
	Delta StreamDelta `json:"delta,omitempty"`
	Error StreamError `json:"error,omitempty"`
}

// StreamDelta represents delta content in streaming
type StreamDelta struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
}

// StreamError represents an error in streaming
type StreamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
