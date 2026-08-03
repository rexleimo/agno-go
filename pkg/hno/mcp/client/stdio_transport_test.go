package client

import (
	"context"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
)

func TestNewStdioTransport(t *testing.T) {
	tests := []struct {
		name    string
		config  StdioConfig
		wantErr bool
	}{
		{
			name: "valid config with validation disabled",
			config: StdioConfig{
				Command:         "echo",
				Args:            []string{"test"},
				ValidateCommand: false, // Disable validation for testing
			},
			wantErr: false,
		},
		{
			name: "valid config with python (whitelisted)",
			config: StdioConfig{
				Command:         "python",
				Args:            []string{"-m", "test"},
				ValidateCommand: true,
			},
			wantErr: false,
		},
		{
			name: "empty command",
			config: StdioConfig{
				Command: "",
			},
			wantErr: true,
		},
		{
			name: "dangerous command rejected",
			config: StdioConfig{
				Command:         "rm",
				Args:            []string{"-rf", "/"},
				ValidateCommand: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := NewStdioTransport(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStdioTransport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if transport == nil {
					t.Error("Expected non-nil transport")
				}
				if transport.IsRunning() {
					t.Error("Transport should not be running before Start()")
				}
			}
		})
	}
}

func TestStdioTransport_StartStop(t *testing.T) {
	// Use a simple command that will run and respond
	// 使用一个简单的将运行并响应的命令
	transport, err := NewStdioTransport(StdioConfig{
		Command:         "cat", // cat will echo back what we send
		ValidateCommand: false, // Disable validation for testing
	})
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start transport
	if err := transport.Start(ctx); err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	if !transport.IsRunning() {
		t.Error("Transport should be running after Start()")
	}

	// Stop transport
	if err := transport.Stop(); err != nil {
		// cat might exit with error when stdin is closed, that's ok
		// cat 在 stdin 关闭时可能会以错误退出，这没关系
		t.Logf("Stop returned error (expected for cat): %v", err)
	}

	if transport.IsRunning() {
		t.Error("Transport should not be running after Stop()")
	}
}

func TestStdioTransport_Send_NotRunning(t *testing.T) {
	transport, err := NewStdioTransport(StdioConfig{
		Command:         "echo",
		ValidateCommand: false,
	})
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	req, err := protocol.NewRequest("test", nil, 1)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	ctx := context.Background()
	_, err = transport.Send(ctx, req)
	if err == nil {
		t.Error("Expected error when sending on non-running transport")
	}
}

func TestStdioTransport_SendNotification_NotRunning(t *testing.T) {
	transport, err := NewStdioTransport(StdioConfig{
		Command:         "echo",
		ValidateCommand: false,
	})
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	notif, err := protocol.NewNotification("test", nil)
	if err != nil {
		t.Fatalf("Failed to create notification: %v", err)
	}

	ctx := context.Background()
	err = transport.SendNotification(ctx, notif)
	if err == nil {
		t.Error("Expected error when sending notification on non-running transport")
	}
}

func TestStdioTransport_DoubleStart(t *testing.T) {
	transport, err := NewStdioTransport(StdioConfig{
		Command:         "cat",
		ValidateCommand: false,
	})
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First start
	if err := transport.Start(ctx); err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}
	defer transport.Stop()

	// Second start should fail
	if err := transport.Start(ctx); err == nil {
		t.Error("Expected error when starting already running transport")
	}
}

func TestStdioTransport_InvalidCommand(t *testing.T) {
	transport, err := NewStdioTransport(StdioConfig{
		Command:         "nonexistent-command-12345",
		ValidateCommand: false, // Disable validation to test Start() failure
	})
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = transport.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting transport with invalid command")
		transport.Stop()
	}
}

// MockEchoServer is a simple test helper that echoes back responses
// MockEchoServer 是一个简单的测试辅助工具，用于回显响应
type MockEchoServer struct {
	responses map[interface{}]*protocol.JSONRPCResponse
}

func TestStdioTransport_SendReceive(t *testing.T) {
	// This test requires a real MCP server for proper request/response flow
	// Testing with 'cat' is not reliable because cat echoes the request, not a response
	// In integration tests, we would use a real MCP server
	// 此测试需要真实的 MCP 服务器才能进行正确的请求/响应流
	// 使用 'cat' 进行测试不可靠，因为 cat 回显请求而不是响应
	// 在集成测试中，我们将使用真实的 MCP 服务器
	t.Skip("Requires real MCP server for proper testing")
}

func TestStdioTransport_ContextCancellation(t *testing.T) {
	transport, err := NewStdioTransport(StdioConfig{
		Command:         "cat",
		ValidateCommand: false,
	})
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	startCtx, startCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer startCancel()

	if err := transport.Start(startCtx); err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}
	defer transport.Stop()

	// Create request with short timeout
	// 创建具有短超时的请求
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := protocol.NewRequest("test", nil, 1)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Send request but don't send response - should timeout
	// 发送请求但不发送响应 - 应该超时
	_, err = transport.Send(ctx, req)
	if err == nil {
		t.Error("Expected timeout error")
	}
	if err != context.DeadlineExceeded {
		t.Logf("Got error: %v (expected context.DeadlineExceeded)", err)
	}
}
