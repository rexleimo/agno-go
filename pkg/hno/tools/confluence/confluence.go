package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// ConfluenceToolkit provides Confluence Cloud REST API integration.
type ConfluenceToolkit struct {
	*toolkit.BaseToolkit
	client *http.Client
}

// New creates a Confluence toolkit. The base URL and credentials are supplied
// per tool call because installations and identities may differ.
func New() *ConfluenceToolkit {
	t := &ConfluenceToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("confluence"),
		client:      &http.Client{Timeout: 30 * time.Second},
	}

	t.RegisterFunction(&toolkit.Function{
		Name:        "list_spaces",
		Description: "List Confluence spaces accessible with the provided credentials",
		Parameters: map[string]toolkit.Parameter{
			"base_url":  {Type: "string", Description: "Confluence base URL, for example https://example.atlassian.net/wiki", Required: true},
			"username":  {Type: "string", Description: "Confluence username or email", Required: true},
			"api_token": {Type: "string", Description: "Confluence API token", Required: true},
		},
		Handler: t.listSpaces,
	})

	t.RegisterFunction(&toolkit.Function{
		Name:        "search_pages",
		Description: "Search for pages in Confluence using CQL",
		Parameters: map[string]toolkit.Parameter{
			"base_url":  {Type: "string", Description: "Confluence base URL", Required: true},
			"username":  {Type: "string", Description: "Confluence username or email", Required: true},
			"api_token": {Type: "string", Description: "Confluence API token", Required: true},
			"query":     {Type: "string", Description: "Full-text search query", Required: true},
			"space_key": {Type: "string", Description: "Optional space key", Required: false},
		},
		Handler: t.searchPages,
	})

	t.RegisterFunction(&toolkit.Function{
		Name:        "get_page_content",
		Description: "Get the storage-format content of a Confluence page",
		Parameters: map[string]toolkit.Parameter{
			"base_url":  {Type: "string", Description: "Confluence base URL", Required: true},
			"username":  {Type: "string", Description: "Confluence username or email", Required: true},
			"api_token": {Type: "string", Description: "Confluence API token", Required: true},
			"page_id":   {Type: "string", Description: "Page ID", Required: true},
		},
		Handler: t.getPageContent,
	})

	return t
}

type confluencePageResponse struct {
	Results []map[string]interface{} `json:"results"`
}

func confluenceString(args map[string]interface{}, name string) (string, error) {
	value, ok := args[name].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", name)
	}
	return strings.TrimSpace(value), nil
}

func confluenceEndpoint(baseURL, path string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid Confluence base_url: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("base_url must be an absolute http or https URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func escapeCQL(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func (c *ConfluenceToolkit) listSpaces(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	baseURL, err := confluenceString(args, "base_url")
	if err != nil {
		return nil, err
	}
	username, err := confluenceString(args, "username")
	if err != nil {
		return nil, err
	}
	token, err := confluenceString(args, "api_token")
	if err != nil {
		return nil, err
	}
	endpoint, err := confluenceEndpoint(baseURL, "/api/v2/spaces")
	if err != nil {
		return nil, err
	}

	resp, err := c.makeConfluenceRequest(ctx, endpoint, username, token)
	if err != nil {
		return nil, err
	}
	var payload confluencePageResponse
	if err := c.parseConfluenceResponse(resp, &payload); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"spaces": payload.Results,
		"count":  len(payload.Results),
	}, nil
}

func (c *ConfluenceToolkit) searchPages(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	baseURL, err := confluenceString(args, "base_url")
	if err != nil {
		return nil, err
	}
	username, err := confluenceString(args, "username")
	if err != nil {
		return nil, err
	}
	token, err := confluenceString(args, "api_token")
	if err != nil {
		return nil, err
	}
	query, err := confluenceString(args, "query")
	if err != nil {
		return nil, err
	}
	spaceKey, _ := args["space_key"].(string)
	spaceKey = strings.TrimSpace(spaceKey)

	endpoint, err := confluenceEndpoint(baseURL, "/rest/api/content/search")
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	cql := `type=page AND text~"` + escapeCQL(query) + `"`
	if spaceKey != "" {
		cql += ` AND space="` + escapeCQL(spaceKey) + `"`
	}
	params := parsed.Query()
	params.Set("cql", cql)
	params.Set("expand", "space,version")
	params.Set("limit", "25")
	parsed.RawQuery = params.Encode()

	resp, err := c.makeConfluenceRequest(ctx, parsed.String(), username, token)
	if err != nil {
		return nil, err
	}
	var payload confluencePageResponse
	if err := c.parseConfluenceResponse(resp, &payload); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"query":     query,
		"pages":     payload.Results,
		"count":     len(payload.Results),
		"space_key": spaceKey,
	}, nil
}

func (c *ConfluenceToolkit) getPageContent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	baseURL, err := confluenceString(args, "base_url")
	if err != nil {
		return nil, err
	}
	username, err := confluenceString(args, "username")
	if err != nil {
		return nil, err
	}
	token, err := confluenceString(args, "api_token")
	if err != nil {
		return nil, err
	}
	pageID, err := confluenceString(args, "page_id")
	if err != nil {
		return nil, err
	}
	endpoint, err := confluenceEndpoint(baseURL, "/rest/api/content/"+url.PathEscape(pageID))
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	params := parsed.Query()
	params.Set("expand", "body.storage,space,version")
	parsed.RawQuery = params.Encode()

	resp, err := c.makeConfluenceRequest(ctx, parsed.String(), username, token)
	if err != nil {
		return nil, err
	}
	var page map[string]interface{}
	if err := c.parseConfluenceResponse(resp, &page); err != nil {
		return nil, err
	}
	return map[string]interface{}{"page": page}, nil
}

func (c *ConfluenceToolkit) makeConfluenceRequest(ctx context.Context, endpoint, username, apiToken string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Confluence request: %w", err)
	}
	req.SetBasicAuth(username, apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Confluence request failed: %w", err)
	}
	return resp, nil
}

func (c *ConfluenceToolkit) parseConfluenceResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		message := strings.TrimSpace(string(body))
		if message == "" {
			return fmt.Errorf("Confluence API request failed with status: %d", resp.StatusCode)
		}
		return fmt.Errorf("Confluence API request failed with status: %d: %s", resp.StatusCode, message)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode Confluence API response: %w", err)
	}
	return nil
}
