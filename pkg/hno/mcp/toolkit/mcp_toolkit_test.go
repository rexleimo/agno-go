package toolkit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/client"
	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
)

// Create mock transport and client helpers
// 创建模拟传输和客户端辅助工具

func createConnectedClient(t *testing.T, tools []protocol.Tool) *client.Client {
	mockTransport := client.NewMockTransport()

	// Setup initialize response
	// 设置初始化响应
	initResult := protocol.InitializeResult{
		ProtocolVersion: "1.0",
		ServerInfo: protocol.ServerInfo{
			Name:    "test-server",
			Version: "1.0.0",
		},
	}
	initData, _ := json.Marshal(initResult)
	mockTransport.SetResponse(1, &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  initData,
		ID:      int64(1),
	})

	// Setup tools/list response
	// 设置 tools/list 响应
	toolsResult := protocol.ToolsListResult{
		Tools: tools,
	}
	toolsData, _ := json.Marshal(toolsResult)
	mockTransport.SetResponse(2, &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  toolsData,
		ID:      int64(2),
	})

	mcpClient, err := client.New(mockTransport, client.Config{
		ClientName: "test-client",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := mcpClient.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}

	return mcpClient
}

func TestNew(t *testing.T) {
	testTools := []protocol.Tool{
		{
			Name:        "test_tool",
			Description: "A test tool",
			InputSchema: protocol.InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"input": map[string]interface{}{
						"type":        "string",
						"description": "Input parameter",
					},
				},
				Required: []string{"input"},
			},
		},
	}

	tests := []struct {
		name      string
		setupFunc func() (Config, bool) // Returns config and shouldFail
		wantErr   bool
	}{
		{
			name: "valid config",
			setupFunc: func() (Config, bool) {
				mcpClient := createConnectedClient(t, testTools)
				return Config{
					Name:   "test-toolkit",
					Client: mcpClient,
				}, false
			},
			wantErr: false,
		},
		{
			name: "nil client",
			setupFunc: func() (Config, bool) {
				return Config{
					Client: nil,
				}, true
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, shouldSkip := tt.setupFunc()
			if shouldSkip && !tt.wantErr {
				t.Skip("Setup failed for test")
			}

			ctx := context.Background()
			toolkit, err := New(ctx, config)

			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if toolkit == nil {
					t.Error("Expected non-nil toolkit")
				}
				if toolkit.Name() == "" {
					t.Error("Expected toolkit to have a name")
				}
				// Cleanup
				toolkit.Close()
			}
		})
	}
}

func TestMCPToolkit_RegisterTools(t *testing.T) {
	testTools := []protocol.Tool{
		{
			Name:        "tool1",
			Description: "First tool",
			InputSchema: protocol.InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"param1": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
		{
			Name:        "tool2",
			Description: "Second tool",
			InputSchema: protocol.InputSchema{
				Type: "object",
			},
		},
	}

	mcpClient := createConnectedClient(t, testTools)
	defer mcpClient.Disconnect()

	ctx := context.Background()
	toolkit, err := New(ctx, Config{
		Client: mcpClient,
	})
	if err != nil {
		t.Fatalf("Failed to create toolkit: %v", err)
	}
	defer toolkit.Close()

	functions := toolkit.Functions()
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	if _, ok := functions["tool1"]; !ok {
		t.Error("Expected tool1 to be registered")
	}
	if _, ok := functions["tool2"]; !ok {
		t.Error("Expected tool2 to be registered")
	}
}

func TestMCPToolkit_ShouldIncludeTool(t *testing.T) {
	tests := []struct {
		name         string
		includeTools []string
		excludeTools []string
		toolName     string
		want         bool
	}{
		{
			name:         "no filters",
			includeTools: nil,
			excludeTools: nil,
			toolName:     "tool1",
			want:         true,
		},
		{
			name:         "whitelist includes tool",
			includeTools: []string{"tool1", "tool2"},
			excludeTools: nil,
			toolName:     "tool1",
			want:         true,
		},
		{
			name:         "whitelist excludes tool",
			includeTools: []string{"tool1", "tool2"},
			excludeTools: nil,
			toolName:     "tool3",
			want:         false,
		},
		{
			name:         "blacklist excludes tool",
			includeTools: nil,
			excludeTools: []string{"tool1"},
			toolName:     "tool1",
			want:         false,
		},
		{
			name:         "blacklist allows tool",
			includeTools: nil,
			excludeTools: []string{"tool1"},
			toolName:     "tool2",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolkit := &MCPToolkit{
				includeTools: tt.includeTools,
				excludeTools: tt.excludeTools,
			}

			got := toolkit.shouldIncludeTool(tt.toolName)
			if got != tt.want {
				t.Errorf("shouldIncludeTool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMCPToolkit_ConvertSchema(t *testing.T) {
	toolkit := &MCPToolkit{}

	tests := []struct {
		name        string
		schema      protocol.InputSchema
		wantParams  int
		checkParams func(map[string]interface{}, *testing.T)
	}{
		{
			name: "simple object schema",
			schema: protocol.InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "User name",
					},
					"age": map[string]interface{}{
						"type": "number",
					},
				},
				Required: []string{"name"},
			},
			wantParams: 2,
			checkParams: func(params map[string]interface{}, t *testing.T) {
				// Check name parameter
				// 检查名称参数
				// Note: The actual implementation stores toolkit.Parameter, not map[string]interface{}
				// This test would need adjustment based on actual return type
			},
		},
		{
			name: "non-object schema",
			schema: protocol.InputSchema{
				Type: "string",
			},
			wantParams: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := toolkit.convertSchema(tt.schema)
			if err != nil {
				t.Errorf("convertSchema() error = %v", err)
				return
			}

			if len(params) != tt.wantParams {
				t.Errorf("convertSchema() returned %d params, want %d", len(params), tt.wantParams)
			}
		})
	}
}

func TestMCPToolkit_WithFilters(t *testing.T) {
	testTools := []protocol.Tool{
		{Name: "tool1", Description: "Tool 1", InputSchema: protocol.InputSchema{Type: "object"}},
		{Name: "tool2", Description: "Tool 2", InputSchema: protocol.InputSchema{Type: "object"}},
		{Name: "tool3", Description: "Tool 3", InputSchema: protocol.InputSchema{Type: "object"}},
	}

	tests := []struct {
		name          string
		includeTools  []string
		excludeTools  []string
		expectedTools []string
	}{
		{
			name:          "include only tool1 and tool2",
			includeTools:  []string{"tool1", "tool2"},
			excludeTools:  nil,
			expectedTools: []string{"tool1", "tool2"},
		},
		{
			name:          "exclude tool3",
			includeTools:  nil,
			excludeTools:  []string{"tool3"},
			expectedTools: []string{"tool1", "tool2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpClient := createConnectedClient(t, testTools)
			defer mcpClient.Disconnect()

			ctx := context.Background()
			toolkit, err := New(ctx, Config{
				Client:       mcpClient,
				IncludeTools: tt.includeTools,
				ExcludeTools: tt.excludeTools,
			})
			if err != nil {
				t.Fatalf("Failed to create toolkit: %v", err)
			}
			defer toolkit.Close()

			functions := toolkit.Functions()
			if len(functions) != len(tt.expectedTools) {
				t.Errorf("Expected %d functions, got %d", len(tt.expectedTools), len(functions))
			}

			for _, expectedTool := range tt.expectedTools {
				if _, ok := functions[expectedTool]; !ok {
					t.Errorf("Expected %s to be registered", expectedTool)
				}
			}
		})
	}
}

func TestMCPToolkit_GetClient(t *testing.T) {
	testTools := []protocol.Tool{
		{Name: "test", InputSchema: protocol.InputSchema{Type: "object"}},
	}

	mcpClient := createConnectedClient(t, testTools)
	defer mcpClient.Disconnect()

	ctx := context.Background()
	toolkit, err := New(ctx, Config{
		Client: mcpClient,
	})
	if err != nil {
		t.Fatalf("Failed to create toolkit: %v", err)
	}
	defer toolkit.Close()

	if toolkit.GetClient() != mcpClient {
		t.Error("GetClient() returned wrong client")
	}
}

func TestMCPToolkit_GetTools(t *testing.T) {
	testTools := []protocol.Tool{
		{Name: "tool1", InputSchema: protocol.InputSchema{Type: "object"}},
		{Name: "tool2", InputSchema: protocol.InputSchema{Type: "object"}},
	}

	mcpClient := createConnectedClient(t, testTools)
	defer mcpClient.Disconnect()

	ctx := context.Background()
	toolkit, err := New(ctx, Config{
		Client: mcpClient,
	})
	if err != nil {
		t.Fatalf("Failed to create toolkit: %v", err)
	}
	defer toolkit.Close()

	tools := toolkit.GetTools()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}
