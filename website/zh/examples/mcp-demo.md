# MCP 演示示例

## 概述

本示例演示如何连接到 MCP (模型上下文协议) 服务器并通过 HNO MCP 客户端使用其工具。您将学习如何设置安全验证、创建传输、连接到 MCP 服务器以及将 MCP 工具与您的 HNO agents 集成。

## 你将学到

- 如何为 MCP 命令创建和配置安全验证
- 如何设置用于子进程通信的 stdio 传输
- 如何连接到 MCP 服务器并发现可用工具
- 如何创建用于 HNO agents 的 MCP 工具包
- 如何直接调用 MCP 工具

## 前置要求

- Go 1.21 或更高版本
- 已安装的 MCP 服务器 (例如: calculator 服务器)

## 设置

### 1. 安装 MCP 服务器

```bash
# 安装 uvx 包管理器
pip install uvx

# 安装 calculator MCP 服务器
uvx mcp install @modelcontextprotocol/server-calculator

# 验证安装
python -m mcp_server_calculator --help
```

### 2. 运行示例

```bash
# 进入示例目录
cd cmd/examples/mcp_demo

# 直接运行
go run main.go

# 或构建并运行
go build -o mcp_demo
./mcp_demo
```

## 完整代码

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/client"
	"github.com/rexleimo/agno-go/pkg/hno/mcp/security"
	mcptoolkit "github.com/rexleimo/agno-go/pkg/hno/mcp/toolkit"
)

func main() {
	fmt.Println("=== HNO MCP Demo ===")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 步骤 1: 创建安全验证器
	fmt.Println("Step 1: Creating security validator...")
	validator := security.NewCommandValidator()

	command := "python"
	args := []string{"-m", "mcp_server_calculator"}

	if err := validator.Validate(command, args); err != nil {
		log.Fatalf("Command validation failed: %v", err)
	}
	fmt.Printf("✓ Command validated: %s %v\n", command, args)

	// 步骤 2: 创建传输
	fmt.Println("Step 2: Creating transport...")
	transport, err := client.NewStdioTransport(client.StdioConfig{
		Command: command,
		Args:    args,
	})
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}
	fmt.Println("✓ Stdio transport created")

	// 步骤 3: 创建 MCP 客户端
	fmt.Println("Step 3: Creating MCP client...")
	mcpClient, err := client.New(transport, client.Config{
		ClientName:    "agno-go-demo",
		ClientVersion: "0.1.0",
	})
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}
	fmt.Println("✓ MCP client created")

	// 步骤 4: 连接到服务器
	fmt.Println("Step 4: Connecting to MCP server...")
	if err := mcpClient.Connect(ctx); err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer mcpClient.Disconnect()

	fmt.Println("✓ Connected to MCP server")
	if serverInfo := mcpClient.GetServerInfo(); serverInfo != nil {
		fmt.Printf("  Server: %s v%s\n", serverInfo.Name, serverInfo.Version)
	}

	// 步骤 5: 发现工具
	fmt.Println("Step 5: Discovering tools...")
	tools, err := mcpClient.ListTools(ctx)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("✓ Found %d tools:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}

	// 步骤 6: 创建 MCP 工具包
	fmt.Println("Step 6: Creating MCP toolkit...")
	toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
		Client: mcpClient,
		Name:   "calculator-tools",
	})
	if err != nil {
		log.Fatalf("Failed to create toolkit: %v", err)
	}
	defer toolkit.Close()

	fmt.Println("✓ MCP toolkit created")
	fmt.Printf("  Toolkit name: %s\n", toolkit.Name())
	fmt.Printf("  Available functions: %d\n", len(toolkit.Functions()))

	// 步骤 7: 直接调用工具
	fmt.Println("Step 7: Calling a tool...")
	result, err := mcpClient.CallTool(ctx, "add", map[string]interface{}{
		"a": 5,
		"b": 3,
	})
	if err != nil {
		log.Fatalf("Failed to call tool: %v", err)
	}

	fmt.Println("✓ Tool call successful")
	fmt.Printf("  Result: %v\n", result.Content)

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("The MCP toolkit can now be passed to an HNO Agent!")
}
```

## 代码解释

### 1. 安全验证

```go
validator := security.NewCommandValidator()
if err := validator.Validate(command, args); err != nil {
    log.Fatalf("Command validation failed: %v", err)
}
```

- 使用默认白名单创建安全验证器
- 验证命令是否安全可执行
- 阻止危险的 shell 元字符

### 2. Stdio 传输

```go
transport, err := client.NewStdioTransport(client.StdioConfig{
    Command: "python",
    Args:    []string{"-m", "mcp_server_calculator"},
})
```

- 创建通过 stdin/stdout 通信的传输
- 将 MCP 服务器作为子进程生成
- 处理双向 JSON-RPC 2.0 消息

### 3. MCP 客户端

```go
mcpClient, err := client.New(transport, client.Config{
    ClientName:    "agno-go-demo",
    ClientVersion: "0.1.0",
})
```

- 使用您的应用程序标识创建 MCP 客户端
- 管理连接生命周期
- 提供工具发现和调用的方法

### 4. 工具发现

```go
tools, err := mcpClient.ListTools(ctx)
for _, tool := range tools {
    fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
}
```

- 查询 MCP 服务器的可用工具
- 返回工具元数据(名称、描述、参数)
- 用于动态工具发现

### 5. MCP 工具包创建

```go
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client: mcpClient,
    Name:   "calculator-tools",
})
defer toolkit.Close()
```

- 将 MCP 工具转换为 HNO 工具包函数
- 从 MCP schema 自动生成函数签名
- 与 `agent.Config.Toolkits` 兼容

### 6. 直接工具调用

```go
result, err := mcpClient.CallTool(ctx, "add", map[string]interface{}{
    "a": 5,
    "b": 3,
})
fmt.Printf("Result: %v\n", result.Content)
```

- 直接调用 MCP 工具,无需 agent
- 将参数作为 map 传递
- 返回结果内容

## 预期输出

```
=== HNO MCP Demo ===

Step 1: Creating security validator...
✓ Command validated: python [-m mcp_server_calculator]

Step 2: Creating transport...
✓ Stdio transport created

Step 3: Creating MCP client...
✓ MCP client created

Step 4: Connecting to MCP server...
✓ Connected to MCP server
  Server: calculator v0.1.0

Step 5: Discovering tools...
✓ Found 4 tools:
  - add: Add two numbers
  - subtract: Subtract two numbers
  - multiply: Multiply two numbers
  - divide: Divide two numbers

Step 6: Creating MCP toolkit...
✓ MCP toolkit created
  Toolkit name: calculator-tools
  Available functions: 4

Step 7: Calling a tool...
✓ Tool call successful
  Result: 8

=== Demo Complete ===
The MCP toolkit can now be passed to an HNO Agent!
```

## 与 HNO Agents 一起使用

一旦您有了 MCP 工具包,您可以将其与任何 HNO agent 一起使用:

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

// 创建模型
model, _ := openai.New("gpt-4o-mini", openai.Config{
    APIKey: "your-api-key",
})

// 使用 MCP 工具包创建 agent
ag, _ := agent.New(agent.Config{
    Name:     "MCP Calculator Agent",
    Model:    model,
    Toolkits: []toolkit.Toolkit{toolkit},  // MCP toolkit here!
})

// 运行 agent
output, _ := ag.Run(context.Background(), "What is 25 * 4 + 15?")
fmt.Println(output.Content)
```

## 故障排除

**错误: "command not allowed"**
- 确保您的 MCP 服务器命令在安全白名单中
- 使用 `validator.AddAllowedCommand("your-command")` 添加

**错误: "failed to start process"**
- 验证 MCP 服务器已安装: `python -m mcp_server_calculator --help`
- 检查 Python 是否在您的 PATH 中

**错误: "connection timeout"**
- MCP 服务器可能启动时间过长
- 增加上下文超时: `context.WithTimeout(ctx, 60*time.Second)`

**工具调用返回错误**
- 验证工具是否存在: 检查 `mcpClient.ListTools(ctx)`
- 确保参数类型与工具 schema 匹配

## 下一步

- 阅读 [MCP 集成指南](../guide/mcp.md)
- 尝试连接到其他 MCP 服务器 (filesystem, git, sqlite)
- 为您的用例构建自定义 MCP 服务器
- 将 MCP 工具与内置的 HNO 工具结合使用
