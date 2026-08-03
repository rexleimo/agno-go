package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

func (a *Anthropic) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	claudeReq := a.buildClaudeRequest(req)
	claudeReq.Stream = true

	resp, err := models.PostJSONRaw(ctx, a.httpClient, a.config.BaseURL+"/messages", a.authHeaders(), claudeReq)
	if err != nil {
		return nil, err
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

		// Anthropic streams use SSE framing (event:/data: lines). Use the
		// shared models.SSEDecoder for parsing.
		// Anthropic 流式使用 SSE 帧（event:/data: 行）。使用共享的
		// models.SSEDecoder 解析。
		decoder := models.NewSSEDecoder(resp.Body)
		for {
			data, err := decoder.Next()
			if err != nil {
				if err != io.EOF {
					chunks <- types.ResponseChunk{
						Done:  true,
						Error: err,
					}
				}
				return
			}
			if len(data) == 0 {
				continue
			}

			var event StreamEvent
			if err := json.Unmarshal(data, &event); err != nil {
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
