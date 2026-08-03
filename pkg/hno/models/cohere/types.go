package cohere

// ChatRequest is a minimal representation for Cohere Chat API
// See Context7 /cohere-ai/cohere-python and docs.cohere.com chat reference
type ChatRequest struct {
    Model       string                 `json:"model,omitempty"`
    Message     string                 `json:"message"`
    ChatHistory []ChatHistoryMessage   `json:"chat_history,omitempty"`
    Preamble    string                 `json:"preamble,omitempty"`
    Temperature float64                `json:"temperature,omitempty"`
    MaxTokens   int                    `json:"max_tokens,omitempty"`

    // Optional: response format control for JSON forcing
    ResponseJSON map[string]interface{} `json:"response_format,omitempty"`
}

// ChatHistoryMessage follows Cohere roles
// Role: USER | CHATBOT | SYSTEM
type ChatHistoryMessage struct {
    Role    string `json:"role"`
    Message string `json:"message"`
}

// ChatResponse matches common fields from Cohere Chat API across versions
type ChatResponse struct {
    ID            string                 `json:"id,omitempty"`
    GenerationID  string                 `json:"generation_id,omitempty"`
    Message       ResponseMessage        `json:"message"`
    ChatHistory   []ChatHistoryMessage   `json:"chat_history,omitempty"`
    FinishReason  string                 `json:"finish_reason,omitempty"`
    Meta          ResponseMeta           `json:"meta,omitempty"`
}

// ResponseMessage contains assistant output blocks
type ResponseMessage struct {
    Role    string         `json:"role"`
    Content []ContentBlock `json:"content"`
}

// ContentBlock represents a single block in the assistant content
type ContentBlock struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
}

// ResponseMeta includes token usage
type ResponseMeta struct {
    APIVersion  map[string]interface{} `json:"api_version,omitempty"`
    Warnings    []string               `json:"warnings,omitempty"`
    BilledUnits *TokenInfo             `json:"billed_units,omitempty"`
    Tokens      *TokenInfo             `json:"tokens,omitempty"`
}

// TokenInfo tracks input/output tokens
type TokenInfo struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
}

