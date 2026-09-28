package run_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/run"
)

// 切片 23（母约 §5 G4：StreamMode 七模式统一流式协议的协议层）。
// 契约 docs/design/v3-test-scope-p1-graph-slice23.json，本文件钉 D1–D6/D11/D12；
// 全部断言只走公共面（导出常量、构造函数、Events 编解码、GenericRunEvent 兜底）。

// TestP4G4_StreamModeConstantsIotaOrder 钉 D1：七常量存在且 iota 序按母约草图
// （Values < Updates < Messages < Tasks < Checkpoints < Debug < Custom）。
func TestP4G4_StreamModeConstantsIotaOrder(t *testing.T) {
	order := []struct {
		name string
		mode run.StreamMode
	}{
		{"StreamValues", run.StreamValues},
		{"StreamUpdates", run.StreamUpdates},
		{"StreamMessages", run.StreamMessages},
		{"StreamTasks", run.StreamTasks},
		{"StreamCheckpoints", run.StreamCheckpoints},
		{"StreamDebug", run.StreamDebug},
		{"StreamCustom", run.StreamCustom},
	}
	seen := map[run.StreamMode]bool{}
	for i, o := range order {
		if int(o.mode) != i {
			t.Fatalf("%s ordinal = %d, want %d (mother-sketch iota order)", o.name, int(o.mode), i)
		}
		if seen[o.mode] {
			t.Fatalf("%s duplicates an earlier constant", o.name)
		}
		seen[o.mode] = true
	}
}

// TestP4G4_NewEventConstructorsStampKindAndTimestamp 钉 D2：六个 New* 构造函数
// 盖章种类与时间戳（EventType() 各归其名、Timestamp() 非零），载荷原样保留。
func TestP4G4_NewEventConstructorsStampKindAndTimestamp(t *testing.T) {
	check := func(t *testing.T, evt run.BaseRunOutputEvent, wantKind string) {
		t.Helper()
		if evt.EventType() != wantKind {
			t.Fatalf("EventType() = %q, want %q", evt.EventType(), wantKind)
		}
		if evt.Timestamp().IsZero() || evt.Timestamp().Unix() <= 0 {
			t.Fatalf("Timestamp() = %v, want a stamped non-zero time", evt.Timestamp())
		}
	}

	nodeStarted := run.NewNodeStartedEvent("run-1", "node-a", 1, map[string]any{"in": "x"})
	check(t, nodeStarted, run.EventTypeNodeStarted)
	if nodeStarted.RunID != "run-1" || nodeStarted.Node != "node-a" || nodeStarted.Attempt != 1 {
		t.Fatalf("NodeStartedEvent payload not preserved: %+v", nodeStarted)
	}
	if nodeStarted.Input == nil {
		t.Fatalf("NodeStartedEvent input not preserved")
	}

	nodeCompleted := run.NewNodeCompletedEvent("run-1", "node-a", 1, "out")
	check(t, nodeCompleted, run.EventTypeNodeCompleted)
	if nodeCompleted.Output != "out" {
		t.Fatalf("NodeCompletedEvent output not preserved: %+v", nodeCompleted)
	}

	taskError := run.NewTaskErrorEvent("run-1", "node-b", "boom")
	check(t, taskError, run.EventTypeTaskError)
	if taskError.Message != "boom" {
		t.Fatalf("TaskErrorEvent message not preserved: %+v", taskError)
	}

	checkpoint := run.NewCheckpointEvent("run-1", "pre-loop", map[string]any{"step": 1})
	check(t, checkpoint, run.EventTypeCheckpoint)
	if checkpoint.Label != "pre-loop" || checkpoint.Payload == nil {
		t.Fatalf("CheckpointEvent payload not preserved: %+v", checkpoint)
	}

	stateUpdate := run.NewStateUpdateEvent("run-1", "node-a", map[string]any{"k": "v"})
	check(t, stateUpdate, run.EventTypeStateUpdate)
	if stateUpdate.Patch == nil {
		t.Fatalf("StateUpdateEvent patch not preserved: %+v", stateUpdate)
	}

	custom := run.NewCustomEvent("run-1", "progress", 42)
	check(t, custom, run.EventTypeCustom)
	if custom.Subtype != "progress" || custom.Data != 42 {
		t.Fatalf("CustomEvent payload not preserved: %+v", custom)
	}
}

// TestP4G4_LegacyEventJSONGoldenShape 钉 D3（反向对照，实现前后都必须绿）：
// 旧事件 JSON 形状逐字段不变 —— run_content 的 M1 golden 字段集
// {event,created_at,run_id,agent_id,sequence,role,content} 与 run_completed 的
// {event,created_at,run_id,agent_id,status,content}。
func TestP4G4_LegacyEventJSONGoldenShape(t *testing.T) {
	contentData, err := json.Marshal(run.Events{
		run.NewRunContentEvent("run-1", "agent-1", "assistant", "hello", 7),
	})
	if err != nil {
		t.Fatalf("marshal content event failed: %v", err)
	}
	var contentRows []map[string]any
	if err := json.Unmarshal(contentData, &contentRows); err != nil {
		t.Fatalf("decode content event failed: %v", err)
	}
	if len(contentRows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(contentRows))
	}
	wantFields := []string{"event", "created_at", "run_id", "agent_id", "sequence", "role", "content"}
	assertExactFields(t, contentRows[0], wantFields)
	assertStringField(t, contentRows[0], "event", "run_content")
	assertStringField(t, contentRows[0], "run_id", "run-1")
	assertStringField(t, contentRows[0], "agent_id", "agent-1")
	assertStringField(t, contentRows[0], "role", "assistant")
	assertStringField(t, contentRows[0], "content", "hello")
	if seq, ok := contentRows[0]["sequence"].(float64); !ok || int(seq) != 7 {
		t.Fatalf("sequence = %v, want 7", contentRows[0]["sequence"])
	}
	if at, ok := contentRows[0]["created_at"].(float64); !ok || at <= 0 {
		t.Fatalf("created_at = %v, want positive unix seconds", contentRows[0]["created_at"])
	}

	completedData, err := json.Marshal(run.Events{
		run.NewRunCompletedEvent("run-1", "agent-1", "", "completed", "done"),
	})
	if err != nil {
		t.Fatalf("marshal completed event failed: %v", err)
	}
	var completedRows []map[string]any
	if err := json.Unmarshal(completedData, &completedRows); err != nil {
		t.Fatalf("decode completed event failed: %v", err)
	}
	assertExactFields(t, completedRows[0], []string{"event", "created_at", "run_id", "agent_id", "status", "content"})
	assertStringField(t, completedRows[0], "event", "run_completed")
	assertStringField(t, completedRows[0], "status", "completed")
	assertStringField(t, completedRows[0], "content", "done")
}

// TestP4G4_NewEventRoundTripTyped 钉 D4：六个新事件经 Events 编解码往返后还原为
// 同型同值（typed，不落 GenericRunEvent/旧类型）；尤其 node_completed 不得被
// contains 归一化吃进 RunCompletedEvent。
func TestP4G4_NewEventRoundTripTyped(t *testing.T) {
	cases := []struct {
		name  string
		evt   run.BaseRunOutputEvent
		check func(t *testing.T, evt run.BaseRunOutputEvent)
	}{
		{
			name: "node_started",
			evt:  run.NewNodeStartedEvent("run-1", "node-a", 2, "in-payload"),
			check: func(t *testing.T, evt run.BaseRunOutputEvent) {
				e, ok := evt.(*run.NodeStartedEvent)
				if !ok {
					t.Fatalf("expected *run.NodeStartedEvent, got %T", evt)
				}
				if e.Node != "node-a" || e.Attempt != 2 || e.Input != "in-payload" {
					t.Fatalf("payload not preserved: %+v", e)
				}
			},
		},
		{
			name: "node_completed",
			evt:  run.NewNodeCompletedEvent("run-1", "node-a", 2, "out-payload"),
			check: func(t *testing.T, evt run.BaseRunOutputEvent) {
				if _, isCompleted := evt.(*run.RunCompletedEvent); isCompleted {
					t.Fatalf("node_completed must not be swallowed into RunCompletedEvent")
				}
				e, ok := evt.(*run.NodeCompletedEvent)
				if !ok {
					t.Fatalf("expected *run.NodeCompletedEvent, got %T", evt)
				}
				if e.Node != "node-a" || e.Attempt != 2 || e.Output != "out-payload" {
					t.Fatalf("payload not preserved: %+v", e)
				}
			},
		},
		{
			name: "task_error",
			evt:  run.NewTaskErrorEvent("run-1", "node-b", "boom"),
			check: func(t *testing.T, evt run.BaseRunOutputEvent) {
				e, ok := evt.(*run.TaskErrorEvent)
				if !ok {
					t.Fatalf("expected *run.TaskErrorEvent, got %T", evt)
				}
				if e.Message != "boom" {
					t.Fatalf("payload not preserved: %+v", e)
				}
			},
		},
		{
			name: "checkpoint",
			evt:  run.NewCheckpointEvent("run-1", "pre-loop", map[string]any{"step": 1}),
			check: func(t *testing.T, evt run.BaseRunOutputEvent) {
				e, ok := evt.(*run.CheckpointEvent)
				if !ok {
					t.Fatalf("expected *run.CheckpointEvent, got %T", evt)
				}
				if e.Label != "pre-loop" || e.Payload == nil {
					t.Fatalf("payload not preserved: %+v", e)
				}
			},
		},
		{
			name: "state_update",
			evt:  run.NewStateUpdateEvent("run-1", "node-a", map[string]any{"k": "v"}),
			check: func(t *testing.T, evt run.BaseRunOutputEvent) {
				e, ok := evt.(*run.StateUpdateEvent)
				if !ok {
					t.Fatalf("expected *run.StateUpdateEvent, got %T", evt)
				}
				if e.Patch == nil {
					t.Fatalf("payload not preserved: %+v", e)
				}
			},
		},
		{
			name: "custom",
			evt:  run.NewCustomEvent("run-1", "progress", "halfway"),
			check: func(t *testing.T, evt run.BaseRunOutputEvent) {
				e, ok := evt.(*run.CustomEvent)
				if !ok {
					t.Fatalf("expected *run.CustomEvent, got %T", evt)
				}
				if e.Subtype != "progress" || e.Data != "halfway" {
					t.Fatalf("payload not preserved: %+v", e)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(run.Events{tc.evt})
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}
			var decoded run.Events
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if len(decoded) != 1 {
				t.Fatalf("expected 1 event, got %d", len(decoded))
			}
			if decoded[0].EventType() != tc.evt.EventType() {
				t.Fatalf("round-tripped EventType() = %q, want %q", decoded[0].EventType(), tc.evt.EventType())
			}
			tc.check(t, decoded[0])
		})
	}
}

// TestP4G4_ZeroValueEventMarshalUsesCanonicalFallback 钉 D2/D4 的第二层防御：
// 字面量构造（种类为空）的新事件序列化时必须兜底 canonical wire 名，
// 往返后仍还原为具体类型而非旧类型。
func TestP4G4_ZeroValueEventMarshalUsesCanonicalFallback(t *testing.T) {
	data, err := json.Marshal(run.Events{&run.NodeCompletedEvent{}})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	assertStringField(t, rows[0], "event", "node_completed")

	var decoded run.Events
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if _, ok := decoded[0].(*run.NodeCompletedEvent); !ok {
		t.Fatalf("empty-kind literal must round-trip typed as NodeCompletedEvent, got %T", decoded[0])
	}
}

// TestP4G4_UnknownKindFallsBackToGeneric 钉 D5（反向钉，实现前后都绿）：
// 完全未知的 kind 仍由 GenericRunEvent 兜底保留载荷。
func TestP4G4_UnknownKindFallsBackToGeneric(t *testing.T) {
	payload := []byte(`[{"event":"mystery_kind","custom_field":"keep-me","created_at":1700000000}]`)
	var decoded run.Events
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	gen, ok := decoded[0].(*run.GenericRunEvent)
	if !ok {
		t.Fatalf("expected *run.GenericRunEvent, got %T", decoded[0])
	}
	if gen.EventType() != "mystery_kind" {
		t.Fatalf("EventType() = %q, want mystery_kind", gen.EventType())
	}
	data, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("re-marshal failed: %v", err)
	}
	var roundtrip []map[string]any
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("roundtrip decode failed: %v", err)
	}
	assertStringField(t, roundtrip[0], "event", "mystery_kind")
	assertStringField(t, roundtrip[0], "custom_field", "keep-me")
}

// TestP4G4_LegacyNormalizationUnchanged 钉 D6（反向对照，实现前后都绿）：
// 既有 contains 归一化的两个现存用例不破 —— team_run_content → RunContentEvent、
// run_completed → RunCompletedEvent；且对旧名字仍宽容（大小写/空白 trim 行为不变，
// 与 D12 共同钉住次序的另一方向）。
func TestP4G4_LegacyNormalizationUnchanged(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		check   func(t *testing.T, evt run.BaseRunOutputEvent)
	}{
		{
			name:    "team_run_content",
			payload: `[{"event":"team_run_content","team_id":"team-1","content":"hi"}]`,
			check: func(t *testing.T, evt run.BaseRunOutputEvent) {
				e, ok := evt.(*run.RunContentEvent)
				if !ok {
					t.Fatalf("expected *run.RunContentEvent, got %T", evt)
				}
				if e.TeamID != "team-1" {
					t.Fatalf("team id = %q, want team-1", e.TeamID)
				}
			},
		},
		{
			name:    "run_completed",
			payload: `[{"event":"run_completed","content":"done"}]`,
			check: func(t *testing.T, evt run.BaseRunOutputEvent) {
				if _, ok := evt.(*run.RunCompletedEvent); !ok {
					t.Fatalf("expected *run.RunCompletedEvent, got %T", evt)
				}
			},
		},
		{
			name:    "case and whitespace tolerance kept",
			payload: `[{"event":"  Team_Run_Content "}]`,
			check: func(t *testing.T, evt run.BaseRunOutputEvent) {
				if _, ok := evt.(*run.RunContentEvent); !ok {
					t.Fatalf("expected *run.RunContentEvent, got %T", evt)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var decoded run.Events
			if err := json.Unmarshal([]byte(tc.payload), &decoded); err != nil {
				t.Fatalf("decode failed: %v", err)
			}
			if len(decoded) != 1 {
				t.Fatalf("expected 1 event, got %d", len(decoded))
			}
			tc.check(t, decoded[0])
		})
	}
}

// TestP4G4_MixedArrayRoundTrip 钉 D11：新旧混合事件数组的 JSON 往返，
// 七个事件逐个还原具体类型与载荷值（组合兼容性）。
func TestP4G4_MixedArrayRoundTrip(t *testing.T) {
	events := run.Events{
		run.NewRunContentEvent("run-1", "agent-1", "assistant", "hello", 0),
		run.NewNodeStartedEvent("run-1", "node-a", 1, "in-payload"),
		run.NewNodeCompletedEvent("run-1", "node-a", 1, "out-payload"),
		run.NewTaskErrorEvent("run-1", "node-b", "boom"),
		run.NewCheckpointEvent("run-1", "pre-loop", map[string]any{"step": 1}),
		run.NewStateUpdateEvent("run-1", "node-a", map[string]any{"k": "v"}),
		run.NewCustomEvent("run-1", "progress", "halfway"),
	}
	data, err := json.Marshal(events)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded run.Events
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decoded) != len(events) {
		t.Fatalf("expected %d events, got %d", len(events), len(decoded))
	}

	wantTypes := []string{
		"*run.RunContentEvent",
		"*run.NodeStartedEvent",
		"*run.NodeCompletedEvent",
		"*run.TaskErrorEvent",
		"*run.CheckpointEvent",
		"*run.StateUpdateEvent",
		"*run.CustomEvent",
	}
	for i, want := range wantTypes {
		if got := typeLabel(decoded[i]); got != want {
			t.Fatalf("decoded[%d] = %s, want %s", i, got, want)
		}
	}

	content, ok := decoded[0].(*run.RunContentEvent)
	if !ok || content.Content != "hello" {
		t.Fatalf("legacy event lost in mixed array: %+v", decoded[0])
	}
	started, ok := decoded[1].(*run.NodeStartedEvent)
	if !ok || started.Node != "node-a" || started.Input != "in-payload" {
		t.Fatalf("node_started payload lost: %+v", decoded[1])
	}
	completed, ok := decoded[2].(*run.NodeCompletedEvent)
	if !ok || completed.Output != "out-payload" {
		t.Fatalf("node_completed payload lost: %+v", decoded[2])
	}
	taskErr, ok := decoded[3].(*run.TaskErrorEvent)
	if !ok || taskErr.Message != "boom" {
		t.Fatalf("task_error payload lost: %+v", decoded[3])
	}
	checkpoint, ok := decoded[4].(*run.CheckpointEvent)
	if !ok || checkpoint.Label != "pre-loop" {
		t.Fatalf("checkpoint payload lost: %+v", decoded[4])
	}
	stateUpdate, ok := decoded[5].(*run.StateUpdateEvent)
	if !ok || stateUpdate.Patch == nil {
		t.Fatalf("state_update payload lost: %+v", decoded[5])
	}
	custom, ok := decoded[6].(*run.CustomEvent)
	if !ok || custom.Subtype != "progress" || custom.Data != "halfway" {
		t.Fatalf("custom payload lost: %+v", decoded[6])
	}
}

// TestP4G4_DecodeOrderExactBeforeContains 钉 D12：decodeEvent 的分派次序语义 ——
// 精确 canonical 名先于 contains 归一化（node_completed 方向），且归一化对旧名字
// 仍宽容（RUN_COMPLETED 大小写/空白方向）。两个方向缺一不可。
func TestP4G4_DecodeOrderExactBeforeContains(t *testing.T) {
	data, err := json.Marshal(run.Events{run.NewNodeCompletedEvent("run-1", "node-a", 1, "out")})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded run.Events
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	e, ok := decoded[0].(*run.NodeCompletedEvent)
	if !ok {
		t.Fatalf("exact canonical name must win over contains normalization, got %T", decoded[0])
	}
	if e.Output != "out" {
		t.Fatalf("payload lost across order boundary: %+v", e)
	}

	var tolerant run.Events
	if err := json.Unmarshal([]byte(`[{"event":" RUN_COMPLETED ","content":"done"}]`), &tolerant); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if _, ok := tolerant[0].(*run.RunCompletedEvent); !ok {
		t.Fatalf("legacy tolerant normalization must keep working, got %T", tolerant[0])
	}
}

func assertExactFields(t *testing.T, row map[string]any, want []string) {
	t.Helper()
	if len(row) != len(want) {
		t.Fatalf("field set = %v (len %d), want exactly %v", keys(row), len(row), want)
	}
	for _, k := range want {
		if _, ok := row[k]; !ok {
			t.Fatalf("missing field %q in %v", k, keys(row))
		}
	}
}

func assertStringField(t *testing.T, row map[string]any, key, want string) {
	t.Helper()
	got, ok := row[key].(string)
	if !ok || got != want {
		t.Fatalf("%s = %v, want %q", key, row[key], want)
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func typeLabel(v any) string {
	return fmt.Sprintf("%T", v)
}
