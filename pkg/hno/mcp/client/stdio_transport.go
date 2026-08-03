package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
	"github.com/rexleimo/agno-go/pkg/hno/mcp/security"
)

// StdioTransport implements Transport using stdin/stdout communication with a subprocess
// StdioTransport 使用与子进程的 stdin/stdout 通信实现 Transport
type StdioTransport struct {
	config  StdioConfig
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	running bool
	mu      sync.RWMutex

	// For request/response correlation
	// 用于请求/响应关联
	pendingRequests map[interface{}]chan *protocol.JSONRPCResponse
	requestMu       sync.RWMutex

	// For reading responses
	// 用于读取响应
	reader *bufio.Scanner

	// Context for shutdown
	// 用于关闭的上下文
	ctx    context.Context
	cancel context.CancelFunc
}

// NewStdioTransport creates a new stdio transport with the given configuration.
// Returns an error if the configuration is invalid or command validation fails.
//
// NewStdioTransport 使用给定配置创建新的 stdio 传输。
// 如果配置无效或命令验证失败则返回错误。
func NewStdioTransport(config StdioConfig) (*StdioTransport, error) {
	if config.Command == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// Validate command if enabled
	// By default, validation is enabled when AllowedCommands is set
	// Or when ValidateCommand is explicitly true
	// 如果启用则验证命令
	// 默认情况下，当设置了 AllowedCommands 或 ValidateCommand 显式为 true 时启用验证
	shouldValidate := config.ValidateCommand || len(config.AllowedCommands) > 0

	if shouldValidate {
		// Create validator with custom or default whitelist
		// 使用自定义或默认白名单创建验证器
		var validator *security.CommandValidator
		if len(config.AllowedCommands) > 0 {
			validator = security.NewCustomCommandValidator(
				config.AllowedCommands,
				security.DefaultBlockedChars(),
			)
		} else {
			validator = security.NewCommandValidator()
		}

		// Create path validator
		// 创建路径验证器
		pathValidator := security.NewPathValidator(validator)

		// Validate the command and arguments
		// 验证命令和参数
		if err := pathValidator.ValidateExecutable(config.Command, config.Args); err != nil {
			return nil, fmt.Errorf("command validation failed: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &StdioTransport{
		config:          config,
		pendingRequests: make(map[interface{}]chan *protocol.JSONRPCResponse),
		ctx:             ctx,
		cancel:          cancel,
	}, nil
}

// Start initializes the transport and begins communication.
// Start 初始化传输并开始通信。
func (t *StdioTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("transport already running")
	}

	// Create command
	// 创建命令
	t.cmd = exec.CommandContext(ctx, t.config.Command, t.config.Args...)
	if t.config.WorkingDir != "" {
		t.cmd.Dir = t.config.WorkingDir
	}
	if len(t.config.Env) > 0 {
		t.cmd.Env = append(t.cmd.Env, t.config.Env...)
	}

	// Setup pipes
	// 设置管道
	var err error
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	// 启动进程
	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Create scanner for reading responses
	// 创建扫描器以读取响应
	t.reader = bufio.NewScanner(t.stdout)
	// Increase buffer size to handle large responses
	// 增加缓冲区大小以处理大响应
	buf := make([]byte, 0, 64*1024)
	t.reader.Buffer(buf, 1024*1024) // 1MB max

	t.running = true

	// Start goroutine to read responses
	// 启动 goroutine 以读取响应
	go t.readLoop()

	// Start goroutine to read stderr (for debugging)
	// 启动 goroutine 以读取 stderr（用于调试）
	go t.readStderr()

	return nil
}

// Stop gracefully shuts down the transport.
// Stop 优雅地关闭传输。
func (t *StdioTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false

	// Cancel context to stop goroutines
	// 取消上下文以停止 goroutines
	t.cancel()

	// Close stdin to signal process to exit
	// 关闭 stdin 以通知进程退出
	if t.stdin != nil {
		t.stdin.Close()
	}

	// Wait for process to exit with timeout
	// 等待进程退出（带超时）
	done := make(chan error, 1)
	go func() {
		done <- t.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Force kill if not exited
		// 如果未退出则强制杀死
		if t.cmd.Process != nil {
			t.cmd.Process.Kill()
		}
		return fmt.Errorf("process did not exit gracefully, killed")
	case err := <-done:
		return err
	}
}

// Send sends a JSON-RPC request and returns the response.
// Send 发送 JSON-RPC 请求并返回响应。
func (t *StdioTransport) Send(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
	if !t.IsRunning() {
		return nil, fmt.Errorf("transport not running")
	}

	// Create response channel
	// 创建响应通道
	respChan := make(chan *protocol.JSONRPCResponse, 1)

	// Register pending request
	// 注册待处理请求
	t.requestMu.Lock()
	t.pendingRequests[req.ID] = respChan
	t.requestMu.Unlock()

	// Cleanup on return
	// 返回时清理
	defer func() {
		t.requestMu.Lock()
		delete(t.pendingRequests, req.ID)
		t.requestMu.Unlock()
	}()

	// Send request
	// 发送请求
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Add newline delimiter
	// 添加换行分隔符
	data = append(data, '\n')

	t.mu.RLock()
	if _, err := t.stdin.Write(data); err != nil {
		t.mu.RUnlock()
		return nil, fmt.Errorf("failed to write request: %w", err)
	}
	t.mu.RUnlock()

	// Wait for response with timeout
	// 等待响应（带超时）
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respChan:
		return resp, nil
	case <-t.ctx.Done():
		return nil, fmt.Errorf("transport stopped")
	}
}

// SendNotification sends a JSON-RPC notification (no response expected).
// SendNotification 发送 JSON-RPC 通知（不期望响应）。
func (t *StdioTransport) SendNotification(ctx context.Context, notif *protocol.JSONRPCNotification) error {
	if !t.IsRunning() {
		return fmt.Errorf("transport not running")
	}

	data, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Add newline delimiter
	// 添加换行分隔符
	data = append(data, '\n')

	t.mu.RLock()
	defer t.mu.RUnlock()

	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write notification: %w", err)
	}

	return nil
}

// IsRunning returns true if the transport is currently active.
// IsRunning 返回传输是否正在运行。
func (t *StdioTransport) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}

// readLoop continuously reads responses from stdout
// readLoop 持续从 stdout 读取响应
func (t *StdioTransport) readLoop() {
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		if !t.reader.Scan() {
			// EOF or error
			// EOF 或错误
			if err := t.reader.Err(); err != nil {
				// Log error (in production, use proper logger)
				// 记录错误（在生产中，使用适当的日志记录器）
				_ = err
			}
			return
		}

		line := t.reader.Bytes()
		if len(line) == 0 {
			continue
		}

		// Try to parse as response
		// 尝试解析为响应
		var resp protocol.JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			// Could be a notification or invalid JSON
			// 可能是通知或无效的 JSON
			continue
		}

		// Dispatch to waiting request
		// 分发到等待的请求
		t.requestMu.RLock()
		if ch, ok := t.pendingRequests[resp.ID]; ok {
			select {
			case ch <- &resp:
			default:
				// Channel full or closed, ignore
				// 通道已满或已关闭，忽略
			}
		}
		t.requestMu.RUnlock()
	}
}

// readStderr reads and discards stderr output
// readStderr 读取并丢弃 stderr 输出
func (t *StdioTransport) readStderr() {
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		// In production, log stderr to proper logger
		// 在生产中，将 stderr 记录到适当的日志记录器
		_ = scanner.Text()
	}
}
