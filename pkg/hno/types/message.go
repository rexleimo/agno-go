package types

import (
    "github.com/google/uuid"
)

// Role represents the role of a message sender
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a conversation message
type Message struct {
    ID         string      `json:"id"`
    Role       Role        `json:"role"`
    Content    string      `json:"content"`
    Name       string      `json:"name,omitempty"`
    ToolCallID string      `json:"tool_call_id,omitempty"`
    ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
    Metadata   interface{} `json:"metadata,omitempty"`

	// ReasoningContent 包含模型的推理过程(仅推理模型)
	// ReasoningContent contains the model's reasoning process (reasoning models only)
	ReasoningContent *ReasoningContent `json:"reasoning_content,omitempty"`
}

// ToolCall represents a tool invocation request from the model
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // typically "function"
	Function ToolCallFunction       `json:"function"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCallFunction contains the function call details
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// NewMessage creates a new message with the given role and content
func NewMessage(role Role, content string) *Message {
    return &Message{
        ID:      "msg-" + uuid.NewString(),
        Role:    role,
        Content: content,
    }
}

// NewSystemMessage creates a system message
func NewSystemMessage(content string) *Message {
	return NewMessage(RoleSystem, content)
}

// NewUserMessage creates a user message
func NewUserMessage(content string) *Message {
	return NewMessage(RoleUser, content)
}

// NewAssistantMessage creates an assistant message
func NewAssistantMessage(content string) *Message {
	return NewMessage(RoleAssistant, content)
}

// NewToolMessage creates a tool response message
func NewToolMessage(toolCallID, content string) *Message {
    msg := NewMessage(RoleTool, content)
    msg.ToolCallID = toolCallID
    return msg
}
