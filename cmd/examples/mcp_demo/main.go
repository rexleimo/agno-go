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
	fmt.Println("=== Agno-Go MCP Demo ===")
	fmt.Println()

	// This example demonstrates how to connect to an MCP server
	// and use its tools through the HNO MCP client.
	// 此示例演示如何连接到 MCP 服务器
	// 并通过 HNO MCP 客户端使用其工具。

	// NOTE: To run this example, you need an actual MCP server running
	// For example, you could use the Python MCP calculator server:
	// 注意: 要运行此示例，您需要一个实际运行的 MCP 服务器
	// 例如，您可以使用 Python MCP 计算器服务器:
	//
	//   uvx mcp install @modelcontextprotocol/server-calculator
	//   OR run: python -m mcp_server_calculator
	//
	// Since we don't have a real server for this demo, we'll show the setup code

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Create and validate security validator
	// 步骤 1: 创建并验证安全验证器
	fmt.Println("Step 1: Creating security validator...")
	validator := security.NewCommandValidator()

	// Example: Validate a command
	// 示例: 验证命令
	command := "python"
	args := []string{"-m", "mcp_server_calculator"}

	if err := validator.Validate(command, args); err != nil {
		log.Fatalf("Command validation failed: %v", err)
	}
	fmt.Printf("✓ Command validated: %s %v\n", command, args)
	fmt.Println()

	// Step 2: Create transport configuration
	// 步骤 2: 创建传输配置
	fmt.Println("Step 2: Creating transport configuration...")
	transportConfig := client.StdioConfig{
		Command: command,
		Args:    args,
	}
	fmt.Printf("✓ Transport configured for: %s\n", transportConfig.Command)
	fmt.Println()

	// Step 3: Create stdio transport
	// 步骤 3: 创建 stdio 传输
	fmt.Println("Step 3: Creating stdio transport...")
	transport, err := client.NewStdioTransport(transportConfig)
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}
	fmt.Println("✓ Stdio transport created")
	fmt.Println()

	// Step 4: Create MCP client
	// 步骤 4: 创建 MCP 客户端
	fmt.Println("Step 4: Creating MCP client...")
	mcpClient, err := client.New(transport, client.Config{
		ClientName:    "agno-go-demo",
		ClientVersion: "0.1.0",
	})
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}
	fmt.Println("✓ MCP client created")
	fmt.Println()

	// Step 5: Connect to MCP server
	// 步骤 5: 连接到 MCP 服务器
	fmt.Println("Step 5: Connecting to MCP server...")
	fmt.Println("(This will fail without a running MCP server)")

	if err := mcpClient.Connect(ctx); err != nil {
		fmt.Printf("⚠ Connection failed (expected): %v\n", err)
		fmt.Println()
		fmt.Println("To run this example with a real MCP server:")
		fmt.Println("1. Install an MCP server (e.g., @modelcontextprotocol/server-calculator)")
		fmt.Println("2. Update the command/args in this example to match your server")
		fmt.Println("3. Run this example again")
		return
	}
	defer mcpClient.Disconnect()

	fmt.Println("✓ Connected to MCP server")
	if serverInfo := mcpClient.GetServerInfo(); serverInfo != nil {
		fmt.Printf("  Server: %s v%s\n", serverInfo.Name, serverInfo.Version)
	}
	fmt.Println()

	// Step 6: List available tools
	// 步骤 6: 列出可用工具
	fmt.Println("Step 6: Discovering tools...")
	tools, err := mcpClient.ListTools(ctx)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("✓ Found %d tools:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	// Step 7: Create MCP toolkit for use with agents
	// 步骤 7: 创建用于 agent 的 MCP 工具包
	fmt.Println("Step 7: Creating MCP toolkit...")
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
	fmt.Println()

	// Step 8: Call a tool directly
	// 步骤 8: 直接调用工具
	fmt.Println("Step 8: Calling a tool...")
	result, err := mcpClient.CallTool(ctx, "add", map[string]interface{}{
		"a": 5,
		"b": 3,
	})
	if err != nil {
		log.Fatalf("Failed to call tool: %v", err)
	}

	fmt.Println("✓ Tool call successful")
	fmt.Printf("  Result: %v\n", result.Content)
	fmt.Println()

	// The toolkit can now be used with HNO agents!
	// 工具包现在可以与 HNO agents 一起使用！
	fmt.Println("=== Demo Complete ===")
	fmt.Println()
	fmt.Println("The MCP toolkit can now be passed to an HNO Agent:")
	fmt.Println("  agent, _ := agent.New(&agent.Config{")
	fmt.Println("    Model: yourModel,")
	fmt.Println("    Toolkits: []toolkit.Toolkit{toolkit},")
	fmt.Println("  })")
}
