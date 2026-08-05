package hackernews

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newMockHN builds a toolkit against a local server that mimics the
// Hacker News Firebase API: /topstories.json returns story IDs and
// /item/{id}.json returns story objects. Deterministic, no external network.
//
// newMockHN 构建指向本地模拟服务器的 toolkit：/topstories.json 返回
// story ID 列表，/item/{id}.json 返回 story 对象。确定性，无外部网络。
func newMockHN() (*HackerNewsToolkit, func()) {
	mux := http.NewServeMux()
	mux.HandleFunc("/topstories.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[1, 2, 3]`)
	})
	mux.HandleFunc("/item/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/item/"), ".json")
		fmt.Fprintf(w, `{"id": %s, "title": "Story %s", "by": "user%s", "type": "story", "score": 42}`, id, id, id)
	})
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"hits":[{"objectID":"42","title":"Real HN search result","url":"https://example.com/story","author":"alice","points":99,"created_at_i":1700000000,"num_comments":12}],"nbHits":1}`)
	})
	srv := httptest.NewServer(mux)
	return NewWithBase(srv.URL), srv.Close
}

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
	toolkit, close := newMockHN()
	defer close()
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

	// Mock returns exactly 3 stories.
	// Mock 精确返回 3 个故事。
	if len(stories) != 3 {
		t.Errorf("Expected 3 stories, got %d", len(stories))
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
	toolkit, close := newMockHN()
	defer close()
	ctx := context.Background()

	// Test getting top stories with custom limit
	result, err := toolkit.Execute(ctx, "get_top_stories", map[string]interface{}{
		"limit": 2.0, // JSON numbers come as float64
	})

	if err != nil {
		t.Fatalf("Get top stories failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	count, ok := resultMap["count"].(int)
	if !ok {
		t.Fatalf("Expected count to be int, got %T", resultMap["count"])
	}

	if count != 2 {
		t.Errorf("Expected 2 stories (limit), got %d", count)
	}
}

func TestHackerNewsToolkit_GetStoryDetails(t *testing.T) {
	toolkit, close := newMockHN()
	defer close()
	ctx := context.Background()

	// Test getting story details for a known story ID
	result, err := toolkit.Execute(ctx, "get_story_details", map[string]interface{}{
		"story_id": 1.0,
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

	if story["title"] == "" {
		t.Errorf("Expected story title, got empty (story=%v)", story)
	}
}

func TestHackerNewsToolkit_GetStoryDetailsInvalidID(t *testing.T) {
	toolkit, close := newMockHN()
	defer close()
	ctx := context.Background()

	// Test getting story details for a non-existent story ID
	_, err := toolkit.Execute(ctx, "get_story_details", map[string]interface{}{
		"story_id": -1.0, // Invalid story ID
	})

	// This might succeed but return an empty story, or fail
	// Either is acceptable for this test
	if err != nil {
		t.Logf("Invalid ID correctly errored: %v", err)
	}
}

func TestHackerNewsToolkit_SearchStories(t *testing.T) {
	toolkit, close := newMockHN()
	defer close()

	result, err := toolkit.Execute(context.Background(), "search_stories", map[string]interface{}{
		"query": "go",
		"limit": 1.0,
	})
	if err != nil {
		t.Fatalf("search_stories failed: %v", err)
	}
	resultMap := result.(map[string]interface{})
	results := resultMap["results"].([]map[string]interface{})
	if len(results) != 1 || results[0]["title"] != "Real HN search result" {
		t.Fatalf("unexpected search results: %#v", results)
	}
	if resultMap["total_hits"] != 1 {
		t.Fatalf("unexpected total hit count: %#v", resultMap["total_hits"])
	}
}
