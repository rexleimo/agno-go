package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestAggregateResponseStream_ConcatenatesContent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch := make(chan types.ResponseChunk, 3)
	ch <- types.ResponseChunk{Content: "Hello"}
	ch <- types.ResponseChunk{Content: ", "}
	ch <- types.ResponseChunk{Content: "world!"}
	close(ch)

	resp, err := AggregateResponseStream(ctx, ch)
	if err != nil {
		t.Fatalf("AggregateResponseStream returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Content != "Hello, world!" {
		t.Fatalf("unexpected content: %q", resp.Content)
	}
	if len(resp.ToolCalls) != 0 {
		t.Fatalf("expected no tool calls, got %d", len(resp.ToolCalls))
	}
}

func TestAggregateResponseStream_AggregatesToolCalls(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch := make(chan types.ResponseChunk, 2)
	ch <- types.ResponseChunk{
		Content: "call ",
		ToolCalls: []types.ToolCall{
			{ID: "tc1", Function: types.ToolCallFunction{Name: "tool_one"}},
		},
	}
	ch <- types.ResponseChunk{
		Content: "done",
		ToolCalls: []types.ToolCall{
			{ID: "tc2", Function: types.ToolCallFunction{Name: "tool_two"}},
		},
	}
	close(ch)

	resp, err := AggregateResponseStream(ctx, ch)
	if err != nil {
		t.Fatalf("AggregateResponseStream returned error: %v", err)
	}
	if resp.Content != "call done" {
		t.Fatalf("unexpected content: %q", resp.Content)
	}
	if len(resp.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "tc1" || resp.ToolCalls[1].ID != "tc2" {
		t.Fatalf("unexpected tool call IDs: %#v", resp.ToolCalls)
	}
}

func TestAggregateResponseStream_StopsOnErrorChunk(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch := make(chan types.ResponseChunk, 2)
	ch <- types.ResponseChunk{Content: "partial"}
	ch <- types.ResponseChunk{Error: errors.New("stream error")}
	close(ch)

	resp, err := AggregateResponseStream(ctx, ch)
	if err == nil {
		t.Fatal("expected error from AggregateResponseStream, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil response when error occurs, got %#v", resp)
	}
}
