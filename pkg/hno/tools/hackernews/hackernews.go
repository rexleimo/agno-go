package hackernews

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// HackerNewsToolkit provides Hacker News API capabilities
// This implementation uses the official Hacker News API

// HackerNewsToolkit provides Hacker News capabilities
type HackerNewsToolkit struct {
	*toolkit.BaseToolkit
	client *http.Client
	baseURL string
}

// New creates a new HackerNews toolkit
func New() *HackerNewsToolkit {
	t := &HackerNewsToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("hacker_news"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://hacker-news.firebaseio.com/v0",
	}

	// Register top stories function
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_top_stories",
		Description: "Get the top stories from Hacker News",
		Parameters: map[string]toolkit.Parameter{
			"limit": {
				Type:        "integer",
				Description: "Number of top stories to return (default: 10)",
				Required:    false,
				Default:     10,
			},
		},
		Handler: t.getTopStories,
	})

	// Register story details function
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_story_details",
		Description: "Get detailed information about a specific Hacker News story",
		Parameters: map[string]toolkit.Parameter{
			"story_id": {
				Type:        "integer",
				Description: "The Hacker News story ID",
				Required:    true,
			},
		},
		Handler: t.getStoryDetails,
	})

	// Register search function
	t.RegisterFunction(&toolkit.Function{
		Name:        "search_stories",
		Description: "Search for Hacker News stories by keyword",
		Parameters: map[string]toolkit.Parameter{
			"query": {
				Type:        "string",
				Description: "The search query",
				Required:    true,
			},
			"limit": {
				Type:        "integer",
				Description: "Number of results to return (default: 10)",
				Required:    false,
				Default:     10,
			},
		},
		Handler: t.searchStories,
	})

	return t
}

// HNItem represents a Hacker News item
type HNItem struct {
	ID          int    `json:"id"`
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"`
	Text        string `json:"text,omitempty"`
	Score       int    `json:"score,omitempty"`
	By          string `json:"by,omitempty"`
	Time        int64  `json:"time,omitempty"`
	Type        string `json:"type,omitempty"`
	Descendants int    `json:"descendants,omitempty"`
}

// getTopStories gets the top stories from Hacker News
func (h *HackerNewsToolkit) getTopStories(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit := 10
	if limitArg, ok := args["limit"].(float64); ok {
		limit = int(limitArg)
	}

	// Get top story IDs
	var storyIDs []int
	resp, err := h.client.Get(h.baseURL + "/topstories.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top stories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch top stories: HTTP %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&storyIDs); err != nil {
		return nil, fmt.Errorf("failed to decode top stories: %w", err)
	}

	// Limit the number of stories
	if len(storyIDs) > limit {
		storyIDs = storyIDs[:limit]
	}

	// Get details for each story
	var stories []map[string]interface{}
	for _, id := range storyIDs {
		story, err := h.getItemDetails(id)
		if err != nil {
			continue // Skip failed stories
		}
		stories = append(stories, story)
	}

	return map[string]interface{}{
		"stories": stories,
		"count":   len(stories),
	}, nil
}

// getStoryDetails gets detailed information about a specific story
func (h *HackerNewsToolkit) getStoryDetails(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	storyID, ok := args["story_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("story_id must be a number")
	}

	story, err := h.getItemDetails(int(storyID))
	if err != nil {
		return nil, fmt.Errorf("failed to get story details: %w", err)
	}

	return story, nil
}

// searchStories searches for Hacker News stories
// Note: Hacker News doesn't have a built-in search API, so this is a placeholder
// In a real implementation, you might use Algolia's HN search API
func (h *HackerNewsToolkit) searchStories(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	limit := 10
	if limitArg, ok := args["limit"].(float64); ok {
		limit = int(limitArg)
	}

	// For now, return a mock response since Hacker News doesn't have a built-in search API
	// In a real implementation, this would use Algolia's HN search API
	mockResults := []map[string]interface{}{
		{
			"id":    123456,
			"title": fmt.Sprintf("Article about %s", query),
			"url":   "https://example.com/article",
			"score": 256,
			"by":    "user123",
			"time":  time.Now().Unix(),
		},
		{
			"id":    123457,
			"title": fmt.Sprintf("Tutorial: %s", query),
			"url":   "https://example.com/tutorial",
			"score": 128,
			"by":    "user456",
			"time":  time.Now().Unix() - 86400, // 1 day ago
		},
	}

	// Limit results
	if len(mockResults) > limit {
		mockResults = mockResults[:limit]
	}

	return map[string]interface{}{
		"query":   query,
		"results": mockResults,
		"count":   len(mockResults),
		"note":    "This is a placeholder implementation. For real search, integrate with Algolia's Hacker News search API.",
	}, nil
}

// getItemDetails fetches details for a specific item ID
func (h *HackerNewsToolkit) getItemDetails(id int) (map[string]interface{}, error) {
	resp, err := h.client.Get(fmt.Sprintf("%s/item/%d.json", h.baseURL, id))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch item %d: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch item %d: HTTP %d", id, resp.StatusCode)
	}

	var item HNItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, fmt.Errorf("failed to decode item %d: %w", id, err)
	}

	// Convert to map for easier use
	result := map[string]interface{}{
		"id":          item.ID,
		"title":       item.Title,
		"url":         item.URL,
		"text":        item.Text,
		"score":       item.Score,
		"by":          item.By,
		"time":        item.Time,
		"type":        item.Type,
		"descendants": item.Descendants,
		"time_string": time.Unix(item.Time, 0).Format("2006-01-02 15:04:05"),
	}

	return result, nil
}