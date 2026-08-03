package debug

import (
    "testing"

    "github.com/rexleimo/agno-go/pkg/hno/models"
    "github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestDumpInvokeRequest(t *testing.T) {
    req := &models.InvokeRequest{ Messages: []*types.Message{ types.NewUserMessage("hello") }, Temperature: 0.2, MaxTokens: 10 }
    s := DumpInvokeRequest(req)
    if s == "" { t.Fatalf("empty dump") }
    if want := "hello"; !contains(s, want) { t.Fatalf("dump missing content %q: %s", want, s) }
}

func TestDumpModelResponse(t *testing.T) {
    resp := &types.ModelResponse{ ID: "id1", Model: "m", Content: "ok", Usage: types.Usage{TotalTokens: 3} }
    s := DumpModelResponse(resp)
    if s == "" { t.Fatalf("empty dump") }
    if want := "\"total_tokens\":3"; !contains(s, want) { t.Fatalf("dump missing usage: %s", s) }
}

func contains(s, sub string) bool { return (len(s) >= len(sub)) && (indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
    for i := 0; i+len(sub) <= len(s); i++ { if s[i:i+len(sub)] == sub { return i } }
    return -1
}

