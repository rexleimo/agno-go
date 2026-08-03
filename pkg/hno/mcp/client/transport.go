package client

import (
	"context"
	"io"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
)

// Transport defines the interface for MCP communication transports
// Transport 定义 MCP 通信传输的接口
type Transport interface {
	// Start initializes the transport and begins communication
	// Start 初始化传输并开始通信
	Start(ctx context.Context) error

	// Stop gracefully shuts down the transport
	// Stop 优雅地关闭传输
	Stop() error

	// Send sends a JSON-RPC request and returns the response
	// Send 发送 JSON-RPC 请求并返回响应
	Send(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error)

	// SendNotification sends a JSON-RPC notification (no response expected)
	// SendNotification 发送 JSON-RPC 通知（不期望响应）
	SendNotification(ctx context.Context, notif *protocol.JSONRPCNotification) error

	// IsRunning returns true if the transport is currently active
	// IsRunning 返回传输是否正在运行
	IsRunning() bool
}

// StdioConfig contains configuration for stdio transport
// StdioConfig 包含 stdio 传输的配置
type StdioConfig struct {
	// Command is the command to execute (e.g., "python", "node")
	// Command 是要执行的命令（例如 "python", "node"）
	Command string

	// Args are the command arguments
	// Args 是命令参数
	Args []string

	// Env contains additional environment variables (key=value format)
	// Env 包含额外的环境变量（key=value 格式）
	Env []string

	// WorkingDir is the working directory for the command
	// WorkingDir 是命令的工作目录
	WorkingDir string

	// ValidateCommand enables command validation before execution (default: true)
	// ValidateCommand 在执行前启用命令验证（默认: true）
	ValidateCommand bool

	// AllowedCommands is a custom whitelist of allowed commands
	// If nil, default whitelist will be used
	// AllowedCommands 是允许的命令自定义白名单
	// 如果为 nil，将使用默认白名单
	AllowedCommands []string
}

// StreamCallback is called when a message is received from the transport
// StreamCallback 在从传输接收到消息时调用
type StreamCallback func(message interface{}) error

// ReadWriter wraps io.Reader and io.Writer for testing purposes
// ReadWriter 包装 io.Reader 和 io.Writer 用于测试
type ReadWriter struct {
	Reader io.Reader
	Writer io.Writer
}

// Read reads from the underlying reader
func (rw *ReadWriter) Read(p []byte) (n int, err error) {
	return rw.Reader.Read(p)
}

// Write writes to the underlying writer
func (rw *ReadWriter) Write(p []byte) (n int, err error) {
	return rw.Writer.Write(p)
}
