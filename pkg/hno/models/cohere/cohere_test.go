package cohere

import (
    "testing"
    "github.com/rexleimo/agno-go/pkg/hno/models"
    "github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestNew_MissingAPIKey(t *testing.T) {
    _, err := New("command", Config{})
    if err == nil {
        t.Fatalf("expected error for missing API key")
    }
}

func TestBuildChatRequest_Basic(t *testing.T) {
    c, err := New("command", Config{APIKey: "test"})
    if err != nil {
        t.Fatalf("New error: %v", err)
    }

    req := &models.InvokeRequest{
        Messages: []*types.Message{
            types.NewSystemMessage("You are helpful."),
            types.NewUserMessage("Hello"),
            types.NewAssistantMessage("Hi! How can I help?"),
            types.NewUserMessage("Tell me a joke"),
        },
        Temperature: 0.7,
        MaxTokens:   128,
    }

    cr := c.buildChatRequest(req)
    if cr.Model != "command" {
        t.Errorf("model = %s, want command", cr.Model)
    }
    if cr.Preamble != "You are helpful." {
        t.Errorf("preamble mismatch: %q", cr.Preamble)
    }
    if cr.Message != "Tell me a joke" {
        t.Errorf("message mismatch: %q", cr.Message)
    }
    if len(cr.ChatHistory) == 0 {
        t.Fatalf("expected history entries")
    }
    // Expect USER("Hello"), CHATBOT("Hi! How can I help?")
    if cr.ChatHistory[0].Role != "USER" || cr.ChatHistory[0].Message != "Hello" {
        t.Errorf("history[0] unexpected: %+v", cr.ChatHistory[0])
    }
    if cr.ChatHistory[1].Role != "CHATBOT" || cr.ChatHistory[1].Message != "Hi! How can I help?" {
        t.Errorf("history[1] unexpected: %+v", cr.ChatHistory[1])
    }
    if cr.Temperature != 0.7 || cr.MaxTokens != 128 {
        t.Errorf("temp/tokens unexpected: %v / %v", cr.Temperature, cr.MaxTokens)
    }
}

