package youtube

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// YouTubeToolkit provides YouTube video search and information capabilities
// This is a basic implementation that can be extended with YouTube Data API

// YouTubeToolkit provides YouTube video capabilities
type YouTubeToolkit struct {
	*toolkit.BaseToolkit
	client *http.Client
}

// New creates a new YouTube toolkit
func New() *YouTubeToolkit {
	t := &YouTubeToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("youtube"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Register YouTube search function
	t.RegisterFunction(&toolkit.Function{
		Name:        "search_youtube",
		Description: "Search for YouTube videos based on a query",
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
		Handler: t.searchYouTube,
	})

	// Register video information function
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_video_info",
		Description: "Get information about a specific YouTube video",
		Parameters: map[string]toolkit.Parameter{
			"video_url": {
				Type:        "string",
				Description: "The YouTube video URL",
				Required:    true,
			},
		},
		Handler: t.getVideoInfo,
	})

	return t
}

// searchYouTube searches for YouTube videos
// Note: This is a placeholder implementation that can be extended
// with YouTube Data API for real search results
func (y *YouTubeToolkit) searchYouTube(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	maxResults := 5
	if maxResultsArg, ok := args["max_results"].(float64); ok {
		maxResults = int(maxResultsArg)
	}

	// For now, return a mock response since we need YouTube Data API key for real search
	// In a real implementation, this would call the YouTube Data API
	mockResults := []map[string]interface{}{
		{
			"title":       fmt.Sprintf("Video about %s", query),
			"video_id":    "dQw4w9WgXcQ",
			"url":         "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			"channel":     "Example Channel",
			"description": fmt.Sprintf("This is a mock video about %s", query),
			"duration":    "4:30",
			"views":       "1,000,000",
			"published":   "2023-01-01",
		},
		{
			"title":       fmt.Sprintf("Tutorial: %s", query),
			"video_id":    "abc123def456",
			"url":         "https://www.youtube.com/watch?v=abc123def456",
			"channel":     "Tutorial Channel",
			"description": fmt.Sprintf("Learn how to %s with this tutorial", query),
			"duration":    "10:15",
			"views":       "500,000",
			"published":   "2023-02-15",
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
		"note":    "This is a placeholder implementation. Integrate with YouTube Data API for real search results.",
	}, nil
}

// getVideoInfo gets information about a specific YouTube video
func (y *YouTubeToolkit) getVideoInfo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	videoURL, ok := args["video_url"].(string)
	if !ok {
		return nil, fmt.Errorf("video_url must be a string")
	}

	// Validate URL
	parsedURL, err := url.Parse(videoURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Check if it's a YouTube URL
	if !strings.Contains(parsedURL.Host, "youtube.com") && !strings.Contains(parsedURL.Host, "youtu.be") {
		return nil, fmt.Errorf("URL must be a YouTube video URL")
	}

	// Extract video ID from URL
	var videoID string
	if strings.Contains(parsedURL.Host, "youtube.com") {
		queryParams := parsedURL.Query()
		videoID = queryParams.Get("v")
	} else if strings.Contains(parsedURL.Host, "youtu.be") {
		videoID = strings.TrimPrefix(parsedURL.Path, "/")
	}

	if videoID == "" {
		return nil, fmt.Errorf("could not extract video ID from URL")
	}

	// For now, return mock video info
	// In a real implementation, this would call the YouTube Data API
	mockVideoInfo := map[string]interface{}{
		"video_id":    videoID,
		"url":         videoURL,
		"title":       "Example YouTube Video",
		"channel":     "Example Channel",
		"description": "This is a mock video description. In a real implementation, this would fetch actual video metadata from YouTube Data API.",
		"duration":    "10:30",
		"views":       "2,500,000",
		"published":   "2023-03-20",
		"likes":       "150,000",
		"dislikes":    "5,000",
		"category":    "Education",
		"tags":        []string{"example", "tutorial", "learning"},
		"note":        "This is a placeholder implementation. Integrate with YouTube Data API for real video information.",
	}

	return mockVideoInfo, nil
}