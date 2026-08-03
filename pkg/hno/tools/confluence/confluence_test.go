package confluence

import (
	"context"
	"testing"
)

func TestConfluenceToolkit_ListSpaces(t *testing.T) {
	toolkit := New()

	// Test with valid parameters
	result, err := toolkit.listSpaces(context.Background(), map[string]interface{}{
		"base_url":  "https://example.atlassian.net/wiki",
		"username":  "testuser@example.com",
		"api_token": "test-token",
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check that spaces are returned
	spaces, ok := resultMap["spaces"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected spaces array, got: %T", resultMap["spaces"])
	}

	if len(spaces) == 0 {
		t.Error("Expected at least one space")
	}

	// Check count field
	count, ok := resultMap["count"].(int)
	if !ok {
		t.Fatalf("Expected count integer, got: %T", resultMap["count"])
	}

	if count != len(spaces) {
		t.Errorf("Count mismatch: expected %d, got %d", len(spaces), count)
	}
}

func TestConfluenceToolkit_SearchPages(t *testing.T) {
	toolkit := New()

	// Test with valid parameters
	result, err := toolkit.searchPages(context.Background(), map[string]interface{}{
		"base_url":  "https://example.atlassian.net/wiki",
		"username":  "testuser@example.com",
		"api_token": "test-token",
		"query":     "documentation",
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check that pages are returned
	pages, ok := resultMap["pages"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected pages array, got: %T", resultMap["pages"])
	}

	if len(pages) == 0 {
		t.Error("Expected at least one page")
	}

	// Check query field
	query, ok := resultMap["query"].(string)
	if !ok {
		t.Fatalf("Expected query string, got: %T", resultMap["query"])
	}

	if query != "documentation" {
		t.Errorf("Expected query 'documentation', got '%s'", query)
	}
}

func TestConfluenceToolkit_SearchPages_WithSpaceKey(t *testing.T) {
	toolkit := New()

	// Test with space key parameter
	result, err := toolkit.searchPages(context.Background(), map[string]interface{}{
		"base_url":  "https://example.atlassian.net/wiki",
		"username":  "testuser@example.com",
		"api_token": "test-token",
		"query":     "documentation",
		"space_key": "DS",
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check space_key field
	spaceKey, ok := resultMap["space_key"].(string)
	if !ok {
		t.Fatalf("Expected space_key string, got: %T", resultMap["space_key"])
	}

	if spaceKey != "DS" {
		t.Errorf("Expected space_key 'DS', got '%s'", spaceKey)
	}
}

func TestConfluenceToolkit_GetPageContent(t *testing.T) {
	toolkit := New()

	// Test with valid parameters
	result, err := toolkit.getPageContent(context.Background(), map[string]interface{}{
		"base_url":  "https://example.atlassian.net/wiki",
		"username":  "testuser@example.com",
		"api_token": "test-token",
		"page_id":   "123456",
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check that page content is returned
	page, ok := resultMap["page"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected page map, got: %T", resultMap["page"])
	}

	// Check page ID
	pageID, ok := page["id"].(string)
	if !ok {
		t.Fatalf("Expected page id string, got: %T", page["id"])
	}

	if pageID != "123456" {
		t.Errorf("Expected page id '123456', got '%s'", pageID)
	}
}

func TestConfluenceToolkit_ListSpaces_MissingParameters(t *testing.T) {
	toolkit := New()

	// Test missing base_url
	_, err := toolkit.listSpaces(context.Background(), map[string]interface{}{
		"username":  "testuser@example.com",
		"api_token": "test-token",
	})

	if err == nil {
		t.Error("Expected error for missing base_url")
	}

	// Test missing username
	_, err = toolkit.listSpaces(context.Background(), map[string]interface{}{
		"base_url":  "https://example.atlassian.net/wiki",
		"api_token": "test-token",
	})

	if err == nil {
		t.Error("Expected error for missing username")
	}

	// Test missing api_token
	_, err = toolkit.listSpaces(context.Background(), map[string]interface{}{
		"base_url": "https://example.atlassian.net/wiki",
		"username": "testuser@example.com",
	})

	if err == nil {
		t.Error("Expected error for missing api_token")
	}
}

func TestConfluenceToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created")
	}

	// Check that functions are registered
	functions := toolkit.Functions()
	if len(functions) != 3 {
		t.Errorf("Expected 3 functions, got %d", len(functions))
	}

	expectedFunctions := []string{"list_spaces", "search_pages", "get_page_content"}
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