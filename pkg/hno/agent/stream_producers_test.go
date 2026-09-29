package agent_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// 切片 34（P4 第 3 片，G4 剩余五模式生产者接线）：Values/Updates/Checkpoints/Debug/Custom
// 的 agent 侧判据 D1–D8。协议层（常量、事件类型、编解码）归切片 23 且本片零改 run 包；
// 三族按「模型回合」键控（agent 路径的诚实单位，不是图节点）。次序断言全部写成绝对序列
// ——沿用切片 31 的纪律：生产者缺席时两两对照会空转成立，绝对序列才会真实判红。

// p4g4pTurn 是一回合的脚本：吐哪些内容分块、请求哪些工具调用。
type p4g4pTurn struct {
	chunks    []string
	toolCalls []types.ToolCall
}

func p4g4pToolTurn(calls ...types.ToolCall) p4g4pTurn { return p4g4pTurn{toolCalls: calls} }
func p4g4pContentTurn(chunks ...string) p4g4pTurn     { return p4g4pTurn{chunks: chunks} }

func p4g4pCall(id, name, arguments string) types.ToolCall {
	return types.ToolCall{
		ID:       id,
		Type:     "function",
		Function: types.ToolCallFunction{Name: name, Arguments: arguments},
	}
}

// p4g4pToolThenContent 是最常用的两回合脚本：回合 1 只要工具调用（不吐内容、增量诚实为空），
// 回合 2 吐两个内容分块。夹具的 Memory 确定性：agent.New 不预置系统消息，回合完成时
// 消息数实测为 [2, 5]（user+assistant₁ → +tool+tool → +assistant₂）。
func p4g4pToolThenContent() []p4g4pTurn {
	return []p4g4pTurn{
		p4g4pToolTurn(
			p4g4pCall("call-weather", "weather", `{"city":"Oslo"}`),
			p4g4pCall("call-boom", "boom", `{}`),
		),
		p4g4pContentTurn("fin", "al"),
	}
}

// p4g4pModel 是手写稳定流式模型：按脚本逐回合吐分块，零网络零真模型。被要求交出脚本
// 之外的回合时它返回错误（宁可让测试响亮地失败，也不静默缩短序列）。
type p4g4pModel struct {
	models.BaseModel
	turns       []p4g4pTurn
	turn        int
	streamCalls int
}

func p4g4pNewModel(turns ...p4g4pTurn) *p4g4pModel {
	return &p4g4pModel{
		BaseModel: models.BaseModel{ID: "p4g4p", Provider: "p4g4p"},
		turns:     turns,
	}
}

func (m *p4g4pModel) Invoke(_ context.Context, _ *models.InvokeRequest) (*types.ModelResponse, error) {
	return &types.ModelResponse{ID: "p4g4p", Model: "p4g4p"}, nil
}

func (m *p4g4pModel) InvokeStream(_ context.Context, _ *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	m.streamCalls++
	if m.turn >= len(m.turns) {
		return nil, fmt.Errorf("p4g4p model asked for turn %d but the script has %d", m.turn+1, len(m.turns))
	}
	turn := m.turns[m.turn]
	m.turn++

	ch := make(chan types.ResponseChunk)
	go func() {
		defer close(ch)
		for _, chunk := range turn.chunks {
			ch <- types.ResponseChunk{Content: chunk}
		}
		if len(turn.toolCalls) > 0 {
			ch <- types.ResponseChunk{ToolCalls: turn.toolCalls}
		}
	}()
	return ch, nil
}

// p4g4pScribe 是 Custom 写入处理器：经 agent.WriteCustomEvent 从 ctx（writer 的安装处）
// 写两条自定义事件，并把两次调用交回的 error 存下来供断言（fail-closed 的三个失败面
// 都要从 handler 侧看得见）。
type p4g4pScribe struct {
	subtypeA string
	dataA    any
	subtypeB string
	dataB    any
	errA     error
	errB     error
}

func (s *p4g4pScribe) handler(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	s.errA = agent.WriteCustomEvent(ctx, s.subtypeA, s.dataA)
	s.errB = agent.WriteCustomEvent(ctx, s.subtypeB, s.dataB)
	return "scribe done", nil
}

// p4g4pToolkit 提供成功 / 真失败 / custom 写入三类处理器。参数一律可选。
func p4g4pToolkit(scribe *p4g4pScribe) toolkit.Toolkit {
	tk := toolkit.NewBaseToolkit("p4g4p")
	tk.RegisterFunction(&toolkit.Function{
		Name:        "weather",
		Description: "reports the weather",
		Parameters:  map[string]toolkit.Parameter{"city": {Type: "string"}},
		Handler: func(_ context.Context, args map[string]interface{}) (interface{}, error) {
			if city, ok := args["city"].(string); ok {
				return "sunny in " + city, nil
			}
			return "sunny", nil
		},
	})
	tk.RegisterFunction(&toolkit.Function{
		Name:        "boom",
		Description: "always fails",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
			return nil, errors.New("handler exploded")
		},
	})
	if scribe != nil {
		tk.RegisterFunction(&toolkit.Function{
			Name:        "scribe",
			Description: "writes two custom events",
			Parameters:  map[string]toolkit.Parameter{},
			Handler:     scribe.handler,
		})
	}
	return tk
}

func p4g4pAgent(t *testing.T, model models.Model, scribe *p4g4pScribe) *agent.Agent {
	t.Helper()
	ag, err := agent.New(agent.Config{
		Name:     "p4g4p-agent",
		Model:    model,
		Toolkits: []toolkit.Toolkit{p4g4pToolkit(scribe)},
	})
	if err != nil {
		t.Fatalf("agent.New() error = %v", err)
	}
	return ag
}

// p4g4pCollect 排空活流再取终值：两个观察面（通道 / 记账）都要读到尽头才可比。
func p4g4pCollect(t *testing.T, result *agent.RunStreamResult) ([]run.BaseRunOutputEvent, agent.RunStreamDone) {
	t.Helper()
	var events []run.BaseRunOutputEvent
	for evt := range result.Events {
		events = append(events, evt)
	}
	return events, <-result.Done
}

func p4g4pKinds(events []run.BaseRunOutputEvent) []string {
	kinds := make([]string, 0, len(events))
	for _, evt := range events {
		kinds = append(kinds, evt.EventType())
	}
	return kinds
}

func p4g4pAssertKinds(t *testing.T, events []run.BaseRunOutputEvent, want []string) {
	t.Helper()
	got := p4g4pKinds(events)
	if len(got) != len(want) {
		t.Fatalf("stream kinds = %v (%d events), want %v (%d events)", got, len(got), want, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] kind = %q, want %q (whole sequence %v)", i, got[i], want[i], got)
		}
	}
}

// p4g4pJSON 把载荷 marshal 成确定性字符串（map 键按字典序），供逐字节形状断言。
func p4g4pJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%+v) error = %v", value, err)
	}
	return string(encoded)
}

// TestP4G4P_FiveModesAreWiredAndStartStreams 钉 D1：接线迁移的正方向 ×5 —— 五个模式
// 逐个返回 err==nil、result!=nil、不报 ErrUnsupportedStreamMode，且模型确实被调用（流启动了）。
func TestP4G4P_FiveModesAreWiredAndStartStreams(t *testing.T) {
	for _, mode := range []run.StreamMode{
		run.StreamValues,
		run.StreamUpdates,
		run.StreamCheckpoints,
		run.StreamDebug,
		run.StreamCustom,
	} {
		t.Run(fmt.Sprintf("mode-%d", int(mode)), func(t *testing.T) {
			model := p4g4pNewModel(p4g4pToolThenContent()...)
			ag := p4g4pAgent(t, model, nil)

			result, err := ag.RunStreamMode(context.Background(), "hello", mode)
			if err != nil {
				t.Fatalf("RunStreamMode(mode %d) error = %v, want nil (mode is wired)", int(mode), err)
			}
			if result == nil {
				t.Fatalf("RunStreamMode(mode %d) result = nil, want a started stream", int(mode))
			}
			if errors.Is(err, run.ErrUnsupportedStreamMode) {
				t.Fatalf("RunStreamMode(mode %d) still reports ErrUnsupportedStreamMode", int(mode))
			}

			_, done := p4g4pCollect(t, result)
			if done.Err != nil {
				t.Fatalf("mode %d: stream finished with error = %v", int(mode), done.Err)
			}
			if done.Output == nil {
				t.Fatalf("mode %d: done output = nil", int(mode))
			}
			if model.streamCalls < 1 {
				t.Fatalf("mode %d: model InvokeStream calls = %d, want >= 1 (the stream must actually start)", int(mode), model.streamCalls)
			}
		})
	}
}

// TestP4G4P_UnknownModesStillFailClosed 钉 D2：fail-closed 收窄后的反向对照 —— 未知模式
// 序数逐个返回可 errors.Is 到 ErrUnsupportedStreamMode 的错误（文案含既有「stream mode N」
// 形状）、result==nil、模型零调用；与 Messages 混选同样整体 fail-closed。
func TestP4G4P_UnknownModesStillFailClosed(t *testing.T) {
	for _, mode := range []run.StreamMode{run.StreamMode(99), run.StreamMode(7)} {
		t.Run(fmt.Sprintf("mode-%d", int(mode)), func(t *testing.T) {
			model := p4g4pNewModel(p4g4pToolThenContent()...)
			ag := p4g4pAgent(t, model, nil)

			result, err := ag.RunStreamMode(context.Background(), "hi", mode)
			if err == nil {
				t.Fatalf("stream mode %d: want fail-closed error, got nil", int(mode))
			}
			if !errors.Is(err, run.ErrUnsupportedStreamMode) {
				t.Fatalf("stream mode %d: error %v does not wrap ErrUnsupportedStreamMode", int(mode), err)
			}
			if result != nil {
				t.Fatalf("stream mode %d: want nil result (stream must not start), got %v", int(mode), result)
			}
			if model.streamCalls != 0 {
				t.Fatalf("stream mode %d: model invoked %d times, want 0 (fail-closed)", int(mode), model.streamCalls)
			}
		})
	}

	t.Run("messages mixed with an unknown ordinal still fails closed", func(t *testing.T) {
		model := p4g4pNewModel(p4g4pToolThenContent()...)
		ag := p4g4pAgent(t, model, nil)

		result, err := ag.RunStreamMode(context.Background(), "hi", run.StreamMessages, run.StreamMode(99))
		if !errors.Is(err, run.ErrUnsupportedStreamMode) {
			t.Fatalf("Messages+99: want ErrUnsupportedStreamMode, got %v", err)
		}
		if !strings.Contains(err.Error(), "stream mode 99") {
			t.Fatalf("Messages+99: error %q must keep the existing ordinal shape", err.Error())
		}
		if result != nil || model.streamCalls != 0 {
			t.Fatalf("Messages+99: stream started (result=%v, calls=%d)", result, model.streamCalls)
		}
	})
}

// TestP4G4P_UpdatesEmitPerTurnDelta 钉 D3：Updates-only 的通道恰两条 StateUpdateEvent
// （每完成回合一条，工具回合的增量诚实为空），Patch 键形恰为 {turn,delta}，delta 是该回合
// 的聚合内容而非累计值；Node 留空；记账 = 通道同序 + 末位旧 run_completed。
func TestP4G4P_UpdatesEmitPerTurnDelta(t *testing.T) {
	model := p4g4pNewModel(p4g4pToolThenContent()...)
	ag := p4g4pAgent(t, model, nil)

	result, err := ag.RunStreamMode(context.Background(), "hello", run.StreamUpdates)
	if err != nil {
		t.Fatalf("RunStreamMode(StreamUpdates) error = %v", err)
	}
	events, done := p4g4pCollect(t, result)
	if done.Err != nil {
		t.Fatalf("stream finished with error = %v", done.Err)
	}

	p4g4pAssertKinds(t, events, []string{run.EventTypeStateUpdate, run.EventTypeStateUpdate})

	wantPatches := []string{`{"delta":"","turn":1}`, `{"delta":"final","turn":2}`}
	for i, evt := range events {
		update, ok := evt.(*run.StateUpdateEvent)
		if !ok {
			t.Fatalf("event[%d] = %T, want *run.StateUpdateEvent", i, evt)
		}
		if update.RunID == "" || update.Timestamp().IsZero() {
			t.Fatalf("event[%d] = {run_id %q, zero timestamp %v}, want non-empty run id and live timestamp", i, update.RunID, update.Timestamp().IsZero())
		}
		if update.Node != "" {
			t.Fatalf("event[%d].Node = %q, want empty (the agent path has no node identity)", i, update.Node)
		}
		if got := p4g4pJSON(t, update.Patch); got != wantPatches[i] {
			t.Fatalf("event[%d].Patch = %s, want %s", i, got, wantPatches[i])
		}
	}

	if len(done.Output.Events) != len(events)+1 {
		t.Fatalf("recorded events = %d, want %d (channel sequence + the legacy run_completed)", len(done.Output.Events), len(events)+1)
	}
	if _, ok := done.Output.Events[len(done.Output.Events)-1].(*run.RunCompletedEvent); !ok {
		t.Fatalf("last recorded event = %T, want *run.RunCompletedEvent", done.Output.Events[len(done.Output.Events)-1])
	}
}

// TestP4G4P_ValuesEmitRunningStatePerTurn 钉 D4：Values-only 的通道恰两条 StateUpdateEvent，
// Patch 键形恰为 {turn,content,messages}（与 Updates 的判别键形），content-so-far 含本回合，
// messages 是夹具确定性下的 Memory 计数 [2,5]；Updates+Values 并选时同回合两条事件键形互异。
func TestP4G4P_ValuesEmitRunningStatePerTurn(t *testing.T) {
	t.Run("values only", func(t *testing.T) {
		model := p4g4pNewModel(p4g4pToolThenContent()...)
		ag := p4g4pAgent(t, model, nil)

		result, err := ag.RunStreamMode(context.Background(), "hello", run.StreamValues)
		if err != nil {
			t.Fatalf("RunStreamMode(StreamValues) error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}

		p4g4pAssertKinds(t, events, []string{run.EventTypeStateUpdate, run.EventTypeStateUpdate})
		wantPatches := []string{`{"content":"","messages":2,"turn":1}`, `{"content":"final","messages":5,"turn":2}`}
		for i, evt := range events {
			update, ok := evt.(*run.StateUpdateEvent)
			if !ok {
				t.Fatalf("event[%d] = %T, want *run.StateUpdateEvent", i, evt)
			}
			if got := p4g4pJSON(t, update.Patch); got != wantPatches[i] {
				t.Fatalf("event[%d].Patch = %s, want %s", i, got, wantPatches[i])
			}
		}
	})

	t.Run("updates and values stay distinguishable in one stream", func(t *testing.T) {
		model := p4g4pNewModel(p4g4pToolThenContent()...)
		ag := p4g4pAgent(t, model, nil)

		result, err := ag.RunStreamMode(context.Background(), "hello", run.StreamUpdates, run.StreamValues)
		if err != nil {
			t.Fatalf("RunStreamMode(Updates+Values) error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}

		p4g4pAssertKinds(t, events, []string{
			run.EventTypeStateUpdate, run.EventTypeStateUpdate,
			run.EventTypeStateUpdate, run.EventTypeStateUpdate,
		})
		if got := p4g4pJSON(t, events[0].(*run.StateUpdateEvent).Patch); got != `{"delta":"","turn":1}` {
			t.Fatalf("turn-1 delta patch = %s, want the Updates key shape", got)
		}
		if got := p4g4pJSON(t, events[1].(*run.StateUpdateEvent).Patch); got != `{"content":"","messages":2,"turn":1}` {
			t.Fatalf("turn-1 state patch = %s, want the Values key shape", got)
		}
	})
}

// TestP4G4P_CheckpointsEmitPerTurnSnapshot 钉 D5：Checkpoints-only 的通道恰两条
// CheckpointEvent，label = turn-N，payload 与 Values 快照同形同值；三族并选时同回合内
// 次序钉死 [update, values, checkpoint] 且每回合各一条（「每回合」而非「每分块」由计数证明）。
func TestP4G4P_CheckpointsEmitPerTurnSnapshot(t *testing.T) {
	t.Run("checkpoints only", func(t *testing.T) {
		model := p4g4pNewModel(p4g4pToolThenContent()...)
		ag := p4g4pAgent(t, model, nil)

		result, err := ag.RunStreamMode(context.Background(), "hello", run.StreamCheckpoints)
		if err != nil {
			t.Fatalf("RunStreamMode(StreamCheckpoints) error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}

		p4g4pAssertKinds(t, events, []string{run.EventTypeCheckpoint, run.EventTypeCheckpoint})
		wantLabels := []string{"turn-1", "turn-2"}
		wantPayloads := []string{`{"content":"","messages":2,"turn":1}`, `{"content":"final","messages":5,"turn":2}`}
		for i, evt := range events {
			checkpoint, ok := evt.(*run.CheckpointEvent)
			if !ok {
				t.Fatalf("event[%d] = %T, want *run.CheckpointEvent", i, evt)
			}
			if checkpoint.Label != wantLabels[i] {
				t.Fatalf("event[%d].Label = %q, want %q", i, checkpoint.Label, wantLabels[i])
			}
			if got := p4g4pJSON(t, checkpoint.Payload); got != wantPayloads[i] {
				t.Fatalf("event[%d].Payload = %s, want the values snapshot %s", i, got, wantPayloads[i])
			}
			if checkpoint.Timestamp().IsZero() {
				t.Fatalf("event[%d] has zero timestamp", i)
			}
		}
	})

	t.Run("three families per turn: delta, state, checkpoint", func(t *testing.T) {
		model := p4g4pNewModel(p4g4pToolThenContent()...)
		ag := p4g4pAgent(t, model, nil)

		result, err := ag.RunStreamMode(context.Background(), "hello", run.StreamUpdates, run.StreamValues, run.StreamCheckpoints)
		if err != nil {
			t.Fatalf("RunStreamMode(three families) error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}

		p4g4pAssertKinds(t, events, []string{
			run.EventTypeStateUpdate, run.EventTypeStateUpdate, run.EventTypeCheckpoint,
			run.EventTypeStateUpdate, run.EventTypeStateUpdate, run.EventTypeCheckpoint,
		})

		delta1 := p4g4pJSON(t, events[0].(*run.StateUpdateEvent).Patch)
		state1 := p4g4pJSON(t, events[1].(*run.StateUpdateEvent).Patch)
		if state1 != p4g4pJSON(t, events[2].(*run.CheckpointEvent).Payload) {
			t.Fatalf("turn-1 checkpoint payload %s diverges from the state patch %s", p4g4pJSON(t, events[2].(*run.CheckpointEvent).Payload), state1)
		}
		if delta1 != `{"delta":"","turn":1}` || state1 != `{"content":"","messages":2,"turn":1}` {
			t.Fatalf("turn-1 shapes drifted: delta=%s state=%s", delta1, state1)
		}
		if got := events[5].(*run.CheckpointEvent).Label; got != "turn-2" {
			t.Fatalf("event[5].Label = %q, want turn-2", got)
		}
	})
}

// TestP4G4P_DebugIsTheCheckpointsTasksUnion 钉 D6：Debug = Checkpoints ∪ Tasks 的并集门
// —— 每族事件恰一份、不复制、不漏；不暗含 Messages（模型在吐分块也不泄漏 run_content）；
// Debug+Checkpoints 与 Debug-only 逐事件相同；Debug+Messages 时回合 2 的 checkpoint 落在
// 该回合 content 之后（构造事实被正面钉住）。
func TestP4G4P_DebugIsTheCheckpointsTasksUnion(t *testing.T) {
	t.Run("debug alone is checkpoints plus tasks, exactly once", func(t *testing.T) {
		model := p4g4pNewModel(p4g4pToolThenContent()...)
		ag := p4g4pAgent(t, model, nil)

		result, err := ag.RunStreamMode(context.Background(), "hello", run.StreamDebug)
		if err != nil {
			t.Fatalf("RunStreamMode(StreamDebug) error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}

		p4g4pAssertKinds(t, events, []string{
			run.EventTypeCheckpoint,
			run.EventTypeNodeStarted, run.EventTypeNodeStarted, run.EventTypeNodeCompleted, run.EventTypeTaskError,
			run.EventTypeCheckpoint,
		})
		for i, evt := range events {
			if _, ok := evt.(*run.RunContentEvent); ok {
				t.Fatalf("event[%d] leaked *run.RunContentEvent into a Debug-only stream (Debug does not imply Messages)", i)
			}
		}
		if got := events[0].(*run.CheckpointEvent).Label; got != "turn-1" {
			t.Fatalf("event[0].Label = %q, want turn-1", got)
		}
		if got := events[5].(*run.CheckpointEvent).Label; got != "turn-2" {
			t.Fatalf("event[5].Label = %q, want turn-2", got)
		}
	})

	t.Run("debug plus checkpoints does not duplicate", func(t *testing.T) {
		runOnce := func(t *testing.T, modes ...run.StreamMode) []run.BaseRunOutputEvent {
			t.Helper()
			model := p4g4pNewModel(p4g4pToolThenContent()...)
			ag := p4g4pAgent(t, model, nil)
			result, err := ag.RunStreamMode(context.Background(), "hello", modes...)
			if err != nil {
				t.Fatalf("RunStreamMode(%v) error = %v", modes, err)
			}
			events, done := p4g4pCollect(t, result)
			if done.Err != nil {
				t.Fatalf("stream finished with error = %v", done.Err)
			}
			return events
		}

		alone := runOnce(t, run.StreamDebug)
		withCheckpoints := runOnce(t, run.StreamCheckpoints, run.StreamDebug)
		if len(withCheckpoints) != len(alone) {
			t.Fatalf("Debug+Checkpoints produced %d events, want the same %d as Debug alone (no duplication)", len(withCheckpoints), len(alone))
		}
		p4g4pAssertKinds(t, withCheckpoints, p4g4pKinds(alone))
	})

	t.Run("debug plus messages pins the within-turn-2 order", func(t *testing.T) {
		model := p4g4pNewModel(p4g4pToolThenContent()...)
		ag := p4g4pAgent(t, model, nil)

		result, err := ag.RunStreamMode(context.Background(), "hello", run.StreamDebug, run.StreamMessages)
		if err != nil {
			t.Fatalf("RunStreamMode(Debug+Messages) error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}

		p4g4pAssertKinds(t, events, []string{
			run.EventTypeCheckpoint,
			run.EventTypeNodeStarted, run.EventTypeNodeStarted, run.EventTypeNodeCompleted, run.EventTypeTaskError,
			run.EventTypeRunContent, run.EventTypeRunContent,
			run.EventTypeCheckpoint,
		})
	})
}

// TestP4G4P_CustomWriterWritesFromToolHandlers 钉 D7：Custom-selected 运行中 handler 经
// agent.WriteCustomEvent 写入的事件以 CustomEvent{Subtype,Data} 落通道与账本；写入口
// fail-closed：纯 ctx / 未选 Custom 的运行都返回错误而非静默吞。
func TestP4G4P_CustomWriterWritesFromToolHandlers(t *testing.T) {
	t.Run("handler writes land on the wire", func(t *testing.T) {
		scribe := &p4g4pScribe{subtypeA: "progress", dataA: "half done", subtypeB: "milestone", dataB: 2}
		model := p4g4pNewModel(
			p4g4pToolTurn(p4g4pCall("call-scribe", "scribe", `{}`)),
			p4g4pContentTurn("tail"),
		)
		ag := p4g4pAgent(t, model, scribe)

		result, err := ag.RunStreamMode(context.Background(), "write please", run.StreamCustom)
		if err != nil {
			t.Fatalf("RunStreamMode(StreamCustom) error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}

		p4g4pAssertKinds(t, events, []string{run.EventTypeCustom, run.EventTypeCustom})
		customA, ok := events[0].(*run.CustomEvent)
		if !ok {
			t.Fatalf("event[0] = %T, want *run.CustomEvent", events[0])
		}
		if customA.Subtype != "progress" || customA.Data != "half done" || customA.RunID == "" || customA.Timestamp().IsZero() {
			t.Fatalf("event[0] = {subtype %q, data %#v, run_id %q}, want the handler's write with live identity", customA.Subtype, customA.Data, customA.RunID)
		}
		customB, ok := events[1].(*run.CustomEvent)
		if !ok || customB.Subtype != "milestone" || customB.Data != 2 {
			t.Fatalf("event[1] = %+v, want {milestone, 2}", events[1])
		}
		if scribe.errA != nil || scribe.errB != nil {
			t.Fatalf("handler writes returned errors: %v / %v", scribe.errA, scribe.errB)
		}

		if len(done.Output.Events) != len(events)+1 {
			t.Fatalf("recorded events = %d, want %d (channel sequence + the legacy run_completed)", len(done.Output.Events), len(events)+1)
		}
	})

	t.Run("bare context fails closed", func(t *testing.T) {
		if err := agent.WriteCustomEvent(context.Background(), "orphan", 1); err == nil {
			t.Fatal("WriteCustomEvent on a bare context returned nil, want the fail-closed error")
		}
	})

	t.Run("a run without StreamCustom gives its handlers no writer", func(t *testing.T) {
		scribe := &p4g4pScribe{subtypeA: "sneaky", dataA: 1}
		model := p4g4pNewModel(
			p4g4pToolTurn(p4g4pCall("call-scribe", "scribe", `{}`)),
			p4g4pContentTurn("tail"),
		)
		ag := p4g4pAgent(t, model, scribe)

		result, err := ag.RunStreamMode(context.Background(), "no custom", run.StreamMessages)
		if err != nil {
			t.Fatalf("RunStreamMode(StreamMessages) error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}

		p4g4pAssertKinds(t, events, []string{run.EventTypeRunContent})
		if scribe.errA == nil {
			t.Fatal("handler write on a Messages-only run returned nil, want the fail-closed error (writer absence by construction)")
		}
		for i, evt := range events {
			if _, ok := evt.(*run.CustomEvent); ok {
				t.Fatalf("event[%d] leaked a custom event into a Messages-only run", i)
			}
		}
	})
}

// TestP4G4P_MessagesAndTasksBehaviorUnchanged 钉 D8：旧两族零漂移 —— Messages-only、
// Tasks-only、Messages+Tasks 的通道序列与切片 23/31 行为线逐事件同型。
func TestP4G4P_MessagesAndTasksBehaviorUnchanged(t *testing.T) {
	t.Run("messages only", func(t *testing.T) {
		model := p4g4pNewModel(p4g4pToolThenContent()...)
		ag := p4g4pAgent(t, model, nil)
		result, err := ag.RunStreamMode(context.Background(), "hello", run.StreamMessages)
		if err != nil {
			t.Fatalf("RunStreamMode(StreamMessages) error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}
		p4g4pAssertKinds(t, events, []string{run.EventTypeRunContent, run.EventTypeRunContent})
		for i, evt := range events {
			content := evt.(*run.RunContentEvent)
			if content.Sequence != i {
				t.Fatalf("event[%d].Sequence = %d, want %d (content numbering untouched)", i, content.Sequence, i)
			}
		}
	})

	t.Run("tasks only", func(t *testing.T) {
		model := p4g4pNewModel(p4g4pToolThenContent()...)
		ag := p4g4pAgent(t, model, nil)
		result, err := ag.RunStreamMode(context.Background(), "hello", run.StreamTasks)
		if err != nil {
			t.Fatalf("RunStreamMode(StreamTasks) error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}
		p4g4pAssertKinds(t, events, []string{
			run.EventTypeNodeStarted, run.EventTypeNodeStarted, run.EventTypeNodeCompleted, run.EventTypeTaskError,
		})
	})

	t.Run("messages plus tasks union", func(t *testing.T) {
		model := p4g4pNewModel(p4g4pToolThenContent()...)
		ag := p4g4pAgent(t, model, nil)
		result, err := ag.RunStreamMode(context.Background(), "hello", run.StreamMessages, run.StreamTasks)
		if err != nil {
			t.Fatalf("RunStreamMode(Messages+Tasks) error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}
		p4g4pAssertKinds(t, events, []string{
			run.EventTypeNodeStarted, run.EventTypeNodeStarted, run.EventTypeNodeCompleted, run.EventTypeTaskError,
			run.EventTypeRunContent, run.EventTypeRunContent,
		})
	})

	t.Run("RunStream wrapper unchanged", func(t *testing.T) {
		model := p4g4pNewModel(p4g4pToolThenContent()...)
		ag := p4g4pAgent(t, model, nil)
		result, err := ag.RunStream(context.Background(), "hello")
		if err != nil {
			t.Fatalf("RunStream error = %v", err)
		}
		events, done := p4g4pCollect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}
		p4g4pAssertKinds(t, events, []string{run.EventTypeRunContent, run.EventTypeRunContent})
	})
}
