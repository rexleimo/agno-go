package claude

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
	defaultBaseURL = "https://api.anthropic.com"
	defaultTimeout = 30 * time.Second
)

// Config 控制 Claude 技能工具包行为。
type Config struct {
	BaseURL      string
	APIKey       string
	DefaultSkill string
	HTTPClient   *http.Client
}

// Toolkit 暴露 Claude 技能调用能力。
type Toolkit struct {
	*toolkit.BaseToolkit
	client       *client
	defaultSkill string
}

// skillRequest mirrors Claude Agent Skills API.
type skillRequest struct {
	SkillID        string `json:"skill_id"`
	Input          string `json:"input"`
	ConversationID string `json:"conversation_id,omitempty"`
}

type skillResponse struct {
	SkillID        string                 `json:"skill_id"`
	Output         map[string]interface{} `json:"output"`
	ConversationID string                 `json:"conversation_id,omitempty"`
}

type client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New 创建 Claude 技能工具包。
func New(cfg Config) (*Toolkit, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("claude api key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	c := &client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  cfg.APIKey,
		http:    httpClient,
	}

	tk := &Toolkit{
		BaseToolkit:  toolkit.NewBaseToolkit("claude_skills"),
		client:       c,
		defaultSkill: cfg.DefaultSkill,
	}

	tk.RegisterFunction(&toolkit.Function{
		Name:        "invoke_claude_skill",
		Description: "Invoke a Claude Agent Skill with the provided input",
		Parameters: map[string]toolkit.Parameter{
			"skill_id": {
				Type:        "string",
				Description: "Identifier of the skill to execute",
			},
			"input": {
				Type:        "string",
				Description: "Prompt or payload sent to the skill",
				Required:    true,
			},
			"conversation_id": {
				Type:        "string",
				Description: "Optional conversation identifier for follow-up requests",
			},
		},
		Handler: tk.invokeSkill,
	})

	return tk, nil
}

func (t *Toolkit) resolveSkillID(args map[string]interface{}) (string, error) {
	if raw, ok := args["skill_id"].(string); ok && raw != "" {
		return raw, nil
	}
	if t.defaultSkill != "" {
		return t.defaultSkill, nil
	}
	return "", fmt.Errorf("skill_id is required when no default skill configured")
}

func (t *Toolkit) invokeSkill(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	input, ok := args["input"].(string)
	if !ok || strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("input must be a non-empty string")
	}

	skillID, err := t.resolveSkillID(args)
	if err != nil {
		return nil, err
	}

	var conversationID string
	if raw, ok := args["conversation_id"].(string); ok {
		conversationID = raw
	}

	resp, err := t.client.InvokeSkill(ctx, skillRequest{
		SkillID:        skillID,
		Input:          input,
		ConversationID: conversationID,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"skill_id":        resp.SkillID,
		"output":          resp.Output,
		"conversation_id": resp.ConversationID,
	}, nil
}

func (c *client) InvokeSkill(ctx context.Context, payload skillRequest) (*skillResponse, error) {
	endpoint := fmt.Sprintf("%s/v1/agent-skills/messages", c.baseURL)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude skill request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("claude skill request returned status %d", resp.StatusCode)
	}

	var result skillResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode skill response: %w", err)
	}

	return &result, nil
}
