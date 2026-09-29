package agent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/observability"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// 切片 32（P7 第 2 片，母约 G9 的 run/agent 级 span）的判据 D1–D8。
// 观察缝 = otel SDK 的 InMemoryExporter + SetTracerProvider（与 observability 自己的
// tracing_test.go 同法），断言只经公共入口 Agent.Run / Agent.RunStreamMode 与
// tracetest.SpanStub 的 Parent/Attributes/Start/End。
//
// 纪律：本文件禁止 t.Parallel()（全局 TracerProvider 是进程级状态），每个测试自己
// 保存并在 t.Cleanup 里还原 provider。

const (
	p7g9s2AgentName  = "labeled-agent"
	p7g9s2ModelName  = "p7g9s2-model"
	p7g9s2Provider   = "p7g9s2"
	p7g9s2ToolResult = "sunny"
)

// p7g9s2Turn 是一回合的脚本：同步路径吐什么内容、请求哪些工具调用、供多少 usage。
type p7g9s2Turn struct {
	content   string
	toolCalls []types.ToolCall
	usage     types.Usage
	chunks    []string
}

func p7g9s2ToolTurn(calls ...types.ToolCall) p7g9s2Turn {
	return p7g9s2Turn{toolCalls: calls}
}

func p7g9s2TextTurn(content string, usage types.Usage) p7g9s2Turn {
	return p7g9s2Turn{content: content, usage: usage}
}

func p7g9s2StreamTurn(chunks ...string) p7g9s2Turn {
	return p7g9s2Turn{chunks: chunks}
}

func p7g9s2StreamToolTurn(chunks []string, calls ...types.ToolCall) p7g9s2Turn {
	return p7g9s2Turn{chunks: chunks, toolCalls: calls}
}

// p7g9s2Model 是手写稳定模型：脚本外的回合一律报错（宁可响亮失败，也不静默缩短序列）。
type p7g9s2Model struct {
	models.BaseModel
	turns       []p7g9s2Turn
	turn        int
	invokeCalls int
	streamCalls int
}

func p7g9s2NewModel(turns ...p7g9s2Turn) *p7g9s2Model {
	return &p7g9s2Model{
		BaseModel: models.BaseModel{ID: p7g9s2ModelName, Provider: p7g9s2Provider},
		turns:     turns,
	}
}

func (m *p7g9s2Model) nextTurn() (p7g9s2Turn, error) {
	if m.turn >= len(m.turns) {
		return p7g9s2Turn{}, fmt.Errorf("p7g9s2 model asked for turn %d but the script has %d", m.turn+1, len(m.turns))
	}
	turn := m.turns[m.turn]
	m.turn++
	return turn, nil
}

func (m *p7g9s2Model) Invoke(_ context.Context, _ *models.InvokeRequest) (*types.ModelResponse, error) {
	m.invokeCalls++
	turn, err := m.nextTurn()
	if err != nil {
		return nil, err
	}
	return &types.ModelResponse{
		ID:        p7g9s2ModelName,
		Model:     p7g9s2ModelName,
		Content:   turn.content,
		ToolCalls: turn.toolCalls,
		Usage:     turn.usage,
	}, nil
}

func (m *p7g9s2Model) InvokeStream(_ context.Context, _ *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	m.streamCalls++
	turn, err := m.nextTurn()
	if err != nil {
		return nil, err
	}
	ch := make(chan types.ResponseChunk)
	go func() {
		defer close(ch)
		for _, chunk := range turn.chunks {
			ch <- types.ResponseChunk{Content: chunk}
		}
		if len(turn.toolCalls) > 0 {
			ch <- types.ResponseChunk{ToolCalls: turn.toolCalls}
		}
		if turn.content != "" {
			ch <- types.ResponseChunk{Content: turn.content}
		}
	}()
	return ch, nil
}

func p7g9s2Call(id, name, arguments string) types.ToolCall {
	return types.ToolCall{
		ID:       id,
		Type:     "function",
		Function: types.ToolCallFunction{Name: name, Arguments: arguments},
	}
}

func p7g9s2Toolkit() toolkit.Toolkit {
	tk := toolkit.NewBaseToolkit("p7g9s2")
	tk.RegisterFunction(&toolkit.Function{
		Name:        "weather",
		Description: "reports the weather",
		Parameters:  map[string]toolkit.Parameter{"city": {Type: "string"}},
		Handler: func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
			return p7g9s2ToolResult, nil
		},
	})
	return tk
}

func p7g9s2Agent(t *testing.T, model models.Model) *agent.Agent {
	t.Helper()
	ag, err := agent.New(agent.Config{
		Name:     p7g9s2AgentName,
		Model:    model,
		Toolkits: []toolkit.Toolkit{p7g9s2Toolkit()},
	})
	if err != nil {
		t.Fatalf("agent.New() error = %v", err)
	}
	return ag
}

// p7g9s2Tracer 安装 SDK tracer 与内存 exporter，并保存/还原进程级 provider。
func p7g9s2Tracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	previous := otel.GetTracerProvider()
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		otel.SetTracerProvider(previous)
	})
	return exporter
}

func p7g9s2ByName(spans tracetest.SpanStubs, name string) []tracetest.SpanStub {
	var out []tracetest.SpanStub
	for _, s := range spans {
		if s.Name == name {
			out = append(out, s)
		}
	}
	return out
}

func p7g9s2Attr(t *testing.T, span tracetest.SpanStub, key string) attribute.Value {
	t.Helper()
	for _, kv := range span.Attributes {
		if kv.Key == attribute.Key(key) {
			return kv.Value
		}
	}
	t.Fatalf("attribute %s not found on span %s (attrs %v)", key, span.Name, span.Attributes)
	return attribute.Value{}
}

// p7g9s2ChildOf 断言 child 的父 span 上下文恰是 parent：既要求 Parent 有效，也要求
// SpanID/TraceID 同时相等（「同 trace 但不同父」的实现必须判红）。
func p7g9s2ChildOf(t *testing.T, parent, child tracetest.SpanStub, kind string) {
	t.Helper()
	if !child.Parent.IsValid() {
		t.Fatalf("%s span has an invalid/zero Parent: %v (invoke_agent span id %s)", kind, child.Parent, parent.SpanContext.SpanID())
	}
	if child.Parent.SpanID() != parent.SpanContext.SpanID() {
		t.Errorf("%s parent span id = %s, want invoke_agent id %s", kind, child.Parent.SpanID(), parent.SpanContext.SpanID())
	}
	if child.SpanContext.TraceID() != parent.SpanContext.TraceID() {
		t.Errorf("%s trace id = %s, want invoke_agent trace id %s", kind, child.SpanContext.TraceID(), parent.SpanContext.TraceID())
	}
}

// p7g9s2Contains 断言父 span 的时钟区间包住子 span（父先开后关）。
// 这是把「在驱动点之外 End」这一设计选择证伪的判据：父提前结束时子还没结束。
func p7g9s2Contains(t *testing.T, parent, child tracetest.SpanStub, kind string) {
	t.Helper()
	if parent.EndTime.IsZero() {
		t.Fatalf("invoke_agent span was never ended (End is zero time) while a %s child ended at %s", kind, child.EndTime)
	}
	if child.EndTime.After(parent.EndTime) {
		t.Errorf("%s child ends at %v after invoke_agent ends at %v: the parent closed before its work", kind, child.EndTime, parent.EndTime)
	}
	if child.StartTime.Before(parent.StartTime) {
		t.Errorf("%s child starts at %v before invoke_agent starts at %v", kind, child.StartTime, parent.StartTime)
	}
}

// expectAgentSpan 取恰一条 invoke_agent span（多条/零条都判红）。
func expectAgentSpan(t *testing.T, spans tracetest.SpanStubs) tracetest.SpanStub {
	t.Helper()
	agents := p7g9s2ByName(spans, observability.SpanAgentRun)
	if len(agents) != 1 {
		t.Fatalf("invoke_agent spans = %d, want exactly 1 (exported names %v)", len(agents), p7g9s2Names(spans))
	}
	return agents[0]
}

func p7g9s2Names(spans tracetest.SpanStubs) []string {
	names := make([]string, 0, len(spans))
	for _, s := range spans {
		names = append(names, s.Name)
	}
	return names
}

// D1：Agent.Run 在驱动内核的那一段开恰一条 invoke_agent span，属性三键齐全且取自
// Agent.Name（不是 Agent.ID）与运行实际的 runID。
func TestP7G9S2_RunEmitsOneInvokeAgentSpanWithIdentityAttributes(t *testing.T) {
	exporter := p7g9s2Tracer(t)
	if p7g9s2AgentName == "agent-"+p7g9s2ModelName {
		t.Fatal("fixture stopped discriminating the agent label from its generated id")
	}

	model := p7g9s2NewModel(p7g9s2TextTurn("final answer", types.Usage{PromptTokens: 11, CompletionTokens: 7, TotalTokens: 18}))
	out, err := p7g9s2Agent(t, model).Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	agentSpan := expectAgentSpan(t, exporter.GetSpans())
	if got := p7g9s2Attr(t, agentSpan, observability.AttrAgentName).AsString(); got != p7g9s2AgentName {
		t.Errorf("agent.name = %q, want %q (the agent label, not its id %q)", got, p7g9s2AgentName, out.RunID)
	}
	if got, want := p7g9s2Attr(t, agentSpan, observability.AttrAgentRunID).AsString(), out.RunID; got != want {
		t.Errorf("agent.run_id = %q, want the run's own %q", got, want)
	}
	if got := p7g9s2Attr(t, agentSpan, observability.AttrGenAISystem).AsString(); got != "hno" {
		t.Errorf("gen_ai.system = %q, want \"hno\"", got)
	}
	if agentSpan.EndTime.IsZero() {
		t.Error("invoke_agent span was not ended")
	}
}

// D2：同步路径的层级——chat 与 execute_tool 都是 invoke_agent 的子 span。
func TestP7G9S2_SyncPathParentsChatAndToolSpans(t *testing.T) {
	exporter := p7g9s2Tracer(t)

	model := p7g9s2NewModel(
		p7g9s2ToolTurn(p7g9s2Call("call-1", "weather", `{"city":"Kyoto"}`)),
		p7g9s2TextTurn("done", types.Usage{}),
	)
	if _, err := p7g9s2Agent(t, model).Run(context.Background(), "hello"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	spans := exporter.GetSpans()
	agentSpan := expectAgentSpan(t, spans)

	chats := p7g9s2ByName(spans, observability.SpanChat)
	if len(chats) != 2 {
		t.Fatalf("chat spans = %d, want exactly 2 (two model turns), exported names %v", len(chats), p7g9s2Names(spans))
	}
	tools := p7g9s2ByName(spans, observability.SpanToolExecute)
	if len(tools) != 1 {
		t.Fatalf("execute_tool spans = %d, want exactly 1, exported names %v", len(tools), p7g9s2Names(spans))
	}
	for i, chat := range chats {
		p7g9s2ChildOf(t, agentSpan, chat, fmt.Sprintf("chat[%d]", i))
	}
	p7g9s2ChildOf(t, agentSpan, tools[0], "execute_tool")
	if got := p7g9s2Attr(t, tools[0], observability.AttrGenAIToolName).AsString(); got != "weather" {
		t.Errorf("gen_ai.tool.name = %q, want \"weather\"", got)
	}
	if got := p7g9s2Attr(t, chats[0], observability.AttrGenAIProvider).AsString(); got != p7g9s2Provider {
		t.Errorf("gen_ai.provider.name = %q, want %q", got, p7g9s2Provider)
	}
}

// D3：流式（Messages 模式）跨协程边界仍然认父。
func TestP7G9S2_StreamPathParentsChatSpansAcrossTheGoroutine(t *testing.T) {
	exporter := p7g9s2Tracer(t)

	model := p7g9s2NewModel(
		p7g9s2StreamToolTurn([]string{"par", "is"}, p7g9s2Call("call-1", "weather", `{"city":"Kyoto"}`)),
		p7g9s2StreamTurn("sunny today"),
	)
	result, err := p7g9s2Agent(t, model).RunStreamMode(context.Background(), "hello", run.StreamMessages)
	if err != nil {
		t.Fatalf("RunStreamMode() error = %v", err)
	}
	for range result.Events {
	}
	done := <-result.Done
	if done.Err != nil {
		t.Fatalf("stream done err = %v", done.Err)
	}

	spans := exporter.GetSpans()
	agentSpan := expectAgentSpan(t, spans)
	chats := p7g9s2ByName(spans, observability.SpanChat)
	if len(chats) != 2 {
		t.Fatalf("chat spans = %d, want exactly 2, exported names %v", len(chats), p7g9s2Names(spans))
	}
	tools := p7g9s2ByName(spans, observability.SpanToolExecute)
	if len(tools) != 1 {
		t.Fatalf("execute_tool spans = %d, want exactly 1, exported names %v", len(tools), p7g9s2Names(spans))
	}
	for i, chat := range chats {
		p7g9s2ChildOf(t, agentSpan, chat, fmt.Sprintf("chat[%d]", i))
		p7g9s2Contains(t, agentSpan, chat, fmt.Sprintf("chat[%d]", i))
	}
	p7g9s2ChildOf(t, agentSpan, tools[0], "execute_tool")
	p7g9s2Contains(t, agentSpan, tools[0], "execute_tool")
	if got, want := p7g9s2Attr(t, agentSpan, observability.AttrAgentRunID).AsString(), done.Output.RunID; got != want {
		t.Errorf("agent.run_id = %q, want the streamed run's %q", got, want)
	}
}

// D4：StreamTasks 路径（taskToolExecutor → runToolBatch → executeOneTool）的
// execute_tool 也认 invoke_agent 为父。
func TestP7G9S2_TasksStreamToolSpanIsAChildToo(t *testing.T) {
	exporter := p7g9s2Tracer(t)

	model := p7g9s2NewModel(
		p7g9s2StreamToolTurn(nil, p7g9s2Call("call-1", "weather", `{"city":"Kyoto"}`)),
		p7g9s2StreamTurn("clear"),
	)
	result, err := p7g9s2Agent(t, model).RunStreamMode(context.Background(), "hello", run.StreamTasks)
	if err != nil {
		t.Fatalf("RunStreamMode(Tasks) error = %v", err)
	}
	events, done := p7g9s2Collect(t, result)
	if done.Err != nil {
		t.Fatalf("stream done err = %v", done.Err)
	}
	if len(events) == 0 {
		t.Fatal("Tasks stream produced no events; the fixture stopped exercising the producer")
	}

	spans := exporter.GetSpans()
	agentSpan := expectAgentSpan(t, spans)
	tools := p7g9s2ByName(spans, observability.SpanToolExecute)
	if len(tools) != 1 {
		t.Fatalf("execute_tool spans = %d, want exactly 1, exported names %v", len(tools), p7g9s2Names(spans))
	}
	p7g9s2ChildOf(t, agentSpan, tools[0], "execute_tool")
}

func p7g9s2Collect(t *testing.T, result *agent.RunStreamResult) ([]run.BaseRunOutputEvent, agent.RunStreamDone) {
	t.Helper()
	var events []run.BaseRunOutputEvent
	for evt := range result.Events {
		events = append(events, evt)
	}
	return events, <-result.Done
}

// D5：父 span 的时钟区间包住每一个子 span（同步 + 流式两种驱动形状）。
// 「在驱动点之外 End」的第一段实现形状（入口函数 defer）会被本行判红。
func TestP7G9S2_AgentSpanContainsEveryChild(t *testing.T) {
	t.Run("sync", func(t *testing.T) {
		exporter := p7g9s2Tracer(t)
		model := p7g9s2NewModel(
			p7g9s2ToolTurn(p7g9s2Call("call-1", "weather", `{"city":"Kyoto"}`)),
			p7g9s2TextTurn("done", types.Usage{}),
		)
		if _, err := p7g9s2Agent(t, model).Run(context.Background(), "hello"); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		spans := exporter.GetSpans()
		agentSpan := expectAgentSpan(t, spans)
		for _, child := range spans {
			if child.Name == observability.SpanAgentRun {
				continue
			}
			p7g9s2Contains(t, agentSpan, child, child.Name)
		}
	})

	t.Run("stream", func(t *testing.T) {
		exporter := p7g9s2Tracer(t)
		model := p7g9s2NewModel(p7g9s2StreamTurn("a", "b"))
		result, err := p7g9s2Agent(t, model).RunStream(context.Background(), "hello")
		if err != nil {
			t.Fatalf("RunStream() error = %v", err)
		}
		_, done := p7g9s2Collect(t, result)
		if done.Err != nil {
			t.Fatalf("stream done err = %v", done.Err)
		}
		spans := exporter.GetSpans()
		agentSpan := expectAgentSpan(t, spans)
		for _, child := range spans {
			if child.Name == observability.SpanAgentRun {
				continue
			}
			p7g9s2Contains(t, agentSpan, child, child.Name)
		}
	})
}

// D6：未接线模式的 fail-closed 路径不开 span。同一测试体内先做正对照（Messages 模式
// 确实产出 1 条 invoke_agent），否则「计数为 0」会在观察缝本身失效时空转成立。
func TestP7G9S2_UnwiredStreamModeOpensNoSpan(t *testing.T) {
	exporter := p7g9s2Tracer(t)

	positive, err := p7g9s2Agent(t, p7g9s2NewModel(p7g9s2StreamTurn("hi"))).RunStreamMode(context.Background(), "hello", run.StreamMessages)
	if err != nil {
		t.Fatalf("wired mode RunStreamMode() error = %v", err)
	}
	_, done := p7g9s2Collect(t, positive)
	if done.Err != nil {
		t.Fatalf("wired mode stream err = %v", done.Err)
	}
	before := len(p7g9s2ByName(exporter.GetSpans(), observability.SpanAgentRun))
	if before != 1 {
		t.Fatalf("positive control: invoke_agent spans = %d, want 1 before the fail-closed call", before)
	}

	negative, err := p7g9s2Agent(t, p7g9s2NewModel(p7g9s2StreamTurn("hi"))).RunStreamMode(context.Background(), "hello", run.StreamMode(99))
	if err == nil {
		t.Fatal("RunStreamMode(unknown mode 99) error = nil, want the fail-closed error")
	}
	if negative != nil {
		t.Errorf("RunStreamMode(unknown mode 99) result = %v, want nil", negative)
	}
	after := p7g9s2ByName(exporter.GetSpans(), observability.SpanAgentRun)
	if len(after) != before {
		t.Errorf("invoke_agent spans after the fail-closed call = %d, want %d (unchanged)", len(after), before)
	}
	if got := len(p7g9s2ByName(exporter.GetSpans(), observability.SpanChat)); got != 1 {
		t.Errorf("chat spans = %d, want 1 (only the positive control's turn ran), exported %v", got, p7g9s2Names(exporter.GetSpans()))
	}
}

// D7：放弃消费（既不排空 Events 也不读 Done、且未取消）时 span 不结束；取消之后驱动
// 返回、span 结束。两半各钉一个绝对读数，取消半是实现前必红的行为半。
func TestP7G9S2_SpanLifetimeFollowsTheDrive(t *testing.T) {
	t.Run("abandoned_without_cancel_yields_no_ended_span", func(t *testing.T) {
		exporter := p7g9s2Tracer(t)
		model := p7g9s2NewModel(p7g9s2StreamTurn("one", "two"))
		result, err := p7g9s2Agent(t, model).RunStream(context.Background(), "hello")
		if err != nil {
			t.Fatalf("RunStream() error = %v", err)
		}
		<-result.Events // 只取一条：驱动协程此后卡在第二条发射上
		if got := len(p7g9s2ByName(exporter.GetSpans(), observability.SpanAgentRun)); got != 0 {
			t.Errorf("invoke_agent spans while the drive is blocked on an undrained channel = %d, want 0", got)
		}
	})

	t.Run("cancel_lets_the_drive_end_the_span", func(t *testing.T) {
		exporter := p7g9s2Tracer(t)
		ctx, cancel := context.WithCancel(context.Background())
		model := p7g9s2NewModel(p7g9s2StreamTurn("one", "two"))
		result, err := p7g9s2Agent(t, model).RunStream(ctx, "hello")
		if err != nil {
			t.Fatalf("RunStream() error = %v", err)
		}
		<-result.Events
		cancel()

		done := <-result.Done
		if done.Err == nil {
			t.Fatal("cancelled stream done err = nil, want the cancellation error")
		}
		spans := exporter.GetSpans()
		agentSpan := expectAgentSpan(t, spans)
		if agentSpan.EndTime.IsZero() {
			t.Error("invoke_agent span End = zero, want it ended after the drive returned")
		}
		for _, child := range p7g9s2ByName(spans, observability.SpanChat) {
			p7g9s2ChildOf(t, agentSpan, child, "chat")
		}
	})
}

var p7g9s2RunIDPattern = regexp.MustCompile("(?:run-)?[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")
var p7g9s2CreatedPattern = regexp.MustCompile(`"created_at":\s*[0-9]+`)

// p7g9s2WireJSON 把「运行时刻盖章」的三个 volatile 面（run_id UUID、起止时间戳、事件
// created_at）掩成定形 token，其余字节逐字节保留。这三个面由 P0B/切片 23 的既有锚钉，
// 本行要问的是「缝本身改不改字节」，两侧用同一个掩码 ⇒ 任何内容/次序/消息改动仍然可见。
func p7g9s2WireJSON(t *testing.T, out *agent.RunOutput) string {
	t.Helper()
	out.RunID = ""
	out.StartedAt = time.Time{}
	out.CompletedAt = time.Time{}
	encoded, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	text := p7g9s2RunIDPattern.ReplaceAllString(string(encoded), "<RUN_ID>")
	return p7g9s2CreatedPattern.ReplaceAllString(text, "\"created_at\":<SECONDS>")
}

// p7g9s2TracerInstalled 用 Start+IsRecording 探针判断「全局是否装了真 tracer」。
// 注意 observability.IsRecording(ctx) 量的是「ctx 里当前那条 span 是否记录」，实测
// 装了 SDK 的空 ctx 上它同样返回 false（/tmp 探针 2026-09-28），所以它不能当这个前置守卫。
func p7g9s2TracerInstalled() bool {
	_, span := observability.Tracer().Start(context.Background(), "p7g9s2-probe")
	defer span.End()
	return span.IsRecording()
}

// D8：未配置任何 SDK 时，本缝零记录、零行为改变：同一个脚本在 noop tracer 下与在 SDK
// 下交出的 RunOutput JSON 逐字节相同，且 span.End() 不 panic。
func TestP7G9S2_NoopTracerKeepsWireBytesIdentical(t *testing.T) {
	if p7g9s2TracerInstalled() {
		t.Fatal("a real tracer is already installed globally; the noop half of this row cannot be observed")
	}

	syncJSON := func() string {
		model := p7g9s2NewModel(
			p7g9s2ToolTurn(p7g9s2Call("call-1", "weather", `{"city":"Kyoto"}`)),
			p7g9s2TextTurn("done", types.Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3}),
		)
		out, err := p7g9s2Agent(t, model).Run(context.Background(), "hello")
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		return p7g9s2WireJSON(t, out)
	}

	before := syncJSON()

	exporter := p7g9s2Tracer(t)
	after := syncJSON()

	if after != before {
		t.Errorf("RunOutput JSON changed once a tracer was configured:\n noop: %s\n sdk : %s", before, after)
	}
	if got := len(p7g9s2ByName(exporter.GetSpans(), observability.SpanAgentRun)); got != 1 {
		t.Errorf("invoke_agent spans under a real SDK tracer = %d, want 1: the observation seam for this slice is dead", got)
	}
}
