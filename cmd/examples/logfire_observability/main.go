//go:build logfire
// +build logfire

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	openaiModel "github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func main() {
	ctx := context.Background()

	tp, shutdown, err := setupTracer(ctx)
	if err != nil {
		log.Fatalf("failed to initialise tracer: %v", err)
	}
	if shutdown != nil {
		defer func() {
			_ = shutdown(context.Background())
		}()
	}

	tracer := otel.Tracer("github.com/rexleimo/agno-go/examples/logfire")

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if strings.TrimSpace(openaiKey) == "" {
		log.Fatal("OPENAI_API_KEY is required to run this example")
	}

	model, err := openaiModel.New("o1-preview", openaiModel.Config{
		APIKey: openaiKey,
	})
	if err != nil {
		log.Fatalf("failed to initialise OpenAI model: %v", err)
	}

	ag, err := agent.New(agent.Config{
		Name:         "LogfireInstrumentedAgent",
		Model:        model,
		Instructions: "You are an observability-friendly assistant. Explain your reasoning clearly.",
		MaxLoops:     4,
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	input := "Plan a weekend trip to Lisbon with food, culture, and outdoor activities."

	runCtx, span := tracer.Start(ctx, "agent.run",
		trace.WithAttributes(
			attribute.String("agent.name", ag.Name),
			attribute.String("agent.model", model.GetID()),
			attribute.String("agent.provider", model.GetProvider()),
			attribute.String("agent.input", input),
		),
	)
	start := time.Now()

	output, err := ag.Run(runCtx, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		log.Fatalf("agent run failed: %v", err)
	}

	duration := time.Since(start)
	span.SetAttributes(
		attribute.Float64("agent.duration_ms", float64(duration.Milliseconds())),
		attribute.String("agent.output", truncate(output.Content, 512)),
	)

	if loops, ok := readIntMetadata(output.Metadata, "loops"); ok {
		span.SetAttributes(attribute.Int("agent.loops", loops))
	}

	if usage, ok := readUsageMetadata(output.Metadata, "usage"); ok {
		span.SetAttributes(
			attribute.Int("agent.usage.prompt_tokens", usage.PromptTokens),
			attribute.Int("agent.usage.completion_tokens", usage.CompletionTokens),
			attribute.Int("agent.usage.total_tokens", usage.TotalTokens),
		)
	}

	if reasoning := extractReasoningSummary(output); reasoning != nil {
		attrs := []attribute.KeyValue{
			attribute.String("reasoning.content", truncate(reasoning.Content, 512)),
		}
		if reasoning.TokenCount != nil {
			attrs = append(attrs, attribute.Int("reasoning.token_count", *reasoning.TokenCount))
		}
		if reasoning.RedactedContent != nil {
			attrs = append(attrs, attribute.String("reasoning.redacted", truncate(*reasoning.RedactedContent, 256)))
		}
		span.AddEvent("reasoning.complete", trace.WithAttributes(attrs...))
	}

	span.End()

	log.Println("✅ Agent run completed. Output:")
	fmt.Println(output.Content)
}

func setupTracer(ctx context.Context) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	writeToken := strings.TrimSpace(os.Getenv("LOGFIRE_WRITE_TOKEN"))
	if writeToken == "" {
		log.Println("⚠️  LOGFIRE_WRITE_TOKEN not set. Spans will remain local.")
		return nil, nil, nil
	}

	endpoint := strings.TrimSpace(os.Getenv("LOGFIRE_OTLP_ENDPOINT"))
	if endpoint == "" {
		endpoint = "logfire-eu.pydantic.dev"
	}

	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithURLPath("/v1/traces"),
		otlptracehttp.WithTLSClientConfig(&tls.Config{}),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization": writeToken,
		}),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("agno-go-logfire-demo"),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("deployment.environment", getEnv("LOGFIRE_ENV", "development")),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create otel resource: %w", err)
	}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(traceProvider)

	return traceProvider, traceProvider.Shutdown, nil
}

func extractReasoningSummary(output *agent.RunOutput) *types.ReasoningContent {
	if output == nil {
		return nil
	}

	for _, msg := range output.Messages {
		if msg != nil && msg.ReasoningContent != nil {
			return msg.ReasoningContent
		}
	}
	return nil
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "…"
}

func readIntMetadata(metadata map[string]interface{}, key string) (int, bool) {
	if metadata == nil {
		return 0, false
	}
	value, ok := metadata[key]
	if !ok {
		return 0, false
	}

	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case string:
		if v == "" {
			return 0, false
		}
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func readUsageMetadata(metadata map[string]interface{}, key string) (types.Usage, bool) {
	if metadata == nil {
		return types.Usage{}, false
	}
	raw, ok := metadata[key]
	if !ok {
		return types.Usage{}, false
	}

	switch v := raw.(type) {
	case types.Usage:
		return v, true
	case *types.Usage:
		if v != nil {
			return *v, true
		}
	}
	return types.Usage{}, false
}

func getEnv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}
