package toolkit

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/client"
	"github.com/rexleimo/agno-go/pkg/hno/mcp/content"
	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// MCPToolkit integrates an MCP server as a toolkit for agno agents
// MCPToolkit 将 MCP 服务器集成为 agno agents 的工具包
type MCPToolkit struct {
    *toolkit.BaseToolkit
    client         *client.Client
    contentHandler *content.Handler
    tools          []protocol.Tool
    includeTools   []string // Optional whitelist
    excludeTools   []string // Optional blacklist
    toolNamePrefix string   // Optional name prefix for registered tools
}

// Config contains configuration for MCPToolkit
// Config 包含 MCPToolkit 的配置
type Config struct {
	// Name is the toolkit name (optional, defaults to server name)
	// Name 是工具包名称（可选，默认为服务器名称）
	Name string

	// Client is the MCP client to use
	// Client 是要使用的 MCP 客户端
	Client *client.Client

	// IncludeTools is a whitelist of tool names to include (optional)
	// If set, only these tools will be available
	// IncludeTools 是要包含的工具名称白名单（可选）
	// 如果设置，仅这些工具可用
	IncludeTools []string

	// ExcludeTools is a blacklist of tool names to exclude (optional)
	// ExcludeTools 是要排除的工具名称黑名单（可选）
    ExcludeTools []string

    // ToolNamePrefix is an optional prefix added to registered tool names
    // ToolNamePrefix 是可选的注册工具名前缀
    ToolNamePrefix string
}

// New creates a new MCP toolkit with the given configuration.
// The client must be connected before creating the toolkit.
// Returns an error if the client is not connected or tool discovery fails.
//
// New 使用给定配置创建新的 MCP 工具包。
// 创建工具包前客户端必须已连接。
// 如果客户端未连接或工具发现失败，则返回错误。
func New(ctx context.Context, config Config) (*MCPToolkit, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	if !config.Client.IsConnected() {
		return nil, fmt.Errorf("client must be connected before creating toolkit")
	}

	// Discover tools from the MCP server
	// 从 MCP 服务器发现工具
	tools, err := config.Client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover tools: %w", err)
	}

	// Determine toolkit name
	// 确定工具包名称
	name := config.Name
	if name == "" {
		serverInfo := config.Client.GetServerInfo()
		if serverInfo != nil {
			name = fmt.Sprintf("mcp-%s", serverInfo.Name)
		} else {
			name = "mcp-toolkit"
		}
	}

    t := &MCPToolkit{
        BaseToolkit:    toolkit.NewBaseToolkit(name),
        client:         config.Client,
        contentHandler: content.New(),
        tools:          tools,
        includeTools:   config.IncludeTools,
        excludeTools:   config.ExcludeTools,
        toolNamePrefix: config.ToolNamePrefix,
    }

	// Register tools as functions
	// 将工具注册为函数
	if err := t.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return t, nil
}

// registerTools converts MCP tools to agno toolkit functions
// registerTools 将 MCP 工具转换为 agno 工具包函数
func (t *MCPToolkit) registerTools() error {
	for _, tool := range t.tools {
		// Check if tool should be included
		// 检查是否应包含工具
		if !t.shouldIncludeTool(tool.Name) {
			continue
		}

		// Convert MCP tool schema to agno parameter schema
		// 将 MCP 工具模式转换为 agno 参数模式
		params, err := t.convertSchema(tool.InputSchema)
		if err != nil {
			return fmt.Errorf("failed to convert schema for tool %s: %w", tool.Name, err)
		}

		// Create closure to capture tool name
		// 创建闭包以捕获工具名称
		toolName := tool.Name

        // Register function (apply optional prefix)
        // 注册函数（应用可选前缀）
        regName := tool.Name
        if p := t.toolNamePrefix; p != "" {
            regName = p + tool.Name
        }
        t.RegisterFunction(&toolkit.Function{
            Name:        regName,
            Description: tool.Description,
            Parameters:  params,
            Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
                return t.callTool(ctx, toolName, args)
            },
        })
	}

	return nil
}

// shouldIncludeTool checks if a tool should be included based on whitelist/blacklist
// shouldIncludeTool 根据白名单/黑名单检查是否应包含工具
func (t *MCPToolkit) shouldIncludeTool(toolName string) bool {
	// Check whitelist first (if set)
	// 首先检查白名单（如果已设置）
	if len(t.includeTools) > 0 {
		for _, include := range t.includeTools {
			if include == toolName {
				return true
			}
		}
		return false
	}

	// Check blacklist
	// 检查黑名单
	for _, exclude := range t.excludeTools {
		if exclude == toolName {
			return false
		}
	}

	return true
}

// convertSchema converts MCP InputSchema to agno toolkit parameters
// convertSchema 将 MCP InputSchema 转换为 agno 工具包参数
func (t *MCPToolkit) convertSchema(schema protocol.InputSchema) (map[string]toolkit.Parameter, error) {
	params := make(map[string]toolkit.Parameter)

	// MCP schemas are JSON Schema objects
	// MCP 模式是 JSON Schema 对象
	if schema.Type != "object" {
		return params, nil // Empty params for non-object types
	}

	// Convert properties
	// 转换属性
	for propName, propSchema := range schema.Properties {
		// Extract type from property schema
		// 从属性模式中提取类型
		propMap, ok := propSchema.(map[string]interface{})
		if !ok {
			continue
		}

		propType, _ := propMap["type"].(string)
		propDesc, _ := propMap["description"].(string)

		// Check if required
		// 检查是否必需
		required := false
		for _, reqName := range schema.Required {
			if reqName == propName {
				required = true
				break
			}
		}

		params[propName] = toolkit.Parameter{
			Type:        propType,
			Description: propDesc,
			Required:    required,
		}
	}

	return params, nil
}

// callTool calls an MCP tool and returns the result
// callTool 调用 MCP 工具并返回结果
func (t *MCPToolkit) callTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// Call the tool via MCP client
	// 通过 MCP 客户端调用工具
	result, err := t.client.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, err
	}

	// Extract text content from result
	// 从结果中提取文本内容
	text := t.contentHandler.ExtractText(result.Content)
	if text != "" {
		return text, nil
	}

	// If no text, format all content as string
	// 如果没有文本，则将所有内容格式化为字符串
	return t.contentHandler.FormatAsString(result.Content), nil
}

// GetClient returns the underlying MCP client
// GetClient 返回底层的 MCP 客户端
func (t *MCPToolkit) GetClient() *client.Client {
	return t.client
}

// GetTools returns the list of available MCP tools
// GetTools 返回可用的 MCP 工具列表
func (t *MCPToolkit) GetTools() []protocol.Tool {
	return t.tools
}

// Close disconnects the MCP client
// Close 断开 MCP 客户端连接
func (t *MCPToolkit) Close() error {
	return t.client.Disconnect()
}
