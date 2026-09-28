package agent_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// 切片 23（母约 §5 G4：StreamMode 协议层）agent 侧判据：D8–D10。
// 沿用 agent 包既有稳定流式 stub 的形状（确定性分块输出）；事件序列经公共
// channel 观察，时间戳只参与"非零"判断、不比较数值（契约 forbiddenObservations）。

// p4g4StreamModel 是本文件的确定性流式模型 stub：每次 InvokeStream 吐同一组分块。
type p4g4StreamModel struct {
	models.BaseModel
	chunks      []string
	streamCalls int
}

func (m *p4g4StreamModel) Invoke(_ context.Context, _ *models.InvokeRequest) (*types.ModelResponse, error) {
	return &types.ModelResponse{ID: "p4g4", Content: strings.Join(m.chunks, ""), Model: "p4g4"}, nil
}

func (m *p4g4StreamModel) InvokeStream(_ context.Context, _ *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	m.streamCalls++
	ch := make(chan types.ResponseChunk)
	go func() {
		defer close(ch)
		for _, chunk := range m.chunks {
			ch <- types.ResponseChunk{Content: chunk}
		}
	}()
	return ch, nil
}

func p4g4NewStreamAgent(t *testing.T, chunks []string) (*agent.Agent, *p4g4StreamModel) {
	t.Helper()
	model := &p4g4StreamModel{BaseModel: models.BaseModel{ID: "p4g4", Provider: "p4g4"}, chunks: chunks}
	ag, err := agent.New(agent.Config{Name: "p4g4-stream-agent", Model: model})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return ag, model
}

func p4g4Collect(t *testing.T, result *agent.RunStreamResult) ([]run.BaseRunOutputEvent, agent.RunStreamDone) {
	t.Helper()
	var events []run.BaseRunOutputEvent
	for evt := range result.Events {
		events = append(events, evt)
	}
	done := <-result.Done
	return events, done
}

// p4g4AssertContentSequence 断言事件序列全部为 RunContentEvent 且内容依次相同；
// 时间戳只断非零（一致性按"同型同内容"断，不比较时间戳数值）。
func p4g4AssertContentSequence(t *testing.T, events []run.BaseRunOutputEvent, wantChunks []string) {
	t.Helper()
	if len(events) != len(wantChunks) {
		t.Fatalf("got %d events, want %d", len(events), len(wantChunks))
	}
	for i, evt := range events {
		contentEvt, ok := evt.(*run.RunContentEvent)
		if !ok {
			t.Fatalf("event[%d] = %T, want *run.RunContentEvent", i, evt)
		}
		if contentEvt.Content != wantChunks[i] {
			t.Fatalf("event[%d].Content = %q, want %q", i, contentEvt.Content, wantChunks[i])
		}
		if contentEvt.EventType() != run.EventTypeRunContent {
			t.Fatalf("event[%d].EventType() = %q, want %q", i, contentEvt.EventType(), run.EventTypeRunContent)
		}
		if contentEvt.Sequence != i {
			t.Fatalf("event[%d].Sequence = %d, want %d", i, contentEvt.Sequence, i)
		}
		if contentEvt.Timestamp().IsZero() {
			t.Fatalf("event[%d] has zero timestamp", i)
		}
	}
}

// TestP4G4_RunStreamModeMessagesMatchesRunStream 钉 D8：RunStreamMode(ctx, in,
// StreamMessages) 产出与 RunStream 完全一致的事件序列（同一稳定 stub、同一输入）。
func TestP4G4_RunStreamModeMessagesMatchesRunStream(t *testing.T) {
	chunks := []string{"Hello", " ", "stream"}

	ag1, _ := p4g4NewStreamAgent(t, chunks)
	result1, err := ag1.RunStreamMode(context.Background(), "hi", run.StreamMessages)
	if err != nil {
		t.Fatalf("RunStreamMode(StreamMessages) error = %v", err)
	}
	events1, done1 := p4g4Collect(t, result1)

	ag2, _ := p4g4NewStreamAgent(t, chunks)
	result2, err := ag2.RunStream(context.Background(), "hi")
	if err != nil {
		t.Fatalf("RunStream error = %v", err)
	}
	events2, done2 := p4g4Collect(t, result2)

	p4g4AssertContentSequence(t, events1, chunks)
	if len(events1) != len(events2) {
		t.Fatalf("event sequence lengths diverge: %d vs %d", len(events1), len(events2))
	}
	for i := range events1 {
		e1, ok1 := events1[i].(*run.RunContentEvent)
		e2, ok2 := events2[i].(*run.RunContentEvent)
		if !ok1 || !ok2 {
			t.Fatalf("event[%d] types diverge: %T vs %T", i, events1[i], events2[i])
		}
		if e1.Content != e2.Content || e1.Sequence != e2.Sequence {
			t.Fatalf("event[%d] content diverges: %+v vs %+v", i, e1, e2)
		}
	}
	if done1.Err != nil || done2.Err != nil {
		t.Fatalf("done errors: %v / %v", done1.Err, done2.Err)
	}
	if done1.Output == nil || done2.Output == nil {
		t.Fatalf("done outputs nil: %v / %v", done1.Output, done2.Output)
	}
	if done1.Output.Content != done2.Output.Content {
		t.Fatalf("final content diverges: %q vs %q", done1.Output.Content, done2.Output.Content)
	}
}

// TestP4G4_RunStreamEquivalentToRunStreamModeMessages 钉 D9（包装关系）：
// RunStream(ctx, in) 等价于 RunStreamMode(ctx, in, StreamMessages)。
func TestP4G4_RunStreamEquivalentToRunStreamModeMessages(t *testing.T) {
	chunks := []string{"wrap", "-", "me"}

	ag1, _ := p4g4NewStreamAgent(t, chunks)
	wrapped, err := ag1.RunStream(context.Background(), "hi")
	if err != nil {
		t.Fatalf("RunStream error = %v", err)
	}
	events1, done1 := p4g4Collect(t, wrapped)

	ag2, _ := p4g4NewStreamAgent(t, chunks)
	direct, err := ag2.RunStreamMode(context.Background(), "hi", run.StreamMessages)
	if err != nil {
		t.Fatalf("RunStreamMode(StreamMessages) error = %v", err)
	}
	events2, done2 := p4g4Collect(t, direct)

	p4g4AssertContentSequence(t, events1, chunks)
	if len(events1) != len(events2) {
		t.Fatalf("event sequence lengths diverge: %d vs %d", len(events1), len(events2))
	}
	for i := range events1 {
		e1, ok1 := events1[i].(*run.RunContentEvent)
		e2, ok2 := events2[i].(*run.RunContentEvent)
		if !ok1 || !ok2 {
			t.Fatalf("event[%d] types diverge: %T vs %T", i, events1[i], events2[i])
		}
		if e1.Content != e2.Content || e1.Sequence != e2.Sequence {
			t.Fatalf("event[%d] content diverges: %+v vs %+v", i, e1, e2)
		}
	}
	if done1.Err != nil || done2.Err != nil {
		t.Fatalf("done errors: %v / %v", done1.Err, done2.Err)
	}
	if done1.Output == nil || done2.Output == nil || done1.Output.Content != done2.Output.Content {
		t.Fatalf("final content diverges: %v vs %v", done1.Output, done2.Output)
	}
}

// TestP4G4_RunStreamModeUnsupportedModesFailClosed 钉 D10：其余任一模式 → 返回可
// errors.Is 到 ErrUnsupportedStreamMode 的错误且不启动流（stub 零调用、无结果对象）；
// 无模式默认 StreamMessages（与 RunStream 同形）；与 Messages 混入其它模式同样 fail-closed。
// Tasks 的生产者已随切片 31 落地（母约 §5 的解禁），其正向判据搬家到 stream_tasks_test.go。
func TestP4G4_RunStreamModeUnsupportedModesFailClosed(t *testing.T) {
	unsupported := []run.StreamMode{
		run.StreamValues,
		run.StreamUpdates,
		run.StreamCheckpoints,
		run.StreamDebug,
		run.StreamCustom,
	}
	for _, mode := range unsupported {
		t.Run(fmt.Sprintf("mode-%d", int(mode)), func(t *testing.T) {
			ag, model := p4g4NewStreamAgent(t, []string{"should", "not", "stream"})
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

	t.Run("mixed with messages still fails closed", func(t *testing.T) {
		ag, model := p4g4NewStreamAgent(t, []string{"no"})
		result, err := ag.RunStreamMode(context.Background(), "hi", run.StreamMessages, run.StreamValues)
		if !errors.Is(err, run.ErrUnsupportedStreamMode) {
			t.Fatalf("mixed modes: want ErrUnsupportedStreamMode, got %v", err)
		}
		if result != nil || model.streamCalls != 0 {
			t.Fatalf("mixed modes: stream started (result=%v, calls=%d)", result, model.streamCalls)
		}
	})

	t.Run("no modes defaults to messages semantics", func(t *testing.T) {
		chunks := []string{"default", "-mode"}
		ag1, _ := p4g4NewStreamAgent(t, chunks)
		result1, err := ag1.RunStreamMode(context.Background(), "hi")
		if err != nil {
			t.Fatalf("RunStreamMode() with no modes error = %v", err)
		}
		events1, done1 := p4g4Collect(t, result1)
		p4g4AssertContentSequence(t, events1, chunks)
		if done1.Err != nil || done1.Output == nil || done1.Output.Content != strings.Join(chunks, "") {
			t.Fatalf("no-mode default diverged from messages semantics: %+v", done1)
		}
	})
}
