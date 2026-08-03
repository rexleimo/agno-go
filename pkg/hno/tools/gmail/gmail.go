package gmail

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

const (
	defaultBaseURL = "https://gmail.googleapis.com"
	defaultTimeout = 30 * time.Second
)

// Config 控制 Gmail 工具包行为。
type Config struct {
	AccessToken string
	BaseURL     string
	UserID      string
	HTTPClient  *http.Client
	Timeout     time.Duration
}

// Toolkit 提供 Gmail 标记已读功能。
type Toolkit struct {
	*toolkit.BaseToolkit
	client *client
	userID string
}

type client struct {
	base string
	http *http.Client
	auth string
}

type modifyRequest struct {
	RemoveLabels []string `json:"removeLabelIds,omitempty"`
	AddLabels    []string `json:"addLabelIds,omitempty"`
}

type modifyResponse struct {
	ID string `json:"id"`
}

// New 创建 Gmail 工具包。
func New(cfg Config) (*Toolkit, error) {
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("gmail access token is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	if cfg.Timeout > 0 {
		httpClient.Timeout = cfg.Timeout
	}

	userID := cfg.UserID
	if userID == "" {
		userID = "me"
	}

	c := &client{
		base: strings.TrimRight(baseURL, "/"),
		http: httpClient,
		auth: "Bearer " + cfg.AccessToken,
	}

	tk := &Toolkit{
		BaseToolkit: toolkit.NewBaseToolkit("gmail"),
		client:      c,
		userID:      userID,
	}

	tk.RegisterFunction(&toolkit.Function{
		Name:        "gmail_mark_as_read",
		Description: "Mark a Gmail message as read by removing the UNREAD label",
		Parameters: map[string]toolkit.Parameter{
			"message_id": {
				Type:        "string",
				Description: "Gmail message identifier",
				Required:    true,
			},
			"user_id": {
				Type:        "string",
				Description: "Optional Gmail user ID (defaults to 'me')",
			},
		},
		Handler: tk.markAsRead,
	})

	return tk, nil
}

func (t *Toolkit) markAsRead(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	messageID, ok := args["message_id"].(string)
	if !ok || messageID == "" {
		return nil, fmt.Errorf("message_id is required")
	}

	userID := t.userID
	if raw, ok := args["user_id"].(string); ok && raw != "" {
		userID = raw
	}

	resp, err := t.client.modifyLabels(ctx, userID, messageID, modifyRequest{
		RemoveLabels: []string{"UNREAD"},
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":      resp.ID,
		"status":  "read",
		"user_id": userID,
	}, nil
}

func (c *client) modifyLabels(ctx context.Context, userID, messageID string, payload modifyRequest) (*modifyResponse, error) {
	url := fmt.Sprintf("%s/gmail/v1/users/%s/messages/%s/modify", c.base, userID, messageID)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode modify request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.auth)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gmail modify request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gmail modify request returned status %d", resp.StatusCode)
	}

	var result modifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode gmail modify response: %w", err)
	}
	return &result, nil
}
