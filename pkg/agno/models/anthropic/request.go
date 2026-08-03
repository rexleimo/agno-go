package anthropic

import (
	"fmt"
	"strings"

	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

func (a *Anthropic) buildClaudeRequest(req *models.InvokeRequest) *ClaudeRequest {
	claudeReq := &ClaudeRequest{
		Model:     a.ID,
		Messages:  make([]ClaudeMessage, 0),
		MaxTokens: req.MaxTokens,
		Stream:    false,
	}

	// Set max tokens
	if claudeReq.MaxTokens == 0 {
		claudeReq.MaxTokens = a.config.MaxTokens
	}

	// Set temperature
	if req.Temperature > 0 {
		claudeReq.Temperature = req.Temperature
	} else if a.config.Temperature > 0 {
		claudeReq.Temperature = a.config.Temperature
	}

	var thinkingCfg *ThinkingConfig
	if a.config.Thinking != nil {
		cfgCopy := *a.config.Thinking
		thinkingCfg = &cfgCopy
	}

	if req.Extra != nil {
		if raw, ok := req.Extra["thinking"]; ok {
			if cfgMap, ok := raw.(map[string]interface{}); ok {
				if thinkingCfg == nil {
					thinkingCfg = &ThinkingConfig{}
				}
				if v, ok := cfgMap["type"].(string); ok && v != "" {
					thinkingCfg.Type = v
				}
				if v, ok := cfgMap["Type"].(string); ok && v != "" {
					thinkingCfg.Type = v
				}
				if budget, ok := valueToInt(cfgMap["budget_tokens"]); ok {
					thinkingCfg.BudgetTokens = budget
				}
				if budget, ok := valueToInt(cfgMap["budgetTokens"]); ok {
					thinkingCfg.BudgetTokens = budget
				}
				if maxTokens, ok := valueToInt(cfgMap["max_output_tokens"]); ok {
					thinkingCfg.MaxOutputTokens = maxTokens
				}
				if maxTokens, ok := valueToInt(cfgMap["maxOutputTokens"]); ok {
					thinkingCfg.MaxOutputTokens = maxTokens
				}
			}
		}
		if budget, ok := valueToInt(req.Extra["thinking_budget"]); ok {
			if thinkingCfg == nil {
				thinkingCfg = &ThinkingConfig{Type: "enabled"}
			}
			thinkingCfg.BudgetTokens = budget
			if thinkingCfg.Type == "" {
				thinkingCfg.Type = "enabled"
			}
		}
		if budget, ok := valueToInt(req.Extra["thinkingBudget"]); ok {
			if thinkingCfg == nil {
				thinkingCfg = &ThinkingConfig{Type: "enabled"}
			}
			thinkingCfg.BudgetTokens = budget
			if thinkingCfg.Type == "" {
				thinkingCfg.Type = "enabled"
			}
		}
		if t, ok := req.Extra["thinking_type"].(string); ok && t != "" {
			if thinkingCfg == nil {
				thinkingCfg = &ThinkingConfig{}
			}
			thinkingCfg.Type = t
		}
		if rawBetas, ok := req.Extra["betas"]; ok {
			claudeReq.Betas = appendBetas(claudeReq.Betas, rawBetas)
		}
		if rawCtx, ok := req.Extra["context_management"]; ok {
			if ctxMap, ok := rawCtx.(map[string]interface{}); ok {
				claudeReq.ContextManagement = ctxMap
			}
		}
	}

	if thinkingCfg != nil {
		if thinkingCfg.Type == "" {
			thinkingCfg.Type = "enabled"
		}
		if strings.EqualFold(thinkingCfg.Type, "disabled") || thinkingCfg.BudgetTokens > 0 {
			claudeReq.Thinking = thinkingCfg
		}
	}

	if len(a.config.Betas) > 0 {
		prefixed := make([]string, 0, len(a.config.Betas)+len(claudeReq.Betas))
		prefixed = append(prefixed, a.config.Betas...)
		prefixed = append(prefixed, claudeReq.Betas...)
		claudeReq.Betas = prefixed
	}
	if a.config.ContextManagement != nil {
		merged := cloneContextMap(a.config.ContextManagement)
		if claudeReq.ContextManagement != nil {
			for k, v := range claudeReq.ContextManagement {
				merged[k] = v
			}
		}
		claudeReq.ContextManagement = merged
	}

	// Convert messages
	var systemPrompt string
	for _, msg := range req.Messages {
		switch msg.Role {
		case types.RoleSystem:
			systemPrompt = msg.Content
		case types.RoleUser, types.RoleAssistant:
			claudeMsg := ClaudeMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			}
			claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
		case types.RoleTool:
			// Handle tool results
			claudeMsg := ClaudeMessage{
				Role:    "user",
				Content: fmt.Sprintf("Tool result: %s", msg.Content),
			}
			claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
		}
	}

	if systemPrompt != "" {
		claudeReq.System = systemPrompt
	}

	// Convert tools
	if len(req.Tools) > 0 {
		claudeReq.Tools = make([]ClaudeTool, len(req.Tools))
		for i, tool := range req.Tools {
			claudeReq.Tools[i] = ClaudeTool{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				InputSchema: tool.Function.Parameters,
			}
		}
	}

	return claudeReq
}

// ClaudeRequest represents the Anthropic API request
type ClaudeRequest struct {
	Model             string                 `json:"model"`
	Messages          []ClaudeMessage        `json:"messages"`
	System            string                 `json:"system,omitempty"`
	MaxTokens         int                    `json:"max_tokens"`
	Temperature       float64                `json:"temperature,omitempty"`
	Tools             []ClaudeTool           `json:"tools,omitempty"`
	Stream            bool                   `json:"stream,omitempty"`
	Thinking          *ThinkingConfig        `json:"thinking,omitempty"`
	Betas             []string               `json:"betas,omitempty"`
	ContextManagement map[string]interface{} `json:"context_management,omitempty"`
}

// ClaudeMessage represents a message in the conversation
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeTool represents a tool definition
type ClaudeTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

func appendBetas(existing []string, raw interface{}) []string {
	switch v := raw.(type) {
	case []string:
		return append(existing, v...)
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				existing = append(existing, s)
			}
		}
	case string:
		if v != "" {
			existing = append(existing, v)
		}
	}
	return existing
}

func cloneContextMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	cloned := make(map[string]interface{}, len(src))
	for k, v := range src {
		cloned[k] = v
	}
	return cloned
}
