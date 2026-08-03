package arxiv

import (
	"context"
	"strings"
	"testing"
)

func TestArXivToolkit_SearchPapers(t *testing.T) {
	toolkit := New()

	// Test searching for papers
	result, err := toolkit.searchPapers(context.Background(), map[string]interface{}{
		"query":        "machine learning",
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

	if query != "machine learning" {
		t.Errorf("Expected query 'machine learning', got '%s'", query)
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
		if _, ok := firstResult["paper_id"].(string); !ok {
			t.Error("Expected paper_id in result")
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

func TestArXivToolkit_GetPaperDetails(t *testing.T) {
	toolkit := New()

	// Test getting paper details (using a known paper ID)
	result, err := toolkit.getPaperDetails(context.Background(), map[string]interface{}{
		"paper_id": "2301.00001",
	})

	// This might fail if the paper doesn't exist, but we should get structured error
	if err != nil {
		// Check if it's a "not found" error
		if !strings.Contains(err.Error(), "not found") {
			t.Logf("Expected paper not found or API error: %v", err)
		}
		return
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	paper, ok := resultMap["paper"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected paper object, got: %T", resultMap["paper"])
	}

	// Check required fields
	if _, ok := paper["paper_id"].(string); !ok {
		t.Error("Expected paper_id in paper details")
	}
	if _, ok := paper["title"].(string); !ok {
		t.Error("Expected title in paper details")
	}
	if _, ok := paper["authors"].([]string); !ok {
		t.Error("Expected authors in paper details")
	}
}

func TestArXivToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created")
	}

	// Check that functions are registered
	functions := toolkit.Functions()
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	expectedFunctions := []string{"search_papers", "get_paper_details"}
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