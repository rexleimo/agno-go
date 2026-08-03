package youtube

import (
	"context"
	"testing"
)

func TestYouTubeToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created, got nil")
	}

	if toolkit.Name() != "youtube" {
		t.Errorf("Expected toolkit name 'youtube', got '%s'", toolkit.Name())
	}

	functions := toolkit.Functions()
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	if _, exists := functions["search_youtube"]; !exists {
		t.Error("Expected 'search_youtube' function to exist")
	}

	if _, exists := functions["get_video_info"]; !exists {
		t.Error("Expected 'get_video_info' function to exist")
	}
}

func TestYouTubeToolkit_SearchYouTube(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test basic search
	result, err := toolkit.Execute(ctx, "search_youtube", map[string]interface{}{
		"query": "machine learning tutorial",
	})

	if err != nil {
		t.Fatalf("YouTube search failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["query"] != "machine learning tutorial" {
		t.Errorf("Expected query 'machine learning tutorial', got '%v'", resultMap["query"])
	}

	results, ok := resultMap["results"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected results slice, got %T", resultMap["results"])
	}

	if len(results) == 0 {
		t.Error("Expected at least one mock result")
	}

	// Check first result structure
	firstResult := results[0]
	if firstResult["title"] == "" {
		t.Error("Expected title in result")
	}
	if firstResult["video_id"] == "" {
		t.Error("Expected video_id in result")
	}
	if firstResult["url"] == "" {
		t.Error("Expected url in result")
	}
}

func TestYouTubeToolkit_SearchYouTubeWithMaxResults(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test search with max_results
	result, err := toolkit.Execute(ctx, "search_youtube", map[string]interface{}{
		"query":       "test query",
		"max_results": 1.0, // JSON numbers come as float64
	})

	if err != nil {
		t.Fatalf("YouTube search failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	count, ok := resultMap["count"].(int)
	if !ok {
		t.Fatalf("Expected count to be int, got %T", resultMap["count"])
	}

	if count > 1 {
		t.Errorf("Expected max 1 result, got %d", count)
	}
}

func TestYouTubeToolkit_SearchYouTubeMissingQuery(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test missing required parameter
	_, err := toolkit.Execute(ctx, "search_youtube", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for missing query parameter")
	}
}

func TestYouTubeToolkit_GetVideoInfo(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test getting video info
	result, err := toolkit.Execute(ctx, "get_video_info", map[string]interface{}{
		"video_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	})

	if err != nil {
		t.Fatalf("Get video info failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["video_id"] != "dQw4w9WgXcQ" {
		t.Errorf("Expected video_id 'dQw4w9WgXcQ', got '%v'", resultMap["video_id"])
	}

	if resultMap["url"] != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" {
		t.Errorf("Expected URL 'https://www.youtube.com/watch?v=dQw4w9WgXcQ', got '%v'", resultMap["url"])
	}

	if resultMap["title"] == "" {
		t.Error("Expected title in video info")
	}
}

func TestYouTubeToolkit_GetVideoInfoShortURL(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test getting video info with short URL
	result, err := toolkit.Execute(ctx, "get_video_info", map[string]interface{}{
		"video_url": "https://youtu.be/dQw4w9WgXcQ",
	})

	if err != nil {
		t.Fatalf("Get video info failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["video_id"] != "dQw4w9WgXcQ" {
		t.Errorf("Expected video_id 'dQw4w9WgXcQ', got '%v'", resultMap["video_id"])
	}
}

func TestYouTubeToolkit_GetVideoInfoInvalidURL(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test invalid URL
	_, err := toolkit.Execute(ctx, "get_video_info", map[string]interface{}{
		"video_url": "not-a-valid-url",
	})

	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestYouTubeToolkit_GetVideoInfoNonYouTubeURL(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test non-YouTube URL
	_, err := toolkit.Execute(ctx, "get_video_info", map[string]interface{}{
		"video_url": "https://example.com/video",
	})

	if err == nil {
		t.Error("Expected error for non-YouTube URL")
	}
}

func TestYouTubeToolkit_GetVideoInfoMissingURL(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test missing required parameter
	_, err := toolkit.Execute(ctx, "get_video_info", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for missing video_url parameter")
	}
}