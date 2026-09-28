package hitlbridge_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rexleimo/agno-go/internal/hitlbridge"
	"github.com/rexleimo/agno-go/pkg/hno/graph"
	"github.com/rexleimo/agno-go/pkg/hno/session"
	"github.com/rexleimo/agno-go/pkg/hno/session/sidecar"
)

// memCP 是检查点夹具 sink（切片 24/27 的条目流观察面）：挂起账目的引擎侧读数经它对账。
type memCP struct{ entries []graph.Checkpoint }

func (m *memCP) Append(_ context.Context, cp graph.Checkpoint) error {
	m.entries = append(m.entries, cp)
	return nil
}

// buildApprovalGraph 按同一声明重建审批图（跨进程恢复的「图不序列化、同声明重建」语义）。
// calls 是本次构建私有的 approve 调用计数（每实例一份，互不共享）。
func buildApprovalGraph(opts ...graph.Option) (*graph.Graph, *int) {
	calls := 0
	g := graph.New(opts...).
		AddNode(graph.NodeFunc("start", func(_ context.Context, in any) (any, error) {
			return "prepared", nil
		})).
		AddNode(graph.NodeFunc("approve", func(ctx context.Context, in any) (any, error) {
			calls++
			// Rerun 重入：响应经公共取值器可见（切片 27 D15），取到即通过。
			if resp, ok := graph.InterruptResponse(ctx, "i-approve"); ok {
				return map[string]any{"approved": resp.(map[string]any)["approved"]}, nil
			}
			return nil, graph.RequestInterrupt(graph.Interrupt{
				InterruptID:    "i-approve",
				Message:        "approve the deployment?",
				ResponseSchema: map[string]any{"type": "object", "required": []string{"approved"}, "properties": map[string]any{"approved": map[string]any{"type": "boolean"}}},
				Mode:           graph.ResumeRerun,
			})
		})).
		AddNode(graph.NodeFunc("final", func(_ context.Context, in any) (any, error) {
			return map[string]any{"done": true, "approved": in}, nil
		})).
		AddEdge("start", "approve").AddEdge("approve", "final").
		SetEntry("start").SetOutput("final")
	return g, &calls
}

// TestP5S28_CrossProcessResumeRoundTrip：中断 → 侧车持久化 → 快照字节水合进
// 全新侧车 + 全新图实例（模拟重启）→ 坏响应被拒且挂起保留（切片 27 校验语义跨
// 重启保持）→ 好响应跑完 → 重复 Resume 幂等 → 挂起记录清除；引擎账目（seq/steps）
// 逐值回装——检查点 Seq 跨重启不跳号；全程 pinned 会话 JSON 字节恒等。
func TestP5S28_CrossProcessResumeRoundTrip(t *testing.T) {
	ctx := context.Background()

	// --- 进程 1：跑到中断，挂起连同伴答账目一起入侧车。
	cp1 := &memCP{}
	g1, calls1 := buildApprovalGraph(graph.WithCheckpointer(cp1))
	sess := session.NewSession("sess-x", "agent-1")
	pinned, err := json.Marshal(sess)
	if err != nil {
		t.Fatal(err)
	}
	sc := sidecar.NewMemorySidecar()
	res, err := g1.Run(ctx, map[string]any{"input": "deploy"})
	if !errors.Is(err, graph.ErrSuspended) || res != nil {
		t.Fatalf("run: want suspension, got res=%v err=%v", res, err)
	}
	var susp *graph.Suspension
	if !errors.As(err, &susp) {
		t.Fatalf("errors.As *Suspension failed: %v", err)
	}
	// 账目读侧：挂检查点 sink 时 seq == 完成条目数 + EntryInterrupt 条目数（=2），
	// steps == 已派发激活数（=2）。
	seq, steps := susp.ResumeAccount()
	if seq != 2 || steps != 2 {
		t.Fatalf("resume account: want seq=2 steps=2, got seq=%d steps=%d", seq, steps)
	}
	kinds := map[graph.EntryKind]int{}
	for _, e := range cp1.entries {
		kinds[e.Kind]++
	}
	if kinds[graph.EntryCompletion] != 1 || kinds[graph.EntryInterrupt] != 1 {
		t.Fatalf("checkpoint stream: want 1 completion + 1 interrupt, got %v", kinds)
	}
	if *calls1 != 1 {
		t.Fatalf("approve calls on first instance: want 1, got %d", *calls1)
	}

	if err := hitlbridge.Capture(ctx, sc, "sess-x", "run-1", susp); err != nil {
		t.Fatalf("capture: %v", err)
	}
	// 账目入记录：记录 seq/steps == 挂起产物读出的账目（D7 记录侧）。
	rec, ok, err := sc.SuspendedRun(ctx, "sess-x", "run-1")
	if err != nil || !ok {
		t.Fatalf("suspended record after capture: ok=%v err=%v", ok, err)
	}
	if rec.Seq != seq || rec.Steps != steps {
		t.Fatalf("record account: want %d/%d, got %d/%d", seq, steps, rec.Seq, rec.Steps)
	}
	// 侧车写入后 pinned 字节逐位不变（OPT-1）。
	after, _ := json.Marshal(sess)
	if !bytes.Equal(pinned, after) {
		t.Fatal("pinned session JSON changed after sidecar capture")
	}

	// --- 模拟重启：字节快照 → 全新侧车 + 全新图实例（同一声明重建）。
	if _, err := sc.AppendEvent(ctx, "sess-x", sidecar.Event{Kind: "hitl.interrupt", RunKey: "run-1"}); err != nil {
		t.Fatal(err)
	}
	snap, err := sc.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	sc2 := sidecar.NewMemorySidecar()
	if err := sc2.RestoreSnapshot(snap); err != nil {
		t.Fatal(err)
	}
	// 派生视图在「新进程」可读：待答中断与事件流（D3 跨重启）。
	pending, err := sc2.PendingInterrupts(ctx, "sess-x")
	if err != nil || len(pending) != 1 || pending[0].InterruptID != "i-approve" {
		t.Fatalf("pending view after restart: %+v err=%v", pending, err)
	}
	events, _ := sc2.EventsByRun(ctx, "sess-x", "run-1")
	if len(events) < 1 {
		t.Fatal("events view empty after restart")
	}

	// --- 进程 2：坏响应先被拒（挂起保留、节点不重跑），好响应后跑完。
	cp2 := &memCP{}
	g2, calls2 := buildApprovalGraph(graph.WithCheckpointer(cp2))
	_, err = hitlbridge.ResumeSaved(ctx, sc2, g2, "sess-x", "run-1", map[string]any{"i-approve": map[string]any{"approved": "not-a-bool"}})
	if !errors.Is(err, graph.ErrInvalidResponse) {
		t.Fatalf("bad response after restart: want ErrInvalidResponse, got %v", err)
	}
	if *calls2 != 0 {
		t.Fatalf("rejected validation must not rerun the node, got %d calls", *calls2)
	}
	rec2, ok, _ := sc2.SuspendedRun(ctx, "sess-x", "run-1")
	if !ok {
		t.Fatal("suspended record must survive a rejected response")
	}
	if rec2.Seq != seq || rec2.Steps != steps {
		t.Fatalf("record account after rejection: want %d/%d, got %d/%d", seq, steps, rec2.Seq, rec2.Steps)
	}

	res2, err := hitlbridge.ResumeSaved(ctx, sc2, g2, "sess-x", "run-1", map[string]any{"i-approve": map[string]any{"approved": true}})
	if err != nil {
		t.Fatalf("resume after restart: %v", err)
	}
	if res2 == nil {
		t.Fatal("resume returned nil result")
	}
	final, _ := res2.Value("final")
	if final == nil || final.(map[string]any)["done"] != true {
		t.Fatalf("resume result missing final output: %v", final)
	}
	if *calls2 != 1 {
		t.Fatalf("rerun mode: want approve rerun exactly once on rebuilt instance, got %d", *calls2)
	}
	// 账目装回：重建实例的检查点 Seq 从挂起账目续号（装回清零则从 1 起跳）。
	if len(cp2.entries) != 2 || cp2.entries[0].Seq != seq+1 {
		t.Fatalf("checkpoint seq continuity after restart: want first entry seq=%d, got %+v", seq+1, cp2.entries)
	}

	// 重复 Resume → ErrNothingToResume（幂等，跨重建保持）。
	if _, err := g2.Resume(ctx, map[string]any{"i-approve": map[string]any{"approved": true}}); !errors.Is(err, graph.ErrNothingToResume) {
		t.Fatalf("double resume: want ErrNothingToResume, got %v", err)
	}
	// 挂起记录已清除（fail-closed 数据的生命周期）。
	if _, ok, _ := sc2.SuspendedRun(ctx, "sess-x", "run-1"); ok {
		t.Fatal("suspended record should be cleared after successful resume")
	}
	if left, _ := sc2.PendingInterrupts(ctx, "sess-x"); len(left) != 0 {
		t.Fatalf("pending view after successful resume: want 0, got %d", len(left))
	}
}

// TestP5S28_HandoffAcrossRestart：Handoff 档跨重建——等待节点不再执行，
// 响应直接成为其输出喂给后继。
func TestP5S28_HandoffAcrossRestart(t *testing.T) {
	ctx := context.Background()
	askCalls := 0
	build := func() *graph.Graph {
		return graph.New().
			AddNode(graph.NodeFunc("ask", func(_ context.Context, in any) (any, error) {
				askCalls++
				return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-h", Message: "give name", Mode: graph.ResumeHandoff})
			})).
			AddNode(graph.NodeFunc("sink", func(_ context.Context, in any) (any, error) {
				return map[string]any{"got": in}, nil
			})).
			AddEdge("ask", "sink").SetEntry("ask").SetOutput("sink")
	}
	g := build()
	sc := sidecar.NewMemorySidecar()
	_, err := g.Run(ctx, nil)
	if !errors.Is(err, graph.ErrSuspended) {
		t.Fatalf("run: %v", err)
	}
	var susp *graph.Suspension
	if !errors.As(err, &susp) {
		t.Fatal("errors.As failed")
	}
	if err := hitlbridge.Capture(ctx, sc, "s-h", "r-h", susp); err != nil {
		t.Fatal(err)
	}
	snap, _ := sc.Snapshot()
	sc2 := sidecar.NewMemorySidecar()
	_ = sc2.RestoreSnapshot(snap)
	g2 := build()
	res, err := hitlbridge.ResumeSaved(ctx, sc2, g2, "s-h", "r-h", map[string]any{"i-h": "rex"})
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if res == nil {
		t.Fatal("resume returned nil result")
	}
	if askCalls != 1 {
		t.Fatalf("handoff: want ask executed once (interrupt only, no rerun), got %d", askCalls)
	}
	sink, _ := res.Value("sink")
	if sink.(map[string]any)["got"] != "rex" {
		t.Fatalf("handoff response not fed to successor: %v", sink)
	}
}

// failingSidecar 只在指定方法上失败的夹具（fail-open / fail-closed 的判据面）。
// 互斥锁属夹具不属产品——本夹具无状态，仅按构造参数定向失败。
type failingSidecar struct {
	sidecar.Sidecar
	failSave   error
	failAppend error
}

func (f *failingSidecar) SaveSuspendedRun(ctx context.Context, rec sidecar.SuspendedRun) error {
	if f.failSave != nil {
		return f.failSave
	}
	return f.Sidecar.SaveSuspendedRun(ctx, rec)
}

func (f *failingSidecar) AppendEvent(ctx context.Context, sessionID string, e sidecar.Event) (int64, error) {
	if f.failAppend != nil {
		return 0, f.failAppend
	}
	return f.Sidecar.AppendEvent(ctx, sessionID, e)
}

// TestP5S28_FailureSemantics：事件流 fail-open（写失败不阻塞主路径、挂起记录照常落）、
// 挂起记录 fail-closed（写失败必须上交且可归因）、记录缺失点名上交、
// 主会话路径与侧车故障正交。
func TestP5S28_FailureSemantics(t *testing.T) {
	ctx := context.Background()
	boom := errors.New("sidecar backend down")

	g, _ := buildApprovalGraph()
	inner := sidecar.NewMemorySidecar()
	sc := &failingSidecar{Sidecar: inner, failAppend: boom}
	_, err := g.Run(ctx, nil)
	if !errors.Is(err, graph.ErrSuspended) {
		t.Fatalf("run: %v", err)
	}
	var susp *graph.Suspension
	if !errors.As(err, &susp) {
		t.Fatal("errors.As failed")
	}

	// fail-open：事件追加失败被 Capture 忽略，挂起记录照常保存，主路径无恙。
	if err := hitlbridge.Capture(ctx, sc, "s-f", "r-f", susp); err != nil {
		t.Fatalf("event append failure must not fail capture (fail-open): %v", err)
	}
	if _, ok, _ := inner.SuspendedRun(ctx, "s-f", "r-f"); !ok {
		t.Fatal("suspended record missing after event-append failure")
	}
	// 主会话路径与侧车故障正交：Session 存储照常工作。
	store := session.NewMemoryStorage()
	sess := session.NewSession("s-f", "agent-1")
	if err := store.Create(ctx, sess); err != nil {
		t.Fatalf("session storage blocked by sidecar failure: %v", err)
	}

	// fail-closed：挂起记录写失败必须上交，绝不静默，且可 errors.Is 归因到后端错误。
	sc2 := &failingSidecar{Sidecar: sidecar.NewMemorySidecar(), failSave: boom}
	if err := hitlbridge.Capture(ctx, sc2, "s-f2", "r-f2", susp); err == nil {
		t.Fatal("save failure must surface (fail-closed)")
	} else if !errors.Is(err, boom) {
		t.Fatalf("save failure not attributable: %v", err)
	}

	// 记录缺失：ResumeSaved 上交点名错误（不冒充恢复成功）。
	gM, _ := buildApprovalGraph()
	if _, err := hitlbridge.ResumeSaved(ctx, sidecar.NewMemorySidecar(), gM, "s-none", "r-none", map[string]any{"i-approve": true}); !errors.Is(err, sidecar.ErrNotFound) {
		t.Fatalf("missing record must surface a named error: %v", err)
	}
}

// TestP5S28_ResuspendChainLifecycle：重悬链的记录生命周期与安装守卫——
// 恢复可再次挂起（新挂起由 Resume 自行入账，旧记录保留为陈旧内容）；对已有
// 活挂起的图重复安装不覆盖（ErrPendingInstalled）；新挂起可再 Capture 覆盖记录，
// 二次重启后按新挂起恢复跑完、记录清除。
func TestP5S28_ResuspendChainLifecycle(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{"type": "object", "required": []string{"ok"}, "properties": map[string]any{"ok": map[string]any{"type": "boolean"}}}
	gateCalls := 0
	build := func() *graph.Graph {
		return graph.New().
			AddNode(graph.NodeFunc("gate", func(ctx context.Context, in any) (any, error) {
				gateCalls++
				if _, ok := graph.InterruptResponse(ctx, "i-1"); ok {
					// 第一道已批：重悬链——同一节点再次中断，换第二个中断 ID。
					return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-2", Message: "second confirmation", ResponseSchema: schema, Mode: graph.ResumeRerun})
				}
				if _, ok := graph.InterruptResponse(ctx, "i-2"); ok {
					// 第二道也已批：放行。
					return map[string]any{"gate": "cleared"}, nil
				}
				return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-1", Message: "first gate", ResponseSchema: schema, Mode: graph.ResumeRerun})
			})).
			AddNode(graph.NodeFunc("final", func(_ context.Context, in any) (any, error) {
				return map[string]any{"done": true}, nil
			})).
			AddEdge("gate", "final").SetEntry("gate").SetOutput("final")
	}

	g1 := build()
	sc := sidecar.NewMemorySidecar()
	_, runErr := g1.Run(ctx, nil)
	if !errors.Is(runErr, graph.ErrSuspended) {
		t.Fatalf("run: %v", runErr)
	}
	var susp1 *graph.Suspension
	if !errors.As(runErr, &susp1) {
		t.Fatal("errors.As failed on first suspension")
	}
	if err := hitlbridge.Capture(ctx, sc, "s-c", "r-c", susp1); err != nil {
		t.Fatal(err)
	}

	// 模拟重启 → 恢复第一道：节点重入后再次中断（重悬链）。
	snap, _ := sc.Snapshot()
	sc2 := sidecar.NewMemorySidecar()
	if err := sc2.RestoreSnapshot(snap); err != nil {
		t.Fatal(err)
	}
	g2 := build()
	res, resumeErr := hitlbridge.ResumeSaved(ctx, sc2, g2, "s-c", "r-c", map[string]any{"i-1": map[string]any{"ok": true}})
	if !errors.Is(resumeErr, graph.ErrSuspended) || res != nil {
		t.Fatalf("resume should re-suspend, got res=%v err=%v", res, resumeErr)
	}
	var susp2 *graph.Suspension
	if !errors.As(resumeErr, &susp2) || len(susp2.Interrupts) != 1 || susp2.Interrupts[0].Interrupt.InterruptID != "i-2" {
		t.Fatalf("re-suspension should wait on i-2, got %+v", susp2)
	}
	// 恢复未完成：挂起记录保留（内容是陈旧的第一道），等待链上新挂起。
	if rec, ok, _ := sc2.SuspendedRun(ctx, "s-c", "r-c"); !ok || rec.Waiting[0].Interrupt.InterruptID != "i-1" {
		t.Fatalf("stale record must be retained untouched, got ok=%v rec=%v", ok, rec)
	}
	// 安装守卫：对已有活挂起的图再装旧记录 → ErrPendingInstalled，活挂起不被覆盖。
	if err := hitlbridge.Install(g2, mustRecord(t, sc2, "s-c", "r-c")); !errors.Is(err, graph.ErrPendingInstalled) {
		t.Fatalf("second install on live pending: want ErrPendingInstalled, got %v", err)
	}
	// 新挂起可再 Capture：记录被新等待项覆盖（重悬链记录可再 Capture）。
	if err := hitlbridge.Capture(ctx, sc2, "s-c", "r-c", susp2); err != nil {
		t.Fatal(err)
	}
	if rec, _, _ := sc2.SuspendedRun(ctx, "s-c", "r-c"); rec.Waiting[0].Interrupt.InterruptID != "i-2" {
		t.Fatalf("record should now carry i-2, got %+v", rec)
	}

	// 二次重启：按新挂起恢复 → 跑完、记录清除、待答视图清空。
	snap2, _ := sc2.Snapshot()
	sc3 := sidecar.NewMemorySidecar()
	if err := sc3.RestoreSnapshot(snap2); err != nil {
		t.Fatal(err)
	}
	g3 := build()
	res3, err := hitlbridge.ResumeSaved(ctx, sc3, g3, "s-c", "r-c", map[string]any{"i-2": map[string]any{"ok": true}})
	if err != nil || res3 == nil {
		t.Fatalf("second-hop resume: %v", err)
	}
	if final, _ := res3.Value("final"); final == nil || final.(map[string]any)["done"] != true {
		t.Fatalf("second-hop resume missing final output: %v", final)
	}
	if gateCalls != 3 {
		t.Fatalf("gate calls across the chain (interrupt + rerun + second rerun): want 3, got %d", gateCalls)
	}
	if _, ok, _ := sc3.SuspendedRun(ctx, "s-c", "r-c"); ok {
		t.Fatal("suspended record should be cleared after the chain completes")
	}
	if left, _ := sc3.PendingInterrupts(ctx, "s-c"); len(left) != 0 {
		t.Fatalf("pending view after chain completes: want 0, got %d", len(left))
	}
}

func mustRecord(t *testing.T, sc sidecar.Sidecar, sessionID, runKey string) sidecar.SuspendedRun {
	t.Helper()
	rec, ok, err := sc.SuspendedRun(context.Background(), sessionID, runKey)
	if err != nil || !ok {
		t.Fatalf("record %s/%s missing: ok=%v err=%v", sessionID, runKey, ok, err)
	}
	return rec
}
