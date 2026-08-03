package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/client"
	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// newMockMCPClient builds a client whose transport routes by method:
// initialize -> server info, tools/list -> one calculator tool,
// tools/call -> echo the input.
//
// newMockMCPClient 构建按方法路由的 client：
// initialize -> 服务器信息，tools/list -> 一个 calculator 工具，
// tools/call -> 回显输入。
func newMockMCPClient(t *testing.T) *client.Client {
	t.Helper()
	transport := client.NewMockTransport()
	transport.SetSendFunc(func(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
		method := req.Method
		switch method {
		case "initialize":
			return jsonResponse(req, map[string]interface{}{
				"protocolVersion": "1.0",
				"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
				"serverInfo":      map[string]interface{}{"name": "calc-server", "version": "1.0.0"},
			})
		case "tools/list":
			return jsonResponse(req, map[string]interface{}{
				"tools": []interface{}{
					map[string]interface{}{
						"name":        "add",
						"description": "Adds two numbers",
						"inputSchema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"a": map[string]interface{}{"type": "number", "description": "first"},
								"b": map[string]interface{}{"type": "number", "description": "second"},
							},
							"required": []interface{}{"a", "b"},
						},
					},
				},
			})
		case "tools/call":
			return jsonResponse(req, map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{"type": "text", "text": "3"},
				},
				"isError": false,
			})
		default:
			return jsonResponse(req, map[string]interface{}{})
		}
	})

	c, err := client.New(transport, client.Config{ClientName: "test"})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	return c
}

func jsonResponse(req *protocol.JSONRPCRequest, result interface{}) (*protocol.JSONRPCResponse, error) {
	raw, _ := json.Marshal(result)
	return &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  raw,
		ID:      req.ID,
	}, nil
}

func TestNewToolkit_RegistersTools(t *testing.T) {
	ctx := context.Background()
	c := newMockMCPClient(t)

	tk, err := NewToolkit(ctx, c)
	if err != nil {
		t.Fatalf("NewToolkit: %v", err)
	}

	// Server name sanitized as prefix: calc_server_add
	// 服务器名净化作前缀：calc_server_add
	fn := tk.Functions()["calc_server_add"]
	if fn == nil {
		t.Fatalf("expected calc_server_add, got %v", mapKeys(tk.Functions()))
	}
	if !strings.Contains(fn.Description, "Adds two numbers") {
		t.Errorf("description = %q", fn.Description)
	}
	if fn.Parameters["a"].Type != "number" || !fn.Parameters["a"].Required {
		t.Errorf("param a = %+v, want number+required", fn.Parameters["a"])
	}
	if fn.Parameters["b"].Required != true {
		t.Errorf("param b should be required")
	}
}

func TestNewToolkit_CallTool(t *testing.T) {
	ctx := context.Background()
	c := newMockMCPClient(t)

	tk, err := NewToolkit(ctx, c)
	if err != nil {
		t.Fatalf("NewToolkit: %v", err)
	}

	fn := tk.Functions()["calc_server_add"]
	result, err := fn.Handler(ctx, map[string]interface{}{"a": 1, "b": 2})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if result != "3" {
		t.Errorf("result = %v, want 3", result)
	}
}

func TestNewToolkit_NilClient(t *testing.T) {
	if _, err := NewToolkit(context.Background(), nil); err == nil {
		t.Error("expected error for nil client")
	}
}

func TestSanitizePrefix(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Calc Server", "calc_server"},
		{"My-Server", "my_server"},
		{"123", "123"},
		{"!!!", "mcp"},
	}
	for _, tt := range tests {
		if got := sanitizePrefix(tt.in); got != tt.want {
			t.Errorf("sanitizePrefix(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func mapKeys(m map[string]*toolkit.Function) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
