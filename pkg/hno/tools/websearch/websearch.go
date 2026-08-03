package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// WebSearchToolkit provides web search capabilities
// This is a basic implementation that can be extended with specific search APIs
// like SerpAPI, Tavily, or Baidu Search in the future

// WebSearchToolkit provides web search capabilities
type WebSearchToolkit struct {
	*toolkit.BaseToolkit
	client *http.Client
}

// New creates a new WebSearch toolkit
func New() *WebSearchToolkit {
	t := &WebSearchToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("web_search"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Register basic web search function
	t.RegisterFunction(&toolkit.Function{
		Name:        "web_search",
		Description: "Perform a web search for the given query and return results",
		Parameters: map[string]toolkit.Parameter{
			"query": {
				Type:        "string",
				Description: "The search query",
				Required:    true,
			},
			"max_results": {
				Type:        "integer",
				Description: "Maximum number of results to return (default: 5)",
				Required:    false,
				Default:     5,
			},
		},
		Handler: t.webSearch,
	})

	// Register URL content extraction function
	t.RegisterFunction(&toolkit.Function{
		Name:        "extract_web_content",
		Description: "Extract content from a specific URL",
		Parameters: map[string]toolkit.Parameter{
			"url": {
				Type:        "string",
				Description: "The URL to extract content from",
				Required:    true,
			},
		},
		Handler: t.extractWebContent,
	})

	return t
}

// webSearch performs a basic web search
// Note: This is a placeholder implementation that can be extended
// with actual search APIs like SerpAPI, Tavily, etc.
func (w *WebSearchToolkit) webSearch(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	maxResults := 5
	if maxResultsArg, ok := args["max_results"].(float64); ok {
		maxResults = int(maxResultsArg)
	}

	// For now, return a mock response since we need API keys for real search
	// In a real implementation, this would call SerpAPI, Tavily, or other search APIs
	mockResults := []map[string]interface{}{
		{
			"title":     "Example Search Result 1",
			"url":       "https://example.com/result1",
			"snippet":   fmt.Sprintf("This is a mock result for query: %s", query),
			"position":  1,
		},
		{
			"title":     "Example Search Result 2",
			"url":       "https://example.com/result2",
			"snippet":   fmt.Sprintf("Another mock result for: %s", query),
			"position":  2,
		},
	}

	// Limit results to maxResults
	if len(mockResults) > maxResults {
		mockResults = mockResults[:maxResults]
	}

	return map[string]interface{}{
		"query":   query,
		"results": mockResults,
		"count":   len(mockResults),
		"note":    "This is a placeholder implementation. Integrate with actual search APIs like SerpAPI or Tavily for real search results.",
	}, nil
}

// extractWebContent extracts content from a specific URL
func (w *WebSearchToolkit) extractWebContent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	urlStr, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url must be a string")
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set a user agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Agno-Go/1.0)")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	// Read response body
	bodyBytes := make([]byte, 1024*10) // Read up to 10KB
	n, err := resp.Body.Read(bodyBytes)
	if err != nil && err.Error() != "EOF" {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	content := string(bodyBytes[:n])

	// Basic content extraction - in a real implementation, you might want to
	// parse HTML and extract meaningful text content
	content = strings.TrimSpace(content)
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