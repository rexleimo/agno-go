package hackernews

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

const (
	defaultBaseURL       = "https://hacker-news.firebaseio.com/v0"
	defaultSearchBaseURL = "https://hn.algolia.com/api/v1"
)

// HackerNewsToolkit provides Hacker News Firebase and Algolia Search API access.
type HackerNewsToolkit struct {
	*toolkit.BaseToolkit
	client        *http.Client
	baseURL       string
	searchBaseURL string
}

// New creates a toolkit using the official Hacker News APIs.
func New() *HackerNewsToolkit {
	return NewWithBases(defaultBaseURL, defaultSearchBaseURL)
}

// NewWithBase creates a toolkit against a custom base. The same base is used
// for both Firebase endpoints and /search, which is convenient for tests and
// API-compatible proxies.
func NewWithBase(baseURL string) *HackerNewsToolkit {
	return NewWithBases(baseURL, baseURL)
}

// NewWithBases creates a toolkit with separate Firebase and Algolia endpoints.
func NewWithBases(baseURL, searchBaseURL string) *HackerNewsToolkit {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	if strings.TrimSpace(searchBaseURL) == "" {
		searchBaseURL = defaultSearchBaseURL
	}
	t := &HackerNewsToolkit{
		BaseToolkit:   toolkit.NewBaseToolkit("hacker_news"),
		client:        &http.Client{Timeout: 30 * time.Second},
		baseURL:       strings.TrimRight(baseURL, "/"),
		searchBaseURL: strings.TrimRight(searchBaseURL, "/"),
	}

	t.RegisterFunction(&toolkit.Function{
		Name:        "get_top_stories",
		Description: "Get the top stories from Hacker News",
		Parameters: map[string]toolkit.Parameter{
			"limit": {Type: "integer", Description: "Number of top stories to return (default: 10)", Required: false, Default: 10},
		},
		Handler: t.getTopStories,
	})
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_story_details",
		Description: "Get detailed information about a specific Hacker News story",
		Parameters: map[string]toolkit.Parameter{
			"story_id": {Type: "integer", Description: "The Hacker News story ID", Required: true},
		},
		Handler: t.getStoryDetails,
	})
	t.RegisterFunction(&toolkit.Function{
		Name:        "search_stories",
		Description: "Search Hacker News stories through the Algolia HN Search API",
		Parameters: map[string]toolkit.Parameter{
			"query": {Type: "string", Description: "The search query", Required: true},
			"limit": {Type: "integer", Description: "Number of results to return (default: 10)", Required: false, Default: 10},
		},
		Handler: t.searchStories,
	})
	return t
}

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

type algoliaSearchHit struct {
	ObjectID    string `json:"objectID"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	StoryText   string `json:"story_text"`
	Author      string `json:"author"`
	Points      int    `json:"points"`
	CreatedAtI  int64  `json:"created_at_i"`
	NumComments int    `json:"num_comments"`
}

type algoliaSearchResponse struct {
	Hits   []algoliaSearchHit `json:"hits"`
	NbHits int                `json:"nbHits"`
}

func hnLimit(args map[string]interface{}, defaultValue, max int) (int, error) {
	limit := defaultValue
	if value, ok := args["limit"]; ok {
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
			return 0, fmt.Errorf("limit must be an integer")
		}
	}
	if limit < 1 || limit > max {
		return 0, fmt.Errorf("limit must be between 1 and %d", max)
	}
	return limit, nil
}

func (h *HackerNewsToolkit) getTopStories(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit, err := hnLimit(args, 10, 100)
	if err != nil {
		return nil, err
	}
	var storyIDs []int
	if err := h.getJSON(ctx, h.baseURL+"/topstories.json", &storyIDs); err != nil {
		return nil, fmt.Errorf("failed to fetch top stories: %w", err)
	}
	if len(storyIDs) > limit {
		storyIDs = storyIDs[:limit]
	}

	stories := make([]map[string]interface{}, 0, len(storyIDs))
	for _, id := range storyIDs {
		story, err := h.getItemDetails(ctx, id)
		if err != nil {
			continue
		}
		stories = append(stories, story)
	}
	return map[string]interface{}{
		"stories": stories,
		"count":   len(stories),
	}, nil
}

func (h *HackerNewsToolkit) getStoryDetails(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	value, ok := args["story_id"]
	if !ok {
		return nil, fmt.Errorf("story_id is required")
	}
	var storyID int
	switch number := value.(type) {
	case int:
		storyID = number
	case int64:
		storyID = int(number)
	case float64:
		storyID = int(number)
	case float32:
		storyID = int(number)
	default:
		return nil, fmt.Errorf("story_id must be a number")
	}
	if storyID <= 0 {
		return nil, fmt.Errorf("story_id must be positive")
	}
	story, err := h.getItemDetails(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get story details: %w", err)
	}
	return story, nil
}

func (h *HackerNewsToolkit) searchStories(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query must be a non-empty string")
	}
	query = strings.TrimSpace(query)
	limit, err := hnLimit(args, 10, 100)
	if err != nil {
		return nil, err
	}

	endpoint, err := url.Parse(h.searchBaseURL + "/search")
	if err != nil {
		return nil, fmt.Errorf("invalid Hacker News search URL: %w", err)
	}
	params := endpoint.Query()
	params.Set("query", query)
	params.Set("tags", "story")
	params.Set("hitsPerPage", strconv.Itoa(limit))
	endpoint.RawQuery = params.Encode()

	var payload algoliaSearchResponse
	if err := h.getJSON(ctx, endpoint.String(), &payload); err != nil {
		return nil, fmt.Errorf("failed to search Hacker News: %w", err)
	}
	results := make([]map[string]interface{}, 0, len(payload.Hits))
	for _, hit := range payload.Hits {
		id, _ := strconv.Atoi(hit.ObjectID)
		result := map[string]interface{}{
			"id":          id,
			"object_id":   hit.ObjectID,
			"title":       hit.Title,
			"url":         hit.URL,
			"text":        hit.StoryText,
			"score":       hit.Points,
			"by":          hit.Author,
			"time":        hit.CreatedAtI,
			"type":        "story",
			"descendants": hit.NumComments,
		}
		if hit.CreatedAtI > 0 {
			result["time_string"] = time.Unix(hit.CreatedAtI, 0).Format("2006-01-02 15:04:05")
		}
		results = append(results, result)
	}
	return map[string]interface{}{
		"query":      query,
		"results":    results,
		"count":      len(results),
		"total_hits": payload.NbHits,
	}, nil
}

func (h *HackerNewsToolkit) getItemDetails(ctx context.Context, id int) (map[string]interface{}, error) {
	var item *HNItem
	endpoint := fmt.Sprintf("%s/item/%d.json", h.baseURL, id)
	if err := h.getJSON(ctx, endpoint, &item); err != nil {
		return nil, fmt.Errorf("failed to fetch item %d: %w", id, err)
	}
	if item == nil {
		return nil, fmt.Errorf("item %d was not found", id)
	}
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
	}
	if item.Time > 0 {
		result["time_string"] = time.Unix(item.Time, 0).Format("2006-01-02 15:04:05")
	}
	return result, nil
}

func (h *HackerNewsToolkit) getJSON(ctx context.Context, endpoint string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		message := strings.TrimSpace(string(body))
		if message == "" {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, message)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return err
	}
	return nil
}
