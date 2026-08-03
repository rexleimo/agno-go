package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

const (
	defaultMaxResults = 5
	defaultTimeout    = 10 * time.Second
	duckDuckGoURL     = "https://html.duckduckgo.com/html/"
)

// Search provides web search capabilities using DuckDuckGo
type Search struct {
	*toolkit.BaseToolkit
	httpClient *http.Client
	maxResults int
	timeout    time.Duration
}

// Config contains search tool configuration
type Config struct {
	MaxResults int
	Timeout    time.Duration
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// New creates a new DuckDuckGo search toolkit
func New(config ...Config) *Search {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.MaxResults == 0 {
		cfg.MaxResults = defaultMaxResults
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}

	s := &Search{
		BaseToolkit: toolkit.NewBaseToolkit("search"),
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		maxResults: cfg.MaxResults,
		timeout:    cfg.Timeout,
	}

	s.registerFunctions()
	return s
}

func (s *Search) registerFunctions() {
	s.RegisterFunction(&toolkit.Function{
		Name:        "search",
		Description: "Search the web using DuckDuckGo. Returns a list of search results with title, URL, and snippet.",
		Parameters: map[string]toolkit.Parameter{
			"query": {
				Type:        "string",
				Description: "The search query",
				Required:    true,
			},
			"max_results": {
				Type:        "integer",
				Description: fmt.Sprintf("Maximum number of results to return (default: %d)", defaultMaxResults),
				Required:    false,
			},
		},
		Handler: s.search,
	})
}

func (s *Search) search(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required and must be a non-empty string")
	}

	maxResults := s.maxResults
	if mr, ok := args["max_results"]; ok {
		switch v := mr.(type) {
		case float64:
			maxResults = int(v)
		case int:
			maxResults = v
		}
	}

	if maxResults <= 0 {
		maxResults = defaultMaxResults
	}

	searchCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	results, err := s.performSearch(searchCtx, query, maxResults)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return results, nil
}

func (s *Search) performSearch(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	// Prepare search request
	formData := url.Values{}
	formData.Set("q", query)
	formData.Set("kl", "wt-wt") // No region restriction

	req, err := http.NewRequestWithContext(ctx, "POST", duckDuckGoURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse HTML response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	results, err := s.parseResults(string(body), maxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}

	return results, nil
}

func (s *Search) parseResults(html string, maxResults int) ([]SearchResult, error) {
	results := make([]SearchResult, 0, maxResults)

	// Simple HTML parsing for DuckDuckGo results
	// Look for result blocks: <div class="result__body">
	lines := strings.Split(html, "\n")

	var currentResult *SearchResult
	inResultBlock := false

	for _, line := range lines {
		// Stop if we already have enough results
		if len(results) >= maxResults {
			break
		}

		line = strings.TrimSpace(line)

		// Start of result block
		if strings.Contains(line, `class="result__a"`) {
			if currentResult != nil && currentResult.Title != "" && len(results) < maxResults {
				results = append(results, *currentResult)
			}
			currentResult = &SearchResult{}
			inResultBlock = true

			// Extract URL from href
			if start := strings.Index(line, `href="`); start != -1 {
				start += 6
				if end := strings.Index(line[start:], `"`); end != -1 {
					currentResult.URL = line[start : start+end]
					// Decode URL-encoded characters
					if decoded, err := url.QueryUnescape(currentResult.URL); err == nil {
						currentResult.URL = decoded
					}
				}
			}

			// Extract title (text between > and <)
			if start := strings.Index(line, ">"); start != -1 {
				start++
				if end := strings.Index(line[start:], "<"); end != -1 {
					currentResult.Title = cleanHTML(line[start : start+end])
				}
			}
		}

		// Extract snippet
		if inResultBlock && strings.Contains(line, `class="result__snippet"`) {
			if start := strings.Index(line, ">"); start != -1 {
				start++
				if end := strings.Index(line[start:], "<"); end != -1 {
					snippet := cleanHTML(line[start : start+end])
					if currentResult != nil {
						currentResult.Snippet = snippet
					}
				}
			}
		}
	}

	// Add last result if valid and within limit
	if currentResult != nil && currentResult.Title != "" && len(results) < maxResults {
		results = append(results, *currentResult)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for query")
	}

	return results, nil
}

func cleanHTML(s string) string {
	// Remove any HTML tags first
	for strings.Contains(s, "<") && strings.Contains(s, ">") {
		start := strings.Index(s, "<")
		end := strings.Index(s[start:], ">")
		if end == -1 {
			break
		}
		s = s[:start] + s[start+end+1:]
	}

	// Then decode HTML entities
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")

	return strings.TrimSpace(s)
}

// FormatResults formats search results as a JSON string
func FormatResults(results []SearchResult) (string, error) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format results: %w", err)
	}
	return string(data), nil
}
