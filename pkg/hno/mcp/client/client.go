package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
)

// Client represents an MCP client that communicates with an MCP server
// Client 表示与 MCP 服务器通信的 MCP 客户端
type Client struct {
	transport Transport
	config    Config

	// Server information after initialization
	// 初始化后的服务器信息
	serverInfo   *protocol.ServerInfo
	capabilities map[string]interface{}
	initialized  bool
	initMu       sync.RWMutex

	// Request ID counter
	// 请求 ID 计数器
	requestID atomic.Int64
}

// Config contains configuration for the MCP client
// Config 包含 MCP 客户端的配置
type Config struct {
	// ClientName is the name of this client
	// ClientName 是此客户端的名称
	ClientName string

	// ClientVersion is the version of this client
	// ClientVersion 是此客户端的版本
	ClientVersion string

	// ProtocolVersion is the MCP protocol version to use (default: "1.0")
	// ProtocolVersion 是要使用的 MCP 协议版本（默认: "1.0"）
	ProtocolVersion string

	// Capabilities are the capabilities supported by this client
	// Capabilities 是此客户端支持的功能
	Capabilities map[string]interface{}

	// ReconnectAttempts controls automatic reconnect attempts when transport fails (default: 3)
	ReconnectAttempts int

	// ReconnectBackoff controls initial backoff duration between reconnect attempts (default: 500ms, exponential)
	ReconnectBackoff time.Duration

	// OnUnauthorized is invoked when server returns an unauthorized error (e.g., to refresh tokens).
	OnUnauthorized func(ctx context.Context) error
}

// New creates a new MCP client with the given transport and configuration.
// Returns an error if the configuration is invalid.
//
// New 使用给定的传输和配置创建新的 MCP 客户端。
// 如果配置无效则返回错误。
func New(transport Transport, config Config) (*Client, error) {
	if transport == nil {
		return nil, fmt.Errorf("transport cannot be nil")
	}

	if config.ClientName == "" {
		config.ClientName = "agno-go-mcp-client"
	}

	if config.ClientVersion == "" {
		config.ClientVersion = "0.1.0"
	}

	if config.ProtocolVersion == "" {
		config.ProtocolVersion = "1.0"
	}

	if config.ReconnectAttempts < 0 {
		config.ReconnectAttempts = 0
	}
	if config.ReconnectBackoff <= 0 {
		config.ReconnectBackoff = 500 * time.Millisecond
	}

	return &Client{
		transport: transport,
		config:    config,
	}, nil
}

// Connect starts the transport and initializes the connection with the MCP server.
// Connect 启动传输并初始化与 MCP 服务器的连接。
func (c *Client) Connect(ctx context.Context) error {
	// Start transport
	// 启动传输
	if err := c.transport.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	if err := c.initialize(ctx); err != nil {
		_ = c.transport.Stop()
		return err
	}

	return nil
}

func (c *Client) initialize(ctx context.Context) error {
	initParams := protocol.InitializeParams{
		ProtocolVersion: c.config.ProtocolVersion,
		ClientInfo: protocol.ClientInfo{
			Name:    c.config.ClientName,
			Version: c.config.ClientVersion,
		},
		Capabilities: c.config.Capabilities,
	}

	reqID := c.requestID.Add(1)
	req, err := protocol.NewRequest(protocol.MethodInitialize, initParams, reqID)
	if err != nil {
		return fmt.Errorf("failed to create initialize request: %w", err)
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("failed to initialize: server error [%d] %s", resp.Error.Code, resp.Error.Message)
	}

	var initResult protocol.InitializeResult
	if resp.Result != nil {
		if err := parseResult(resp.Result, &initResult); err != nil {
			return fmt.Errorf("failed to parse initialize result: %w", err)
		}
	}

	c.initMu.Lock()
	c.serverInfo = &initResult.ServerInfo
	c.capabilities = initResult.Capabilities
	c.initialized = true
	c.initMu.Unlock()

	notif, err := protocol.NewNotification("initialized", nil)
	if err != nil {
		return fmt.Errorf("failed to create initialized notification: %w", err)
	}

	if err := c.transport.SendNotification(ctx, notif); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

func (c *Client) reconnect(ctx context.Context, attempt int) error {
	c.initMu.Lock()
	c.initialized = false
	c.serverInfo = nil
	c.capabilities = nil
	c.initMu.Unlock()

	_ = c.transport.Stop()

	backoff := c.config.ReconnectBackoff
	delay := backoff * time.Duration(1<<attempt)
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	if err := c.transport.Start(ctx); err != nil {
		return fmt.Errorf("failed to restart transport: %w", err)
	}

	if err := c.initialize(ctx); err != nil {
		return fmt.Errorf("failed to reinitialize: %w", err)
	}
	return nil
}

// Disconnect closes the connection with the MCP server.
// Disconnect 关闭与 MCP 服务器的连接。
func (c *Client) Disconnect() error {
	c.initMu.Lock()
	c.initialized = false
	c.serverInfo = nil
	c.capabilities = nil
	c.initMu.Unlock()

	return c.transport.Stop()
}

// IsConnected returns true if the client is connected and initialized.
// IsConnected 返回客户端是否已连接并初始化。
func (c *Client) IsConnected() bool {
	c.initMu.RLock()
	defer c.initMu.RUnlock()
	return c.initialized && c.transport.IsRunning()
}

// GetServerInfo returns information about the connected server.
// Returns nil if not connected.
//
// GetServerInfo 返回有关已连接服务器的信息。
// 如果未连接则返回 nil。
func (c *Client) GetServerInfo() *protocol.ServerInfo {
	c.initMu.RLock()
	defer c.initMu.RUnlock()
	return c.serverInfo
}

// GetCapabilities returns the server's capabilities.
// Returns nil if not connected.
//
// GetCapabilities 返回服务器的功能。
// 如果未连接则返回 nil。
func (c *Client) GetCapabilities() map[string]interface{} {
	c.initMu.RLock()
	defer c.initMu.RUnlock()
	return c.capabilities
}

// ListTools retrieves the list of available tools from the server.
// ListTools 从服务器检索可用工具列表。
func (c *Client) ListTools(ctx context.Context) ([]protocol.Tool, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	var result protocol.ToolsListResult
	if err := c.call(ctx, protocol.MethodToolsList, protocol.ToolsListParams{}, &result); err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return result.Tools, nil
}

// CallTool calls a tool on the server with the given arguments.
// CallTool 使用给定参数调用服务器上的工具。
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*protocol.ToolsCallResult, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	params := protocol.ToolsCallParams{
		Name:      name,
		Arguments: arguments,
	}

	var result protocol.ToolsCallResult
	if err := c.call(ctx, protocol.MethodToolsCall, params, &result); err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	if result.IsError {
		return &result, fmt.Errorf("tool execution failed: %v", result.Content)
	}

	return &result, nil
}

// ListResources retrieves the list of available resources from the server.
// ListResources 从服务器检索可用资源列表。
func (c *Client) ListResources(ctx context.Context) ([]protocol.Resource, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	var result protocol.ResourcesListResult
	if err := c.call(ctx, protocol.MethodResourcesList, protocol.ResourcesListParams{}, &result); err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	return result.Resources, nil
}

// ReadResource reads a resource from the server.
// ReadResource 从服务器读取资源。
func (c *Client) ReadResource(ctx context.Context, uri string) ([]protocol.Content, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	params := protocol.ResourcesReadParams{
		URI: uri,
	}

	var result protocol.ResourcesReadResult
	if err := c.call(ctx, protocol.MethodResourcesRead, params, &result); err != nil {
		return nil, fmt.Errorf("failed to read resource: %w", err)
	}

	return result.Contents, nil
}

// ListPrompts retrieves the list of available prompts from the server.
// ListPrompts 从服务器检索可用提示列表。
func (c *Client) ListPrompts(ctx context.Context) ([]protocol.Prompt, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	var result protocol.PromptsListResult
	if err := c.call(ctx, protocol.MethodPromptsList, protocol.PromptsListParams{}, &result); err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	return result.Prompts, nil
}

// GetPrompt retrieves a prompt from the server with the given arguments.
// GetPrompt 使用给定参数从服务器检索提示。
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*protocol.PromptsGetResult, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	params := protocol.PromptsGetParams{
		Name:      name,
		Arguments: arguments,
	}

	var result protocol.PromptsGetResult
	if err := c.call(ctx, protocol.MethodPromptsGet, params, &result); err != nil {
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	return &result, nil
}

// call is a helper method to make JSON-RPC calls
// call 是一个辅助方法，用于进行 JSON-RPC 调用
func (c *Client) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	id := c.requestID.Add(1)
	maxAttempts := c.config.ReconnectAttempts + 1
	unauthorizedRetried := false

	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := protocol.NewRequest(method, params, id)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.transport.Send(ctx, req)
		if err != nil {
			if attempt < c.config.ReconnectAttempts {
				if recErr := c.reconnect(ctx, attempt); recErr == nil {
					continue
				} else {
					return fmt.Errorf("failed to reconnect transport: %w", recErr)
				}
			}
			return fmt.Errorf("failed to send request: %w", err)
		}

		if resp.Error != nil {
			if !unauthorizedRetried && c.handleUnauthorized(ctx, resp.Error, attempt) {
				unauthorizedRetried = true
				continue
			}
			return fmt.Errorf("server error [%d]: %s", resp.Error.Code, resp.Error.Message)
		}

		if result != nil && resp.Result != nil {
			if err := parseResult(resp.Result, result); err != nil {
				return fmt.Errorf("failed to parse result: %w", err)
			}
		}
		return nil
	}

	return fmt.Errorf("request failed after %d attempts", maxAttempts)
}

func (c *Client) handleUnauthorized(ctx context.Context, rpcErr *protocol.JSONRPCError, attempt int) bool {
	if !isUnauthorized(rpcErr) || c.config.OnUnauthorized == nil {
		return false
	}

	if err := c.config.OnUnauthorized(ctx); err != nil {
		return false
	}

	if err := c.reconnect(ctx, attempt); err != nil {
		return false
	}

	return true
}

func isUnauthorized(err *protocol.JSONRPCError) bool {
	if err == nil {
		return false
	}
	if err.Code == 401 {
		return true
	}
	message := strings.ToLower(err.Message)
	return strings.Contains(message, "unauthorized") || strings.Contains(message, "401")
}
