package runner

// M2 feasibility rehearsal for the P7/G9 contract (slice: 观测接线).
// Every TestP7G9_ row maps to a D row of docs/design/v3-test-scope-p7-g9-observability.json.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/observability"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// scriptedModel fails the first n invocations, then succeeds.
type scriptedModel struct {
	failures int
	calls    int
	errBody  string
	usage    types.Usage
}

func (m *scriptedModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	m.calls++
	if m.calls <= m.failures {
		return nil, errors.New(m.errBody)
	}
	return &types.ModelResponse{Content: "ok", Usage: m.usage}, nil
}
func (m *scriptedModel) GetProvider() string { return "stub" }
func (m *scriptedModel) GetID() string       { return "stub-1" }
func (m *scriptedModel) GetName() string     { return "stub-1" }
func (m *scriptedModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	return nil, errors.New("not used")
}

func newG9Runner(t *testing.T, m models.Model, rcfg *observability.RetryConfig, brk *observability.CircuitBreaker, invoker TurnInvoker) *Runner {
	t.Helper()
	cfg := Config{
		Model:        m,
		ToolExecutor: ToolExecutorFunc(func(ctx context.Context, calls []types.ToolCall) ([]ToolCallOutcome, error) { return nil, nil }),
		Retry:        rcfg,
		Breaker:      brk,
	}
	if invoker != nil {
		cfg.Invoker = invoker
	}
	r, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func noWait(a int) *observability.RetryConfig {
	cfg := observability.DefaultRetryConfig()
	cfg.MaxAttempts = a
	cfg.InitialWait = time.Millisecond
	cfg.MaxWait = 2 * time.Millisecond
	cfg.Jitter = false
	return &cfg
}

// D1 retry recovers: 2 failures then success, MaxAttempts=3 -> 3 invocations, run OK.
func TestP7G9_RetryRecovers(t *testing.T) {
	m := &scriptedModel{failures: 2, errBody: "transient"}
	r := newG9Runner(t, m, noWait(3), nil, nil)
	_, _, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if err != nil {
		t.Fatalf("run err: %v", err)
	}
	if reason != StopNoToolCalls {
		t.Errorf("reason = %s, want no_tool_calls", reason)
	}
	if m.calls != 3 {
		t.Errorf("calls = %d, want 3", m.calls)
	}
}

// D2 zero-value reverse control: nil policies -> exactly one call, error shape identical to today.
func TestP7G9_ZeroValueNoPolicy(t *testing.T) {
	m := &scriptedModel{failures: 99, errBody: "boom"}
	r := newG9Runner(t, m, nil, nil, nil)
	_, _, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if err == nil || !strings.Contains(err.Error(), "runner: model invoke: boom") {
		t.Errorf("err = %v, want wrapped 'runner: model invoke: boom'", err)
	}
	if reason != StopModelFailure {
		t.Errorf("reason = %s, want model_failure", reason)
	}
	if m.calls != 1 {
		t.Errorf("calls = %d, want 1", m.calls)
	}
}

// D3 retry exhaustion surfaces the LAST error, once.
func TestP7G9_RetryExhaustedSurfacesLastError(t *testing.T) {
	m := &scriptedModel{failures: 99, errBody: "always"}
	r := newG9Runner(t, m, noWait(2), nil, nil)
	_, _, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if err == nil || !strings.Contains(err.Error(), "always") {
		t.Errorf("err = %v, want last attempt's error", err)
	}
	if strings.Contains(err.Error(), "first") {
		t.Errorf("err = %v, must not be attempt-1's error", err)
	}
	if reason != StopModelFailure {
		t.Errorf("reason = %s, want model_failure", reason)
	}
	if m.calls != 2 {
		t.Errorf("calls = %d, want 2", m.calls)
	}
	if n := strings.Count(err.Error(), "runner: model invoke"); n != 1 {
		t.Errorf("error wrapped %d times, want exactly once: %v", n, err)
	}
}

// D4 fail-closed misconfig: MaxAttempts=0 must clamp to one real attempt, not a silent success.
func TestP7G9_ZeroAttemptsClampedToOne(t *testing.T) {
	m := &scriptedModel{failures: 99, errBody: "boom"}
	rcfg := &observability.RetryConfig{MaxAttempts: 0}
	r := newG9Runner(t, m, rcfg, nil, nil)
	_, _, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if err == nil {
		t.Fatal("err = nil, want the model error (raw Retry would silently succeed)")
	}
	if reason != StopModelFailure {
		t.Errorf("reason = %s, want model_failure", reason)
	}
	if m.calls != 1 {
		t.Errorf("calls = %d, want 1 (clamped, not zero-iteration)", m.calls)
	}
}

// D5 breaker opens across runs after threshold failures; open turn fails fast.
func TestP7G9_BreakerOpens(t *testing.T) {
	m := &scriptedModel{failures: 99, errBody: "boom"}
	brk := observability.NewCircuitBreaker(2, 50*time.Millisecond)
	rcfg := noWait(1)
	for i := 0; i < 2; i++ {
		r := newG9Runner(t, m, rcfg, brk, nil)
		_, _, _, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
		if err == nil {
			t.Fatalf("run %d: want error", i)
		}
	}
	callsAfterOpen := m.calls
	r3 := newG9Runner(t, m, rcfg, brk, nil)
	_, _, reason, err := r3.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if !errors.Is(err, observability.ErrOpen) {
		t.Errorf("err = %v, want errors.Is ErrOpen", err)
	}
	if reason != StopModelFailure {
		t.Errorf("reason = %s, want model_failure", reason)
	}
	if m.calls != callsAfterOpen {
		t.Errorf("calls = %d, want %d (fail fast, invoker untouched)", m.calls, callsAfterOpen)
	}
}

// D6 breaker recovery: after cooldown the probe is allowed and success resets.
func TestP7G9_BreakerRecovers(t *testing.T) {
	m := &scriptedModel{failures: 2, errBody: "boom"}
	brk := observability.NewCircuitBreaker(2, 5*time.Millisecond)
	rcfg := noWait(1)
	for i := 0; i < 2; i++ {
		r := newG9Runner(t, m, rcfg, brk, nil)
		_, _, _, _ = r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	}
	time.Sleep(10 * time.Millisecond) // cooldown elapsed
	m.failures = 0
	r := newG9Runner(t, m, rcfg, brk, nil)
	_, _, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if err != nil || reason != StopNoToolCalls {
		t.Fatalf("recovery run: err=%v reason=%s, want success", err, reason)
	}
	r2 := newG9Runner(t, m, rcfg, brk, nil) // closed again: next run reaches the model
	callsBefore := m.calls
	if _, _, _, err := r2.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")}); err != nil {
		t.Fatalf("post-recovery run: %v", err)
	}
	if m.calls != callsBefore+1 {
		t.Errorf("calls = %d, want %d (breaker closed after success)", m.calls, callsBefore+1)
	}
}

// D7 breaker-open is not retried even when Retry is configured.
func TestP7G9_OpenBreakerNotRetried(t *testing.T) {
	m := &scriptedModel{failures: 99, errBody: "boom"}
	brk := observability.NewCircuitBreaker(1, time.Second)
	rcfg := noWait(3)
	r1 := newG9Runner(t, m, rcfg, brk, nil)
	_, _, _, _ = r1.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")}) // fails, opens
	frozen := m.calls
	r2 := newG9Runner(t, m, rcfg, brk, nil)
	_, _, _, err := r2.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if !errors.Is(err, observability.ErrOpen) {
		t.Fatalf("err = %v, want ErrOpen", err)
	}
	if m.calls != frozen {
		t.Errorf("calls = %d, want %d (ErrOpen must not enter the retry loop)", m.calls, frozen)
	}
}

// D8 breaker records ONE logical outcome per turn, not per retry attempt.
func TestP7G9_BreakerCountsPostRetryOutcome(t *testing.T) {
	m := &scriptedModel{failures: 99, errBody: "boom"}
	brk := observability.NewCircuitBreaker(2, time.Second)
	rcfg := noWait(3)
	r := newG9Runner(t, m, rcfg, brk, nil)
	_, _, _, _ = r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if m.calls != 3 {
		t.Fatalf("calls = %d, want 3 attempts", m.calls)
	}
	r2 := newG9Runner(t, m, rcfg, brk, nil)
	_, _, _, err := r2.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if errors.Is(err, observability.ErrOpen) {
		t.Error("breaker opened after a single failed turn — per-attempt recording leak")
	}
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v, want the model error (turn was allowed)", err)
	}
	if m.calls != 6 {
		t.Errorf("calls = %d, want 6 (two turns x 3 attempts; failures=1 < threshold=2 let turn 2 through)", m.calls)
	}
}

// D9 model-stage chat span with provider/model and usage attributes.
func TestP7G9_ChatSpanWithUsage(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	m := &scriptedModel{usage: types.Usage{PromptTokens: 11, CompletionTokens: 7, TotalTokens: 18}}
	r := newG9Runner(t, m, nil, nil, nil)
	if _, _, _, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")}); err != nil {
		t.Fatalf("run: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1 chat span", len(spans))
	}
	span := spans[0]
	if span.Name != observability.SpanChat {
		t.Errorf("span name = %s, want %s", span.Name, observability.SpanChat)
	}
	attrs := map[string]string{}
	intAttrs := map[string]int64{}
	for _, kv := range span.Attributes {
		if kv.Value.AsString() != "" && kv.Value.Type().String() == "STRING" {
			attrs[string(kv.Key)] = kv.Value.AsString()
		}
		if kv.Value.Type().String() == "INT64" {
			intAttrs[string(kv.Key)] = kv.Value.AsInt64()
		}
	}
	if attrs[observability.AttrGenAIProvider] != "stub" || attrs[observability.AttrGenAIModel] != "stub-1" {
		t.Errorf("provider/model attrs = %v, want stub/stub-1", attrs)
	}
	if intAttrs[observability.AttrGenAIRequestTokens] != 11 || intAttrs[observability.AttrGenAICompletionTokens] != 7 || intAttrs[observability.AttrGenAITotalTokens] != 18 {
		t.Errorf("usage attrs = %v, want 11/7/18", intAttrs)
	}
}

// D10 one span per attempt; failed attempts record the error; last attempt carries usage.
func TestP7G9_SpanPerAttemptWithError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	m := &scriptedModel{failures: 2, errBody: "transient", usage: types.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2}}
	r := newG9Runner(t, m, noWait(3), nil, nil)
	if _, _, _, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")}); err != nil {
		t.Fatalf("run: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) != 3 {
		t.Fatalf("spans = %d, want 3 (one per attempt)", len(spans))
	}
	errSpans := 0
	usageSpans := 0
	for _, s := range spans {
		if len(s.Events) > 0 {
			errSpans++
		}
		if len(s.Attributes) > 2 {
			usageSpans++
		}
	}
	if errSpans != 2 {
		t.Errorf("spans with recorded error = %d, want 2", errSpans)
	}
	if usageSpans != 1 {
		t.Errorf("spans with usage attrs = %d, want 1 (final successful attempt)", usageSpans)
	}
}

// D11 noop tracer: no SDK configured -> identical behavior, nothing recorded.
func TestP7G9_NoopTracerZeroCost(t *testing.T) {
	otel.SetTracerProvider(sdktrace.NewTracerProvider()) // default: no exporter, non-recording spans
	m := &scriptedModel{failures: 99, errBody: "boom"}
	r := newG9Runner(t, m, nil, nil, nil)
	_, _, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if err == nil || !strings.Contains(err.Error(), "runner: model invoke: boom") || reason != StopModelFailure {
		t.Errorf("err=%v reason=%s, want today's exact shape", err, reason)
	}
	if m.calls != 1 {
		t.Errorf("calls = %d, want 1", m.calls)
	}
}

// D12 custom Invoker path is resilient too (wiring is at the InvokeTurn boundary).
func TestP7G9_CustomInvokerAlsoResilient(t *testing.T) {
	m := &scriptedModel{}
	calls := 0
	invoker := TurnInvokerFunc(func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
		calls++
		if calls < 3 {
			return nil, errors.New("transient")
		}
		return &types.ModelResponse{Content: "ok"}, nil
	})
	r := newG9Runner(t, m, noWait(3), nil, invoker)
	if _, _, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")}); err != nil || reason != StopNoToolCalls {
		t.Fatalf("err=%v reason=%s, want success", err, reason)
	}
	if calls != 3 {
		t.Errorf("invoker calls = %d, want 3", calls)
	}
}

// D13 retries never consume the MaxTurns budget.
func TestP7G9_RetryDoesNotConsumeTurnBudget(t *testing.T) {
	m := &scriptedModel{failures: 2, errBody: "transient"}
	cfg := Config{
		Model:        m,
		MaxTurns:     1,
		ToolExecutor: ToolExecutorFunc(func(ctx context.Context, calls []types.ToolCall) ([]ToolCallOutcome, error) { return nil, nil }),
		Retry:        noWait(3),
	}
	r, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, _, reason, runErr := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if runErr != nil || reason != StopNoToolCalls {
		t.Fatalf("err=%v reason=%s, want success (3 attempts inside 1 turn, MaxTurns=1)", runErr, reason)
	}
	if m.calls != 3 {
		t.Errorf("calls = %d, want 3", m.calls)
	}
}
