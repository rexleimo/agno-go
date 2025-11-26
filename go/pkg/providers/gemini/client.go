package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	"github.com/rexleimo/agno-go/pkg/providers/shared"
)

const defaultEndpoint = "https://generativelanguage.googleapis.com/v1beta"

// Client implements lightweight REST calls to Gemini.
type Client struct {
	endpoint string
	apiKey   string
	http     *http.Client
	status   model.ProviderStatus
}

type chatRequest struct {
	Contents       []oaContent `json:"contents"`
	SafetySettings []any       `json:"safetySettings,omitempty"`
}

type oaContent struct {
	Parts []oaPart `json:"parts"`
	Role  string   `json:"role,omitempty"`
}

type oaPart struct {
	Text string `json:"text"`
}

type chatResponse struct {
	Candidates []struct {
		Content oaContent `json:"content"`
	} `json:"candidates"`
}

type embedRequest struct {
	Model   string    `json:"model"`
	Content oaContent `json:"content"`
}

type embedResponse struct {
	Embedding struct {
		Values []float64 `json:"values"`
	} `json:"embedding"`
}

// New constructs a Gemini client.
func New(status model.ProviderStatus, endpoint, apiKey string) *Client {
	ep := endpoint
	if strings.TrimSpace(ep) == "" {
		ep = defaultEndpoint
	}
	if apiKey == "" {
		status.Status = model.ProviderNotConfigured
	}
	return &Client{
		endpoint: strings.TrimSuffix(ep, "/"),
		apiKey:   apiKey,
		http:     shared.DefaultHTTPClient(60 * time.Second),
		status:   status,
	}
}

func (c *Client) Name() agent.Provider { return agent.ProviderGemini }

func (c *Client) Status() model.ProviderStatus { return c.status }

func (c *Client) Chat(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	if c.status.Status != model.ProviderAvailable {
		return nil, model.ErrProviderUnavailable
	}
	body := chatRequest{
		Contents: toContents(req.Messages),
	}
	payload, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.endpoint, req.Model.ModelID, c.apiKey)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, parseGeminiError(resp)
	}
	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	content := ""
	if len(out.Candidates) > 0 && len(out.Candidates[0].Content.Parts) > 0 {
		content = out.Candidates[0].Content.Parts[0].Text
	}
	msg := agent.Message{
		Role:    agent.RoleAssistant,
		Content: content,
	}
	return &model.ChatResponse{
		Message: msg,
	}, nil
}

func (c *Client) Stream(ctx context.Context, req model.ChatRequest, fn model.StreamHandler) error {
	if c.status.Status != model.ProviderAvailable {
		return model.ErrProviderUnavailable
	}
	body := chatRequest{
		Contents: toContents(req.Messages),
	}
	payload, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", c.endpoint, req.Model.ModelID, c.apiKey)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return parseGeminiError(resp)
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
		for _, cand := range chunk.Candidates {
			for _, part := range cand.Content.Parts {
				if strings.TrimSpace(part.Text) == "" {
					continue
				}
				if err := fn(model.ChatStreamEvent{Type: "token", Delta: part.Text}); err != nil {
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
		Content: oaContent{
			Parts: []oaPart{{Text: strings.Join(req.Input, "\n")}},
		},
	}
	payload, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/models/%s:embedContent?key=%s", c.endpoint, req.Model.ModelID, c.apiKey)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, parseGeminiError(resp)
	}
	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &model.EmbeddingResponse{
		Vectors: [][]float64{out.Embedding.Values},
	}, nil
}

func toContents(msgs []agent.Message) []oaContent {
	out := make([]oaContent, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, oaContent{
			Role:  m.RoleString(),
			Parts: []oaPart{{Text: m.Content}},
		})
	}
	return out
}

func parseGeminiError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if len(body) > 0 {
		var parsed struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &parsed); err == nil {
			if msg := strings.TrimSpace(parsed.Error.Message); msg != "" {
				return fmt.Errorf("gemini error: %s (%s)", resp.Status, msg)
			}
		}
		if msg := strings.TrimSpace(string(body)); msg != "" {
			return fmt.Errorf("gemini error: %s (%s)", resp.Status, msg)
		}
	}
	return fmt.Errorf("gemini error: %s", resp.Status)
}
