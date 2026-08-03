package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		transport Transport
		config    Config
		wantErr   bool
	}{
		{
			name:      "valid config",
			transport: NewMockTransport(),
			config: Config{
				ClientName:    "test-client",
				ClientVersion: "1.0.0",
			},
			wantErr: false,
		},
		{
			name:      "nil transport",
			transport: nil,
			config:    Config{},
			wantErr:   true,
		},
		{
			name:      "default values",
			transport: NewMockTransport(),
			config:    Config{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.transport, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if client == nil {
					t.Error("Expected non-nil client")
				}
				if client.IsConnected() {
					t.Error("Client should not be connected before Connect()")
				}
			}
		})
	}
}

func TestClient_Connect(t *testing.T) {
	mockTransport := NewMockTransport()

	// Setup mock response for initialize
	// 设置初始化的模拟响应
	initResult := protocol.InitializeResult{
		ProtocolVersion: "1.0",
		ServerInfo: protocol.ServerInfo{
			Name:    "test-server",
			Version: "1.0.0",
		},
		Capabilities: map[string]interface{}{
			"tools": true,
		},
	}
	resultData, _ := json.Marshal(initResult)
	mockTransport.SetResponse(1, &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  resultData,
		ID:      int64(1),
	})

	client, err := New(mockTransport, Config{
		ClientName:    "test-client",
		ClientVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !client.IsConnected() {
		t.Error("Client should be connected after Connect()")
	}

	serverInfo := client.GetServerInfo()
	if serverInfo == nil {
		t.Fatal("Expected server info, got nil")
	}
	if serverInfo.Name != "test-server" {
		t.Errorf("Expected server name 'test-server', got %s", serverInfo.Name)
	}

	// Cleanup
	client.Disconnect()
}

func TestClient_Connect_TransportError(t *testing.T) {
	mockTransport := NewMockTransport()
	mockTransport.startError = fmt.Errorf("transport start failed")

	client, err := New(mockTransport, Config{})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected error when transport fails to start")
	}
}

func TestClient_Disconnect(t *testing.T) {
	mockTransport := NewMockTransport()

	// Setup mock response for initialize
	// 设置初始化的模拟响应
	initResult := protocol.InitializeResult{
		ProtocolVersion: "1.0",
		ServerInfo: protocol.ServerInfo{
			Name:    "test-server",
			Version: "1.0.0",
		},
	}
	resultData, _ := json.Marshal(initResult)
	mockTransport.SetResponse(1, &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  resultData,
		ID:      int64(1),
	})

	client, err := New(mockTransport, Config{})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if err := client.Disconnect(); err != nil {
		t.Errorf("Disconnect() failed: %v", err)
	}

	if client.IsConnected() {
		t.Error("Client should not be connected after Disconnect()")
	}

	if client.GetServerInfo() != nil {
		t.Error("Server info should be nil after disconnect")
	}
}

func TestClient_ReconnectOnSendFailure(t *testing.T) {
	mockTransport := NewMockTransport()

	initResult := protocol.InitializeResult{
		ProtocolVersion: "1.0",
		ServerInfo:      protocol.ServerInfo{Name: "test-server", Version: "1.0.0"},
	}
	initData, _ := json.Marshal(initResult)

	toolsResult := protocol.ToolsListResult{
		Tools: []protocol.Tool{{Name: "reconnect_tool", InputSchema: protocol.InputSchema{Type: "object"}}},
	}
	toolsData, _ := json.Marshal(toolsResult)

	var sendCount int
	mockTransport.SetSendFunc(func(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
		switch req.Method {
		case protocol.MethodInitialize:
			return &protocol.JSONRPCResponse{JSONRPC: protocol.JSONRPCVersion, Result: initData, ID: req.ID}, nil
		case protocol.MethodToolsList:
			if sendCount == 0 {
				sendCount++
				return nil, errors.New("transport failure")
			}
			return &protocol.JSONRPCResponse{JSONRPC: protocol.JSONRPCVersion, Result: toolsData, ID: req.ID}, nil
		default:
			return &protocol.JSONRPCResponse{JSONRPC: protocol.JSONRPCVersion, Result: json.RawMessage(`{}`), ID: req.ID}, nil
		}
	})

	client, err := New(mockTransport, Config{ReconnectAttempts: 2, ReconnectBackoff: time.Millisecond})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	if mockTransport.StartCount() < 2 {
		t.Fatalf("expected transport restart, start count %d", mockTransport.StartCount())
	}
}

func TestClient_OnUnauthorizedRefresh(t *testing.T) {
	mockTransport := NewMockTransport()

	initResult := protocol.InitializeResult{
		ProtocolVersion: "1.0",
		ServerInfo:      protocol.ServerInfo{Name: "test-server", Version: "1.0.0"},
	}
	initData, _ := json.Marshal(initResult)

	toolsResult := protocol.ToolsListResult{
		Tools: []protocol.Tool{{Name: "refreshed_tool", InputSchema: protocol.InputSchema{Type: "object"}}},
	}
	toolsData, _ := json.Marshal(toolsResult)

	unauthorized := true
	mockTransport.SetSendFunc(func(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
		switch req.Method {
		case protocol.MethodInitialize:
			return &protocol.JSONRPCResponse{JSONRPC: protocol.JSONRPCVersion, Result: initData, ID: req.ID}, nil
		case protocol.MethodToolsList:
			if unauthorized {
				unauthorized = false
				resp, _ := protocol.NewErrorResponse(401, "unauthorized", nil, req.ID)
				return resp, nil
			}
			return &protocol.JSONRPCResponse{JSONRPC: protocol.JSONRPCVersion, Result: toolsData, ID: req.ID}, nil
		default:
			return &protocol.JSONRPCResponse{JSONRPC: protocol.JSONRPCVersion, Result: json.RawMessage(`{}`), ID: req.ID}, nil
		}
	})

	refreshed := false
	client, err := New(mockTransport, Config{
		ReconnectAttempts: 2,
		ReconnectBackoff:  time.Millisecond,
		OnUnauthorized: func(ctx context.Context) error {
			refreshed = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}
	if !refreshed {
		t.Fatalf("expected OnUnauthorized to be invoked")
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
}

func TestClient_ListTools(t *testing.T) {
	mockTransport := NewMockTransport()

	// Setup mock response for initialize
	// 设置初始化的模拟响应
	initResult := protocol.InitializeResult{
		ProtocolVersion: "1.0",
		ServerInfo:      protocol.ServerInfo{Name: "test-server", Version: "1.0.0"},
	}
	initData, _ := json.Marshal(initResult)
	mockTransport.SetResponse(1, &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  initData,
		ID:      int64(1),
	})

	// Setup mock response for tools/list
	// 设置 tools/list 的模拟响应
	toolsResult := protocol.ToolsListResult{
		Tools: []protocol.Tool{
			{
				Name:        "test_tool",
				Description: "A test tool",
				InputSchema: protocol.InputSchema{
					Type: "object",
				},
			},
		},
	}
	toolsData, _ := json.Marshal(toolsResult)
	mockTransport.SetResponse(2, &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  toolsData,
		ID:      int64(2),
	})

	client, err := New(mockTransport, Config{})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools() failed: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	if tools[0].Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %s", tools[0].Name)
	}
}

func TestClient_ListTools_NotConnected(t *testing.T) {
	mockTransport := NewMockTransport()
	client, err := New(mockTransport, Config{})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	_, err = client.ListTools(ctx)
	if err == nil {
		t.Error("Expected error when calling ListTools on disconnected client")
	}
}

func TestClient_CallTool(t *testing.T) {
	mockTransport := NewMockTransport()

	// Setup mock response for initialize
	// 设置初始化的模拟响应
	initResult := protocol.InitializeResult{
		ProtocolVersion: "1.0",
		ServerInfo:      protocol.ServerInfo{Name: "test-server", Version: "1.0.0"},
	}
	initData, _ := json.Marshal(initResult)
	mockTransport.SetResponse(1, &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  initData,
		ID:      int64(1),
	})

	// Setup mock response for tools/call
	// 设置 tools/call 的模拟响应
	callResult := protocol.ToolsCallResult{
		Content: []protocol.Content{
			{
				Type: protocol.ContentTypeText,
				Text: "Result: 3",
			},
		},
		IsError: false,
	}
	callData, _ := json.Marshal(callResult)
	mockTransport.SetResponse(2, &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  callData,
		ID:      int64(2),
	})

	client, err := New(mockTransport, Config{})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	result, err := client.CallTool(ctx, "add", map[string]interface{}{
		"a": 1,
		"b": 2,
	})
	if err != nil {
		t.Fatalf("CallTool() failed: %v", err)
	}

	if result.IsError {
		t.Error("Expected successful tool call")
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(result.Content))
	}
}

func TestClient_CallTool_Error(t *testing.T) {
	mockTransport := NewMockTransport()

	// Setup mock response for initialize
	// 设置初始化的模拟响应
	initResult := protocol.InitializeResult{
		ProtocolVersion: "1.0",
		ServerInfo:      protocol.ServerInfo{Name: "test-server", Version: "1.0.0"},
	}
	initData, _ := json.Marshal(initResult)
	mockTransport.SetResponse(1, &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  initData,
		ID:      int64(1),
	})

	// Setup mock error response for tools/call
	// 设置 tools/call 的模拟错误响应
	callResult := protocol.ToolsCallResult{
		Content: []protocol.Content{
			{
				Type: protocol.ContentTypeText,
				Text: "Tool execution failed",
			},
		},
		IsError: true,
	}
	callData, _ := json.Marshal(callResult)
	mockTransport.SetResponse(2, &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  callData,
		ID:      int64(2),
	})

	client, err := New(mockTransport, Config{})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	_, err = client.CallTool(ctx, "failing_tool", nil)
	if err == nil {
		t.Error("Expected error when tool execution fails")
	}
}
