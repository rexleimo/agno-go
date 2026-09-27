package agent

import (
	"context"
	"github.com/rexleimo/agno-go/pkg/hno/types"
	"reflect"
	"sort"
	"testing"
)

func p0sRunStream(t *testing.T, ag *Agent) *RunOutput {
	t.Helper()
	result, err := ag.RunStream(context.Background(), "test input")
	if err != nil {
		t.Fatalf("RunStream failed: %v", err)
	}
	for range result.Events {
	}
	done := <-result.Done
	if done.Err != nil {
		t.Fatalf("RunStream done error: %v", done.Err)
	}
	if done.Output == nil {
		t.Fatal("RunStream returned nil output")
	}
	return done.Output
}

func p0sToolMessages(messages []*types.Message) []*types.Message {
	var tools []*types.Message
	for _, message := range messages {
		if message.Role == types.RoleTool {
			tools = append(tools, message)
		}
	}
	return tools
}

func p0sToolMessagePairs(messages []*types.Message) []string {
	pairs := make([]string, 0)
	for _, message := range p0sToolMessages(messages) {
		pairs = append(pairs, message.ToolCallID+"\x00"+message.Content)
	}
	sort.Strings(pairs)
	return pairs
}

func TestP0S_ToolCallLimitTruncatesInStream(t *testing.T) {
	model := &p0sStreamModel{chunks: p0sThreeCallChunks()}
	ag, err := New(Config{
		Name:          "p0s-limit-stream",
		Model:         model,
		Toolkits:      p0sToolkits(false),
		ToolCallLimit: 2,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	output := p0sRunStream(t, ag)
	tools := p0sToolMessages(output.Messages)
	if len(tools) != 3 {
		t.Errorf("期望恰好 3 条 RoleTool 消息，实际为 %d", len(tools))
	}
	got := make(map[string]string)
	for _, message := range tools {
		got[message.ToolCallID] = message.Content
	}
	want := map[string]string{
		"c1": `"A"`,
		"c2": `"B"`,
		"c3": "tool call limit reached; call not executed",
	}
	for id, content := range want {
		if got[id] != content {
			t.Errorf("工具调用 %s：期望内容 %q，实际为 %q", id, content, got[id])
		}
	}
}

func TestP0S_ToolCallLimitExhaustedAcrossRounds(t *testing.T) {
	model := &p0sStreamModel{chunks: [][]types.ResponseChunk{
		{{ToolCalls: []types.ToolCall{p0sToolCall("c1", "p0s_a")}}},
		{{ToolCalls: []types.ToolCall{p0sToolCall("c2", "p0s_b")}}},
		{{ToolCalls: []types.ToolCall{p0sToolCall("c3", "p0s_a")}}},
	}}
	ag, err := New(Config{
		Name:          "p0s-limit-rounds",
		Model:         model,
		Toolkits:      p0sToolkits(false),
		ToolCallLimit: 2,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	output := p0sRunStream(t, ag)
	tools := p0sToolMessages(output.Messages)
	if len(tools) != 3 {
		t.Errorf("期望恰好 3 条 RoleTool 消息（2 条已执行 + 1 条配对跳过），实际为 %d", len(tools))
	}
	got := make(map[string]string)
	for _, message := range tools {
		got[message.ToolCallID] = message.Content
	}
	for id, content := range map[string]string{
		"c1": `"A"`,
		"c2": `"B"`,
		"c3": "tool call limit reached; call not executed",
	} {
		if got[id] != content {
			t.Errorf("工具调用 %s：期望内容 %q，实际为 %q", id, content, got[id])
		}
	}
	if model.invocations != 3 {
		t.Errorf("期望 InvokeStream 调用 3 次，实际为 %d 次", model.invocations)
	}
}

func TestP0S_StopLoopHaltsStream(t *testing.T) {
	model := &p0sStreamModel{chunks: [][]types.ResponseChunk{
		{{ToolCalls: []types.ToolCall{p0sToolCall("c1", "p0s_a")}}},
		{{Content: "第二轮内容不应被读取"}},
	}}
	ag, err := New(Config{
		Name:     "p0s-stop-loop",
		Model:    model,
		Toolkits: p0sToolkits(true),
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	output := p0sRunStream(t, ag)
	if model.invocations != 1 {
		t.Errorf("期望 InvokeStream 只调用 1 次，实际为 %d 次", model.invocations)
	}
	found := false
	for _, message := range p0sToolMessages(output.Messages) {
		if message.ToolCallID == "c1" && message.Content == `"A"` {
			found = true
		}
	}
	if !found {
		t.Error("Done.Output.Messages 中应保留 c1 的工具结果消息")
	}
}

func TestP0S_SyncAndStreamProduceSameToolMessages(t *testing.T) {
	syncModel := &p0sStreamModel{chunks: p0sThreeCallChunks()}
	syncAgent, err := New(Config{
		Name:          "p0s-sync-parity",
		Model:         syncModel,
		Toolkits:      p0sToolkits(false),
		ToolCallLimit: 2,
	})
	if err != nil {
		t.Fatalf("failed to create sync agent: %v", err)
	}
	syncOutput, err := syncAgent.Run(context.Background(), "test input")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	streamModel := &p0sStreamModel{chunks: p0sThreeCallChunks()}
	streamAgent, err := New(Config{
		Name:          "p0s-stream-parity",
		Model:         streamModel,
		Toolkits:      p0sToolkits(false),
		ToolCallLimit: 2,
	})
	if err != nil {
		t.Fatalf("failed to create stream agent: %v", err)
	}
	streamOutput := p0sRunStream(t, streamAgent)

	want := p0sToolMessagePairs(syncOutput.Messages)
	got := p0sToolMessagePairs(streamOutput.Messages)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("同步与流式工具消息不一致：期望 %q，实际为 %q", want, got)
	}
}

func TestP0S_StopReasonObservable(t *testing.T) {
	model := &p0sStreamModel{chunks: [][]types.ResponseChunk{{{Content: "final"}}}}
	ag, err := New(Config{
		Name:  "p0s-stop-reason",
		Model: model,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result, err := ag.RunStream(context.Background(), "test input")
	if err != nil {
		t.Fatalf("RunStream failed: %v", err)
	}
	for range result.Events {
	}
	done := <-result.Done
	if done.Err != nil {
		t.Fatalf("RunStream done error: %v", done.Err)
	}
	if done.StopReason != "no_tool_calls" {
		t.Errorf("StopReason = %q, want %q", done.StopReason, "no_tool_calls")
	}
}
