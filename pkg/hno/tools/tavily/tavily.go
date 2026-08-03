package tavily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

const (
	defaultBaseURL = "https://api.tavily.com"
)

// Config 配置 Tavily 工具包。
type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// Toolkit 提供 Tavily 快速回答与阅读能力。
type Toolkit struct {
	*toolkit.BaseToolkit
	client *client
}

type client struct {
	apiKey string
	http   *http.Client
	base   string
}

type searchRequest struct {
	APIKey     string `json:"api_key"`
	Query      string `json:"query"`
	SearchType string `json:"search_type,omitempty"`
}

type readerRequest struct {
	APIKey  string `json:"api_key"`
	Query   string `json:"query"`
	Extract bool   `json:"extract"`
}

type searchResponse struct {
	Answer  string                   `json:"answer"`
	Results []map[string]interface{} `json:"results"`
}

type readerResponse struct {
	Results []map[string]interface{} `json:"results"`
}

// New 创建 Tavily 工具包。
func New(cfg Config) (*Toolkit, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("tavily api key is required")
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	if cfg.HTTPClient != nil {
		httpClient = cfg.HTTPClient
	}
	if cfg.Timeout > 0 {
		httpClient.Timeout = cfg.Timeout
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	c := &client{
		apiKey: cfg.APIKey,
		http:   httpClient,
		base:   strings.TrimRight(baseURL, "/"),
	}

	tk := &Toolkit{
		BaseToolkit: toolkit.NewBaseToolkit("tavily"),
		client:      c,
	}

	tk.RegisterFunction(&toolkit.Function{
		Name:        "tavily_search",
		Description: "Perform a Tavily quick answer search",
		Parameters: map[string]toolkit.Parameter{
			"query": {
				Type:        "string",
				Description: "Search query",
				Required:    true,
			},
			"search_type": {
				Type:        "string",
				Description: "Optional search type (e.g. news, academic)",
			},
		},
		Handler: tk.search,
	})

	tk.RegisterFunction(&toolkit.Function{
		Name:        "tavily_reader",
		Description: "Execute Tavily reader mode with extract flag",
		Parameters: map[string]toolkit.Parameter{
			"query": {
				Type:        "string",
				Description: "Reader query or URL",
				Required:    true,
			},
			"extract": {
				Type:        "boolean",
				Description: "Whether to return extracted article content",
				Required:    false,
			},
		},
		Handler: tk.reader,
	})

	return tk, nil
}

func (t *Toolkit) search(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query must be a non-empty string")
	}

	var searchType string
	if raw, ok := args["search_type"].(string); ok {
		searchType = raw
	}

	return t.client.search(ctx, query, searchType)
}

func (t *Toolkit) reader(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query must be a non-empty string")
	}
	extract := true
	if raw, ok := args["extract"].(bool); ok {
		extract = raw
	}
	return t.client.reader(ctx, query, extract)
}

func (c *client) search(ctx context.Context, query, searchType string) (interface{}, error) {
	payload := searchRequest{
		APIKey: c.apiKey,
		Query:  query,
	}
	if searchType != "" {
		payload.SearchType = searchType
	}

	var resp searchResponse
	if err := c.post(ctx, "/search", payload, &resp); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"answer":  resp.Answer,
		"results": resp.Results,
	}, nil
}

func (c *client) reader(ctx context.Context, query string, extract bool) (interface{}, error) {
	payload := readerRequest{
		APIKey:  c.apiKey,
		Query:   query,
		Extract: extract,
	}

	var resp readerResponse
	if err := c.post(ctx, "/reader", payload, &resp); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"results": resp.Results,
	}, nil
}

func (c *client) post(ctx context.Context, path string, payload interface{}, out interface{}) error {
	url := c.base + "/v1" + path
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode tavily payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("tavily request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("tavily request returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode tavily response: %w", err)
	}
	return nil
}
