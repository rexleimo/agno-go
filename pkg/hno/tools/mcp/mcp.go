// Package mcp bridges MCP servers into the standard toolkit system: every
// tool exposed by an MCP server becomes a regular toolkit.Function that can
// be registered on an agent like any built-in tool.
//
// mcp 包将 MCP 服务器桥接到标准 toolkit 体系：MCP 服务器暴露的每个
// 工具都变成常规 toolkit.Function，可像内置工具一样注册到 agent。
package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/client"
	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// Toolkit adapts an MCP client's tools into a standard toolkit.
// Toolkit 将 MCP 客户端的工具适配为标准 toolkit。
type Toolkit struct {
	*toolkit.BaseToolkit
	client *client.Client
}

// NewToolkit connects to the MCP server, lists its tools, and registers each
// as a toolkit.Function. The server name prefixes function names to avoid
// collisions when multiple MCP servers are mounted.
//
// NewToolkit 连接 MCP 服务器，列出其工具，并将每个工具注册为
// toolkit.Function。服务器名作为函数名前缀，避免多个 MCP 服务器
// 挂载时冲突。
func NewToolkit(ctx context.Context, c *client.Client) (*Toolkit, error) {
	if c == nil {
		return nil, fmt.Errorf("mcp: client is required")
	}
	if !c.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return nil, fmt.Errorf("mcp: connect: %w", err)
		}
	}

	tools, err := c.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("mcp: list tools: %w", err)
	}

	name := "mcp"
	if info := c.GetServerInfo(); info != nil && info.Name != "" {
		name = sanitizePrefix(info.Name)
	}

	bt := toolkit.NewBaseToolkit(name)
	tk := &Toolkit{BaseToolkit: bt, client: c}

	for _, tool := range tools {
		registered := buildFunction(name, tool, c)
		bt.RegisterFunction(registered)
	}

	return tk, nil
}

// buildFunction converts one MCP tool into a toolkit.Function.
// buildFunction 将单个 MCP 工具转换为 toolkit.Function。
func buildFunction(prefix string, tool protocol.Tool, c *client.Client) *toolkit.Function {
	fnName := prefix + "_" + tool.Name

	params := make(map[string]toolkit.Parameter)
	for name, raw := range tool.InputSchema.Properties {
		prop, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		ptype, _ := prop["type"].(string)
		desc, _ := prop["description"].(string)
		params[name] = toolkit.Parameter{
			Type:        ptype,
			Description: desc,
			Required:    contains(tool.InputSchema.Required, name),
		}
	}

	return &toolkit.Function{
		Name:        fnName,
		Description: tool.Description,
		Parameters:  params,
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			result, err := c.CallTool(ctx, tool.Name, args)
			if err != nil {
				return nil, err
			}
			if result == nil {
				return nil, fmt.Errorf("mcp: empty result from %s", tool.Name)
			}
			if result.IsError {
				return nil, fmt.Errorf("mcp: tool %s returned error: %s", tool.Name, contentText(result.Content))
			}
			return contentText(result.Content), nil
		},
	}
}

// contentText concatenates text content blocks into a single string.
// contentText 将文本内容块拼接为单个字符串。
func contentText(content []protocol.Content) string {
	var sb strings.Builder
	for _, c := range content {
		if c.Type == "text" && c.Text != "" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String()
}

// sanitizePrefix makes a server name a safe function prefix.
// sanitizePrefix 将服务器名转为安全的函数前缀。
func sanitizePrefix(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			sb.WriteRune(r + 32) // lowercase
		} else {
			sb.WriteRune('_')
		}
	}
	out := sb.String()
	out = strings.Trim(out, "_")
	if out == "" {
		return "mcp"
	}
	return out
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
