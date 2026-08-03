// Package observability provides OpenTelemetry-based tracing for agents,
// with a no-op default so it is zero-cost when not configured.
//
// Span hierarchy follows the GenAI semantic conventions:
//
//	invoke_agent  (agent run)
//	  ├── execute_tool  (each tool call)
//	  └── chat          (each model invocation, with gen_ai.* attributes)
//
// observability 包为 agent 提供基于 OpenTelemetry 的追踪，默认 no-op
// 零开销（未配置时不产生任何成本）。
//
// Span 层级遵循 GenAI 语义约定：
//
//	invoke_agent  (agent 运行)
//	  ├── execute_tool  (每次工具调用)
//	  └── chat          (每次模型调用，带 gen_ai.* 属性)
package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracerName is the OpenTelemetry tracer name for the framework.
// TracerName 是框架的 OpenTelemetry tracer 名称。
const TracerName = "hno"

// Span names (GenAI semantic conventions).
// Span 名称（GenAI 语义约定）。
const (
	SpanAgentRun    = "invoke_agent"
	SpanToolExecute = "execute_tool"
	SpanChat        = "chat"
)

// GenAI attribute keys (semantic conventions).
// GenAI 属性键（语义约定）。
const (
	AttrGenAIProvider         = "gen_ai.provider.name"
	AttrGenAIModel            = "gen_ai.request.model"
	AttrGenAIRequestTokens    = "gen_ai.usage.input_tokens"
	AttrGenAICompletionTokens = "gen_ai.usage.output_tokens"
	AttrGenAITotalTokens      = "gen_ai.usage.total_tokens"
	AttrGenAIToolName         = "gen_ai.tool.name"
	AttrGenAISystem           = "gen_ai.system"
	AttrAgentName             = "agent.name"
	AttrAgentRunID            = "agent.run_id"
)

// Tracer returns the framework tracer. When no SDK is configured, the
// OpenTelemetry default no-op tracer is returned (zero overhead).
//
// Tracer 返回框架 tracer。未配置 SDK 时返回 OpenTelemetry 默认的
// no-op tracer（零开销）。
func Tracer() trace.Tracer {
	return otel.Tracer(TracerName)
}

// StartAgentSpan starts an invoke_agent span.
// StartAgentSpan 启动 invoke_agent span。
func StartAgentSpan(ctx context.Context, name, runID string) (context.Context, trace.Span) {
	ctx, span := Tracer().Start(ctx, SpanAgentRun,
		trace.WithAttributes(
			attribute.String(AttrAgentName, name),
			attribute.String(AttrAgentRunID, runID),
			attribute.String(AttrGenAISystem, "hno"),
		),
	)
	return ctx, span
}

// StartToolSpan starts an execute_tool span as a child of ctx.
// StartToolSpan 启动 execute_tool span，作为 ctx 的子 span。
func StartToolSpan(ctx context.Context, toolName string) (context.Context, trace.Span) {
	ctx, span := Tracer().Start(ctx, SpanToolExecute,
		trace.WithAttributes(attribute.String(AttrGenAIToolName, toolName)),
	)
	return ctx, span
}

// StartChatSpan starts a chat span as a child of ctx.
// StartChatSpan 启动 chat span，作为 ctx 的子 span。
func StartChatSpan(ctx context.Context, provider, model string) (context.Context, trace.Span) {
	ctx, span := Tracer().Start(ctx, SpanChat,
		trace.WithAttributes(
			attribute.String(AttrGenAIProvider, provider),
			attribute.String(AttrGenAIModel, model),
		),
	)
	return ctx, span
}

// SetUsage records token usage on a span.
// SetUsage 在 span 上记录 token 用量。
func SetUsage(span trace.Span, input, output, total int) {
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(
		attribute.Int(AttrGenAIRequestTokens, input),
		attribute.Int(AttrGenAICompletionTokens, output),
		attribute.Int(AttrGenAITotalTokens, total),
	)
}

// IsRecording reports whether a real (non-noop) tracer is configured.
// IsRecording 报告是否配置了真实（非 noop）tracer。
func IsRecording(ctx context.Context) bool {
	return trace.SpanFromContext(ctx).IsRecording()
}
