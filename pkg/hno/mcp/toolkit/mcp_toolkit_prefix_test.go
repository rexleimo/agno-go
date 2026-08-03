package toolkit

import (
	"context"
	"encoding/json"
	"testing"

	mcpclient "github.com/rexleimo/agno-go/pkg/hno/mcp/client"
	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
)

func TestMCPToolkit_ToolNamePrefix(t *testing.T) {
	// Mock transport that handles initialize and tools/list
	mt := mcpclient.NewMockTransport()
	mt.SetSendFunc(func(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
		switch req.Method {
		case protocol.MethodInitialize:
			res := protocol.InitializeResult{ProtocolVersion: "1.0", ServerInfo: protocol.ServerInfo{Name: "mock", Version: "0.0.1"}}
			b, _ := json.Marshal(res)
			return &protocol.JSONRPCResponse{JSONRPC: protocol.JSONRPCVersion, Result: json.RawMessage(b), ID: req.ID}, nil
		case protocol.MethodToolsList:
			tools := []protocol.Tool{
				{Name: "sum", Description: "add numbers", InputSchema: protocol.InputSchema{Type: "object", Properties: map[string]interface{}{"a": map[string]interface{}{"type": "number"}, "b": map[string]interface{}{"type": "number"}}, Required: []string{"a", "b"}}},
			}
			res := protocol.ToolsListResult{Tools: tools}
			b, _ := json.Marshal(res)
			return &protocol.JSONRPCResponse{JSONRPC: protocol.JSONRPCVersion, Result: json.RawMessage(b), ID: req.ID}, nil
		default:
			// Default empty result
			return &protocol.JSONRPCResponse{JSONRPC: protocol.JSONRPCVersion, Result: json.RawMessage(`{}`), ID: req.ID}, nil
		}
	})

	cli, err := mcpclient.New(mt, mcpclient.Config{})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := cli.Connect(context.Background()); err != nil {
		t.Fatalf("connect: %v", err)
	}

	tk, err := New(context.Background(), Config{Client: cli, ToolNamePrefix: "acme_"})
	if err != nil {
		t.Fatalf("new toolkit: %v", err)
	}

	// Should register function with prefixed name
	found := false
	for name := range tk.Functions() {
		if name == "acme_sum" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected prefixed tool name 'acme_sum'")
	}
}
