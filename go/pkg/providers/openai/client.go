package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	"github.com/rexleimo/agno-go/pkg/providers/shared"
)

const defaultEndpoint = "https://api.openai.com/v1"

// Client implements model.ChatProvider and EmbeddingProvider against OpenAI REST API.
type Client struct {
	endpoint string
	apiKey   string
	http     *http.Client
	status   model.ProviderStatus
}

type chatRequest struct {
	Model       string      `json:"model"`
	Messages    []oaMessage `json:"messages"`
	Stream      bool        `json:"stream,omitempty"`
	Tools       []any       `json:"tools,omitempty"`
	MaxTokens   *int        `json:"max_tokens,omitempty"`
	Temperature float64     `json:"temperature,omitempty"`
}

type oaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func New(status model.ProviderStatus, endpoint, apiKey string) *Client {
	ep := endpoint
	if strings.TrimSpace(ep) == "" {
		ep = defaultEndpoint
	}
	if apiKey == "" {
		status.Status = model.ProviderNotConfigured
	}
	client := shared.DefaultHTTPClient(60 * time.Second)
	return &Client{
		endpoint: strings.TrimSuffix(ep, "/"),
		apiKey:   apiKey,
		http:     client,
		status:   status,
	}
}

func (c *Client) Name() agent.Provider { return agent.ProviderOpenAI }

func (c *Client) Status() model.ProviderStatus { return c.status }

func (c *Client) Chat(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	if c.status.Status != model.ProviderAvailable {
		return nil, model.ErrProviderUnavailable
	}
	body := chatRequest{
		Model:       req.Model.ModelID,
		Messages:    toOAMessages(req.Messages),
		Stream:      false,
		MaxTokens:   req.Model.MaxTokens,
		Temperature: req.Model.Temperature,
	}
	payload, _ := json.Marshal(body)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/chat/completions", bytes.NewReader(payload))
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, mapHTTPError(resp)
	}
	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Choices) == 0 {
		return nil, errors.New("empty response")
	}
	content := out.Choices[0].Message.Content
	msg := agent.Message{
		Role:    agent.RoleAssistant,
		Content: content,
	}
	return &model.ChatResponse{
		Message: msg,
		Usage: agent.Usage{
			PromptTokens:     out.Usage.PromptTokens,
			CompletionTokens: out.Usage.CompletionTokens,
		},
	}, nil
}

func (c *Client) Stream(ctx context.Context, req model.ChatRequest, fn model.StreamHandler) error {
	if c.status.Status != model.ProviderAvailable {
		return model.ErrProviderUnavailable
	}
	body := chatRequest{
		Model:       req.Model.ModelID,
		Messages:    toOAMessages(req.Messages),
		Stream:      true,
		MaxTokens:   req.Model.MaxTokens,
		Temperature: req.Model.Temperature,
	}
	payload, _ := json.Marshal(body)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/chat/completions", bytes.NewReader(payload))
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return mapHTTPError(resp)
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk chatResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return err
		}
		for _, choice := range chunk.Choices {
			if delta := strings.TrimSpace(choice.Delta.Content); delta != "" {
				if err := fn(model.ChatStreamEvent{Type: "token", Delta: delta}); err != nil {
					return err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return fn(model.ChatStreamEvent{Type: "end", Done: true})
}

func (c *Client) Embed(ctx context.Context, req model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	if c.status.Status != model.ProviderAvailable {
		return nil, model.ErrProviderUnavailable
	}
	body := embedRequest{
		Model: req.Model.ModelID,
		Input: req.Input,
	}
	payload, _ := json.Marshal(body)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/embeddings", bytes.NewReader(payload))
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, mapHTTPError(resp)
	}
	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	vectors := make([][]float64, len(out.Data))
	for i, d := range out.Data {
		vectors[i] = d.Embedding
	}
	return &model.EmbeddingResponse{Vectors: vectors}, nil
}

func toOAMessages(msgs []agent.Message) []oaMessage {
	out := make([]oaMessage, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, oaMessage{Role: string(m.Role), Content: m.Content})
	}
	return out
}
