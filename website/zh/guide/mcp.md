# MCP 集成

## 什么是 MCP?

**模型上下文协议 (Model Context Protocol, MCP)** 是一个开放标准,能够在 LLM 应用程序和外部数据源及工具之间实现无缝集成。由 Anthropic 开发,MCP 通过标准化接口为 AI 模型与各种服务的连接提供了通用协议。

**Model Context Protocol (MCP)** is an open standard that enables seamless integration between LLM applications and external data sources and tools. Developed by Anthropic, MCP provides a universal protocol for connecting AI models with various services through a standardized interface.

## 为什么在 Agno-Go 中使用 MCP?

- **🔌 可扩展性** - 将您的 agent 连接到任何兼容 MCP 的服务器
  - **Extensibility** - Connect your agents to any MCP-compatible server
- **🔒 安全性** - 内置命令验证和 shell 注入保护
  - **Security** - Built-in command validation and shell injection protection
- **🚀 性能** - 快速初始化 (<100μs) 和低内存占用 (<10KB)
  - **Performance** - Fast initialization (<100μs) and low memory footprint (<10KB)
- **📦 可重用性** - 利用现有的 MCP 服务器,无需重新造轮子
  - **Reusability** - Leverage existing MCP servers without reinventing the wheel

## 架构

Agno-Go 的 MCP 实现由几个关键组件组成:

Agno-Go's MCP implementation consists of several key components:

```
pkg/hno/mcp/
├── protocol/       # JSON-RPC 2.0 和 MCP 消息类型 | JSON-RPC 2.0 and MCP message types
├── client/         # MCP 客户端核心和传输 | MCP client core and transports
├── security/       # 命令验证和安全 | Command validation and security
├── content/        # 内容类型处理 | Content type handling
└── toolkit/        # 与 agno 工具包系统集成 | Integration with agno toolkit system
```

## 快速开始

### 前置要求 | Prerequisites

- Go 1.21 或更高版本 | Go 1.21 or later
- 一个 MCP 服务器 (例如: calculator, filesystem, git)
  - An MCP server (e.g., calculator, filesystem, git)

### 安装 | Installation

```bash
# 安装 uvx 以管理 MCP 服务器
# Install uvx for managing MCP servers
pip install uvx

# 安装示例 MCP 服务器
# Install a sample MCP server
uvx mcp install @modelcontextprotocol/server-calculator
```

### 基本用法 | Basic Usage

#### 1. 创建安全验证器 | Create Security Validator

```go
import "github.com/rexleimo/agno-go/pkg/hno/mcp/security"

// 使用默认安全命令创建验证器
// Create validator with default safe commands
validator := security.NewCommandValidator()

// 使用前验证命令
// Validate command before use
if err := validator.Validate("python", []string{"-m", "mcp_server"}); err != nil {
    log.Fatal(err)
}
```

#### 2. 设置传输 | Setup Transport

```go
import "github.com/rexleimo/agno-go/pkg/hno/mcp/client"

// 创建 stdio 传输以进行子进程通信
// Create stdio transport for subprocess communication
transport, err := client.NewStdioTransport(client.StdioConfig{
    Command: "python",
    Args:    []string{"-m", "mcp_server_calculator"},
})
if err != nil {
    log.Fatal(err)
}
```

#### 3. 连接到 MCP 服务器 | Connect to MCP Server

```go
// 创建 MCP 客户端
// Create MCP client
mcpClient, err := client.New(transport, client.Config{
    ClientName:    "my-agent",
    ClientVersion: "1.0.0",
})
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()
if err := mcpClient.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer mcpClient.Disconnect()

// 获取服务器信息
// Get server information
serverInfo := mcpClient.GetServerInfo()
fmt.Printf("Connected to: %s v%s\n", serverInfo.Name, serverInfo.Version)
```

#### 4. 为 Agent 创建 MCP 工具包 | Create MCP Toolkit for Agents

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    mcptoolkit "github.com/rexleimo/agno-go/pkg/hno/mcp/toolkit"
)

// 从 MCP 客户端创建工具包
// Create toolkit from MCP client
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client: mcpClient,
    Name:   "calculator-tools",
    // 可选: 过滤特定工具
    // Optional: filter specific tools
    IncludeTools: []string{"add", "subtract", "multiply"},
})
if err != nil {
    log.Fatal(err)
}
defer toolkit.Close()

// 与 agno agent 一起使用
// Use with agno agent
ag, err := agent.New(agent.Config{
    Name:     "Math Assistant",
    Model:    yourModel,
    Toolkits: []toolkit.Toolkit{toolkit},
})
```

## 安全功能 | Security Features

Agno-Go 的 MCP 实现将安全放在首位:

Agno-Go's MCP implementation prioritizes security:

### 命令白名单 | Command Whitelist

默认只允许特定命令:

Only specific commands are allowed by default:

- `python`, `python3`
- `node`, `npm`, `npx`
- `uvx`
- `docker`

### Shell 注入保护 | Shell Injection Protection

所有命令参数都经过验证以防止 shell 注入:

All command arguments are validated to prevent shell injection:

**阻止的字符 | Blocked characters:**
- `;` (命令分隔符 | command separator)
- `|` (管道 | pipe)
- `&` (后台执行 | background execution)
- `` ` `` (命令替换 | command substitution)
- `$` (变量扩展 | variable expansion)
- `>`, `<` (重定向 | redirection)

### 自定义安全策略 | Custom Security Policies

```go
// 使用特定允许的命令创建自定义验证器
// Create custom validator with specific allowed commands
validator := security.NewCustomCommandValidator(
    []string{"go", "rust"},     // 允许的命令 | allowed commands
    []string{";", "|", "&"},    // 阻止的字符 | blocked chars
)

// 动态添加或删除命令
// Add or remove commands dynamically
validator.AddAllowedCommand("ruby")
validator.RemoveAllowedCommand("go")
```

## 工具过滤 | Tool Filtering

您可以选择性地从 MCP 服务器包含或排除工具:

You can selectively include or exclude tools from MCP servers:

```go
// 仅包含特定工具
// Include only specific tools
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client:       mcpClient,
    IncludeTools: []string{"add", "subtract", "multiply"},
})

// 或排除某些工具
// Or exclude certain tools
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client:       mcpClient,
    ExcludeTools: []string{"divide"},  // 排除除法 | exclude division
})
```

## 内容处理 | Content Handling

MCP 支持不同的内容类型:

MCP supports different content types:

```go
import "github.com/rexleimo/agno-go/pkg/hno/mcp/content"

handler := content.New()

// 从内容中提取文本
// Extract text from content
text := handler.ExtractText(contents)

// 提取图像
// Extract images
images, err := handler.ExtractImages(contents)
for _, img := range images {
    fmt.Printf("Image: %s (%d bytes)\n", img.MimeType, len(img.Data))
}

// 创建不同的内容类型
// Create different content types
textContent := handler.CreateTextContent("Hello, world!")
imageContent := handler.CreateImageContent(imageData, "image/png")
resourceContent := handler.CreateResourceContent("file:///path", "text/plain")
```

## 已知的 MCP 服务器 | Known MCP Servers

可与 Agno-Go 一起使用的兼容 MCP 服务器:

Compatible MCP servers you can use with Agno-Go:

| 服务器 Server | 描述 Description | 安装 Installation |
|--------|-------------|--------------|
| **@modelcontextprotocol/server-calculator** | 数学运算 Math operations | `uvx mcp install @modelcontextprotocol/server-calculator` |
| **@modelcontextprotocol/server-filesystem** | 文件操作 File operations | `uvx mcp install @modelcontextprotocol/server-filesystem` |
| **@modelcontextprotocol/server-git** | Git 操作 Git operations | `uvx mcp install @modelcontextprotocol/server-git` |
| **@modelcontextprotocol/server-sqlite** | SQLite 数据库 SQLite database | `uvx mcp install @modelcontextprotocol/server-sqlite` |

更多服务器请访问 [MCP 服务器注册表](https://github.com/modelcontextprotocol/servers)。

More servers available at [MCP Servers Registry](https://github.com/modelcontextprotocol/servers).

## 性能 | Performance

Agno-Go 的 MCP 实现经过高度优化:

Agno-Go's MCP implementation is highly optimized:

- **MCP 客户端初始化 | MCP Client Init**: <100μs
- **工具发现 | Tool Discovery**: 每个服务器 <50μs | <50μs per server
- **内存 | Memory**: 每个连接 <10KB | <10KB per connection
- **测试覆盖率 | Test Coverage**: >80%

## 限制 | Limitations

当前实现状态:

Current implementation status:

- ✅ Stdio transport (已实现 | implemented)
- ⏳ SSE transport (计划中 | planned)
- ⏳ HTTP transport (计划中 | planned)
- ✅ Tools (已实现 | implemented)
- ✅ Resources (已实现 | implemented)
- ✅ Prompts (已实现 | implemented)

## 最佳实践 | Best Practices

1. **始终使用安全验证** - 永不绕过命令验证
   - **Always use security validation** - Never bypass command validation

2. **适当过滤工具** - 仅公开您的 agent 需要的工具
   - **Filter tools appropriately** - Only expose tools your agent needs

3. **优雅地处理错误** - MCP 服务器可能会失败或超时
   - **Handle errors gracefully** - MCP servers may fail or timeout

4. **关闭连接** - 始终 defer `toolkit.Close()` 以清理资源
   - **Close connections** - Always defer `toolkit.Close()` to clean up resources

5. **使用模拟服务器测试** - 使用 `pkg/hno/mcp/client/testing.go` 中的测试工具
   - **Test with mock servers** - Use the testing utilities in `pkg/hno/mcp/client/testing.go`

## 下一步 | Next Steps

- 尝试 [MCP 演示示例](../examples/mcp-demo.md) | Try the [MCP Demo Example](../examples/mcp-demo.md)
- 阅读 [MCP 实现指南](../../pkg/hno/mcp/IMPLEMENTATION.md) | Read the [MCP Implementation Guide](../../pkg/hno/mcp/IMPLEMENTATION.md)
- 探索 [MCP 协议规范](https://spec.modelcontextprotocol.io/) | Explore the [MCP Protocol Specification](https://spec.modelcontextprotocol.io/)
- 在 [GitHub](https://github.com/rexleimo/agno-Go/discussions) 上参与讨论 | Join discussions on [GitHub](https://github.com/rexleimo/agno-Go/discussions)

## 故障排除 | Troubleshooting

**错误: "command not allowed"**
- 检查您的命令是否在白名单中 | Check that your command is in the whitelist
- 使用 `validator.AddAllowedCommand()` 添加自定义命令 | Use `validator.AddAllowedCommand()` to add custom commands

**错误: "shell metacharacters detected"**
- 您的命令参数包含危险字符 | Your command arguments contain dangerous characters
- 确保参数不包含 `;`, `|`, `&` 等 | Ensure arguments don't contain `;`, `|`, `&`, etc.

**错误: "failed to start MCP server"**
- 验证 MCP 服务器已安装 | Verify the MCP server is installed
- 检查命令路径是否正确 | Check that the command path is correct
- 确保您具有必要的权限 | Ensure you have necessary permissions

**MCP 服务器无响应**
- 检查服务器日志中的错误 | Check server logs for errors
- 验证 JSON-RPC 消息格式是否正确 | Verify JSON-RPC messages are correctly formatted
- 尝试使用 `mcpClient.Connect(ctx)` 重新连接 | Try reconnecting with `mcpClient.Connect(ctx)`
