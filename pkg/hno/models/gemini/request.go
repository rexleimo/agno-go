package gemini

import (
	"encoding/json"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func (g *Gemini) buildGeminiRequest(req *models.InvokeRequest) *GeminiRequest {
	geminiReq := &GeminiRequest{
		Contents:         make([]Content, 0),
		GenerationConfig: &GenerationConfig{},
	}

	// Set generation config
	if req.Temperature > 0 {
		geminiReq.GenerationConfig.Temperature = req.Temperature
	} else if g.config.Temperature > 0 {
		geminiReq.GenerationConfig.Temperature = g.config.Temperature
	}

	if req.MaxTokens > 0 {
		geminiReq.GenerationConfig.MaxOutputTokens = req.MaxTokens
	} else if g.config.MaxTokens > 0 {
		geminiReq.GenerationConfig.MaxOutputTokens = g.config.MaxTokens
	}

	var thinkingBudget int
	var includeThoughtsPtr *bool

	if g.config.ThinkingBudget > 0 {
		thinkingBudget = g.config.ThinkingBudget
	}
	if g.config.IncludeThoughts != nil {
		includeThoughtsPtr = g.config.IncludeThoughts
	}

	if req.Extra != nil {
		if val, ok := req.Extra["thinking_budget"]; ok {
			if budget, ok := models.ToInt(val); ok {
				thinkingBudget = budget
			}
		}
		if val, ok := req.Extra["include_thoughts"]; ok {
			if include, ok := models.ToBool(val); ok {
				includeThoughtsPtr = &include
			}
		}
		if val, ok := req.Extra["includeThoughts"]; ok {
			if include, ok := models.ToBool(val); ok {
				includeThoughtsPtr = &include
			}
		}
		if cfgRaw, ok := req.Extra["thinking_config"]; ok {
			if cfg, ok := cfgRaw.(map[string]interface{}); ok {
				if v, ok := cfg["budget_tokens"]; ok {
					if budget, ok := models.ToInt(v); ok {
						thinkingBudget = budget
					}
				}
				if v, ok := cfg["budgetTokens"]; ok {
					if budget, ok := models.ToInt(v); ok {
						thinkingBudget = budget
					}
				}
				if v, ok := cfg["include_thoughts"]; ok {
					if include, ok := models.ToBool(v); ok {
						includeThoughtsPtr = &include
					}
				}
				if v, ok := cfg["includeThoughts"]; ok {
					if include, ok := models.ToBool(v); ok {
						includeThoughtsPtr = &include
					}
				}
			}
		}
	}

	if thinkingBudget > 0 || includeThoughtsPtr != nil {
		tc := &ThinkingConfig{}
		if thinkingBudget > 0 {
			tc.BudgetTokens = thinkingBudget
		}
		if includeThoughtsPtr != nil {
			tc.IncludeThoughts = includeThoughtsPtr
		}
		geminiReq.ThinkingConfig = tc
	}

	// Convert messages to Gemini format
	var systemInstruction string
	for _, msg := range req.Messages {
		switch msg.Role {
		case types.RoleSystem:
			systemInstruction = msg.Content
		case types.RoleUser:
			geminiReq.Contents = append(geminiReq.Contents, Content{
				Role: "user",
				Parts: []Part{
					{Text: msg.Content},
				},
			})
		case types.RoleAssistant:
			content := Content{
				Role:  "model",
				Parts: make([]Part, 0),
			}

			// Add text content
			if msg.Content != "" {
				content.Parts = append(content.Parts, Part{Text: msg.Content})
			}

			// Add tool calls if present
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					var args map[string]interface{}
					json.Unmarshal([]byte(tc.Function.Arguments), &args)
					content.Parts = append(content.Parts, Part{
						FunctionCall: &FunctionCall{
							Name: tc.Function.Name,
							Args: args,
						},
					})
				}
			}

			geminiReq.Contents = append(geminiReq.Contents, content)
		case types.RoleTool:
			// Tool results are sent as function responses
			geminiReq.Contents = append(geminiReq.Contents, Content{
				Role: "function",
				Parts: []Part{
					{
						FunctionResponse: &FunctionResponse{
							Name: msg.Name,
							Response: map[string]interface{}{
								"result": msg.Content,
							},
						},
					},
				},
			})
		}
	}

	// Add system instruction if present
	if systemInstruction != "" {
		geminiReq.SystemInstruction = &Content{
			Parts: []Part{
				{Text: systemInstruction},
			},
		}
	}

	// Convert tools
	if len(req.Tools) > 0 {
		geminiReq.Tools = make([]Tool, 1)
		geminiReq.Tools[0].FunctionDeclarations = make([]FunctionDeclaration, len(req.Tools))

		for i, tool := range req.Tools {
			geminiReq.Tools[0].FunctionDeclarations[i] = FunctionDeclaration{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			}
		}
	}

	return geminiReq
}

// GeminiRequest represents the Gemini API request
type GeminiRequest struct {
	Contents          []Content         `json:"contents"`
	SystemInstruction *Content          `json:"systemInstruction,omitempty"`
	Tools             []Tool            `json:"tools,omitempty"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
	ThinkingConfig    *ThinkingConfig   `json:"thinkingConfig,omitempty"`
}

// Content represents content in the request/response
type Content struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts"`
}

// Part represents a part of the content
type Part struct {
	Text             string            `json:"text,omitempty"`
	FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
	Thought          bool              `json:"thought,omitempty"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// FunctionResponse represents a function response
type FunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

// Tool represents a tool definition
type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations,omitempty"`
}

// FunctionDeclaration represents a function declaration
type FunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// GenerationConfig represents generation configuration
type GenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
}

// ThinkingConfig represents reasoning configuration for Gemini thinking models
type ThinkingConfig struct {
	IncludeThoughts *bool `json:"includeThoughts,omitempty"`
	BudgetTokens    int   `json:"budgetTokens,omitempty"`
}
