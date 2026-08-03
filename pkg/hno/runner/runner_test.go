package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// mockModel is a scripted model for tests.
// mockModel 是测试用的脚本化模型。
type mockModel struct {
	responses []*types.ModelResponse // scripted responses in order / 按顺序的脚本化响应
	index     int
	err       error
	repeat    *types.ModelResponse // if set, every Invoke returns this / 若设置，每次 Invoke 都返回它
}

func (m *mockModel) Invoke(_ context.Context, _ *models.InvokeRequest) (*types.ModelResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.repeat != nil {
		return m.repeat, nil
	}
	if m.index >= len(m.responses) {
		return &types.ModelResponse{Content: "fallback"}, nil
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

func (m *mockModel) InvokeStream(context.Context, *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	return nil, errors.New("not implemented")
}

func (m *mockModel) GetProvider() string { return "mock" }
func (m *mockModel) GetID() string       { return "mock-model" }
func (m *mockModel) GetName() string     { return "Mock Model" }

func toolCall(id, name string) types.ToolCall {
	return types.ToolCall{
		ID:   id,
		Type: "function",
		Function: types.ToolCallFunction{
			Name:      name,
			Arguments: `{}`,
		},
	}
}

// recordingExecutor records executed calls and returns scripted outcomes.
// recordingExecutor 记录执行的调用并返回脚本化结果。
type recordingExecutor struct {
	calls     []string
	stopLoop  bool
	hitlBlock bool
	execErr   error
}

func (e *recordingExecutor) Execute(_ context.Context, calls []types.ToolCall) ([]ToolCallOutcome, error) {
	if e.execErr != nil {
		return nil, e.execErr
	}
	outcomes := make([]ToolCallOutcome, 0, len(calls))
	for _, c := range calls {
		e.calls = append(e.calls, c.Function.Name)
		outcomes = append(outcomes, ToolCallOutcome{
			Call: c,
			Message: &types.Message{
				Role:       types.RoleTool,
				ToolCallID: c.ID,
				Content:    "result:" + c.Function.Name,
			},
			StopLoop:    e.stopLoop,
			HITLBlocked: e.hitlBlock,
		})
	}
	return outcomes, nil
}

func newTestRunner(m *mockModel, exec ToolExecutor, opts ...func(*Config)) (*Runner, error) {
	cfg := Config{
		Model:        m,
		ToolExecutor: exec,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return New(cfg)
}

func withMaxTurns(n int) func(*Config) {
	return func(c *Config) { c.MaxTurns = n }
}

func withToolCallLimit(n int) func(*Config) {
	return func(c *Config) { c.ToolCallLimit = n }
}

func TestNewValidation(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected error when model is nil")
	}
	if _, err := New(Config{Model: &mockModel{}}); err == nil {
		t.Fatal("expected error when tool executor is nil")
	}
	r, err := New(Config{Model: &mockModel{}, ToolExecutor: &recordingExecutor{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.maxTurns != DefaultMaxTurns {
		t.Errorf("expected default max turns %d, got %d", DefaultMaxTurns, r.maxTurns)
	}
}

func TestRunnerDirectAnswer(t *testing.T) {
	m := &mockModel{responses: []*types.ModelResponse{{Content: "hello"}}}
	exec := &recordingExecutor{}
	r, err := newTestRunner(m, exec)
	if err != nil {
		t.Fatal(err)
	}

	resp, msgs, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != StopNoToolCalls {
		t.Errorf("expected StopNoToolCalls, got %s", reason)
	}
	if resp.Content != "hello" {
		t.Errorf("expected 'hello', got %q", resp.Content)
	}
	if len(exec.calls) != 0 {
		t.Errorf("expected no tool executions, got %v", exec.calls)
	}
	// user + assistant = 2 messages
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestRunnerSingleToolLoop(t *testing.T) {
	m := &mockModel{responses: []*types.ModelResponse{
		{Content: "", ToolCalls: []types.ToolCall{toolCall("call_1", "add")}},
		{Content: "42"},
	}}
	exec := &recordingExecutor{}
	r, err := newTestRunner(m, exec)
	if err != nil {
		t.Fatal(err)
	}

	resp, msgs, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("2+2")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != StopNoToolCalls {
		t.Errorf("expected StopNoToolCalls, got %s", reason)
	}
	if resp.Content != "42" {
		t.Errorf("expected '42', got %q", resp.Content)
	}
	if len(exec.calls) != 1 || exec.calls[0] != "add" {
		t.Errorf("expected [add], got %v", exec.calls)
	}
	// user + assistant(toolcall) + tool + assistant(final) = 4
	if len(msgs) != 4 {
		t.Errorf("expected 4 messages, got %d", len(msgs))
	}
}

func TestRunnerStopAfterToolCall(t *testing.T) {
	m := &mockModel{responses: []*types.ModelResponse{
		{Content: "", ToolCalls: []types.ToolCall{toolCall("call_1", "approve")}},
	}}
	exec := &recordingExecutor{stopLoop: true}
	r, err := newTestRunner(m, exec)
	if err != nil {
		t.Fatal(err)
	}

	_, _, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("go")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != StopAfterToolCall {
		t.Errorf("expected StopAfterToolCall, got %s", reason)
	}
}

func TestRunnerHITLBlocked(t *testing.T) {
	m := &mockModel{responses: []*types.ModelResponse{
		{Content: "", ToolCalls: []types.ToolCall{toolCall("call_1", "dangerous_op")}},
	}}
	exec := &recordingExecutor{hitlBlock: true}
	r, err := newTestRunner(m, exec)
	if err != nil {
		t.Fatal(err)
	}

	_, _, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("do it")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != StopHITLBlocked {
		t.Errorf("expected StopHITLBlocked, got %s", reason)
	}
}

func TestRunnerMaxTurnsLimit(t *testing.T) {
	// Model always requests a tool → loop must terminate at max turns.
	// 模型每次都请求工具 → 循环必须在 max_turns 处终止。
	m := &mockModel{repeat: &types.ModelResponse{
		ToolCalls: []types.ToolCall{toolCall("call", "loop")},
	}}
	exec := &recordingExecutor{}
	r, err := newTestRunner(m, exec, withMaxTurns(3))
	if err != nil {
		t.Fatal(err)
	}

	_, _, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("x")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != StopLimitReached {
		t.Errorf("expected StopLimitReached, got %s", reason)
	}
	// 3 model calls happened; the 3rd returned tool calls but hit max_turns,
	// so only the first 2 batches were executed.
	// 发生了 3 次模型调用；第 3 次返回工具调用但已达 max_turns，
	// 因此只执行了前 2 批工具。
	if len(exec.calls) != 2 {
		t.Errorf("expected exactly 2 tool executions, got %d", len(exec.calls))
	}
}

func TestRunnerToolCallLimit(t *testing.T) {
	m := &mockModel{repeat: &types.ModelResponse{
		ToolCalls: []types.ToolCall{toolCall("call", "loop")},
	}}
	exec := &recordingExecutor{}
	r, err := newTestRunner(m, exec, withToolCallLimit(4))
	if err != nil {
		t.Fatal(err)
	}

	_, msgs, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("x")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != StopLimitReached {
		t.Errorf("expected StopLimitReached, got %s", reason)
	}
	if len(exec.calls) != 4 {
		t.Errorf("expected exactly 4 tool executions, got %d", len(exec.calls))
	}
	// 4 executed tool messages + user + 4 assistant (with tool calls) + 1 final assistant = 10
	// 4 条已执行工具消息 + 用户 + 4 条带工具调用的助手消息 + 1 条最终助手消息 = 10
	if len(msgs) != 10 {
		t.Errorf("expected 10 messages, got %d", len(msgs))
	}
}

func TestRunnerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled / 预先取消

	m := &mockModel{responses: []*types.ModelResponse{{Content: "hi"}}}
	exec := &recordingExecutor{}
	r, err := newTestRunner(m, exec)
	if err != nil {
		t.Fatal(err)
	}

	_, _, reason, err := r.Run(ctx, []*types.Message{types.NewUserMessage("hi")})
	if reason != StopCancelled {
		t.Errorf("expected StopCancelled, got %s", reason)
	}
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestRunnerModelError(t *testing.T) {
	m := &mockModel{err: errors.New("api down")}
	exec := &recordingExecutor{}
	r, err := newTestRunner(m, exec)
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if err == nil {
		t.Fatal("expected model error to propagate")
	}
}

func TestRunnerToolExecutorError(t *testing.T) {
	m := &mockModel{responses: []*types.ModelResponse{
		{Content: "", ToolCalls: []types.ToolCall{toolCall("call_1", "boom")}},
	}}
	exec := &recordingExecutor{execErr: errors.New("tool failed")}
	r, err := newTestRunner(m, exec)
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if err == nil {
		t.Fatal("expected tool executor error to propagate")
	}
}

func TestRunnerOnStepCallback(t *testing.T) {
	m := &mockModel{responses: []*types.ModelResponse{
		{Content: "", ToolCalls: []types.ToolCall{toolCall("call_1", "add")}},
		{Content: "42"},
	}}
	exec := &recordingExecutor{}

	var steps []StepEvent
	r, err := New(Config{
		Model:        m,
		ToolExecutor: exec,
		OnStep: func(ev StepEvent) {
			steps = append(steps, ev)
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = r.Run(context.Background(), []*types.Message{types.NewUserMessage("2+2")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// turn 1: await_model (with tool calls) + tool_calls_pending, turn 2: await_model (final)
	if len(steps) != 3 {
		t.Errorf("expected 3 step events, got %d", len(steps))
	}
	if steps[0].State != StateAwaitModel || steps[0].Turn != 1 {
		t.Errorf("unexpected first step: %+v", steps[0])
	}
}
