package evolink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Config contains EvoLink provider configuration
type Config struct {
	APIKey     string
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// Client is a thin HTTP wrapper for EvoLink endpoints
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewClient creates a new EvoLink client
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, types.NewInvalidConfigError("evolink api key is required", nil)
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.evolink.ai"
	}
	hc := cfg.HTTPClient
	if hc == nil {
		to := cfg.Timeout
		if to == 0 {
			to = 60 * time.Second
		}
		hc = &http.Client{Timeout: to}
	}
	return &Client{baseURL: strings.TrimRight(base, "/"), apiKey: cfg.APIKey, http: hc}, nil
}

// post issues a POST request and decodes JSON into out if non-nil
func (c *Client) post(ctx context.Context, path string, payload interface{}, out interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return types.NewAPIError("api request failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return types.NewAPIError(fmt.Sprintf("api error (status %d): %s", resp.StatusCode, string(body)), nil)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// get issues a GET request and decodes JSON into out
func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return types.NewAPIError("api request failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return types.NewAPIError(fmt.Sprintf("api error (status %d): %s", resp.StatusCode, string(body)), nil)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// PostJSON is an exported helper for subpackages
func (c *Client) PostJSON(ctx context.Context, path string, payload interface{}, out interface{}) error {
	return c.post(ctx, path, payload, out)
}

// GetJSON is an exported helper for subpackages
func (c *Client) GetJSON(ctx context.Context, path string, out interface{}) error {
	return c.get(ctx, path, out)
}

// TaskResult represents a generic EvoLink task payload
type TaskResult struct {
	ID     string                 `json:"id"`
	Status string                 `json:"status"`
	Data   map[string]interface{} `json:"data,omitempty"`
	Error  interface{}            `json:"error,omitempty"`
}

// PollTask polls GET /v1/tasks/{id} until terminal status
func (c *Client) PollTask(ctx context.Context, taskID string, interval time.Duration) (*TaskResult, error) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		var tr TaskResult
		if err := c.get(ctx, "/v1/tasks/"+taskID, &tr); err != nil {
			return nil, err
		}
		switch strings.ToLower(tr.Status) {
		case "completed", "succeeded", "success":
			return &tr, nil
		case "failed", "cancelled", "canceled", "error":
			return nil, types.NewAPIError("task not completed", nil)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
