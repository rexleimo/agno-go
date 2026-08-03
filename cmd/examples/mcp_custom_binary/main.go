package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/client"
)

// This example demonstrates using custom binary executables with MCP
// 此示例演示如何使用自定义二进制可执行文件与 MCP

func main() {
	// Example 1: Using a whitelisted command (e.g., python)
	// 示例 1: 使用白名单命令（例如 python）
	example1WhitelistedCommand()

	// Example 2: Using a relative path script
	// 示例 2: 使用相对路径脚本
	// example2RelativePathScript()

	// Example 3: Using an absolute path script
	// 示例 3: 使用绝对路径脚本
	// example3AbsolutePathScript()

	// Example 4: Using custom whitelist
	// 示例 4: 使用自定义白名单
	// example4CustomWhitelist()

	// Example 5: Disabling validation (not recommended)
	// 示例 5: 禁用验证（不推荐）
	// example5DisableValidation()
}

func example1WhitelistedCommand() {
	fmt.Println("=== Example 1: Whitelisted Command ===")

	// This will succeed because 'python' is in the default whitelist
	// 这将成功，因为 'python' 在默认白名单中
	config := client.StdioConfig{
		Command:         "python",
		Args:            []string{"-m", "mcp_server"},
		ValidateCommand: true, // Enable validation
	}

	transport, err := client.NewStdioTransport(config)
	if err != nil {
		log.Printf("Failed to create transport: %v\n", err)
		return
	}

	fmt.Printf("✓ Transport created successfully for command: %s\n", config.Command)
	transport.Stop()
}

func example2RelativePathScript() {
	fmt.Println("\n=== Example 2: Relative Path Script ===")

	// Create a temporary script
	// 创建临时脚本
	scriptPath := "./my_mcp_server.sh"
	content := "#!/bin/bash\necho 'MCP Server Running'\n"

	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		log.Printf("Failed to create script: %v\n", err)
		return
	}
	defer os.Remove(scriptPath)

	// This will validate the relative path
	// 这将验证相对路径
	config := client.StdioConfig{
		Command:         scriptPath,
		Args:            []string{},
		ValidateCommand: true,
	}

	transport, err := client.NewStdioTransport(config)
	if err != nil {
		log.Printf("Failed to create transport: %v\n", err)
		return
	}

	fmt.Printf("✓ Transport created successfully for script: %s\n", scriptPath)
	transport.Stop()
}

func example3AbsolutePathScript() {
	fmt.Println("\n=== Example 3: Absolute Path Script ===")

	// Create a temporary directory and script
	// 创建临时目录和脚本
	tmpDir, err := os.MkdirTemp("", "mcp-example-*")
	if err != nil {
		log.Printf("Failed to create temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := tmpDir + "/mcp_server.sh"
	content := "#!/bin/bash\necho 'MCP Server Running'\n"

	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		log.Printf("Failed to create script: %v\n", err)
		return
	}

	// Add the script name to allowed commands
	// 将脚本名称添加到允许的命令
	config := client.StdioConfig{
		Command:         scriptPath,
		Args:            []string{},
		ValidateCommand: true,
		AllowedCommands: []string{"mcp_server.sh"}, // Allow this specific script
	}

	transport, err := client.NewStdioTransport(config)
	if err != nil {
		log.Printf("Failed to create transport: %v\n", err)
		return
	}

	fmt.Printf("✓ Transport created successfully for absolute path: %s\n", scriptPath)
	transport.Stop()
}

func example4CustomWhitelist() {
	fmt.Println("\n=== Example 4: Custom Whitelist ===")

	// Use a custom whitelist of allowed commands
	// 使用自定义的允许命令白名单
	config := client.StdioConfig{
		Command:         "node",
		Args:            []string{"server.js"},
		ValidateCommand: true,
		AllowedCommands: []string{"node", "deno", "bun"}, // Custom whitelist
	}

	transport, err := client.NewStdioTransport(config)
	if err != nil {
		log.Printf("Failed to create transport: %v\n", err)
		return
	}

	fmt.Printf("✓ Transport created successfully with custom whitelist\n")
	transport.Stop()
}

func example5DisableValidation() {
	fmt.Println("\n=== Example 5: Disable Validation (Not Recommended) ===")

	// Disable validation (not recommended for production)
	// 禁用验证（不推荐用于生产）
	config := client.StdioConfig{
		Command:         "bash",
		Args:            []string{"-c", "echo 'WARNING: This bypasses security validation'"},
		ValidateCommand: false, // Explicitly disable validation
	}

	transport, err := client.NewStdioTransport(config)
	if err != nil {
		log.Printf("Failed to create transport: %v\n", err)
		return
	}

	fmt.Printf("⚠ Transport created with validation disabled\n")
	transport.Stop()
}

func exampleRealWorld() {
	fmt.Println("\n=== Real World Example: MCP Server ===")

	// Connect to an actual MCP server
	// 连接到实际的 MCP 服务器
	config := client.StdioConfig{
		Command:         "python",
		Args:            []string{"-m", "mcp_server_example"},
		ValidateCommand: true,
		WorkingDir:      "/path/to/mcp/server",
		Env: []string{
			"PYTHONUNBUFFERED=1",
			"MCP_SERVER_DEBUG=1",
		},
	}

	transport, err := client.NewStdioTransport(config)
	if err != nil {
		log.Printf("Failed to create transport: %v\n", err)
		return
	}

	// Create MCP client
	// 创建 MCP 客户端
	mcpClient, err := client.New(transport, client.Config{
		ClientName:    "mcp-custom-binary-example",
		ClientVersion: "1.0.0",
	})
	if err != nil {
		log.Printf("Failed to create client: %v\n", err)
		return
	}

	// Connect to the server
	// 连接到服务器
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := mcpClient.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v\n", err)
		return
	}

	fmt.Println("✓ Successfully connected to MCP server")

	// List available tools
	// 列出可用的工具
	tools, err := mcpClient.ListTools(ctx)
	if err != nil {
		log.Printf("Failed to list tools: %v\n", err)
		mcpClient.Disconnect()
		return
	}

	fmt.Printf("Available tools: %d\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}

	// Disconnect
	// 断开连接
	mcpClient.Disconnect()
	fmt.Println("✓ Disconnected from MCP server")
}
