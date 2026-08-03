package protocol

import (
	"encoding/json"
	"testing"
)

func TestInitializeParams(t *testing.T) {
	params := InitializeParams{
		ProtocolVersion: "1.0",
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "0.1.0",
		},
		Capabilities: map[string]interface{}{
			"tools": true,
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal InitializeParams: %v", err)
	}

	var decoded InitializeParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal InitializeParams: %v", err)
	}

	if decoded.ProtocolVersion != params.ProtocolVersion {
		t.Errorf("Expected protocol version %s, got %s", params.ProtocolVersion, decoded.ProtocolVersion)
	}
	if decoded.ClientInfo.Name != params.ClientInfo.Name {
		t.Errorf("Expected client name %s, got %s", params.ClientInfo.Name, decoded.ClientInfo.Name)
	}
}

func TestToolSchema(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "First parameter",
				},
			},
			Required: []string{"param1"},
		},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Failed to marshal Tool: %v", err)
	}

	var decoded Tool
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Tool: %v", err)
	}

	if decoded.Name != tool.Name {
		t.Errorf("Expected tool name %s, got %s", tool.Name, decoded.Name)
	}
	if decoded.InputSchema.Type != tool.InputSchema.Type {
		t.Errorf("Expected schema type %s, got %s", tool.InputSchema.Type, decoded.InputSchema.Type)
	}
	if len(decoded.InputSchema.Required) != len(tool.InputSchema.Required) {
		t.Errorf("Expected %d required fields, got %d", len(tool.InputSchema.Required), len(decoded.InputSchema.Required))
	}
}

func TestToolsCallParams(t *testing.T) {
	params := ToolsCallParams{
		Name: "calculate",
		Arguments: map[string]interface{}{
			"operation": "add",
			"a":         1,
			"b":         2,
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal ToolsCallParams: %v", err)
	}

	var decoded ToolsCallParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ToolsCallParams: %v", err)
	}

	if decoded.Name != params.Name {
		t.Errorf("Expected tool name %s, got %s", params.Name, decoded.Name)
	}
	if len(decoded.Arguments) != len(params.Arguments) {
		t.Errorf("Expected %d arguments, got %d", len(params.Arguments), len(decoded.Arguments))
	}
}

func TestToolsCallResult(t *testing.T) {
	result := ToolsCallResult{
		Content: []Content{
			{
				Type: ContentTypeText,
				Text: "Result: 3",
			},
		},
		IsError: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal ToolsCallResult: %v", err)
	}

	var decoded ToolsCallResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ToolsCallResult: %v", err)
	}

	if len(decoded.Content) != len(result.Content) {
		t.Errorf("Expected %d content items, got %d", len(result.Content), len(decoded.Content))
	}
	if decoded.IsError != result.IsError {
		t.Errorf("Expected isError %v, got %v", result.IsError, decoded.IsError)
	}
}

func TestContentTypes(t *testing.T) {
	tests := []struct {
		name       string
		content    Content
		wantType   string
		wantNotNil []string
		wantNil    []string
	}{
		{
			name: "text content",
			content: Content{
				Type: ContentTypeText,
				Text: "Hello, world!",
			},
			wantType:   ContentTypeText,
			wantNotNil: []string{"Text"},
			wantNil:    []string{"Data", "MimeType", "URI"},
		},
		{
			name: "image content",
			content: Content{
				Type:     ContentTypeImage,
				Data:     "base64encodeddata==",
				MimeType: "image/png",
			},
			wantType:   ContentTypeImage,
			wantNotNil: []string{"Data", "MimeType"},
			wantNil:    []string{"Text", "URI"},
		},
		{
			name: "resource content",
			content: Content{
				Type:     ContentTypeResource,
				URI:      "file:///path/to/resource",
				MimeType: "application/json",
			},
			wantType:   ContentTypeResource,
			wantNotNil: []string{"URI", "MimeType"},
			wantNil:    []string{"Text", "Data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.content)
			if err != nil {
				t.Fatalf("Failed to marshal Content: %v", err)
			}

			var decoded Content
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal Content: %v", err)
			}

			if decoded.Type != tt.wantType {
				t.Errorf("Expected content type %s, got %s", tt.wantType, decoded.Type)
			}

			// Check non-nil fields
			for _, field := range tt.wantNotNil {
				switch field {
				case "Text":
					if decoded.Text == "" {
						t.Errorf("Expected %s to be non-empty", field)
					}
				case "Data":
					if decoded.Data == "" {
						t.Errorf("Expected %s to be non-empty", field)
					}
				case "MimeType":
					if decoded.MimeType == "" {
						t.Errorf("Expected %s to be non-empty", field)
					}
				case "URI":
					if decoded.URI == "" {
						t.Errorf("Expected %s to be non-empty", field)
					}
				}
			}

			// Check nil fields
			for _, field := range tt.wantNil {
				switch field {
				case "Text":
					if decoded.Text != "" {
						t.Errorf("Expected %s to be empty, got %s", field, decoded.Text)
					}
				case "Data":
					if decoded.Data != "" {
						t.Errorf("Expected %s to be empty, got %s", field, decoded.Data)
					}
				case "URI":
					if decoded.URI != "" {
						t.Errorf("Expected %s to be empty, got %s", field, decoded.URI)
					}
				}
			}
		})
	}
}

func TestResource(t *testing.T) {
	resource := Resource{
		URI:         "file:///path/to/resource.json",
		Name:        "test-resource",
		Description: "A test resource",
		MimeType:    "application/json",
	}

	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal Resource: %v", err)
	}

	var decoded Resource
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Resource: %v", err)
	}

	if decoded.URI != resource.URI {
		t.Errorf("Expected URI %s, got %s", resource.URI, decoded.URI)
	}
	if decoded.Name != resource.Name {
		t.Errorf("Expected name %s, got %s", resource.Name, decoded.Name)
	}
}

func TestPrompt(t *testing.T) {
	prompt := Prompt{
		Name:        "test-prompt",
		Description: "A test prompt",
		Arguments: []Argument{
			{
				Name:        "arg1",
				Description: "First argument",
				Required:    true,
			},
			{
				Name:        "arg2",
				Description: "Second argument",
				Required:    false,
			},
		},
	}

	data, err := json.Marshal(prompt)
	if err != nil {
		t.Fatalf("Failed to marshal Prompt: %v", err)
	}

	var decoded Prompt
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Prompt: %v", err)
	}

	if decoded.Name != prompt.Name {
		t.Errorf("Expected prompt name %s, got %s", prompt.Name, decoded.Name)
	}
	if len(decoded.Arguments) != len(prompt.Arguments) {
		t.Errorf("Expected %d arguments, got %d", len(prompt.Arguments), len(decoded.Arguments))
	}
}

func TestMessage(t *testing.T) {
	message := Message{
		Role: "user",
		Content: []Content{
			{
				Type: ContentTypeText,
				Text: "Hello",
			},
		},
	}

	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Failed to marshal Message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Message: %v", err)
	}

	if decoded.Role != message.Role {
		t.Errorf("Expected role %s, got %s", message.Role, decoded.Role)
	}
	if len(decoded.Content) != len(message.Content) {
		t.Errorf("Expected %d content items, got %d", len(message.Content), len(decoded.Content))
	}
}
