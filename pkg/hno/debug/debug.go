package debug

import (
    "encoding/json"

    "github.com/rexleimo/agno-go/pkg/hno/models"
    "github.com/rexleimo/agno-go/pkg/hno/types"
)

// DumpInvokeRequest returns a redacted JSON view of an InvokeRequest for debugging
func DumpInvokeRequest(req *models.InvokeRequest) string {
    type m struct{ Role string `json:"role"`; Content string `json:"content"` }
    out := struct{
        Messages    []m                      `json:"messages"`
        Tools       []models.ToolDefinition `json:"tools,omitempty"`
        Temperature float64                 `json:"temperature,omitempty"`
        MaxTokens   int                     `json:"max_tokens,omitempty"`
    }{ Temperature: req.Temperature, MaxTokens: req.MaxTokens }
    if len(req.Messages) > 0 {
        out.Messages = make([]m, len(req.Messages))
        for i, msg := range req.Messages {
            // avoid dumping tool metadata/IDs
            out.Messages[i] = m{ Role: string(msg.Role), Content: msg.Content }
        }
    }
    if len(req.Tools) > 0 { out.Tools = req.Tools }
    b, _ := json.Marshal(out)
    return string(b)
}

// DumpModelResponse returns a compact JSON view of a ModelResponse
func DumpModelResponse(resp *types.ModelResponse) string {
    out := struct{
        ID      string      `json:"id,omitempty"`
        Model   string      `json:"model,omitempty"`
        Content string      `json:"content,omitempty"`
        Usage   types.Usage `json:"usage"`
    }{ ID: resp.ID, Model: resp.Model, Content: resp.Content, Usage: resp.Usage }
    b, _ := json.Marshal(out)
    return string(b)
}

