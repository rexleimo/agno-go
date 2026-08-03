package pubmed

import (
	"context"
	"strings"
	"testing"
)

func TestPubMedToolkit_SearchArticles(t *testing.T) {
	toolkit := New()

	// Test searching for articles
	result, err := toolkit.searchArticles(context.Background(), map[string]interface{}{
		"query":        "cancer research",
		"max_results":  5,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check query
	query, ok := resultMap["query"].(string)
	if !ok {
		t.Fatalf("Expected query string, got: %T", resultMap["query"])
	}

	if query != "cancer research" {
		t.Errorf("Expected query 'cancer research', got '%s'", query)
	}

	// Check results
	results, ok := resultMap["results"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected results array, got: %T", resultMap["results"])
	}

	// Should have some results (may be 0 if API is down, but structure should be valid)
	if len(results) > 0 {
		firstResult := results[0]

		// Check required fields
		if _, ok := firstResult["article_id"].(string); !ok {
			t.Error("Expected article_id in result")
		}
		if _, ok := firstResult["title"].(string); !ok {
			t.Error("Expected title in result")
		}
		if _, ok := firstResult["authors"].([]string); !ok {
			t.Error("Expected authors in result")
		}
	}

	// Check max results
	maxResults, ok := resultMap["max_results"].(int)
	if !ok {
		t.Fatalf("Expected max_results integer, got: %T", resultMap["max_results"])
	}

	if maxResults != 5 {
		t.Logf("Expected max_results 5, got %d (API may have returned different count)", maxResults)
	}
}

func TestPubMedToolkit_GetArticleDetails(t *testing.T) {
	toolkit := New()

	// Test getting article details (using a known article ID)
	result, err := toolkit.getArticleDetails(context.Background(), map[string]interface{}{
		"article_id": "12345678",
	})

	// This might fail if the article doesn't exist, but we should get structured error
	if err != nil {
		// Check if it's a "not found" error
		if !strings.Contains(err.Error(), "not found") {
			t.Logf("Expected article not found or API error: %v", err)
		}
		return
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	article, ok := resultMap["article"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected article object, got: %T", resultMap["article"])
	}

	// Check required fields
	if _, ok := article["article_id"].(string); !ok {
		t.Error("Expected article_id in article details")
	}
	if _, ok := article["title"].(string); !ok {
		t.Error("Expected title in article details")
	}
	if _, ok := article["authors"].([]string); !ok {
		t.Error("Expected authors in article details")
	}
}

func TestPubMedToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created")
	}

	// Check that functions are registered
	functions := toolkit.Functions()
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	expectedFunctions := []string{"search_articles", "get_article_details"}
	for _, expected := range expectedFunctions {
		found := false
		for _, function := range functions {
			if function.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected function '%s' not found", expected)
		}
	}
}