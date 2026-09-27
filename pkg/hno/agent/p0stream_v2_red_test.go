package agent

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// p0sLimitMessage 是内核 skipped 工具消息的文案（runner/tool_batch.go:5）。
// 断言里重复字面量是刻意的：若实现把它改掉，本文件必须失败。
const p0sLimitMessage = "tool call limit reached; call not executed"

// p0sFailingStreamModel 的流式调用总是失败，用于观察失败路径的终止原因。
type p0sFailingStreamModel struct {
	err error
}

func (m *p0sFailingStreamModel) Invoke(context.Context, *models.InvokeRequest) (*types.ModelResponse, error) {
	return nil, m.err
}

func (m *p0sFailingStreamModel) InvokeStream(context.Context, *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	return nil, m.err
}

func (m *p0sFailingStreamModel) GetProvider() string { return "p0sv2" }
func (m *p0sFailingStreamModel) GetID() string       { return "p0sv2-failing-model" }
func (m *p0sFailingStreamModel) GetName() string     { return "P0SV2 Failing Model" }

// p0sSeedToolHistory 向 Memory 预置 n 条历史 tool 消息，模拟同一 agent 之前的运行。
func p0sSeedToolHistory(ag *Agent, n int) {
	for i := 0; i < n; i++ {
		ag.Memory.Add(types.NewToolMessage("seeded", "stale tool result from a previous run"), ag.UserID)
	}
}

// p0sToolContent 返回本次运行中该 tool call id 对应的 tool 消息内容，缺失返回空串。
func p0sToolContent(messages []*types.Message, callID string) string {
	for _, message := range messages {
		if message.Role == types.RoleTool && message.ToolCallID == callID {
			return message.Content
		}
	}
	return ""
}

func p0sOneCallThenFinal() [][]types.ResponseChunk {
	return [][]types.ResponseChunk{
		{{ToolCalls: []types.ToolCall{p0sToolCall("c1", "p0s_a")}}},
		{{Content: "final"}},
	}
}

// TestP0SV2_ToolCallLimitCountsPerRunInStream 断言流式路径的 ToolCallLimit 按「每次运行」计数：
// 历史里的 tool 消息不得占用本轮额度。
func TestP0SV2_ToolCallLimitCountsPerRunInStream(t *testing.T) {
	model := &p0sStreamModel{chunks: p0sOneCallThenFinal()}
	ag, err := New(Config{
		Name:          "p0sv2-stream-per-run",
		Model:         model,
		Toolkits:      p0sToolkits(false),
		ToolCallLimit: 1,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}
	p0sSeedToolHistory(ag, 2)

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

	got := p0sToolContent(done.Output.Messages, "c1")
	if got == p0sLimitMessage {
		t.Error("c1 被跳过：历史 tool 消息占用了本轮额度")
	}
	if got != `"A"` {
		t.Errorf("c1 期望执行结果 %q，实际为 %q", `"A"`, got)
	}
	if done.Output.Content != "final" {
		t.Errorf("期望最终内容 %q，实际为 %q", "final", done.Output.Content)
	}
	if model.invocations != 2 {
		t.Errorf("期望 InvokeStream 调用 2 次，实际为 %d 次", model.invocations)
	}
}

// TestP0SV2_ToolCallLimitCountsPerRunInSync 对同步路径提出同一要求，钉住两条路径共用同一计数语义。
func TestP0SV2_ToolCallLimitCountsPerRunInSync(t *testing.T) {
	model := &p0sStreamModel{chunks: p0sOneCallThenFinal()}
	ag, err := New(Config{
		Name:          "p0sv2-sync-per-run",
		Model:         model,
		Toolkits:      p0sToolkits(false),
		ToolCallLimit: 1,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}
	p0sSeedToolHistory(ag, 2)

	output, err := ag.Run(context.Background(), "test input")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	got := p0sToolContent(output.Messages, "c1")
	if got == p0sLimitMessage {
		t.Error("c1 被跳过：历史 tool 消息占用了本轮额度")
	}
	if got != `"A"` {
		t.Errorf("c1 期望执行结果 %q，实际为 %q", `"A"`, got)
	}
	if output.Content != "final" {
		t.Errorf("期望最终内容 %q，实际为 %q", "final", output.Content)
	}
}

// p0sCancelWrappedModel 的失败内部带着 context.Canceled，但运行上下文本身没有取消。
// 用来钉住「失败阶段由内核判定，不由错误的包装链猜」。
type p0sCancelWrappedModel struct {
	err error
}

func (m *p0sCancelWrappedModel) Invoke(context.Context, *models.InvokeRequest) (*types.ModelResponse, error) {
	return nil, m.err
}

func (m *p0sCancelWrappedModel) InvokeStream(context.Context, *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	return nil, m.err
}

func (m *p0sCancelWrappedModel) GetProvider() string { return "p0sv2" }
func (m *p0sCancelWrappedModel) GetID() string       { return "p0sv2-cancel-wrapped-model" }
func (m *p0sCancelWrappedModel) GetName() string     { return "P0SV2 Cancel-Wrapped Model" }

// TestP0SV2_CancelWrappedModelFailureIsNotReportedAsCancellation 断言：运行上下文仍然存活时，
// 即使模型返回的错误内部包着 context.Canceled，两条路径都不得把它上报成「本次运行被取消」。
// 取消与否由内核的阶段判定决定，而不是由错误的包装链反推。
func TestP0SV2_CancelWrappedModelFailureIsNotReportedAsCancellation(t *testing.T) {
	sentinel := fmt.Errorf("upstream closed: %w", context.Canceled)

	syncAgent, err := New(Config{
		Name:  "p0sv2-cancel-wrap-sync",
		Model: &p0sCancelWrappedModel{err: sentinel},
	})
	if err != nil {
		t.Fatalf("failed to create sync agent: %v", err)
	}
	syncOutput, syncErr := syncAgent.Run(context.Background(), "test input")
	if syncErr == nil {
		t.Fatal("模型失败时 Run 应返回错误")
	}
	if syncOutput != nil {
		t.Error("模型失败时 Run 不应返回成品 Output")
	}
	var syncAPIErr *types.HnoError
	if !errors.As(syncErr, &syncAPIErr) {
		t.Fatalf("Run 错误类型 = %T，期望 *types.HnoError", syncErr)
	}
	if syncAPIErr.Code == types.ErrCodeCancelled {
		t.Errorf("Run 错误码 = %q，上下文未取消时不得报成取消", syncAPIErr.Code)
	}

	streamAgent, err := New(Config{
		Name:  "p0sv2-cancel-wrap-stream",
		Model: &p0sCancelWrappedModel{err: sentinel},
	})
	if err != nil {
		t.Fatalf("failed to create stream agent: %v", err)
	}
	result, err := streamAgent.RunStream(context.Background(), "test input")
	if err != nil {
		t.Fatalf("RunStream failed: %v", err)
	}
	for range result.Events {
	}
	done := <-result.Done
	if done.Err == nil {
		t.Fatal("模型失败时 RunStream 应经 Done.Err 上报")
	}
	var streamAPIErr *types.HnoError
	if !errors.As(done.Err, &streamAPIErr) {
		t.Fatalf("RunStream 错误类型 = %T，期望 *types.HnoError", done.Err)
	}
	if streamAPIErr.Code == types.ErrCodeCancelled {
		t.Errorf("RunStream 错误码 = %q，上下文未取消时不得报成取消", streamAPIErr.Code)
	}
	if done.StopReason != "model_failure" {
		t.Errorf("流式 StopReason = %q，期望 %q", done.StopReason, "model_failure")
	}
}

// TestP0SV2_ModelFailureStopReasonIsNotNoToolCalls 断言模型调用失败时不得上报 no_tool_calls。
func TestP0SV2_ModelFailureStopReasonIsNotNoToolCalls(t *testing.T) {
	ag, err := New(Config{
		Name:  "p0sv2-model-failure",
		Model: &p0sFailingStreamModel{err: errors.New("api down")},
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
	if done.Err == nil {
		t.Fatal("模型调用失败应经 Done.Err 上报")
	}
	if done.Output != nil {
		t.Error("模型调用失败不应返回成品 Output")
	}
	if got := done.StopReason; got != "model_failure" {
		t.Errorf("模型失败路径的 StopReason = %q，期望 %q", got, "model_failure")
	}
}

// p0sMessageSequence 把消息渲染为有序的可比对列，覆盖角色、工具调用配对、内容与工具调用清单。
func p0sMessageSequence(messages []*types.Message) []string {
	seq := make([]string, 0, len(messages))
	for _, message := range messages {
		callIDs := make([]string, 0, len(message.ToolCalls))
		for _, call := range message.ToolCalls {
			callIDs = append(callIDs, call.ID)
		}
		seq = append(seq, strings.Join([]string{string(message.Role), message.ToolCallID, message.Content, strings.Join(callIDs, ",")}, "|"))
	}
	return seq
}

type p0sParityScenario struct {
	name          string
	chunks        [][]types.ResponseChunk
	toolCallLimit int
	stopLoop      bool

	// wantSequence 与 wantReason 是本次运行产出的绝对期望，不来自另一条路径。
	// 它们必须与下面的等式对照同时成立：否则两条路径可以同时坏掉而测试仍绿。
	wantSequence []string
	wantReason   string
}

// TestP0SV2_SyncAndStreamProduceSameMessageSequence 用同一场景分别跑同步与流式，
// 先各自对照硬编码的绝对期望，再要求两条路径逐条一致。
func TestP0SV2_SyncAndStreamProduceSameMessageSequence(t *testing.T) {
	userTurn := "user||test input|"
	skippedC3 := "tool|c3|" + p0sLimitMessage + "|"

	scenarios := []p0sParityScenario{
		{
			name:          "truncated-batch",
			chunks:        p0sThreeCallChunks(),
			toolCallLimit: 2,
			wantSequence: []string{
				userTurn,
				"assistant|||c1,c2,c3",
				`tool|c1|"A"|`,
				`tool|c2|"B"|`,
				skippedC3,
			},
			wantReason: "limit_reached",
		},
		{
			name: "limit-exhausted-across-rounds",
			chunks: [][]types.ResponseChunk{
				{{ToolCalls: []types.ToolCall{p0sToolCall("c1", "p0s_a")}}},
				{{ToolCalls: []types.ToolCall{p0sToolCall("c2", "p0s_b")}}},
				{{ToolCalls: []types.ToolCall{p0sToolCall("c3", "p0s_a")}}},
			},
			toolCallLimit: 2,
			wantSequence: []string{
				userTurn,
				"assistant|||c1",
				`tool|c1|"A"|`,
				"assistant|||c2",
				`tool|c2|"B"|`,
				"assistant|||c3",
				skippedC3,
			},
			wantReason: "limit_reached",
		},
		{
			name: "stop-loop",
			chunks: [][]types.ResponseChunk{
				{{ToolCalls: []types.ToolCall{p0sToolCall("c1", "p0s_a")}}},
				{{Content: "must not be reached"}},
			},
			stopLoop: true,
			wantSequence: []string{
				userTurn,
				"assistant|||c1",
				`tool|c1|"A"|`,
			},
			wantReason: "stop_after_tool_call",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			syncAgent, err := New(Config{
				Name:          "p0sv2-parity-sync",
				Model:         &p0sStreamModel{chunks: scenario.chunks},
				Toolkits:      p0sToolkits(scenario.stopLoop),
				ToolCallLimit: scenario.toolCallLimit,
			})
			if err != nil {
				t.Fatalf("failed to create sync agent: %v", err)
			}
			syncOutput, err := syncAgent.Run(context.Background(), "test input")
			if err != nil {
				t.Fatalf("Run failed: %v", err)
			}

			streamAgent, err := New(Config{
				Name:          "p0sv2-parity-stream",
				Model:         &p0sStreamModel{chunks: scenario.chunks},
				Toolkits:      p0sToolkits(scenario.stopLoop),
				ToolCallLimit: scenario.toolCallLimit,
			})
			if err != nil {
				t.Fatalf("failed to create stream agent: %v", err)
			}
			result, err := streamAgent.RunStream(context.Background(), "test input")
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

			want := p0sMessageSequence(syncOutput.Messages)
			got := p0sMessageSequence(done.Output.Messages)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("完整消息序列不一致：期望 %q，实际为 %q", want, got)
			}
			if done.StopReason != syncOutput.StopReason {
				t.Errorf("StopReason 不一致：同步 %q，流式 %q", syncOutput.StopReason, done.StopReason)
			}
			if reason := result.StopReason(); reason != done.StopReason {
				t.Errorf("RunStreamResult.StopReason() = %q，与 Done.StopReason %q 不一致", reason, done.StopReason)
			}

			if !reflect.DeepEqual(got, scenario.wantSequence) {
				t.Errorf("流式绝对期望不符：期望 %q，实际为 %q", scenario.wantSequence, got)
			}
			if !reflect.DeepEqual(want, scenario.wantSequence) {
				t.Errorf("同步绝对期望不符：期望 %q，实际为 %q", scenario.wantSequence, want)
			}
			if done.StopReason != scenario.wantReason {
				t.Errorf("流式 StopReason = %q，期望 %q", done.StopReason, scenario.wantReason)
			}
			if syncOutput.StopReason != scenario.wantReason {
				t.Errorf("同步 StopReason = %q，期望 %q", syncOutput.StopReason, scenario.wantReason)
			}
		})
	}
}

// TestP0SV2_StopReasonExposedOnRunStreamResult 断言终止原因经 RunStreamResult 公共入口可读，
// 并与 Done、Output 上的值一致。
func TestP0SV2_StopReasonExposedOnRunStreamResult(t *testing.T) {
	ag, err := New(Config{
		Name:  "p0sv2-result-stop-reason",
		Model: &p0sStreamModel{chunks: [][]types.ResponseChunk{{{Content: "final"}}}},
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

	if got := result.StopReason(); got != "no_tool_calls" {
		t.Errorf("RunStreamResult.StopReason() = %q，期望 %q", got, "no_tool_calls")
	}
	if done.Output != nil && done.Output.StopReason != result.StopReason() {
		t.Errorf("Output.StopReason = %q，与 RunStreamResult.StopReason() %q 不一致", done.Output.StopReason, result.StopReason())
	}
}
