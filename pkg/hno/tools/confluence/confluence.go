package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// ConfluenceToolkit provides Confluence API integration capabilities
// This is a basic implementation that can be extended with specific Confluence API endpoints

// ConfluenceToolkit provides Confluence API integration
type ConfluenceToolkit struct {
	*toolkit.BaseToolkit
	client *http.Client
}

// New creates a new Confluence toolkit
func New() *ConfluenceToolkit {
	t := &ConfluenceToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("confluence"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Register Confluence space list function
	t.RegisterFunction(&toolkit.Function{
		Name:        "list_spaces",
		Description: "List Confluence spaces accessible with the provided credentials",
		Parameters: map[string]toolkit.Parameter{
			"base_url": {
				Type:        "string",
				Description: "Confluence base URL (e.g., https://your-domain.atlassian.net/wiki)",
				Required:    true,
			},
			"username": {
				Type:        "string",
				Description: "Confluence username or email",
				Required:    true,
			},
			"api_token": {
				Type:        "string",
				Description: "Confluence API token",
				Required:    true,
			},
		},
		Handler: t.listSpaces,
	})

	// Register Confluence page search function
	t.RegisterFunction(&toolkit.Function{
		Name:        "search_pages",
		Description: "Search for pages in Confluence",
		Parameters: map[string]toolkit.Parameter{
			"base_url": {
				Type:        "string",
				Description: "Confluence base URL",
				Required:    true,
			},
			"username": {
				Type:        "string",
				Description: "Confluence username or email",
				Required:    true,
			},
			"api_token": {
				Type:        "string",
				Description: "Confluence API token",
				Required:    true,
			},
			"query": {
				Type:        "string",
				Description: "Search query",
				Required:    true,
			},
			"space_key": {
				Type:        "string",
				Description: "Space key to search in (optional)",
				Required:    false,
			},
		},
		Handler: t.searchPages,
	})

	// Register Confluence page content function
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_page_content",
		Description: "Get the content of a specific Confluence page",
		Parameters: map[string]toolkit.Parameter{
			"base_url": {
				Type:        "string",
				Description: "Confluence base URL",
				Required:    true,
			},
			"username": {
				Type:        "string",
				Description: "Confluence username or email",
				Required:    true,
			},
			"api_token": {
				Type:        "string",
				Description: "Confluence API token",
				Required:    true,
			},
			"page_id": {
				Type:        "string",
				Description: "Page ID",
				Required:    true,
			},
		},
		Handler: t.getPageContent,
	})

	return t
}

// listSpaces lists accessible Confluence spaces
func (c *ConfluenceToolkit) listSpaces(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, ok := args["base_url"].(string)
	if !ok {
		return nil, fmt.Errorf("base_url must be a string")
	}

	_, ok = args["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username must be a string")
	}

	_, ok = args["api_token"].(string)
	if !ok {
		return nil, fmt.Errorf("api_token must be a string")
	}

	// For now, return a mock response since we need actual Confluence API integration
	// In a real implementation, this would call the Confluence API
	mockSpaces := []map[string]interface{}{
		{
			"id":    "12345",
			"key":   "DS",
			"name":  "Documentation Space",
			"type":  "global",
			"_links": map[string]interface{}{
				"webui": "/spaces/DS",
			},
		},
		{
			"id":    "67890",
			"key":   "DEV",
			"name":  "Development Space",
			"type":  "global",
			"_links": map[string]interface{}{
				"webui": "/spaces/DEV",
			},
		},
	}

	return map[string]interface{}{
		"spaces": mockSpaces,
		"count":  len(mockSpaces),
		"note":   "This is a placeholder implementation. Integrate with Confluence API for real space data.",
	}, nil
}

// searchPages searches for pages in Confluence
func (c *ConfluenceToolkit) searchPages(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, ok := args["base_url"].(string)
	if !ok {
		return nil, fmt.Errorf("base_url must be a string")
	}

	_, ok = args["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username must be a string")
	}

	_, ok = args["api_token"].(string)
	if !ok {
		return nil, fmt.Errorf("api_token must be a string")
	}

	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	spaceKey := ""
	if spaceKeyArg, ok := args["space_key"].(string); ok {
		spaceKey = spaceKeyArg
	}

	// For now, return a mock response
	mockPages := []map[string]interface{}{
		{
			"id":      "123456",
			"title":   "Getting Started Guide",
			"space": map[string]interface{}{
				"key":  "DS",
				"name": "Documentation Space",
			},
			"version": map[string]interface{}{
				"number": 3,
			},
			"_links": map[string]interface{}{
				"webui": "/spaces/DS/pages/123456/Getting+Started+Guide",
			},
		},
		{
			"id":      "789012",
			"title":   "API Documentation",
			"space": map[string]interface{}{
				"key":  "DEV",
				"name": "Development Space",
			},
			"version": map[string]interface{}{
				"number": 5,
			},
			"_links": map[string]interface{}{
				"webui": "/spaces/DEV/pages/789012/API+Documentation",
			},
		},
	}

	result := map[string]interface{}{
		"query":     query,
		"pages":     mockPages,
		"count":     len(mockPages),
		"space_key": spaceKey,
		"note":      "This is a placeholder implementation. Integrate with Confluence API for real search results.",
	}

	return result, nil
}

// getPageContent gets the content of a specific Confluence page
func (c *ConfluenceToolkit) getPageContent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, ok := args["base_url"].(string)
	if !ok {
		return nil, fmt.Errorf("base_url must be a string")
	}

	_, ok = args["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username must be a string")
	}

	_, ok = args["api_token"].(string)
	if !ok {
		return nil, fmt.Errorf("api_token must be a string")
	}

	pageID, ok := args["page_id"].(string)
	if !ok {
		return nil, fmt.Errorf("page_id must be a string")
	}

	// For now, return a mock response
	mockContent := map[string]interface{}{
		"id":    pageID,
		"title": "Example Page Title",
		"space": map[string]interface{}{
			"key":  "DS",
			"name": "Documentation Space",
		},
		"body": map[string]interface{}{
			"storage": map[string]interface{}{
				"value": "<p>This is example page content for page " + pageID + ".</p><p>In a real implementation, this would contain the actual Confluence page content.</p>",
				"representation": "storage",
			},
		},
		"version": map[string]interface{}{
			"number":    2,
			"message":   "Updated documentation",
			"minorEdit": false,
		},
		"_links": map[string]interface{}{
			"webui": "/spaces/DS/pages/" + pageID + "/Example+Page+Title",
		},
	}

	return map[string]interface{}{
		"page": mockContent,
		"note": "This is a placeholder implementation. Integrate with Confluence API for real page content.",
	}, nil
}

// Helper function to make authenticated Confluence API requests
func (c *ConfluenceToolkit) makeConfluenceRequest(ctx context.Context, url, username, apiToken string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set basic auth for Confluence API
	req.SetBasicAuth(username, apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// Helper function to parse Confluence API response
func (c *ConfluenceToolkit) parseConfluenceResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Confluence API request failed with status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode Confluence API response: %w", err)
	}

	return nil
}