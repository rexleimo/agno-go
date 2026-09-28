package sidecar_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/session"
	"github.com/rexleimo/agno-go/pkg/hno/session/sidecar"
)

// TestP5S28_ByteIdentity：事件/挂起记录写入侧车后，同一 Session 值的对外 JSON
// 字节逐位不变（D1=OPT-1 核心命题）。附 [B2] 对照：Storage.Update 会改字节
// （UpdatedAt 重打）——这正是事件写入绝不路由过 Update 的原因。
func TestP5S28_ByteIdentity(t *testing.T) {
	ctx := context.Background()
	sess := session.NewSession("sess-b1", "agent-1")
	before, err := json.Marshal(sess)
	if err != nil {
		t.Fatal(err)
	}

	sc := sidecar.NewMemorySidecar()
	if _, err := sc.AppendEvent(ctx, "sess-b1", sidecar.Event{Kind: "node.completed", Payload: map[string]any{"n": "a"}}); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	if err := sc.SaveSuspendedRun(ctx, sidecar.SuspendedRun{
		SessionID: "sess-b1",
		RunKey:    "run-1",
		Waiting: []sidecar.WaitingNodeRecord{{
			Node:      "approve",
			Interrupt: sidecar.InterruptRecord{InterruptID: "i-1", Mode: sidecar.ModeRerun},
		}},
	}); err != nil {
		t.Fatalf("SaveSuspendedRun: %v", err)
	}

	after, err := json.Marshal(sess)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("pinned session JSON changed after sidecar writes: %d vs %d bytes", len(before), len(after))
	}

	// [B2] 对照：主存储 Update 本身（与事件无关）就改变字节——纪律的由来。
	store := session.NewMemoryStorage()
	_ = store.Create(ctx, sess)
	sess.State = map[string]interface{}{"touched": true}
	_ = store.Update(ctx, sess)
	afterUpdate, _ := json.Marshal(sess)
	if bytes.Equal(before, afterUpdate) {
		t.Fatal("expected Storage.Update to change bytes (UpdatedAt re-stamp); control failed")
	}
}

// TestP5S28_StoreRoundTripAndViews：事件流追加（Seq 分配）、派生视图
// （Events / EventsByRun / PendingInterrupts）与挂起记录读写清除。
func TestP5S28_StoreRoundTripAndViews(t *testing.T) {
	ctx := context.Background()
	sc := sidecar.NewMemorySidecar()

	if _, err := sc.AppendEvent(ctx, "s1", sidecar.Event{Kind: "run.started", RunKey: "r1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := sc.AppendEvent(ctx, "s1", sidecar.Event{Kind: "hitl.interrupt", RunKey: "r1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := sc.AppendEvent(ctx, "s1", sidecar.Event{Kind: "run.started", RunKey: "r2"}); err != nil {
		t.Fatal(err)
	}

	events, err := sc.Events(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[0].Seq >= events[1].Seq || events[1].Seq >= events[2].Seq {
		t.Fatalf("events view: want 3 ascending seq, got %+v", events)
	}
	byRun, err := sc.EventsByRun(ctx, "s1", "r1")
	if err != nil {
		t.Fatal(err)
	}
	if len(byRun) != 2 {
		t.Fatalf("EventsByRun r1: want 2, got %d", len(byRun))
	}

	rec := sidecar.SuspendedRun{
		SessionID: "s1",
		RunKey:    "r1",
		Waiting: []sidecar.WaitingNodeRecord{
			{Node: "approve", Interrupt: sidecar.InterruptRecord{InterruptID: "i-1", Mode: sidecar.ModeRerun}},
		},
		Values:    map[string]any{"a": "va"},
		Completed: []string{"a"},
		Seq:       2,
		Steps:     5,
	}
	if err := sc.SaveSuspendedRun(ctx, rec); err != nil {
		t.Fatal(err)
	}
	pending, err := sc.PendingInterrupts(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].InterruptID != "i-1" {
		t.Fatalf("PendingInterrupts: want [i-1], got %+v", pending)
	}

	got, ok, err := sc.SuspendedRun(ctx, "s1", "r1")
	if err != nil || !ok {
		t.Fatalf("SuspendedRun: ok=%v err=%v", ok, err)
	}
	if got.Seq != 2 || got.Steps != 5 || got.Values["a"] != "va" {
		t.Fatalf("record round-trip mismatch: %+v", got)
	}
	if err := sc.ClearSuspendedRun(ctx, "s1", "r1"); err != nil {
		t.Fatal(err)
	}
	pending, _ = sc.PendingInterrupts(ctx, "s1")
	if len(pending) != 0 {
		t.Fatalf("PendingInterrupts after clear: want 0, got %d", len(pending))
	}
	if _, ok, _ := sc.SuspendedRun(ctx, "s1", "r1"); ok {
		t.Fatal("SuspendedRun after clear: want ok=false")
	}
	if err := sc.ClearSuspendedRun(ctx, "s1", "r1"); err != nil {
		t.Fatalf("ClearSuspendedRun idempotent: %v", err)
	}
}

// TestP5S28_InvalidRecordRejected：缺关键键的挂起记录被点名拒绝
// （fail-closed 数据不接受半截记录）。
func TestP5S28_InvalidRecordRejected(t *testing.T) {
	ctx := context.Background()
	sc := sidecar.NewMemorySidecar()
	if err := sc.SaveSuspendedRun(ctx, sidecar.SuspendedRun{RunKey: "r1"}); err == nil {
		t.Fatal("want error for empty SessionID")
	}
	if err := sc.SaveSuspendedRun(ctx, sidecar.SuspendedRun{SessionID: "s1"}); err == nil {
		t.Fatal("want error for empty RunKey")
	}
	if err := sc.SaveSuspendedRun(ctx, sidecar.SuspendedRun{SessionID: "s1", RunKey: "r1"}); err == nil {
		t.Fatal("want error for zero waiting nodes")
	}
}

// TestP5S28_SnapshotRestart：快照字节 → 全新侧车水合 → 记录与序号状态原样
// 回来（模拟重启的存储半边）。
func TestP5S28_SnapshotRestart(t *testing.T) {
	ctx := context.Background()
	sc := sidecar.NewMemorySidecar()
	_, _ = sc.AppendEvent(ctx, "s1", sidecar.Event{Kind: "hitl.interrupt", RunKey: "r1"})
	if err := sc.SaveSuspendedRun(ctx, sidecar.SuspendedRun{
		SessionID: "s1",
		RunKey:    "r1",
		Waiting: []sidecar.WaitingNodeRecord{
			{Node: "approve", In: map[string]any{"q": "ok?"}, Interrupt: sidecar.InterruptRecord{InterruptID: "i-1", Mode: sidecar.ModeHandoff, ResponseSchema: map[string]any{"type": "object"}}},
		},
		Seq:   3,
		Steps: 7,
	}); err != nil {
		t.Fatal(err)
	}

	snap, err := sc.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	fresh := sidecar.NewMemorySidecar()
	if err := fresh.RestoreSnapshot(snap); err != nil {
		t.Fatal(err)
	}
	events, _ := fresh.Events(ctx, "s1")
	if len(events) != 1 || events[0].Kind != "hitl.interrupt" {
		t.Fatalf("events after restore: %+v", events)
	}
	rec, ok, err := fresh.SuspendedRun(ctx, "s1", "r1")
	if err != nil || !ok {
		t.Fatalf("suspended after restore: ok=%v err=%v", ok, err)
	}
	if rec.Steps != 7 || rec.Waiting[0].In.(map[string]any)["q"] != "ok?" || rec.Waiting[0].Interrupt.Mode != sidecar.ModeHandoff {
		t.Fatalf("record after restore mismatch: %+v", rec)
	}
	// 序号续号：水合后新事件 Seq 接着旧流（检查点 Seq 不跳号的存储半边）。
	seq, _ := fresh.AppendEvent(ctx, "s1", sidecar.Event{Kind: "hitl.resumed", RunKey: "r1"})
	if seq != 2 {
		t.Fatalf("seq continuity after restore: want 2, got %d", seq)
	}
}
