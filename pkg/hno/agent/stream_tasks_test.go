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

// 切片 31（母约 §5 G4 生产者接线，第 2 片）：StreamTasks 的 agent 侧判据 D1–D9。
// 协议层（常量、事件类型、编解码）归切片 23，本片只接生产者，因此一条断言都不许
// 落在 run 包的新符号上（D10：run 包零改动）。
//
// 次序断言全部写成**绝对序列**（不是「A 与 B 相等」这类两两对照）：Tasks 未接线时
// 通道是空的，两两对照会空转成立，绝对序列才会真实判红。

// p4s31LookalikeText 是 D5 的可分辨夹具核心：一个**成功**的处理器返回与失败文案
// 一字不差的文本。按结果文本判成败的分类器会把它错认成 task_error。
const p4s31LookalikeText = "tool execution error: handler exploded"

// p4s31HandlerError 是失败处理器交回的哨兵。
var p4s31HandlerError = errors.New("handler exploded")

// p4s31Turn 是一回合的脚本：吐哪些内容分块、请求哪些工具调用。
type p4s31Turn struct {
	chunks    []string
	toolCalls []types.ToolCall
}

func p4s31ToolTurn(calls ...types.ToolCall) p4s31Turn {
	return p4s31Turn{toolCalls: calls}
}

func p4s31ContentTurn(chunks ...string) p4s31Turn {
	return p4s31Turn{chunks: chunks}
}

// p4s31Model 是手写稳定流式模型：按脚本逐回合吐分块，零网络零真模型。
// 被要求交出脚本之外的回合时它返回错误（宁可让测试响亮地失败，也不静默缩短序列）。
type p4s31Model struct {
	models.BaseModel
	turns       []p4s31Turn
	turn        int
	streamCalls int
	invokeCalls int
}

func p4s31NewModel(turns ...p4s31Turn) *p4s31Model {
	return &p4s31Model{
		BaseModel: models.BaseModel{ID: "p4s31", Provider: "p4s31"},
		turns:     turns,
	}
}

func (m *p4s31Model) Invoke(_ context.Context, _ *models.InvokeRequest) (*types.ModelResponse, error) {
	m.invokeCalls++
	return &types.ModelResponse{ID: "p4s31", Model: "p4s31"}, nil
}

func (m *p4s31Model) InvokeStream(_ context.Context, _ *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	m.streamCalls++
	if m.turn >= len(m.turns) {
		return nil, fmt.Errorf("p4s31 model asked for turn %d but the script has %d", m.turn+1, len(m.turns))
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

func p4s31Call(id, name, arguments string) types.ToolCall {
	return types.ToolCall{
		ID:       id,
		Type:     "function",
		Function: types.ToolCallFunction{Name: name, Arguments: arguments},
	}
}

func p4s31ArgString(args map[string]interface{}, key string) string {
	if value, ok := args[key].(string); ok {
		return value
	}
	return ""
}

// p4s31Toolkit 提供四类处理器：成功、真失败、文本 lookalike、另一个成功工具。
// 参数一律可选：被上限截断的夹具只关心「谁发声」，不关心校验。
func p4s31Toolkit() toolkit.Toolkit {
	tk := toolkit.NewBaseToolkit("p4s31")
	tk.RegisterFunction(&toolkit.Function{
		Name:        "weather",
		Description: "reports the weather",
		Parameters:  map[string]toolkit.Parameter{"city": {Type: "string"}},
		Handler: func(_ context.Context, args map[string]interface{}) (interface{}, error) {
			return "sunny in " + p4s31ArgString(args, "city"), nil
		},
	})
	tk.RegisterFunction(&toolkit.Function{
		Name:        "boom",
		Description: "always fails",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
			return nil, p4s31HandlerError
		},
	})
	tk.RegisterFunction(&toolkit.Function{
		Name:        "lookalike",
		Description: "succeeds with failure-looking text",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
			return p4s31LookalikeText, nil
		},
	})
	tk.RegisterFunction(&toolkit.Function{
		Name:        "echo",
		Description: "echoes its argument",
		Parameters:  map[string]toolkit.Parameter{"text": {Type: "string"}},
		Handler: func(_ context.Context, args map[string]interface{}) (interface{}, error) {
			return p4s31ArgString(args, "text"), nil
		},
	})
	return tk
}

func p4s31Agent(t *testing.T, model models.Model, toolCallLimit int) *agent.Agent {
	t.Helper()
	ag, err := agent.New(agent.Config{
		Name:          "p4s31-agent",
		Model:         model,
		Toolkits:      []toolkit.Toolkit{p4s31Toolkit()},
		ToolCallLimit: toolCallLimit,
	})
	if err != nil {
		t.Fatalf("agent.New() error = %v", err)
	}
	return ag
}

// p4s31Collect 排空活流再取终值：两个观察面（通道 / 记账）都要读到尽头才可比。
func p4s31Collect(t *testing.T, result *agent.RunStreamResult) ([]run.BaseRunOutputEvent, agent.RunStreamDone) {
	t.Helper()
	var events []run.BaseRunOutputEvent
	for evt := range result.Events {
		events = append(events, evt)
	}
	return events, <-result.Done
}

func p4s31Kinds(events []run.BaseRunOutputEvent) []string {
	kinds := make([]string, 0, len(events))
	for _, evt := range events {
		kinds = append(kinds, evt.EventType())
	}
	return kinds
}

func p4s31AssertKinds(t *testing.T, events []run.BaseRunOutputEvent, want []string) {
	t.Helper()
	got := p4s31Kinds(events)
	if len(got) != len(want) {
		t.Fatalf("stream kinds = %v (%d events), want %v (%d events)", got, len(got), want, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] kind = %q, want %q (whole sequence %v)", i, got[i], want[i], got)
		}
	}
}

func p4s31JSONText(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%q) error = %v", value, err)
	}
	return string(encoded)
}

// 一批「1 成 1 败」的调用在流上的期望形状（契约 designConsequence 第 2/4 条）。
var p4s31Batch1Kinds = []string{
	run.EventTypeNodeStarted,
	run.EventTypeNodeStarted,
	run.EventTypeNodeCompleted,
	run.EventTypeTaskError,
}

// p4s31ToolThenContent 是最常用的两回合脚本：回合 1 只要工具调用（不吐内容），
// 回合 2 只吐两个内容分块。Messages+Tasks 的并集序列即 [任务 4 条, 内容 2 条]。
func p4s31WeatherBoomTurn() p4s31Turn {
	return p4s31ToolTurn(
		p4s31Call("call-weather", "weather", `{"city":"Oslo"}`),
		p4s31Call("call-boom", "boom", `{}`),
	)
}

func p4s31ToolThenContent() []p4s31Turn {
	return []p4s31Turn{p4s31WeatherBoomTurn(), p4s31ContentTurn("fin", "al")}
}

// TestP4S31_TasksModeIsWiredAndStartsTheStream 钉 D1：接线迁移的正方向 —— StreamTasks
// 从 fail-closed 集合移出，流真的被启动（模型被调用），且不再报 ErrUnsupportedStreamMode。
func TestP4S31_TasksModeIsWiredAndStartsTheStream(t *testing.T) {
	model := p4s31NewModel(p4s31ToolThenContent()...)
	ag := p4s31Agent(t, model, 0)

	result, err := ag.RunStreamMode(context.Background(), "weather please", run.StreamTasks)
	if err != nil {
		t.Fatalf("RunStreamMode(StreamTasks) error = %v, want nil (mode is wired)", err)
	}
	if result == nil {
		t.Fatal("RunStreamMode(StreamTasks) result = nil, want a started stream")
	}
	if errors.Is(err, run.ErrUnsupportedStreamMode) {
		t.Fatal("RunStreamMode(StreamTasks) still reports ErrUnsupportedStreamMode")
	}

	_, done := p4s31Collect(t, result)
	if done.Err != nil {
		t.Fatalf("stream finished with error = %v", done.Err)
	}
	if done.Output == nil {
		t.Fatal("done output = nil")
	}
	if model.streamCalls < 1 {
		t.Fatalf("model InvokeStream calls = %d, want >= 1 (the stream must actually start)", model.streamCalls)
	}
}

// TestP4S31_UnwiredModesStillFailClosed 钉 D2：接线迁移的反方向 —— 五个未接线的模式
// 逐个仍 fail-closed（错误、result==nil、模型零调用），与 Tasks 混选时整次调用仍
// fail-closed 且点名**首个**未接线模式。十条子用例。
func TestP4S31_UnwiredModesStillFailClosed(t *testing.T) {
	unwired := []run.StreamMode{
		run.StreamValues,
		run.StreamUpdates,
		run.StreamCheckpoints,
		run.StreamDebug,
		run.StreamCustom,
	}

	for _, mode := range unwired {
		t.Run(fmt.Sprintf("single-mode-%d", int(mode)), func(t *testing.T) {
			model := p4s31NewModel(p4s31ToolThenContent()...)
			ag := p4s31Agent(t, model, 0)

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

	for _, mode := range unwired {
		t.Run(fmt.Sprintf("tasks-mixed-with-mode-%d", int(mode)), func(t *testing.T) {
			model := p4s31NewModel(p4s31ToolThenContent()...)
			ag := p4s31Agent(t, model, 0)

			result, err := ag.RunStreamMode(context.Background(), "hi", run.StreamTasks, mode)
			if !errors.Is(err, run.ErrUnsupportedStreamMode) {
				t.Fatalf("Tasks+mode-%d: want ErrUnsupportedStreamMode, got %v", int(mode), err)
			}
			if result != nil || model.streamCalls != 0 {
				t.Fatalf("Tasks+mode-%d: stream started (result=%v, calls=%d)", int(mode), result, model.streamCalls)
			}
			wantNamed := fmt.Sprintf("stream mode %d", int(mode))
			if !strings.Contains(err.Error(), wantNamed) {
				t.Fatalf("Tasks+mode-%d: error %q must name the first unwired mode (%q)", int(mode), err.Error(), wantNamed)
			}
		})
	}
}

// TestP4S31_TasksOnlyStreamCarriesTaskEventsOnly 钉 D3：按族门控 —— Tasks-only 的
// 通道只含任务三事件、绝不泄漏 run_content；记账序列 = 通道同序 + 末位旧 run_completed。
func TestP4S31_TasksOnlyStreamCarriesTaskEventsOnly(t *testing.T) {
	model := p4s31NewModel(p4s31ToolThenContent()...)
	ag := p4s31Agent(t, model, 0)

	result, err := ag.RunStreamMode(context.Background(), "weather please", run.StreamTasks)
	if err != nil {
		t.Fatalf("RunStreamMode(StreamTasks) error = %v", err)
	}
	events, done := p4s31Collect(t, result)
	if done.Err != nil {
		t.Fatalf("stream finished with error = %v", done.Err)
	}

	p4s31AssertKinds(t, events, p4s31Batch1Kinds)

	// 内容族未选中：通道上不允许出现任何 run_content，即使模型确实在吐分块。
	for i, evt := range events {
		if _, ok := evt.(*run.RunContentEvent); ok {
			t.Fatalf("event[%d] leaked *run.RunContentEvent into a Tasks-only stream", i)
		}
	}

	if done.Output == nil {
		t.Fatal("done output = nil")
	}
	recorded := done.Output.Events
	if len(recorded) != len(events)+1 {
		t.Fatalf("recorded events = %d, want %d (channel sequence + the legacy run_completed)", len(recorded), len(events)+1)
	}
	for i := range events {
		if recorded[i].EventType() != events[i].EventType() {
			t.Fatalf("recorded[%d] kind = %q, want %q (ledger must mirror the channel prefix)", i, recorded[i].EventType(), events[i].EventType())
		}
	}
	if _, ok := recorded[len(recorded)-1].(*run.RunCompletedEvent); !ok {
		t.Fatalf("last recorded event = %T, want *run.RunCompletedEvent", recorded[len(recorded)-1])
	}
}

// TestP4S31_TaskEventPayloadsAndWireShape 钉 D4：逐字段线格式（designConsequence 第 2 条）
// 与 JSON 往返（复用切片 23 协议，本片不改它）。
func TestP4S31_TaskEventPayloadsAndWireShape(t *testing.T) {
	model := p4s31NewModel(p4s31ToolThenContent()...)
	ag := p4s31Agent(t, model, 0)

	result, err := ag.RunStreamMode(context.Background(), "weather please", run.StreamTasks)
	if err != nil {
		t.Fatalf("RunStreamMode(StreamTasks) error = %v", err)
	}
	events, done := p4s31Collect(t, result)
	if done.Err != nil {
		t.Fatalf("stream finished with error = %v", done.Err)
	}
	p4s31AssertKinds(t, events, p4s31Batch1Kinds)

	startedWeather, ok := events[0].(*run.NodeStartedEvent)
	if !ok {
		t.Fatalf("event[0] = %T, want *run.NodeStartedEvent", events[0])
	}
	if startedWeather.RunID == "" {
		t.Fatal("node_started.RunID is empty")
	}
	if startedWeather.Node != "weather" {
		t.Fatalf("node_started.Node = %q, want the tool function name %q", startedWeather.Node, "weather")
	}
	if startedWeather.Attempt != 1 {
		t.Fatalf("node_started.Attempt = %d, want 1", startedWeather.Attempt)
	}
	if startedWeather.Input != `{"city":"Oslo"}` {
		t.Fatalf("node_started.Input = %#v, want the raw arguments text %q", startedWeather.Input, `{"city":"Oslo"}`)
	}
	if startedWeather.Timestamp().IsZero() {
		t.Fatal("node_started timestamp is zero")
	}

	startedBoom, ok := events[1].(*run.NodeStartedEvent)
	if !ok {
		t.Fatalf("event[1] = %T, want *run.NodeStartedEvent", events[1])
	}
	if startedBoom.Node != "boom" || startedBoom.Input != `{}` {
		t.Fatalf("second node_started = {node %q, input %#v}, want {boom, {}}", startedBoom.Node, startedBoom.Input)
	}

	completed, ok := events[2].(*run.NodeCompletedEvent)
	if !ok {
		t.Fatalf("event[2] = %T, want *run.NodeCompletedEvent", events[2])
	}
	if completed.RunID == "" || completed.Node != "weather" || completed.Attempt != 1 {
		t.Fatalf("node_completed = {run_id %q, node %q, attempt %d}, want non-empty run_id, weather, 1", completed.RunID, completed.Node, completed.Attempt)
	}
	if want := p4s31JSONText(t, "sunny in Oslo"); completed.Output != want {
		t.Fatalf("node_completed.Output = %#v, want the tool message text %q", completed.Output, want)
	}
	if completed.Timestamp().IsZero() {
		t.Fatal("node_completed timestamp is zero")
	}

	taskError, ok := events[3].(*run.TaskErrorEvent)
	if !ok {
		t.Fatalf("event[3] = %T, want *run.TaskErrorEvent", events[3])
	}
	if taskError.RunID == "" || taskError.Node != "boom" {
		t.Fatalf("task_error = {run_id %q, node %q}, want non-empty run_id and boom", taskError.RunID, taskError.Node)
	}
	if !strings.Contains(taskError.Message, p4s31HandlerError.Error()) {
		t.Fatalf("task_error.Message = %q, want it to carry the handler error %q", taskError.Message, p4s31HandlerError)
	}
	if taskError.Timestamp().IsZero() {
		t.Fatal("task_error timestamp is zero")
	}

	// JSON 往返：run.Events 必须把三类事件解回具体类型并保住载荷（切片 23 的解码表）。
	encoded, err := json.Marshal(run.Events(events))
	if err != nil {
		t.Fatalf("json.Marshal(run.Events) error = %v", err)
	}
	var decoded run.Events
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(run.Events) error = %v", err)
	}
	if len(decoded) != len(events) {
		t.Fatalf("decoded %d events, want %d", len(decoded), len(events))
	}
	for i := range events {
		if fmt.Sprintf("%T", decoded[i]) != fmt.Sprintf("%T", events[i]) {
			t.Fatalf("decoded[%d] = %T, want the same concrete type as %T", i, decoded[i], events[i])
		}
		if decoded[i].EventType() != events[i].EventType() {
			t.Fatalf("decoded[%d].EventType() = %q, want %q", i, decoded[i].EventType(), events[i].EventType())
		}
	}
	decodedStarted, ok := decoded[0].(*run.NodeStartedEvent)
	if !ok || decodedStarted.Node != startedWeather.Node || decodedStarted.Input != startedWeather.Input || decodedStarted.Attempt != startedWeather.Attempt {
		t.Fatalf("decoded node_started lost payload: %+v", decoded[0])
	}
	decodedCompleted, ok := decoded[2].(*run.NodeCompletedEvent)
	if !ok || decodedCompleted.Output != completed.Output {
		t.Fatalf("decoded node_completed lost payload: %+v", decoded[2])
	}
	decodedError, ok := decoded[3].(*run.TaskErrorEvent)
	if !ok || decodedError.Message != taskError.Message {
		t.Fatalf("decoded task_error lost payload: %+v", decoded[3])
	}
}

// TestP4S31_TaskFailureClassifiedByTheHandlerError 钉 D5：成败分类只看 handler 交回的
// error。真失败 → task_error；成功但文本与失败文案一字不差 → node_completed。
func TestP4S31_TaskFailureClassifiedByTheHandlerError(t *testing.T) {
	t.Run("handler error becomes task_error", func(t *testing.T) {
		model := p4s31NewModel(
			p4s31ToolTurn(p4s31Call("call-boom", "boom", `{}`)),
			p4s31ContentTurn("done"),
		)
		ag := p4s31Agent(t, model, 0)

		result, err := ag.RunStreamMode(context.Background(), "explode", run.StreamTasks)
		if err != nil {
			t.Fatalf("RunStreamMode error = %v", err)
		}
		events, done := p4s31Collect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}
		p4s31AssertKinds(t, events, []string{run.EventTypeNodeStarted, run.EventTypeTaskError})
		taskError, ok := events[1].(*run.TaskErrorEvent)
		if !ok {
			t.Fatalf("event[1] = %T, want *run.TaskErrorEvent", events[1])
		}
		if !strings.Contains(taskError.Message, p4s31HandlerError.Error()) {
			t.Fatalf("task_error.Message = %q, want it to carry %q", taskError.Message, p4s31HandlerError)
		}
	})

	t.Run("success whose text looks like the failure stays completed", func(t *testing.T) {
		model := p4s31NewModel(
			p4s31ToolTurn(p4s31Call("call-lookalike", "lookalike", `{}`)),
			p4s31ContentTurn("done"),
		)
		ag := p4s31Agent(t, model, 0)

		result, err := ag.RunStreamMode(context.Background(), "pretend", run.StreamTasks)
		if err != nil {
			t.Fatalf("RunStreamMode error = %v", err)
		}
		events, done := p4s31Collect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}
		p4s31AssertKinds(t, events, []string{run.EventTypeNodeStarted, run.EventTypeNodeCompleted})
		completed, ok := events[1].(*run.NodeCompletedEvent)
		if !ok {
			t.Fatalf("event[1] = %T, want *run.NodeCompletedEvent", events[1])
		}
		if want := p4s31JSONText(t, p4s31LookalikeText); completed.Output != want {
			t.Fatalf("node_completed.Output = %#v, want the JSON form of the tool message %q", completed.Output, want)
		}
	})
}

// TestP4S31_ConcurrentBatchEmitsInDeclarationOrder 钉 D6：并发批次的确定性次序 ——
// started 恰 N 条且按调用声明序，结束事件在批次汇合后按结果序全部跟随其后；
// 跨回合不与本批交织。绑定场景命令以 -race -count=30 复跑。
func TestP4S31_ConcurrentBatchEmitsInDeclarationOrder(t *testing.T) {
	weather := p4s31Call("call-weather", "weather", `{"city":"Oslo"}`)
	boom := p4s31Call("call-boom", "boom", `{}`)
	echo := p4s31Call("call-echo", "echo", `{"text":"second batch"}`)
	lookalike := p4s31Call("call-lookalike", "lookalike", `{}`)

	model := p4s31NewModel(
		p4s31ToolTurn(weather, boom),
		p4s31ToolTurn(echo, lookalike),
		p4s31ContentTurn("tail"),
	)
	ag := p4s31Agent(t, model, 0)

	result, err := ag.RunStreamMode(context.Background(), "two batches", run.StreamTasks)
	if err != nil {
		t.Fatalf("RunStreamMode error = %v", err)
	}
	events, done := p4s31Collect(t, result)
	if done.Err != nil {
		t.Fatalf("stream finished with error = %v", done.Err)
	}

	wantKinds := []string{
		run.EventTypeNodeStarted, run.EventTypeNodeStarted, run.EventTypeNodeCompleted, run.EventTypeTaskError,
		run.EventTypeNodeStarted, run.EventTypeNodeStarted, run.EventTypeNodeCompleted, run.EventTypeNodeCompleted,
	}
	p4s31AssertKinds(t, events, wantKinds)

	wantNodes := []string{"weather", "boom", "weather", "boom", "echo", "lookalike", "echo", "lookalike"}
	for i, evt := range events {
		var node string
		switch e := evt.(type) {
		case *run.NodeStartedEvent:
			node = e.Node
		case *run.NodeCompletedEvent:
			node = e.Node
		case *run.TaskErrorEvent:
			node = e.Node
		default:
			t.Fatalf("event[%d] = %T, want one of the three task event types", i, evt)
		}
		if node != wantNodes[i] {
			t.Fatalf("event[%d] node = %q, want %q (whole node sequence %v)", i, node, wantNodes[i], wantNodes)
		}
	}
}

// TestP4S31_MessagesPathsUnchangedByTaskWiring 钉 D7（反向对照）：含工具调用的运行里
// RunStream、RunStreamMode(无模式)、RunStreamMode(Messages) 三者通道序列同型同内容，
// 且都不含任务事件。
func TestP4S31_MessagesPathsUnchangedByTaskWiring(t *testing.T) {
	turns := p4s31ToolThenContent()
	chunks := []string{"fin", "al"}

	cases := []struct {
		name         string
		entries      []run.StreamMode
		useRunStream bool
	}{
		{name: "RunStream", useRunStream: true},
		{name: "no modes", entries: nil},
		{name: "StreamMessages", entries: []run.StreamMode{run.StreamMessages}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			model := p4s31NewModel(turns...)
			ag := p4s31Agent(t, model, 0)
			ctx := context.Background()

			var (
				result *agent.RunStreamResult
				err    error
			)
			if testCase.useRunStream {
				result, err = ag.RunStream(ctx, "weather please")
			} else {
				result, err = ag.RunStreamMode(ctx, "weather please", testCase.entries...)
			}
			if err != nil {
				t.Fatalf("stream entry returned error = %v", err)
			}

			events, done := p4s31Collect(t, result)
			if done.Err != nil {
				t.Fatalf("stream finished with error = %v", done.Err)
			}
			p4s31AssertKinds(t, events, []string{run.EventTypeRunContent, run.EventTypeRunContent})
			for i, evt := range events {
				content, ok := evt.(*run.RunContentEvent)
				if !ok {
					t.Fatalf("event[%d] = %T, want *run.RunContentEvent (no task events on the messages path)", i, evt)
				}
				if content.Content != chunks[i] || content.Sequence != i {
					t.Fatalf("event[%d] = {content %q, sequence %d}, want {%q, %d}", i, content.Content, content.Sequence, chunks[i], i)
				}
			}
		})
	}
}

// TestP4S31_ModeSelectionIsAnOrderIndependentUnion 钉 D8：并集语义、参数次序无关、
// 重复模式不重发、无模式等于 Messages、Debug 不得被当成 Tasks 别名。
func TestP4S31_ModeSelectionIsAnOrderIndependentUnion(t *testing.T) {
	unionKinds := []string{
		run.EventTypeNodeStarted, run.EventTypeNodeStarted, run.EventTypeNodeCompleted, run.EventTypeTaskError,
		run.EventTypeRunContent, run.EventTypeRunContent,
	}

	collect := func(t *testing.T, modes ...run.StreamMode) []run.BaseRunOutputEvent {
		t.Helper()
		model := p4s31NewModel(p4s31ToolThenContent()...)
		ag := p4s31Agent(t, model, 0)
		result, err := ag.RunStreamMode(context.Background(), "weather please", modes...)
		if err != nil {
			t.Fatalf("RunStreamMode(%v) error = %v", modes, err)
		}
		events, done := p4s31Collect(t, result)
		if done.Err != nil {
			t.Fatalf("stream finished with error = %v", done.Err)
		}
		return events
	}

	t.Run("swapping the mode arguments changes nothing", func(t *testing.T) {
		p4s31AssertKinds(t, collect(t, run.StreamMessages, run.StreamTasks), unionKinds)
		p4s31AssertKinds(t, collect(t, run.StreamTasks, run.StreamMessages), unionKinds)
	})

	t.Run("repeating a mode does not re-emit", func(t *testing.T) {
		p4s31AssertKinds(t, collect(t, run.StreamTasks), p4s31Batch1Kinds)
		p4s31AssertKinds(t, collect(t, run.StreamTasks, run.StreamTasks), p4s31Batch1Kinds)

		contentKinds := []string{run.EventTypeRunContent, run.EventTypeRunContent}
		p4s31AssertKinds(t, collect(t, run.StreamMessages), contentKinds)
		p4s31AssertKinds(t, collect(t, run.StreamMessages, run.StreamMessages), contentKinds)
	})

	t.Run("no modes equals messages", func(t *testing.T) {
		p4s31AssertKinds(t, collect(t), []string{run.EventTypeRunContent, run.EventTypeRunContent})
	})

	t.Run("debug is not accepted as a tasks alias", func(t *testing.T) {
		model := p4s31NewModel(p4s31ToolThenContent()...)
		ag := p4s31Agent(t, model, 0)
		result, err := ag.RunStreamMode(context.Background(), "hi", run.StreamTasks, run.StreamDebug)
		if !errors.Is(err, run.ErrUnsupportedStreamMode) {
			t.Fatalf("Tasks+Debug: want ErrUnsupportedStreamMode, got %v", err)
		}
		if result != nil || model.streamCalls != 0 {
			t.Fatalf("Tasks+Debug: stream started (result=%v, calls=%d)", result, model.streamCalls)
		}
	})
}

// TestP4S31_CallsTruncatedByTheLimitStaySilent 钉 D9：被 ToolCallLimit 截断的调用不发声。
func TestP4S31_CallsTruncatedByTheLimitStaySilent(t *testing.T) {
	model := p4s31NewModel(
		p4s31ToolTurn(
			p4s31Call("call-weather", "weather", `{"city":"Oslo"}`),
			p4s31Call("call-boom", "boom", `{}`),
		),
	)
	ag := p4s31Agent(t, model, 1)

	result, err := ag.RunStreamMode(context.Background(), "weather please", run.StreamTasks)
	if err != nil {
		t.Fatalf("RunStreamMode error = %v", err)
	}
	events, done := p4s31Collect(t, result)
	if done.Err != nil {
		t.Fatalf("stream finished with error = %v", done.Err)
	}

	p4s31AssertKinds(t, events, []string{run.EventTypeNodeStarted, run.EventTypeNodeCompleted})
	for _, evt := range events {
		switch e := evt.(type) {
		case *run.NodeStartedEvent:
			if e.Node == "boom" {
				t.Fatal("a call truncated by ToolCallLimit emitted node_started")
			}
		case *run.NodeCompletedEvent:
			if e.Node == "boom" {
				t.Fatal("a call truncated by ToolCallLimit emitted node_completed")
			}
		case *run.TaskErrorEvent:
			t.Fatalf("a truncated call emitted task_error: %+v", e)
		}
	}
}
