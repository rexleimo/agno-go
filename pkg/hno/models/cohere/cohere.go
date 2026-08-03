package cohere

import (
    "context"
    "strings"

    "github.com/rexleimo/agno-go/pkg/hno/models"
    "github.com/rexleimo/agno-go/pkg/hno/types"
)

const (
    // DefaultBaseURL Cohere Chat API base
    DefaultBaseURL = "https://api.cohere.ai/v1"
)

// Cohere minimal Chat provider
type Cohere struct {
    models.BaseModel
    http   *models.HTTPClient
    config Config
}

// Config for Cohere provider
type Config struct {
    APIKey      string
    BaseURL     string
    Temperature float64
    MaxTokens   int
}

// New creates a Cohere model
func New(modelID string, config Config) (*Cohere, error) {
    if config.APIKey == "" {
        return nil, types.NewInvalidConfigError("Cohere API key is required", nil)
    }
    baseURL := config.BaseURL
    if baseURL == "" {
        baseURL = DefaultBaseURL
    }
    return &Cohere{
        BaseModel: models.BaseModel{ID: modelID, Provider: "cohere"},
        http:      models.NewHTTPClient(),
        config: Config{
            APIKey:      config.APIKey,
            BaseURL:     baseURL,
            Temperature: config.Temperature,
            MaxTokens:   config.MaxTokens,
        },
    }, nil
}

// Invoke implements synchronous chat
func (c *Cohere) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
    chatReq := c.buildChatRequest(req)

    headers := map[string]string{
        "Authorization": "Bearer " + c.config.APIKey,
    }

    resp, err := c.http.PostJSON(ctx, c.config.BaseURL+"/chat", headers, chatReq)
    if err != nil {
        return nil, err
    }

    var chatResp ChatResponse
    if err := models.ReadJSONResponse(resp, &chatResp); err != nil {
        return nil, err
    }

    // Extract assistant text
    content := extractAssistantText(chatResp)

    mr := &types.ModelResponse{
        ID:      firstNonEmpty(chatResp.ID, chatResp.GenerationID),
        Content: content,
        Model:   c.ID,
        Metadata: types.Metadata{
            FinishReason: chatResp.FinishReason,
        },
    }

    if chatResp.Meta.Tokens != nil {
        mr.Usage = types.Usage{
            PromptTokens:     chatResp.Meta.Tokens.InputTokens,
            CompletionTokens: chatResp.Meta.Tokens.OutputTokens,
            TotalTokens:      chatResp.Meta.Tokens.InputTokens + chatResp.Meta.Tokens.OutputTokens,
        }
    } else if chatResp.Meta.BilledUnits != nil {
        // fallback to billed units if tokens section missing
        mr.Usage = types.Usage{
            PromptTokens:     chatResp.Meta.BilledUnits.InputTokens,
            CompletionTokens: chatResp.Meta.BilledUnits.OutputTokens,
            TotalTokens:      chatResp.Meta.BilledUnits.InputTokens + chatResp.Meta.BilledUnits.OutputTokens,
        }
    }

    return mr, nil
}

// InvokeStream provides a simple non-SSE fallback by calling Invoke
func (c *Cohere) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
    ch := make(chan types.ResponseChunk, 1)
    go func() {
        defer close(ch)
        resp, err := c.Invoke(ctx, req)
        if err != nil {
            ch <- types.ResponseChunk{Error: err, Done: true}
            return
        }
        // Emit as a single chunk
        ch <- types.ResponseChunk{Content: resp.Content}
        ch <- types.ResponseChunk{Done: true}
    }()
    return ch, nil
}

// buildChatRequest maps InvokeRequest to Cohere Chat API
func (c *Cohere) buildChatRequest(req *models.InvokeRequest) ChatRequest {
    // Separate preamble (first system message) and chat history
    var preamble string
    var history []ChatHistoryMessage
    var lastUser string

    // Build history excluding the final user message which becomes `message`
    // Follow Cohere guidance: chat_history should exclude current user turn
    for i, m := range req.Messages {
        role := strings.ToLower(string(m.Role))
        switch role {
        case "system":
            if preamble == "" {
                preamble = m.Content
                continue
            }
            // additional system prompts go into history as SYSTEM
            history = append(history, ChatHistoryMessage{Role: "SYSTEM", Message: m.Content})
        case "user":
            // assume the last user message is the current turn
            lastUser = m.Content
            // previous user messages form history
            if i != len(req.Messages)-1 {
                history = append(history, ChatHistoryMessage{Role: "USER", Message: m.Content})
            }
        case "assistant":
            history = append(history, ChatHistoryMessage{Role: "CHATBOT", Message: m.Content})
        default:
            // ignore tool messages for minimal implementation
        }
    }

    temperature, maxTokens := models.MergeConfig(req.Temperature, c.config.Temperature, req.MaxTokens, c.config.MaxTokens)

    chatReq := ChatRequest{
        Model:        c.ID,
        Message:      lastUser,
        ChatHistory:  history,
        Preamble:     preamble,
        Temperature:  temperature,
        MaxTokens:    maxTokens,
        ResponseJSON: nil,
    }

    // Basic tool/function schema could be added via response_format when needed
    return chatReq
}

func extractAssistantText(resp ChatResponse) string {
    // Prefer top-level message content
    if resp.Message.Content != nil {
        var b strings.Builder
        for _, blk := range resp.Message.Content {
            if strings.ToLower(blk.Type) == "text" {
                b.WriteString(blk.Text)
            }
        }
        return b.String()
    }

    // Fallback: take last CHATBOT message from chat_history
    for i := len(resp.ChatHistory) - 1; i >= 0; i-- {
        if strings.ToUpper(resp.ChatHistory[i].Role) == "CHATBOT" {
            return resp.ChatHistory[i].Message
        }
    }
    return ""
}

func firstNonEmpty(vals ...string) string {
    for _, v := range vals {
        if v != "" {
            return v
        }
    }
    return ""
}
