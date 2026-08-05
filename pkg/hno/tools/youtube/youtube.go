package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

const defaultBaseURL = "https://www.googleapis.com/youtube/v3"

// Config configures access to the YouTube Data API v3.
type Config struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

// YouTubeToolkit provides YouTube Data API video search and metadata.
type YouTubeToolkit struct {
	*toolkit.BaseToolkit
	client  *http.Client
	apiKey  string
	baseURL string
}

// New creates a toolkit using YOUTUBE_API_KEY from the environment.
func New() *YouTubeToolkit {
	return NewWithConfig(Config{APIKey: os.Getenv("YOUTUBE_API_KEY")})
}

// NewWithBase creates a toolkit against a custom API base URL. The API key is
// read from YOUTUBE_API_KEY; use NewWithConfig for explicit configuration.
func NewWithBase(baseURL string) *YouTubeToolkit {
	return NewWithConfig(Config{APIKey: os.Getenv("YOUTUBE_API_KEY"), BaseURL: baseURL})
}

// NewWithConfig creates a toolkit with explicit API settings.
func NewWithConfig(config Config) *YouTubeToolkit {
	if strings.TrimSpace(config.BaseURL) == "" {
		config.BaseURL = defaultBaseURL
	}
	if config.Client == nil {
		config.Client = &http.Client{Timeout: 30 * time.Second}
	}

	t := &YouTubeToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("youtube"),
		client:      config.Client,
		apiKey:      strings.TrimSpace(config.APIKey),
		baseURL:     strings.TrimRight(config.BaseURL, "/"),
	}

	t.RegisterFunction(&toolkit.Function{
		Name:        "search_youtube",
		Description: "Search for YouTube videos using the YouTube Data API",
		Parameters: map[string]toolkit.Parameter{
			"query":       {Type: "string", Description: "The search query", Required: true},
			"max_results": {Type: "integer", Description: "Maximum number of results to return (default: 5)", Required: false, Default: 5},
		},
		Handler: t.searchYouTube,
	})

	t.RegisterFunction(&toolkit.Function{
		Name:        "get_video_info",
		Description: "Get information about a YouTube video using the YouTube Data API",
		Parameters: map[string]toolkit.Parameter{
			"video_url": {Type: "string", Description: "The YouTube video URL", Required: true},
		},
		Handler: t.getVideoInfo,
	})

	return t
}

type youtubeSnippet struct {
	PublishedAt  string   `json:"publishedAt"`
	ChannelID    string   `json:"channelId"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	ChannelTitle string   `json:"channelTitle"`
	CategoryID   string   `json:"categoryId"`
	Tags         []string `json:"tags"`
}

type youtubeSearchItem struct {
	ID struct {
		VideoID string `json:"videoId"`
	} `json:"id"`
	Snippet youtubeSnippet `json:"snippet"`
}

type youtubeSearchResponse struct {
	Items []youtubeSearchItem `json:"items"`
}

type youtubeVideoItem struct {
	ID             string         `json:"id"`
	Snippet        youtubeSnippet `json:"snippet"`
	ContentDetails struct {
		Duration string `json:"duration"`
	} `json:"contentDetails"`
	Statistics struct {
		ViewCount    string `json:"viewCount"`
		LikeCount    string `json:"likeCount"`
		CommentCount string `json:"commentCount"`
	} `json:"statistics"`
}

type youtubeVideoResponse struct {
	Items []youtubeVideoItem `json:"items"`
}

func youtubeString(args map[string]interface{}, name string) (string, error) {
	value, ok := args[name].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", name)
	}
	return strings.TrimSpace(value), nil
}

func youtubeLimit(args map[string]interface{}) (int, error) {
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
	if limit < 1 || limit > 50 {
		return 0, fmt.Errorf("max_results must be between 1 and 50")
	}
	return limit, nil
}

func (y *YouTubeToolkit) searchYouTube(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, err := youtubeString(args, "query")
	if err != nil {
		return nil, err
	}
	limit, err := youtubeLimit(args)
	if err != nil {
		return nil, err
	}
	if y.apiKey == "" {
		return nil, fmt.Errorf("YouTube API key is required; set YOUTUBE_API_KEY or configure the toolkit")
	}

	endpoint := y.baseURL + "/search"
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid YouTube API base URL: %w", err)
	}
	params := parsed.Query()
	params.Set("part", "snippet")
	params.Set("type", "video")
	params.Set("q", query)
	params.Set("maxResults", strconv.Itoa(limit))
	params.Set("key", y.apiKey)
	parsed.RawQuery = params.Encode()

	var payload youtubeSearchResponse
	if err := y.doJSON(ctx, parsed.String(), &payload); err != nil {
		return nil, err
	}
	results := make([]map[string]interface{}, 0, len(payload.Items))
	for _, item := range payload.Items {
		if item.ID.VideoID == "" {
			continue
		}
		results = append(results, map[string]interface{}{
			"video_id":    item.ID.VideoID,
			"url":         "https://www.youtube.com/watch?v=" + item.ID.VideoID,
			"title":       item.Snippet.Title,
			"channel":     item.Snippet.ChannelTitle,
			"channel_id":  item.Snippet.ChannelID,
			"description": item.Snippet.Description,
			"published":   item.Snippet.PublishedAt,
		})
	}
	return map[string]interface{}{
		"query":   query,
		"results": results,
		"count":   len(results),
	}, nil
}

func (y *YouTubeToolkit) getVideoInfo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	videoURL, err := youtubeString(args, "video_url")
	if err != nil {
		return nil, err
	}
	videoID, err := extractVideoID(videoURL)
	if err != nil {
		return nil, err
	}
	if y.apiKey == "" {
		return nil, fmt.Errorf("YouTube API key is required; set YOUTUBE_API_KEY or configure the toolkit")
	}

	endpoint := y.baseURL + "/videos"
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid YouTube API base URL: %w", err)
	}
	params := parsed.Query()
	params.Set("part", "snippet,contentDetails,statistics")
	params.Set("id", videoID)
	params.Set("key", y.apiKey)
	parsed.RawQuery = params.Encode()

	var payload youtubeVideoResponse
	if err := y.doJSON(ctx, parsed.String(), &payload); err != nil {
		return nil, err
	}
	if len(payload.Items) == 0 {
		return nil, fmt.Errorf("YouTube video %q was not found", videoID)
	}
	item := payload.Items[0]
	return map[string]interface{}{
		"video_id":         videoID,
		"url":              videoURL,
		"title":            item.Snippet.Title,
		"channel":          item.Snippet.ChannelTitle,
		"channel_id":       item.Snippet.ChannelID,
		"description":      item.Snippet.Description,
		"duration":         item.ContentDetails.Duration,
		"duration_seconds": iso8601Seconds(item.ContentDetails.Duration),
		"views":            item.Statistics.ViewCount,
		"likes":            item.Statistics.LikeCount,
		"comments":         item.Statistics.CommentCount,
		"published":        item.Snippet.PublishedAt,
		"category":         item.Snippet.CategoryID,
		"tags":             item.Snippet.Tags,
	}, nil
}

func (y *YouTubeToolkit) doJSON(ctx context.Context, endpoint string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create YouTube request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := y.client.Do(req)
	if err != nil {
		return fmt.Errorf("YouTube request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		message := strings.TrimSpace(string(body))
		if message == "" {
			return fmt.Errorf("YouTube API request failed with status: %d", resp.StatusCode)
		}
		return fmt.Errorf("YouTube API request failed with status: %d: %s", resp.StatusCode, message)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode YouTube API response: %w", err)
	}
	return nil
}

func extractVideoID(videoURL string) (string, error) {
	parsed, err := url.Parse(videoURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	host := strings.ToLower(parsed.Hostname())
	var videoID string
	switch {
	case host == "youtu.be":
		videoID = strings.Trim(strings.TrimPrefix(parsed.Path, "/"), "/")
	case host == "youtube.com" || strings.HasSuffix(host, ".youtube.com"):
		videoID = parsed.Query().Get("v")
		if videoID == "" {
			parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
			if len(parts) == 2 && (parts[0] == "shorts" || parts[0] == "embed" || parts[0] == "live") {
				videoID = parts[1]
			}
		}
	default:
		return "", fmt.Errorf("URL must be a YouTube video URL")
	}
	if videoID == "" || strings.ContainsAny(videoID, "/?#") {
		return "", fmt.Errorf("could not extract video ID from URL")
	}
	return videoID, nil
}

var iso8601Duration = regexp.MustCompile(`^P(?:([0-9]+)D)?T(?:([0-9]+)H)?(?:([0-9]+)M)?(?:([0-9]+(?:\.[0-9]+)?)S)?$`)

func iso8601Seconds(value string) int64 {
	matches := iso8601Duration.FindStringSubmatch(value)
	if matches == nil {
		return 0
	}
	parse := func(raw string) float64 {
		if raw == "" {
			return 0
		}
		result, _ := strconv.ParseFloat(raw, 64)
		return result
	}
	return int64(parse(matches[1])*86400 + parse(matches[2])*3600 + parse(matches[3])*60 + parse(matches[4]))
}
