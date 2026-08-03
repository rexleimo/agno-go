package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// BitbucketToolkit provides Bitbucket API integration capabilities
// This is a basic implementation that can be extended with specific Bitbucket API endpoints

// BitbucketToolkit provides Bitbucket API integration
type BitbucketToolkit struct {
	*toolkit.BaseToolkit
	client *http.Client
}

// New creates a new Bitbucket toolkit
func New() *BitbucketToolkit {
	t := &BitbucketToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("bitbucket"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Register Bitbucket workspace list function
	t.RegisterFunction(&toolkit.Function{
		Name:        "list_workspaces",
		Description: "List Bitbucket workspaces accessible with the provided credentials",
		Parameters: map[string]toolkit.Parameter{
			"username": {
				Type:        "string",
				Description: "Bitbucket username",
				Required:    true,
			},
			"app_password": {
				Type:        "string",
				Description: "Bitbucket app password",
				Required:    true,
			},
		},
		Handler: t.listWorkspaces,
	})

	// Register Bitbucket repository list function
	t.RegisterFunction(&toolkit.Function{
		Name:        "list_repositories",
		Description: "List repositories in a Bitbucket workspace",
		Parameters: map[string]toolkit.Parameter{
			"username": {
				Type:        "string",
				Description: "Bitbucket username",
				Required:    true,
			},
			"app_password": {
				Type:        "string",
				Description: "Bitbucket app password",
				Required:    true,
			},
			"workspace": {
				Type:        "string",
				Description: "Workspace slug",
				Required:    true,
			},
		},
		Handler: t.listRepositories,
	})

	// Register Bitbucket pull request list function
	t.RegisterFunction(&toolkit.Function{
		Name:        "list_pull_requests",
		Description: "List pull requests in a Bitbucket repository",
		Parameters: map[string]toolkit.Parameter{
			"username": {
				Type:        "string",
				Description: "Bitbucket username",
				Required:    true,
			},
			"app_password": {
				Type:        "string",
				Description: "Bitbucket app password",
				Required:    true,
			},
			"workspace": {
				Type:        "string",
				Description: "Workspace slug",
				Required:    true,
			},
			"repository": {
				Type:        "string",
				Description: "Repository slug",
				Required:    true,
			},
		},
		Handler: t.listPullRequests,
	})

	return t
}

// listWorkspaces lists accessible Bitbucket workspaces
func (b *BitbucketToolkit) listWorkspaces(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, ok := args["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username must be a string")
	}

	_, ok = args["app_password"].(string)
	if !ok {
		return nil, fmt.Errorf("app_password must be a string")
	}

	// For now, return a mock response since we need actual Bitbucket API integration
	// In a real implementation, this would call the Bitbucket API
	mockWorkspaces := []map[string]interface{}{
		{
			"uuid":       "{workspace-uuid-1}",
			"slug":       "example-workspace",
			"name":       "Example Workspace",
			"type":       "workspace",
			"is_private": true,
		},
		{
			"uuid":       "{workspace-uuid-2}",
			"slug":       "another-workspace",
			"name":       "Another Workspace",
			"type":       "workspace",
			"is_private": false,
		},
	}

	return map[string]interface{}{
		"workspaces": mockWorkspaces,
		"count":      len(mockWorkspaces),
		"note":       "This is a placeholder implementation. Integrate with Bitbucket API for real workspace data.",
	}, nil
}

// listRepositories lists repositories in a Bitbucket workspace
func (b *BitbucketToolkit) listRepositories(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, ok := args["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username must be a string")
	}

	_, ok = args["app_password"].(string)
	if !ok {
		return nil, fmt.Errorf("app_password must be a string")
	}

	workspace, ok := args["workspace"].(string)
	if !ok {
		return nil, fmt.Errorf("workspace must be a string")
	}

	// For now, return a mock response
	mockRepositories := []map[string]interface{}{
		{
			"uuid":       "{repo-uuid-1}",
			"slug":       "example-repo",
			"name":       "Example Repository",
			"full_name":  fmt.Sprintf("%s/example-repo", workspace),
			"description": "An example repository",
			"is_private": true,
			"language":   "Go",
		},
		{
			"uuid":       "{repo-uuid-2}",
			"slug":       "another-repo",
			"name":       "Another Repository",
			"full_name":  fmt.Sprintf("%s/another-repo", workspace),
			"description": "Another example repository",
			"is_private": false,
			"language":   "Python",
		},
	}

	return map[string]interface{}{
		"workspace":    workspace,
		"repositories": mockRepositories,
		"count":        len(mockRepositories),
		"note":         "This is a placeholder implementation. Integrate with Bitbucket API for real repository data.",
	}, nil
}

// listPullRequests lists pull requests in a Bitbucket repository
func (b *BitbucketToolkit) listPullRequests(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, ok := args["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username must be a string")
	}

	_, ok = args["app_password"].(string)
	if !ok {
		return nil, fmt.Errorf("app_password must be a string")
	}

	workspace, ok := args["workspace"].(string)
	if !ok {
		return nil, fmt.Errorf("workspace must be a string")
	}

	repository, ok := args["repository"].(string)
	if !ok {
		return nil, fmt.Errorf("repository must be a string")
	}

	// For now, return a mock response
	mockPullRequests := []map[string]interface{}{
		{
			"id":    1,
			"title": "Add new feature",
			"state": "OPEN",
			"author": map[string]interface{}{
				"display_name": "John Doe",
				"uuid":         "{author-uuid-1}",
			},
			"source": map[string]interface{}{
				"branch": map[string]interface{}{
					"name": "feature/new-feature",
				},
			},
			"destination": map[string]interface{}{
				"branch": map[string]interface{}{
					"name": "main",
				},
			},
			"created_on": "2024-01-15T10:00:00Z",
			"updated_on": "2024-01-15T10:30:00Z",
		},
		{
			"id":    2,
			"title": "Fix bug in authentication",
			"state": "MERGED",
			"author": map[string]interface{}{
				"display_name": "Jane Smith",
				"uuid":         "{author-uuid-2}",
			},
			"source": map[string]interface{}{
				"branch": map[string]interface{}{
					"name": "fix/auth-bug",
				},
			},
			"destination": map[string]interface{}{
				"branch": map[string]interface{}{
					"name": "main",
				},
			},
			"created_on": "2024-01-14T14:00:00Z",
			"updated_on": "2024-01-14T16:00:00Z",
		},
	}

	return map[string]interface{}{
		"workspace":      workspace,
		"repository":     repository,
		"pull_requests": mockPullRequests,
		"count":          len(mockPullRequests),
		"note":           "This is a placeholder implementation. Integrate with Bitbucket API for real pull request data.",
	}, nil
}

// Helper function to make authenticated Bitbucket API requests
func (b *BitbucketToolkit) makeBitbucketRequest(ctx context.Context, url, username, appPassword string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set basic auth for Bitbucket API
	req.SetBasicAuth(username, appPassword)
	req.Header.Set("Accept", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// Helper function to parse Bitbucket API response
func (b *BitbucketToolkit) parseBitbucketResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bitbucket API request failed with status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode Bitbucket API response: %w", err)
	}

	return nil
}