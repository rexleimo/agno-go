# MCP Demo Example

## Overview

This example demonstrates how to connect to an MCP (Model Context Protocol) server and use its tools through the HNO MCP client. You'll learn how to set up security validation, create transports, connect to MCP servers, and integrate MCP tools with your Agno agents.

## You Will Learn

- How to create and configure security validation for MCP commands
- How to set up stdio transport for subprocess communication
- How to connect to an MCP server and discover available tools
- How to create an MCP toolkit for use with Agno agents
- How to call MCP tools directly

## Prerequisites

- Go 1.21 or later
- An MCP server installed (e.g., calculator server)

## Setup

### 1. Install an MCP Server

```bash
# Install uvx package manager
pip install uvx

# Install the calculator MCP server
uvx mcp install @modelcontextprotocol/server-calculator

# Verify installation
python -m mcp_server_calculator --help
```

### 2. Run the Example

```bash
# Navigate to example directory
cd cmd/examples/mcp_demo

# Run directly
go run main.go

# Or build and run
go build -o mcp_demo
./mcp_demo
```

## Complete Code

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

	// Step 1: Create security validator
	fmt.Println("Step 1: Creating security validator...")
	validator := security.NewCommandValidator()

	command := "python"
	args := []string{"-m", "mcp_server_calculator"}

	if err := validator.Validate(command, args); err != nil {
		log.Fatalf("Command validation failed: %v", err)
	}
	fmt.Printf("✓ Command validated: %s %v\n", command, args)

	// Step 2: Create transport
	fmt.Println("Step 2: Creating transport...")
	transport, err := client.NewStdioTransport(client.StdioConfig{
		Command: command,
		Args:    args,
	})
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}
	fmt.Println("✓ Stdio transport created")

	// Step 3: Create MCP client
	fmt.Println("Step 3: Creating MCP client...")
	mcpClient, err := client.New(transport, client.Config{
		ClientName:    "agno-go-demo",
		ClientVersion: "0.1.0",
	})
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}
	fmt.Println("✓ MCP client created")

	// Step 4: Connect to server
	fmt.Println("Step 4: Connecting to MCP server...")
	if err := mcpClient.Connect(ctx); err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer mcpClient.Disconnect()

	fmt.Println("✓ Connected to MCP server")
	if serverInfo := mcpClient.GetServerInfo(); serverInfo != nil {
		fmt.Printf("  Server: %s v%s\n", serverInfo.Name, serverInfo.Version)
	}

	// Step 5: Discover tools
	fmt.Println("Step 5: Discovering tools...")
	tools, err := mcpClient.ListTools(ctx)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("✓ Found %d tools:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}

	// Step 6: Create MCP toolkit
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

	// Step 7: Call a tool directly
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
	fmt.Println("The MCP toolkit can now be passed to an agno Agent!")
}
```

## Code Explanation

### 1. Security Validation

```go
validator := security.NewCommandValidator()
if err := validator.Validate(command, args); err != nil {
    log.Fatalf("Command validation failed: %v", err)
}
```

- Creates a security validator with default whitelist
- Validates that the command is safe to execute
- Blocks dangerous shell metacharacters

### 2. Stdio Transport

```go
transport, err := client.NewStdioTransport(client.StdioConfig{
    Command: "python",
    Args:    []string{"-m", "mcp_server_calculator"},
})
```

- Creates a transport that communicates via stdin/stdout
- Spawns the MCP server as a subprocess
- Handles bidirectional JSON-RPC 2.0 messages

### 3. MCP Client

```go
mcpClient, err := client.New(transport, client.Config{
    ClientName:    "agno-go-demo",
    ClientVersion: "0.1.0",
})
```

- Creates an MCP client with your application identity
- Manages the connection lifecycle
- Provides methods for tool discovery and invocation

### 4. Tool Discovery

```go
tools, err := mcpClient.ListTools(ctx)
for _, tool := range tools {
    fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
}
```

- Queries the MCP server for available tools
- Returns tool metadata (name, description, parameters)
- Used for dynamic tool discovery

### 5. MCP Toolkit Creation

```go
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client: mcpClient,
    Name:   "calculator-tools",
})
defer toolkit.Close()
```

- Converts MCP tools into Agno toolkit functions
- Automatically generates function signatures from MCP schemas
- Compatible with `agent.Config.Toolkits`

### 6. Direct Tool Call

```go
result, err := mcpClient.CallTool(ctx, "add", map[string]interface{}{
    "a": 5,
    "b": 3,
})
fmt.Printf("Result: %v\n", result.Content)
```

- Calls an MCP tool directly without an agent
- Passes parameters as a map
- Returns the result content

## Expected Output

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
The MCP toolkit can now be passed to an agno Agent!
```

## Using with Agno Agents

Once you have the MCP toolkit, you can use it with any Agno agent:

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

// Create model
model, _ := openai.New("gpt-4o-mini", openai.Config{
    APIKey: "your-api-key",
})

// Create agent with MCP toolkit
ag, _ := agent.New(agent.Config{
    Name:     "MCP Calculator Agent",
    Model:    model,
    Toolkits: []toolkit.Toolkit{toolkit},  // MCP toolkit here!
})

// Run agent
output, _ := ag.Run(context.Background(), "What is 25 * 4 + 15?")
fmt.Println(output.Content)
```

## Troubleshooting

**Error: "command not allowed"**
- Make sure your MCP server command is in the security whitelist
- Add it with `validator.AddAllowedCommand("your-command")`

**Error: "failed to start process"**
- Verify the MCP server is installed: `python -m mcp_server_calculator --help`
- Check that Python is in your PATH

**Error: "connection timeout"**
- The MCP server may be taking too long to start
- Increase the context timeout: `context.WithTimeout(ctx, 60*time.Second)`

**Tool calls return errors**
- Verify the tool exists: check `mcpClient.ListTools(ctx)`
- Ensure parameter types match the tool schema

## Next Steps

- Read the [MCP Integration Guide](../guide/mcp.md)
- Try connecting to other MCP servers (filesystem, git, sqlite)
- Build a custom MCP server for your use case
- Combine MCP tools with built-in Agno tools
