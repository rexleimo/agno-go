package bitbucket

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

const defaultBaseURL = "https://api.bitbucket.org/2.0"

// BitbucketToolkit provides Bitbucket Cloud API integration.
type BitbucketToolkit struct {
	*toolkit.BaseToolkit
	client  *http.Client
	baseURL string
}

// New creates a toolkit using Bitbucket Cloud's public API.
func New() *BitbucketToolkit {
	return NewWithBase(defaultBaseURL)
}

// NewWithBase creates a toolkit against a custom Bitbucket API base URL.
// It is useful for Bitbucket Server-compatible proxies and HTTP contract tests.
func NewWithBase(baseURL string) *BitbucketToolkit {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}

	t := &BitbucketToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("bitbucket"),
		client:      &http.Client{Timeout: 30 * time.Second},
		baseURL:     strings.TrimRight(baseURL, "/"),
	}

	t.RegisterFunction(&toolkit.Function{
		Name:        "list_workspaces",
		Description: "List Bitbucket workspaces accessible with the provided credentials",
		Parameters: map[string]toolkit.Parameter{
			"username":     {Type: "string", Description: "Bitbucket username", Required: true},
			"app_password": {Type: "string", Description: "Bitbucket app password", Required: true},
		},
		Handler: t.listWorkspaces,
	})

	t.RegisterFunction(&toolkit.Function{
		Name:        "list_repositories",
		Description: "List repositories in a Bitbucket workspace",
		Parameters: map[string]toolkit.Parameter{
			"username":     {Type: "string", Description: "Bitbucket username", Required: true},
			"app_password": {Type: "string", Description: "Bitbucket app password", Required: true},
			"workspace":    {Type: "string", Description: "Workspace slug", Required: true},
		},
		Handler: t.listRepositories,
	})

	t.RegisterFunction(&toolkit.Function{
		Name:        "list_pull_requests",
		Description: "List pull requests in a Bitbucket repository",
		Parameters: map[string]toolkit.Parameter{
			"username":     {Type: "string", Description: "Bitbucket username", Required: true},
			"app_password": {Type: "string", Description: "Bitbucket app password", Required: true},
			"workspace":    {Type: "string", Description: "Workspace slug", Required: true},
			"repository":   {Type: "string", Description: "Repository slug", Required: true},
		},
		Handler: t.listPullRequests,
	})

	return t
}

type bitbucketPage struct {
	Values []map[string]interface{} `json:"values"`
	Next   string                   `json:"next,omitempty"`
}

func requiredString(args map[string]interface{}, name string) (string, error) {
	value, ok := args[name].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", name)
	}
	return strings.TrimSpace(value), nil
}

func (b *BitbucketToolkit) endpoint(path string) string {
	return b.baseURL + "/" + strings.TrimLeft(path, "/")
}

// listWorkspaces lists accessible Bitbucket workspaces using the authenticated API.
func (b *BitbucketToolkit) listWorkspaces(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	username, err := requiredString(args, "username")
	if err != nil {
		return nil, err
	}
	password, err := requiredString(args, "app_password")
	if err != nil {
		return nil, err
	}

	resp, err := b.makeBitbucketRequest(ctx, b.endpoint("/workspaces"), username, password)
	if err != nil {
		return nil, err
	}
	var payload bitbucketPage
	if err := b.parseBitbucketResponse(resp, &payload); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"workspaces": payload.Values,
		"count":      len(payload.Values),
	}
	if payload.Next != "" {
		result["next"] = payload.Next
	}
	return result, nil
}

// listRepositories lists repositories in a Bitbucket workspace.
func (b *BitbucketToolkit) listRepositories(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	username, err := requiredString(args, "username")
	if err != nil {
		return nil, err
	}
	password, err := requiredString(args, "app_password")
	if err != nil {
		return nil, err
	}
	workspace, err := requiredString(args, "workspace")
	if err != nil {
		return nil, err
	}

	resp, err := b.makeBitbucketRequest(ctx,
		b.endpoint("/repositories/"+url.PathEscape(workspace)), username, password)
	if err != nil {
		return nil, err
	}
	var payload bitbucketPage
	if err := b.parseBitbucketResponse(resp, &payload); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"workspace":    workspace,
		"repositories": payload.Values,
		"count":        len(payload.Values),
	}
	if payload.Next != "" {
		result["next"] = payload.Next
	}
	return result, nil
}

// listPullRequests lists pull requests in a Bitbucket repository.
func (b *BitbucketToolkit) listPullRequests(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	username, err := requiredString(args, "username")
	if err != nil {
		return nil, err
	}
	password, err := requiredString(args, "app_password")
	if err != nil {
		return nil, err
	}
	workspace, err := requiredString(args, "workspace")
	if err != nil {
		return nil, err
	}
	repository, err := requiredString(args, "repository")
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests", url.PathEscape(workspace), url.PathEscape(repository))
	resp, err := b.makeBitbucketRequest(ctx, b.endpoint(path), username, password)
	if err != nil {
		return nil, err
	}
	var payload bitbucketPage
	if err := b.parseBitbucketResponse(resp, &payload); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"workspace":     workspace,
		"repository":    repository,
		"pull_requests": payload.Values,
		"count":         len(payload.Values),
	}
	if payload.Next != "" {
		result["next"] = payload.Next
	}
	return result, nil
}

func (b *BitbucketToolkit) makeBitbucketRequest(ctx context.Context, endpoint, username, appPassword string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bitbucket request: %w", err)
	}
	req.SetBasicAuth(username, appPassword)
	req.Header.Set("Accept", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Bitbucket request failed: %w", err)
	}
	return resp, nil
}

func (b *BitbucketToolkit) parseBitbucketResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		message := strings.TrimSpace(string(body))
		if message == "" {
			return fmt.Errorf("Bitbucket API request failed with status: %d", resp.StatusCode)
		}
		return fmt.Errorf("Bitbucket API request failed with status: %d: %s", resp.StatusCode, message)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode Bitbucket API response: %w", err)
	}
	return nil
}
