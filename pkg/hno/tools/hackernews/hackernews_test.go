package hackernews

import (
	"context"
	"testing"
)

func TestHackerNewsToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created, got nil")
	}

	if toolkit.Name() != "hacker_news" {
		t.Errorf("Expected toolkit name 'hacker_news', got '%s'", toolkit.Name())
	}

	functions := toolkit.Functions()
	if len(functions) != 3 {
		t.Errorf("Expected 3 functions, got %d", len(functions))
	}

	if _, exists := functions["get_top_stories"]; !exists {
		t.Error("Expected 'get_top_stories' function to exist")
	}

	if _, exists := functions["get_story_details"]; !exists {
		t.Error("Expected 'get_story_details' function to exist")
	}

	if _, exists := functions["search_stories"]; !exists {
		t.Error("Expected 'search_stories' function to exist")
	}
}

func TestHackerNewsToolkit_GetTopStories(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test getting top stories with default limit
	result, err := toolkit.Execute(ctx, "get_top_stories", map[string]interface{}{})

	if err != nil {
		t.Fatalf("Get top stories failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	stories, ok := resultMap["stories"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected stories slice, got %T", resultMap["stories"])
	}

	// Should have some stories (actual number depends on API response)
	if len(stories) == 0 {
		t.Log("Warning: No stories returned from Hacker News API")
	}

	count, ok := resultMap["count"].(int)
	if !ok {
		t.Fatalf("Expected count to be int, got %T", resultMap["count"])
	}

	if count != len(stories) {
		t.Errorf("Count %d doesn't match stories length %d", count, len(stories))
	}
}

func TestHackerNewsToolkit_GetTopStoriesWithLimit(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test getting top stories with custom limit
	result, err := toolkit.Execute(ctx, "get_top_stories", map[string]interface{}{
		"limit": 5.0, // JSON numbers come as float64
	})

	if err != nil {
		t.Fatalf("Get top stories failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	stories, ok := resultMap["stories"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected stories slice, got %T", resultMap["stories"])
	}

	count, ok := resultMap["count"].(int)
	if !ok {
		t.Fatalf("Expected count to be int, got %T", resultMap["count"])
	}

	if count > 5 {
		t.Errorf("Expected max 5 stories, got %d", count)
	}

	if count != len(stories) {
		t.Errorf("Count %d doesn't match stories length %d", count, len(stories))
	}
}

func TestHackerNewsToolkit_GetStoryDetails(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test getting story details for a known story ID
	// Using a story ID that likely exists
	result, err := toolkit.Execute(ctx, "get_story_details", map[string]interface{}{
		"story_id": 1.0, // First Hacker News story
	})

	if err != nil {
		t.Fatalf("Get story details failed: %v", err)
	}

	story, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if story["id"] == nil {
		t.Error("Expected story to have an ID")
	}

	// Check that we got some basic fields
	if story["title"] == "" && story["text"] == "" {
		t.Log("Warning: Story has no title or text")
	}
}

func TestHackerNewsToolkit_GetStoryDetailsInvalidID(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test getting story details for a non-existent story ID
	_, err := toolkit.Execute(ctx, "get_story_details", map[string]interface{}{
		"story_id": -1.0, // Invalid story ID
	})

	// This might succeed but return an empty story, or fail
	// Either is acceptable for this test
	if err != nil {
		t.Logf("Get story details failed as expected: %v", err)
	}
}

func TestHackerNewsToolkit_GetStoryDetailsMissingID(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test missing required parameter
	_, err := toolkit.Execute(ctx, "get_story_details", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for missing story_id parameter")
	}
}

func TestHackerNewsToolkit_SearchStories(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test searching stories
	result, err := toolkit.Execute(ctx, "search_stories", map[string]interface{}{
		"query": "artificial intelligence",
	})

	if err != nil {
		t.Fatalf("Search stories failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["query"] != "artificial intelligence" {
		t.Errorf("Expected query 'artificial intelligence', got '%v'", resultMap["query"])
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
	if firstResult["id"] == nil {
		t.Error("Expected id in result")
	}
}

func TestHackerNewsToolkit_SearchStoriesWithLimit(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test searching stories with limit
	result, err := toolkit.Execute(ctx, "search_stories", map[string]interface{}{
		"query": "test query",
		"limit": 1.0, // JSON numbers come as float64
	})

	if err != nil {
		t.Fatalf("Search stories failed: %v", err)
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

func TestHackerNewsToolkit_SearchStoriesMissingQuery(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test missing required parameter
	_, err := toolkit.Execute(ctx, "search_stories", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for missing query parameter")
	}
}