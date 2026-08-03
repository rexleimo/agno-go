package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// setupTracer installs a test SDK exporter and returns it.
// setupTracer 安装测试 SDK exporter 并返回它。
func setupTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	return exporter
}

func TestSpanHierarchyAndAttributes(t *testing.T) {
	exporter := setupTracer(t)

	ctx := context.Background()
	ctx, agentSpan := StartAgentSpan(ctx, "test-agent", "run-123")
	ctx, toolSpan := StartToolSpan(ctx, "calculator")
	_, chatSpan := StartChatSpan(ctx, "openai", "gpt-4o-mini")

	SetUsage(chatSpan, 10, 20, 30)

	agentSpan.End()
	toolSpan.End()
	chatSpan.End()

	spans := exporter.GetSpans()
	if len(spans) != 3 {
		t.Fatalf("expected 3 spans, got %d", len(spans))
	}

	// Verify agent span attributes.
	// 验证 agent span 属性。
	agent := findSpan(spans, SpanAgentRun)
	if agent == nil {
		t.Fatal("agent span not found")
	}
	assertAttr(t, agent, AttrAgentName, "test-agent")
	assertAttr(t, agent, AttrAgentRunID, "run-123")
	assertAttr(t, agent, AttrGenAISystem, "hno")

	// Verify tool span.
	// 验证工具 span。
	tool := findSpan(spans, SpanToolExecute)
	if tool == nil {
		t.Fatal("tool span not found")
	}
	assertAttr(t, tool, AttrGenAIToolName, "calculator")

	// Verify chat span with usage.
	// 验证 chat span 及用量。
	chat := findSpan(spans, SpanChat)
	if chat == nil {
		t.Fatal("chat span not found")
	}
	assertAttr(t, chat, AttrGenAIProvider, "openai")
	assertAttr(t, chat, AttrGenAIModel, "gpt-4o-mini")
	assertAttr(t, chat, AttrGenAIRequestTokens, 10)
	assertAttr(t, chat, AttrGenAICompletionTokens, 20)
	assertAttr(t, chat, AttrGenAITotalTokens, 30)

	// Hierarchy: tool and chat share the agent span's trace.
	// 层级：tool 和 chat 与 agent span 共享同一 trace。
	if tool.SpanContext.TraceID() != agent.SpanContext.TraceID() {
		t.Error("tool span not in agent trace")
	}
	if chat.SpanContext.TraceID() != agent.SpanContext.TraceID() {
		t.Error("chat span not in agent trace")
	}
	// Children's parent span IDs equal the agent span ID.
	// 子 span 的父 span ID 等于 agent span ID。
	if tool.Parent.SpanID() != agent.SpanContext.SpanID() {
		t.Error("tool span parent mismatch")
	}
	// chat was started from the tool-derived ctx, so its parent is tool.
	// chat 从 tool 派生出的 ctx 启动，因此其父 span 是 tool。
	if chat.Parent.SpanID() != tool.SpanContext.SpanID() {
		t.Error("chat span parent mismatch")
	}
}

func TestNoopTracerByDefault(t *testing.T) {
	// Without an installed provider, spans must be no-op (not recording).
	// 未安装 provider 时，span 必须是 no-op（不记录）。
	ctx := context.Background()
	ctx, span := StartChatSpan(ctx, "openai", "gpt-4o")
	if span.IsRecording() {
		t.Error("expected no-op span by default")
	}
	span.End()
}

func findSpan(spans []tracetest.SpanStub, name string) *tracetest.SpanStub {
	for i := range spans {
		if spans[i].Name == name {
			return &spans[i]
		}
	}
	return nil
}

func assertAttr(t *testing.T, span *tracetest.SpanStub, key string, want interface{}) {
	t.Helper()
	for _, kv := range span.Attributes {
		if kv.Key == attribute.Key(key) {
			switch want.(type) {
			case string:
				if kv.Value.AsString() != want {
					t.Errorf("%s = %q, want %v", key, kv.Value.AsString(), want)
				}
			case int:
				if int(kv.Value.AsInt64()) != want {
					t.Errorf("%s = %d, want %v", key, kv.Value.AsInt64(), want)
				}
			}
			return
		}
	}
	t.Errorf("attribute %s not found on span %s", key, span.Name)
}
