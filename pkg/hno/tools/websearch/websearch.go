package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

const defaultBaseURL = "https://api.tavily.com"

// Config configures the Tavily Search API used by this toolkit.
type Config struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

// WebSearchToolkit provides real web search through Tavily and direct URL
// content extraction through net/http.
type WebSearchToolkit struct {
	*toolkit.BaseToolkit
	client  *http.Client
	apiKey  string
	baseURL string
}

// New creates a toolkit using TAVILY_API_KEY from the environment.
func New() *WebSearchToolkit {
	return NewWithConfig(Config{APIKey: os.Getenv("TAVILY_API_KEY")})
}

// NewWithBase creates a toolkit against a custom Tavily-compatible endpoint.
func NewWithBase(baseURL string) *WebSearchToolkit {
	return NewWithConfig(Config{APIKey: os.Getenv("TAVILY_API_KEY"), BaseURL: baseURL})
}

// NewWithConfig creates a toolkit with explicit API settings.
func NewWithConfig(config Config) *WebSearchToolkit {
	if strings.TrimSpace(config.BaseURL) == "" {
		config.BaseURL = defaultBaseURL
	}
	if config.Client == nil {
		config.Client = &http.Client{Timeout: 30 * time.Second}
	}

	t := &WebSearchToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("web_search"),
		client:      config.Client,
		apiKey:      strings.TrimSpace(config.APIKey),
		baseURL:     strings.TrimRight(config.BaseURL, "/"),
	}
	t.RegisterFunction(&toolkit.Function{
		Name:        "web_search",
		Description: "Search the web using the Tavily Search API",
		Parameters: map[string]toolkit.Parameter{
			"query":       {Type: "string", Description: "The search query", Required: true},
			"max_results": {Type: "integer", Description: "Maximum number of results to return (default: 5)", Required: false, Default: 5},
		},
		Handler: t.webSearch,
	})
	t.RegisterFunction(&toolkit.Function{
		Name:        "extract_web_content",
		Description: "Extract content from a specific URL",
		Parameters: map[string]toolkit.Parameter{
			"url": {Type: "string", Description: "The URL to extract content from", Required: true},
		},
		Handler: t.extractWebContent,
	})
	return t
}

type tavilySearchRequest struct {
	APIKey        string `json:"api_key"`
	Query         string `json:"query"`
	SearchDepth   string `json:"search_depth"`
	MaxResults    int    `json:"max_results"`
	IncludeAnswer bool   `json:"include_answer"`
	IncludeRaw    bool   `json:"include_raw_content"`
}

type tavilySearchResponse struct {
	Results []struct {
		Title      string  `json:"title"`
		URL        string  `json:"url"`
		Content    string  `json:"content"`
		Score      float64 `json:"score"`
		RawContent string  `json:"raw_content"`
	} `json:"results"`
}

func webSearchString(args map[string]interface{}, name string) (string, error) {
	value, ok := args[name].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", name)
	}
	return strings.TrimSpace(value), nil
}

func webSearchLimit(args map[string]interface{}) (int, error) {
	limit := 5
	if value, ok := args["max_results"]; ok {
		switch number := value.(type) {
		case int:
			limit = number
		case int64:
			limit = int(number)
		case float64:
			limit = int(number)
		case float32:
			limit = int(number)
		default:
			return 0, fmt.Errorf("max_results must be an integer")
		}
	}
	if limit < 1 || limit > 20 {
		return 0, fmt.Errorf("max_results must be between 1 and 20")
	}
	return limit, nil
}

func (w *WebSearchToolkit) webSearch(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, err := webSearchString(args, "query")
	if err != nil {
		return nil, err
	}
	limit, err := webSearchLimit(args)
	if err != nil {
		return nil, err
	}
	if w.apiKey == "" {
		return nil, fmt.Errorf("Tavily API key is required; set TAVILY_API_KEY or configure the toolkit")
	}

	body, err := json.Marshal(tavilySearchRequest{
		APIKey:        w.apiKey,
		Query:         query,
		SearchDepth:   "basic",
		MaxResults:    limit,
		IncludeAnswer: false,
		IncludeRaw:    false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode Tavily request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create Tavily request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Tavily request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, apiError("Tavily", resp)
	}
	var payload tavilySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode Tavily response: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(payload.Results))
	for _, item := range payload.Results {
		results = append(results, map[string]interface{}{
			"title":   item.Title,
			"url":     item.URL,
			"snippet": item.Content,
			"score":   item.Score,
		})
	}
	return map[string]interface{}{
		"query":   query,
		"results": results,
		"count":   len(results),
	}, nil
}

func (w *WebSearchToolkit) extractWebContent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	urlStr, err := webSearchString(args, "url")
	if err != nil {
		return nil, err
	}
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return nil, fmt.Errorf("URL must be an absolute http or https URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; HNO/2.0)")
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, apiError("web content", resp)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	content := strings.TrimSpace(string(body))
	if len(content) > 1000 {
		content = content[:1000] + "..."
	}
	return map[string]interface{}{
		"url":     urlStr,
		"content": content,
		"status":  resp.StatusCode,
		"length":  len(content),
	}, nil
}

func apiError(service string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	message := strings.TrimSpace(string(body))
	if message == "" {
		return fmt.Errorf("%s API request failed with status: %d", service, resp.StatusCode)
	}
	return fmt.Errorf("%s API request failed with status: %d: %s", service, resp.StatusCode, message)
}
