package types

import (
	"testing"
)

func TestNewMessage(t *testing.T) {
	tests := []struct {
		name    string
		role    Role
		content string
		want    *Message
	}{
		{
			name:    "system message",
			role:    RoleSystem,
			content: "You are a helpful assistant",
			want: &Message{
				Role:    RoleSystem,
				Content: "You are a helpful assistant",
			},
		},
		{
			name:    "user message",
			role:    RoleUser,
			content: "Hello",
			want: &Message{
				Role:    RoleUser,
				Content: "Hello",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		got := NewMessage(tt.role, tt.content)
		if got.Role != tt.want.Role {
			t.Errorf("Role = %v, want %v", got.Role, tt.want.Role)
		}
		if got.Content != tt.want.Content {
			t.Errorf("Content = %v, want %v", got.Content, tt.want.Content)
		}
		if got.ID == "" {
			t.Error("expected message ID to be generated, got empty string")
		}
	})
	}
}

func TestNewSystemMessage(t *testing.T) {
	content := "test system message"
	msg := NewSystemMessage(content)

	if msg.Role != RoleSystem {
		t.Errorf("expected role %v, got %v", RoleSystem, msg.Role)
	}
	if msg.Content != content {
		t.Errorf("expected content %v, got %v", content, msg.Content)
	}
}

func TestNewUserMessage(t *testing.T) {
	content := "Hello"
	msg := NewUserMessage(content)

	if msg.Role != RoleUser {
		t.Errorf("expected role %v, got %v", RoleUser, msg.Role)
	}
	if msg.Content != content {
		t.Errorf("expected content %v, got %v", content, msg.Content)
	}

	if msg.ID == "" {
		t.Error("expected message ID to be generated for user message")
	}
}

func TestNewAssistantMessage(t *testing.T) {
	content := "Hi there"
	msg := NewAssistantMessage(content)

	if msg.Role != RoleAssistant {
		t.Errorf("expected role %v, got %v", RoleAssistant, msg.Role)
	}
	if msg.Content != content {
		t.Errorf("expected content %v, got %v", content, msg.Content)
	}

	if msg.ID == "" {
		t.Error("expected message ID to be generated for assistant message")
	}
}

func TestNewToolMessage(t *testing.T) {
	toolCallID := "call_123"
	content := "tool result"
	msg := NewToolMessage(toolCallID, content)

	if msg.Role != RoleTool {
		t.Errorf("expected role %v, got %v", RoleTool, msg.Role)
	}
	if msg.ToolCallID != toolCallID {
		t.Errorf("expected toolCallID %v, got %v", toolCallID, msg.ToolCallID)
	}
	if msg.Content != content {
		t.Errorf("expected content %v, got %v", content, msg.Content)
	}

	if msg.ID == "" {
		t.Error("expected message ID to be generated for tool message")
	}
}

func TestMessageIDUniqueness(t *testing.T) {
    msg1 := NewUserMessage("hello")
    msg2 := NewUserMessage("hello again")

    if msg1.ID == msg2.ID {
        t.Errorf("expected different IDs for different messages, got same: %s", msg1.ID)
    }
}
