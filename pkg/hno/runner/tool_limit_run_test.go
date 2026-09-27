package runner

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// quietLogger  keeps kernel test output readable.
// quietLogger 让内核测试输出保持可读。
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func toolMessage(callID, content string) *types.Message {
	return types.NewToolMessage(callID, content)
}

// countingInvoker 是被调用的注入式回合调用器，不依赖 models.Model。
type countingInvoker struct {
	responses []*types.ModelResponse
	calls     int
}

func (i *countingInvoker) InvokeTurn(context.Context, *models.InvokeRequest) (*types.ModelResponse, error) {
	index := i.calls
	i.calls++
	if index >= len(i.responses) {
		return &types.ModelResponse{Content: "final"}, nil
	}
	return i.responses[index], nil
}

func TestDecideToolBatchContract(t *testing.T) {
	calls := []types.ToolCall{toolCall("a", "fa"), toolCall("b", "fb"), toolCall("c", "fc")}

	tests := []struct {
		name        string
		executed    int
		limit       int
		in          []types.ToolCall
		wantRun     []string
		wantSkipped []string
		wantHit     bool
	}{
		{name: "no-limit-runs-everything", limit: 0, in: calls, wantRun: []string{"a", "b", "c"}},
		{name: "partial-room-truncates", limit: 2, in: calls, wantRun: []string{"a", "b"}, wantSkipped: []string{"c"}, wantHit: true},
		{name: "full-room-not-hit", limit: 3, in: calls, wantRun: []string{"a", "b", "c"}, wantHit: false},
		{
			name:        "exhausted-before-round-skips-all",
			executed:    3,
			limit:       3,
			in:          calls,
			wantSkipped: []string{"a", "b", "c"},
			wantHit:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			toRun, skipped, hit := decideToolBatch(tc.executed, tc.limit, tc.in)
			if hit != tc.wantHit {
				t.Errorf("limitHit = %v，期望 %v", hit, tc.wantHit)
			}
			if got, want := callIDs(toRun), tc.wantRun; !equalStrings(got, want) {
				t.Errorf("toRun = %q，期望 %q", got, want)
			}
			if got, want := callIDs(skipped), tc.wantSkipped; !equalStrings(got, want) {
				t.Errorf("skipped = %q，期望 %q", got, want)
			}
		})
	}
}

// TestToolCallLimitCountsPerRunNotPerHistory 断言上限按本次运行已消耗的调用计数，
// 历史里的 tool 消息不得占用额度。
func TestToolCallLimitCountsPerRunNotPerHistory(t *testing.T) {
	model := &mockModel{responses: []*types.ModelResponse{
		{ToolCalls: []types.ToolCall{toolCall("r1", "fa")}},
		{ToolCalls: []types.ToolCall{toolCall("r2", "fb")}},
		{Content: "final"},
	}}
	exec := &recordingExecutor{}
	r, err := New(Config{
		Model:         model,
		ToolExecutor:  exec,
		ToolCallLimit: 2,
		Logger:        quietLogger(),
	})
	if err != nil {
		t.Fatal(err)
	}

	history := []*types.Message{
		types.NewUserMessage("hi"),
		toolMessage("old-1", "stale"),
		toolMessage("old-2", "stale"),
		toolMessage("old-3", "stale"),
	}
	_, messages, reason, err := r.Run(context.Background(), history)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got, want := exec.calls, []string{"fa", "fb"}; !equalStrings(got, want) {
		t.Errorf("执行的工具 = %q，期望历史不占额度、两条都执行 %q", got, want)
	}
	if reason != StopNoToolCalls {
		t.Errorf("StopReason = %q，期望 %q", reason, StopNoToolCalls)
	}
	if got := skippedContents(messages); len(got) != 0 {
		t.Errorf("不该有 skipped 消息，实际为 %q", got)
	}
}

// TestExhaustedLimitStillPairsSkippedCalls 断言「本轮开始前上限已用尽」时，
// 模型新请求的每个调用都拿到配对 tool 消息，且经 OnSkippedToolCalls 上报。
// 注意与「同轮部分截断」区分：后者由 DecideToolBatch 的 partial-room 分支覆盖。
func TestExhaustedLimitStillPairsSkippedCalls(t *testing.T) {
	model := &mockModel{responses: []*types.ModelResponse{
		{ToolCalls: []types.ToolCall{toolCall("x1", "fa")}},
		{ToolCalls: []types.ToolCall{toolCall("x2", "fb"), toolCall("x3", "fc")}},
		{Content: "must not be reached"},
	}}
	exec := &recordingExecutor{}
	var reported [][]string
	r, err := New(Config{
		Model:         model,
		ToolExecutor:  exec,
		ToolCallLimit: 1,
		Logger:        quietLogger(),
		OnSkippedToolCalls: func(skipped []types.ToolCall) {
			reported = append(reported, callIDs(skipped))
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, messages, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if reason != StopLimitReached {
		t.Errorf("StopReason = %q，期望 %q", reason, StopLimitReached)
	}
	if got, want := exec.calls, []string{"fa"}; !equalStrings(got, want) {
		t.Errorf("执行的工具 = %q，期望只有 %q", got, want)
	}
	if len(reported) != 1 || !equalStrings(reported[0], []string{"x2", "x3"}) {
		t.Errorf("OnSkippedToolCalls 上报 %q，期望恰好 [x2 x3]", reported)
	}
	paired := map[string]string{}
	for _, message := range messages {
		if message.Role == types.RoleTool {
			paired[message.ToolCallID] = message.Content
		}
	}
	for id, want := range map[string]string{
		"x1": "result:fa",
		"x2": toolCallLimitMessage,
		"x3": toolCallLimitMessage,
	} {
		if paired[id] != want {
			t.Errorf("%s 的 tool 消息 = %q，期望 %q", id, paired[id], want)
		}
	}
}

// TestFailureReasonClassification 用表钉住四类终止原因，避免失败被误报为取消或正常收尾。
func TestFailureReasonClassification(t *testing.T) {
	tests := []struct {
		name       string
		builder    MessageBuilder
		model      *mockModel
		exec       ToolExecutor
		cancelled  bool
		wantReason StopReason
	}{
		{
			name:       "model-failure",
			model:      &mockModel{err: errors.New("api down")},
			exec:       &recordingExecutor{},
			wantReason: StopModelFailure,
		},
		{
			name:       "tool-failure",
			model:      &mockModel{responses: []*types.ModelResponse{{ToolCalls: []types.ToolCall{toolCall("t1", "boom")}}}},
			exec:       &recordingExecutor{execErr: errors.New("tool failed")},
			wantReason: StopToolFailure,
		},
		{
			name:       "tool-failure-under-cancelled-context",
			model:      &mockModel{responses: []*types.ModelResponse{{ToolCalls: []types.ToolCall{toolCall("t1", "boom")}}}},
			exec:       &recordingExecutor{execErr: errors.New("tool failed")},
			cancelled:  true,
			wantReason: StopCancelled,
		},
		{
			name:  "builder-failure",
			model: &mockModel{},
			exec:  &recordingExecutor{},
			builder: MessageBuilderFunc(func(context.Context, []*types.Message, []models.ToolDefinition) (*models.InvokeRequest, error) {
				return nil, errors.New("cannot build request")
			}),
			wantReason: StopModelFailure,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := New(Config{
				Model:          tc.model,
				ToolExecutor:   tc.exec,
				MessageBuilder: tc.builder,
				Logger:         quietLogger(),
			})
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			if tc.cancelled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			_, _, reason, err := r.Run(ctx, []*types.Message{types.NewUserMessage("hi")})
			if err == nil {
				t.Fatal("期望失败并返回错误")
			}
			if reason != tc.wantReason {
				t.Errorf("StopReason = %q，期望 %q", reason, tc.wantReason)
			}
		})
	}
}

// TestInvokerTakesPrecedenceOverModel 断言注入的 TurnInvoker 是唯一被调用的回合入口，
// 流的聚合发生在 agent 侧而非内核侧。
func TestInvokerTakesPrecedenceOverModel(t *testing.T) {
	model := &mockModel{responses: []*types.ModelResponse{{Content: "from model"}}}
	invoker := &countingInvoker{responses: []*types.ModelResponse{{Content: "from invoker"}}}
	exec := &recordingExecutor{}

	r, err := New(Config{
		Model:        model,
		Invoker:      invoker,
		ToolExecutor: exec,
		Logger:       quietLogger(),
	})
	if err != nil {
		t.Fatal(err)
	}

	response, _, reason, err := r.Run(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if invoker.calls != 1 {
		t.Errorf("TurnInvoker 被调用 %d 次，期望 1", invoker.calls)
	}
	if model.index != 0 {
		t.Errorf("注入 Invoker 后仍调用了 Model.Invoke %d 次", model.index)
	}
	if response == nil || response.Content != "from invoker" {
		t.Errorf("响应内容 = %v，期望来自注入的 invoker", response)
	}
	if reason != StopNoToolCalls {
		t.Errorf("StopReason = %q，期望 %q", reason, StopNoToolCalls)
	}
}

func callIDs(calls []types.ToolCall) []string {
	ids := make([]string, 0, len(calls))
	for _, call := range calls {
		ids = append(ids, call.ID)
	}
	return ids
}

func skippedContents(messages []*types.Message) []string {
	var out []string
	for _, message := range messages {
		if message.Role == types.RoleTool && message.Content == toolCallLimitMessage {
			out = append(out, message.Content)
		}
	}
	return out
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
