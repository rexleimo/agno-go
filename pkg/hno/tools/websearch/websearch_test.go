package websearch

import (
	"context"
	"testing"
)

func TestWebSearchToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created, got nil")
	}

	if toolkit.Name() != "web_search" {
		t.Errorf("Expected toolkit name 'web_search', got '%s'", toolkit.Name())
	}

	functions := toolkit.Functions()
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	if _, exists := functions["web_search"]; !exists {
		t.Error("Expected 'web_search' function to exist")
	}

	if _, exists := functions["extract_web_content"]; !exists {
		t.Error("Expected 'extract_web_content' function to exist")
	}
}

func TestWebSearchToolkit_WebSearch(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test basic search
	result, err := toolkit.Execute(ctx, "web_search", map[string]interface{}{
		"query": "test query",
	})

	if err != nil {
		t.Fatalf("Web search failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["query"] != "test query" {
		t.Errorf("Expected query 'test query', got '%v'", resultMap["query"])
	}

	results, ok := resultMap["results"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected results slice, got %T", resultMap["results"])
	}

	if len(results) == 0 {
		t.Error("Expected at least one mock result")
	}
}

func TestWebSearchToolkit_WebSearchWithMaxResults(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test search with max_results
	result, err := toolkit.Execute(ctx, "web_search", map[string]interface{}{
		"query":       "test query",
		"max_results": 1.0, // JSON numbers come as float64
	})

	if err != nil {
		t.Fatalf("Web search failed: %v", err)
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

func TestWebSearchToolkit_WebSearchMissingQuery(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test missing required parameter
	_, err := toolkit.Execute(ctx, "web_search", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for missing query parameter")
	}
}

func TestWebSearchToolkit_ExtractWebContent(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test URL content extraction with a simple URL
	// Note: This will make an actual HTTP request
	result, err := toolkit.Execute(ctx, "extract_web_content", map[string]interface{}{
		"url": "https://httpbin.org/get",
	})

	if err != nil {
		t.Fatalf("Extract web content failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["url"] != "https://httpbin.org/get" {
		t.Errorf("Expected URL 'https://httpbin.org/get', got '%v'", resultMap["url"])
	}

	if resultMap["status"] != 200 {
		t.Errorf("Expected status 200, got %v", resultMap["status"])
	}
}

func TestWebSearchToolkit_ExtractWebContentInvalidURL(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test invalid URL
	_, err := toolkit.Execute(ctx, "extract_web_content", map[string]interface{}{
		"url": "not-a-valid-url",
	})

	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestWebSearchToolkit_ExtractWebContentMissingURL(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test missing required parameter
	_, err := toolkit.Execute(ctx, "extract_web_content", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for missing URL parameter")
	}
}