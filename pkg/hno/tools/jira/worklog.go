package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// Config 控制 Jira 工具包。
type Config struct {
	BaseURL    string
	AuthToken  string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// Toolkit 提供 Jira 工时登记功能。
type Toolkit struct {
	*toolkit.BaseToolkit
	client *client
}

type client struct {
	base string
	auth string
	http *http.Client
}

type worklogRequest struct {
	Comment          string `json:"comment,omitempty"`
	Started          string `json:"started"`
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
}

type worklogResponse struct {
	ID      string                 `json:"id"`
	Self    string                 `json:"self"`
	Details map[string]interface{} `json:"details,omitempty"`
}

const defaultTimeout = 30 * time.Second

// New 创建 Jira 工具包。
func New(cfg Config) (*Toolkit, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("jira base url is required")
	}
	if cfg.AuthToken == "" {
		return nil, fmt.Errorf("jira auth token is required")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	if cfg.Timeout > 0 {
		httpClient.Timeout = cfg.Timeout
	}

	c := &client{
		base: strings.TrimRight(cfg.BaseURL, "/"),
		auth: cfg.AuthToken,
		http: httpClient,
	}

	tk := &Toolkit{
		BaseToolkit: toolkit.NewBaseToolkit("jira"),
		client:      c,
	}

	tk.RegisterFunction(&toolkit.Function{
		Name:        "jira_add_worklog",
		Description: "Add a worklog entry to a Jira issue",
		Parameters: map[string]toolkit.Parameter{
			"issue_id": {
				Type:        "string",
				Description: "Jira issue key or ID",
				Required:    true,
			},
			"time_spent_seconds": {
				Type:        "number",
				Description: "Time spent in seconds",
				Required:    true,
			},
			"started": {
				Type:        "string",
				Description: "ISO8601 timestamp when work started",
				Required:    true,
			},
			"comment": {
				Type:        "string",
				Description: "Optional comment to store with the worklog",
			},
		},
		Handler: tk.addWorklog,
	})

	return tk, nil
}

func (t *Toolkit) addWorklog(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	issueID, ok := args["issue_id"].(string)
	if !ok || strings.TrimSpace(issueID) == "" {
		return nil, fmt.Errorf("issue_id must be a non-empty string")
	}

	started, ok := args["started"].(string)
	if !ok || strings.TrimSpace(started) == "" {
		return nil, fmt.Errorf("started must be provided")
	}

	timeSpentFloat, ok := args["time_spent_seconds"].(float64)
	if !ok {
		return nil, fmt.Errorf("time_spent_seconds must be a number")
	}
	timeSpentSeconds := int(timeSpentFloat)

	var comment string
	if raw, ok := args["comment"].(string); ok {
		comment = raw
	}

	resp, err := t.client.addWorklog(ctx, issueID, worklogRequest{
		Comment:          comment,
		Started:          started,
		TimeSpentSeconds: timeSpentSeconds,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":     resp.ID,
		"self":   resp.Self,
		"status": "logged",
	}, nil
}

func (c *client) addWorklog(ctx context.Context, issueID string, payload worklogRequest) (*worklogResponse, error) {
	endpoint := fmt.Sprintf("%s/rest/api/3/issue/%s/worklog", c.base, issueID)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode worklog payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.auth)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira worklog request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("jira worklog request returned status %d", resp.StatusCode)
	}

	var result worklogResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode worklog response: %w", err)
	}
	return &result, nil
}
