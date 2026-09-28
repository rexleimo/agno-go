package graph_test

// 本文件是切片 27（母约 §9 G7：图引擎 HITL Interrupt/Resume 核心）的行为族，对应契约
// docs/design/v3-test-scope-p1-graph-slice27.json（D1–D15）。与既有各片同构：单独成文件
// 让本片的行为族与文件边界对齐，便于按切片审差分。
//
// 观察面守契约 forbiddenObservations：挂起的唯一证据是 Run/Resume 的公共返回值
//（errors.Is(ErrSuspended) / errors.As(*Suspension) / InterruptResponse 取值器）与夹具
// sink 收到的条目；scheduler 内部字段、pending 表形状、goroutine 数、time.Sleep 定序都
// 不是本族的证据。并行双中断只断言清单的集合形状（ID 集与节点名集），不断言完成次序
//（契约 observabilityLimit 第 2 条）。
//
// 夹具自备内存 Checkpointer 与调用计数：互斥锁属夹具不属引擎——Sync/Exit 的 Append 在
// 消费者 goroutine，节点体计数在生产者 goroutine，夹具自己保证并发安全
//（契约 allowedTestSeam.add）。

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// errP5G7Sink 是 D13 用的夹具错误：sink 拒绝提交时交出的可归因错误。
var errP5G7Sink = errors.New("p5g7 fixture: sink refused")

// p5g7Sink 是本片自备的内存 Checkpointer：failOn 对某条条目返回 true 时，该次 Append
// 返回夹具错误（D13 用它只点名 EntryInterrupt 条目）。互斥锁属夹具不属引擎。
type p5g7Sink struct {
	mu      sync.Mutex
	entries []graph.Checkpoint
	failOn  func(graph.Checkpoint) bool
}

func (s *p5g7Sink) Append(_ context.Context, cp graph.Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failOn != nil && s.failOn(cp) {
		return errP5G7Sink
	}
	s.entries = append(s.entries, cp)
	return nil
}

func (s *p5g7Sink) snapshot() []graph.Checkpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]graph.Checkpoint(nil), s.entries...)
}

// p5g7Counter 是节点体调用计数：*int 由同一测试内的多个激活累加（激活之间经队列
// channel 建立 happens-before，无需额外的锁）。
func p5g7Counter() *int { n := 0; return &n }

// TestP5G7_D1_InterruptSuspendsRun 覆盖契约 D1：节点返回 RequestInterrupt → Run 交
// (nil, err)，errors.Is(err, ErrSuspended) 且 errors.As 取到 *Suspension（含
// InterruptID/Message/Payload/Mode 与已完成前驱状态）；不交半截 Result。
// 对把挂起当失败/当完成的实现判红。
func TestP5G7_D1_InterruptSuspendsRun(t *testing.T) {
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
		AddNode(graph.NodeFunc("b", func(_ context.Context, in any) (any, error) {
			return nil, graph.RequestInterrupt(graph.Interrupt{
				InterruptID: "i-1",
				Message:     "need human approval",
				Payload:     map[string]any{"amount": 5},
			})
		})).
		AddEdge("a", "b").
		SetEntry("a").SetOutput("b")

	res, err := g.Run(context.Background(), "go")
	if res != nil {
		t.Fatalf("挂起时 Run 仍返回 Result %v，调用方会把半截执行当成结论（R19-Q1 批评的形状）", res)
	}
	if !errors.Is(err, graph.ErrSuspended) {
		t.Fatalf("Run 的错误 = %v, want errors.Is(err, graph.ErrSuspended)", err)
	}
	var susp *graph.Suspension
	if !errors.As(err, &susp) {
		t.Fatalf("errors.As(err, *graph.Suspension) 不成立: %v", err)
	}
	if len(susp.Interrupts) != 1 {
		t.Fatalf("Suspension 待答中断数 = %d, want 1: %+v", len(susp.Interrupts), susp.Interrupts)
	}
	w := susp.Interrupts[0]
	if w.Node != "b" {
		t.Errorf("等待节点 = %q, want %q", w.Node, "b")
	}
	if w.In != "a#1" {
		t.Errorf("等待节点激活输入 = %v, want a#1（前驱输出被原样记住）", w.In)
	}
	i := w.Interrupt
	if i.InterruptID != "i-1" || i.Message != "need human approval" {
		t.Errorf("中断 ID/Message = %q/%q, want i-1/need human approval", i.InterruptID, i.Message)
	}
	if !reflect.DeepEqual(i.Payload, map[string]any{"amount": 5}) {
		t.Errorf("中断 Payload = %v, want map[amount:5]", i.Payload)
	}
	if i.Mode != graph.ResumeRerun {
		t.Errorf("Mode 零值 = %v, want ResumeRerun（默认档）", i.Mode)
	}
	if got, want := susp.Completed, []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("已完成前驱 = %v, want %v（挂起前完成的状态留在 Suspension）", got, want)
	}
	if v, ok := susp.Values["a"]; !ok || v != "a#1" {
		t.Errorf("Suspension.Values[a] = %v,%v, want a#1,true", v, ok)
	}
}

// TestP5G7_D2_SchemaRejectionKeepsSuspension 覆盖契约 D2：响应未过 ResponseSchema
// （类型错 / 缺必填）→ Resume 返回包 ErrInvalidResponse 的错误，挂起原样保留；
// 修正后的同一挂起可恢复成功。对「校验失败即丢弃挂起」或「不校验」的实现判红。
func TestP5G7_D2_SchemaRejectionKeepsSuspension(t *testing.T) {
	schema := map[string]any{
		"type":     "object",
		"required": []any{"approve"},
		"properties": map[string]any{
			"approve": map[string]any{"type": "boolean"},
		},
	}
	calls := p5g7Counter()
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
		AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
			*calls++
			if *calls == 1 {
				return nil, graph.RequestInterrupt(graph.Interrupt{
					InterruptID: "i-1", Message: "approve?", ResponseSchema: schema,
				})
			}
			return "b#resumed", nil
		})).
		AddEdge("a", "b").
		SetEntry("a").SetOutput("b")

	if _, err := g.Run(context.Background(), "go"); !errors.Is(err, graph.ErrSuspended) {
		t.Fatalf("首跑未挂起: %v", err)
	}
	// 响应集以 InterruptID 为键，值是该中断的响应对象。
	// 方向一：类型错。
	if _, err := g.Resume(context.Background(), map[string]any{"i-1": map[string]any{"approve": "yes"}}); !errors.Is(err, graph.ErrInvalidResponse) {
		t.Fatalf("类型错的响应 = %v, want errors.Is(ErrInvalidResponse)", err)
	}
	// 方向二：缺必填。
	if _, err := g.Resume(context.Background(), map[string]any{"i-1": map[string]any{}}); !errors.Is(err, graph.ErrInvalidResponse) {
		t.Fatalf("缺必填的响应 = %v, want errors.Is(ErrInvalidResponse)", err)
	}
	if *calls != 1 {
		t.Errorf("校验失败后节点体被再次执行 %d 次, want 1（节点保持 Waiting）", *calls)
	}
	// 修正后的同一挂起仍可恢复：挂起没有被校验失败销毁。
	res, err := g.Resume(context.Background(), map[string]any{"i-1": map[string]any{"approve": true}})
	if err != nil {
		t.Fatalf("修正后的 Resume 失败: %v", err)
	}
	if got, want := res.Output(), "b#resumed"; got != want {
		t.Errorf("Output = %v, want %v", got, want)
	}
}

// TestP5G7_D3_RerunModeRerunsWithResponse 覆盖契约 D3：恢复后等待节点带着响应重新执行
// （同输入再跑、响应经 ctx 取值器可见），图从该节点继续跑完。
// 对不重跑直接用旧输出的实现判红。
func TestP5G7_D3_RerunModeRerunsWithResponse(t *testing.T) {
	calls := p5g7Counter()
	var inputs []any
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
		AddNode(graph.NodeFunc("b", func(ctx context.Context, in any) (any, error) {
			*calls++
			inputs = append(inputs, in)
			if *calls == 1 {
				return nil, graph.RequestInterrupt(graph.Interrupt{
					InterruptID: "i-1", Message: "rerun me", Mode: graph.ResumeRerun,
				})
			}
			v, _ := graph.InterruptResponse(ctx, "i-1")
			return fmt.Sprintf("b saw %v", v), nil
		})).
		AddNode(graph.NodeFunc("c", func(_ context.Context, in any) (any, error) { return in, nil })).
		AddEdge("a", "b").AddEdge("b", "c").
		SetEntry("a").SetOutput("c")

	if _, err := g.Run(context.Background(), "go"); !errors.Is(err, graph.ErrSuspended) {
		t.Fatalf("首跑未挂起: %v", err)
	}
	res, err := g.Resume(context.Background(), map[string]any{"i-1": "human-says-go"})
	if err != nil {
		t.Fatalf("Rerun 档 Resume 失败: %v", err)
	}
	if *calls != 2 {
		t.Errorf("节点体执行次数 = %d, want 2（恢复后带响应重跑）", *calls)
	}
	if len(inputs) != 2 || inputs[0] != "a#1" || inputs[1] != "a#1" {
		t.Errorf("两次激活输入 = %v, want 两次都是 a#1（同输入再跑）", inputs)
	}
	if got, want := res.Output(), "b saw human-says-go"; got != want {
		t.Errorf("Output = %v, want %v（输出含响应内容）", got, want)
	}
	if got, want := res.Completed(), []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Completed = %v, want %v（图从该节点继续跑完）", got, want)
	}
}

// TestP5G7_D4_HandoffModeHandsOff 覆盖契约 D4：恢复后等待节点不再执行（体内调用数
// 不变），响应直接成为其输出喂给后继。对「Handoff 也重跑」或「后继收不到」的实现判红。
func TestP5G7_D4_HandoffModeHandsOff(t *testing.T) {
	calls := p5g7Counter()
	var cIn any
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
		AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
			*calls++
			return nil, graph.RequestInterrupt(graph.Interrupt{
				InterruptID: "i-1", Message: "hand it over", Mode: graph.ResumeHandoff,
			})
		})).
		AddNode(graph.NodeFunc("c", func(_ context.Context, in any) (any, error) { cIn = in; return in, nil })).
		AddEdge("a", "b").AddEdge("b", "c").
		SetEntry("a").SetOutput("c")

	if _, err := g.Run(context.Background(), "go"); !errors.Is(err, graph.ErrSuspended) {
		t.Fatalf("首跑未挂起: %v", err)
	}
	if *calls != 1 {
		t.Fatalf("挂起前节点体已执行 %d 次, want 1", *calls)
	}
	res, err := g.Resume(context.Background(), map[string]any{"i-1": "human-value"})
	if err != nil {
		t.Fatalf("Handoff 档 Resume 失败: %v", err)
	}
	if *calls != 1 {
		t.Errorf("恢复后节点体执行次数 = %d, want 仍 1（Handoff 不偷跑节点体）", *calls)
	}
	if v, ok := res.Value("b"); !ok || v != "human-value" {
		t.Errorf("Value(b) = %v,%v, want human-value,true（响应即该节点输出）", v, ok)
	}
	if cIn != "human-value" {
		t.Errorf("后继 c 收到的输入 = %v, want human-value", cIn)
	}
}

// TestP5G7_D5_NothingToResumeIdempotent 覆盖契约 D5：对同一挂起二次 Resume、以及对
// 从未挂起的图 Resume → ErrNothingToResume（errors.Is）。对二次恢复重跑图或 panic
// 的实现判红。
func TestP5G7_D5_NothingToResumeIdempotent(t *testing.T) {
	t.Run("never-suspended", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
			SetEntry("a").SetOutput("a")
		if _, err := g.Run(context.Background(), "go"); err != nil {
			t.Fatalf("普通图执行失败: %v", err)
		}
		res, err := g.Resume(context.Background(), nil)
		if !errors.Is(err, graph.ErrNothingToResume) {
			t.Fatalf("从未挂起的 Resume = %v, want errors.Is(ErrNothingToResume)", err)
		}
		if res != nil {
			t.Errorf("无处可恢复时仍返回 Result %v", res)
		}
	})
	t.Run("double-resume", func(t *testing.T) {
		calls := p5g7Counter()
		g := graph.New().
			AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
			AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
				*calls++
				if *calls == 1 {
					return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-1"})
				}
				return "b#resumed", nil
			})).
			AddEdge("a", "b").
			SetEntry("a").SetOutput("b")
		if _, err := g.Run(context.Background(), "go"); !errors.Is(err, graph.ErrSuspended) {
			t.Fatalf("首跑未挂起: %v", err)
		}
		if _, err := g.Resume(context.Background(), map[string]any{"i-1": "ok"}); err != nil {
			t.Fatalf("首次 Resume 失败: %v", err)
		}
		res, err := g.Resume(context.Background(), map[string]any{"i-1": "ok"})
		if !errors.Is(err, graph.ErrNothingToResume) {
			t.Fatalf("二次 Resume = %v, want errors.Is(ErrNothingToResume)（幂等）", err)
		}
		if res != nil {
			t.Errorf("二次 Resume 仍返回 Result %v", res)
		}
	})
}

// TestP5G7_D6_ResponseSetExactlyCovers 覆盖契约 D6：响应集恰好覆盖全部待答中断——
// 多给（未知 InterruptID）与少给（缺某中断）都报错且文案点名 ID，挂起均保留。
// 对静默忽略多余/缺失的实现判红。
func TestP5G7_D6_ResponseSetExactlyCovers(t *testing.T) {
	calls := p5g7Counter()
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
		AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
			*calls++
			if *calls == 1 {
				return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-1"})
			}
			return "b#resumed", nil
		})).
		AddEdge("a", "b").
		SetEntry("a").SetOutput("b")

	if _, err := g.Run(context.Background(), "go"); !errors.Is(err, graph.ErrSuspended) {
		t.Fatalf("首跑未挂起: %v", err)
	}
	// 多给：未知 InterruptID 被点名拒绝，不静默取交集。
	if _, err := g.Resume(context.Background(), map[string]any{"i-1": "x", "ghost": "y"}); err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("多余响应的错误 = %v, want 点名未知 InterruptID ghost", err)
	}
	// 少给：缺失的中断被点名拒绝。
	if _, err := g.Resume(context.Background(), map[string]any{}); err == nil || !strings.Contains(err.Error(), "i-1") {
		t.Fatalf("缺失响应的错误 = %v, want 点名缺失的 InterruptID i-1", err)
	}
	// 两个方向都不销毁挂起：正确响应仍可恢复。
	res, err := g.Resume(context.Background(), map[string]any{"i-1": "x"})
	if err != nil {
		t.Fatalf("修正后的 Resume 失败: %v", err)
	}
	if got, want := res.Output(), "b#resumed"; got != want {
		t.Errorf("Output = %v, want %v", got, want)
	}
}

// TestP5G7_D7_InterruptCheckpointEntries 覆盖契约 D7：Sync 档挂起时 Checkpointer 收到
// 完成条目 + EntryInterrupt 条目（节点名/输入/Interrupt 全量）；普通完成的条目种类零值
// 不变（既有 durability 反向对照）。对挂起不落条目或改既有条目形状的实现判红。
func TestP5G7_D7_InterruptCheckpointEntries(t *testing.T) {
	t.Run("suspend-persists-interrupt", func(t *testing.T) {
		sink := &p5g7Sink{}
		g := graph.New(graph.WithDurability(graph.DurabilitySync), graph.WithCheckpointer(sink)).
			AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
			AddNode(graph.NodeFunc("b", func(_ context.Context, in any) (any, error) {
				return nil, graph.RequestInterrupt(graph.Interrupt{
					InterruptID: "i-1", Message: "persist me",
					Payload: map[string]any{"k": "v"},
				})
			})).
			AddEdge("a", "b").
			SetEntry("a").SetOutput("b")
		if _, err := g.Run(context.Background(), "go"); !errors.Is(err, graph.ErrSuspended) {
			t.Fatalf("首跑未挂起: %v", err)
		}
		entries := sink.snapshot()
		if len(entries) != 2 {
			t.Fatalf("sink 收到 %d 条条目, want 2（完成条目 a + 中断条目 b）: %+v", len(entries), entries)
		}
		if entries[0].Kind != graph.EntryCompletion || entries[0].Node != "a" || entries[0].Output != "a#1" {
			t.Errorf("第 1 条 = %+v, want 完成条目 {Seq:1 Node:a Output:a#1}", entries[0])
		}
		cp := entries[1]
		if cp.Kind != graph.EntryInterrupt {
			t.Fatalf("第 2 条 Kind = %v, want EntryInterrupt", cp.Kind)
		}
		if cp.Node != "b" || cp.Input != "a#1" {
			t.Errorf("中断条目 Node/Input = %q/%v, want b/a#1", cp.Node, cp.Input)
		}
		if cp.Interrupt == nil || cp.Interrupt.InterruptID != "i-1" || cp.Interrupt.Message != "persist me" {
			t.Errorf("中断条目的 Interrupt = %+v, want i-1/persist me 全量携带", cp.Interrupt)
		}
		if cp.Seq != 2 {
			t.Errorf("中断条目 Seq = %d, want 2（与完成条目同一条流、Seq 连续）", cp.Seq)
		}
	})
	t.Run("plain-completion-zero-kind", func(t *testing.T) {
		sink := &p5g7Sink{}
		g := graph.New(graph.WithDurability(graph.DurabilitySync), graph.WithCheckpointer(sink)).
			AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
			AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) { return "b#1", nil })).
			AddEdge("a", "b").
			SetEntry("a").SetOutput("b")
		if _, err := g.Run(context.Background(), "go"); err != nil {
			t.Fatalf("普通图执行失败: %v", err)
		}
		for i, cp := range sink.snapshot() {
			if cp.Kind != graph.EntryCompletion || cp.Input != nil || cp.Interrupt != nil {
				t.Errorf("普通完成的第 %d 条 = %+v, want Kind 零值且 Input/Interrupt 为 nil（既有条目形状不变）", i, cp)
			}
		}
	})
}

// TestP5G7_D8_RetryDoesNotSwallowInterrupt 覆盖契约 D8：带 WithRetry(MaxAttempts=5) 的
// 节点第 1 次执行即返回中断 → 恰执行 1 次后挂起（RequestInterrupt 不是可重试错误）。
// 对把中断重试 5 次的实现判红。
func TestP5G7_D8_RetryDoesNotSwallowInterrupt(t *testing.T) {
	calls := p5g7Counter()
	retry := graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
		*calls++
		return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-1"})
	})
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
		AddNode(retry, graph.WithRetry(graph.RetryConfig{
			MaxAttempts: 5,
			ShouldRetry: func(error) bool { return true },
		})).
		AddEdge("a", "b").
		SetEntry("a").SetOutput("b")

	res, err := g.Run(context.Background(), "go")
	if res != nil || !errors.Is(err, graph.ErrSuspended) {
		t.Fatalf("重试包络下的中断 = %v, %v, want 挂起（RequestInterrupt 不进重试循环）", res, err)
	}
	if *calls != 1 {
		t.Errorf("节点体执行次数 = %d, want 1（重试不重发中断）", *calls)
	}
}

// TestP5G7_D9_ParallelInterruptsThenJoin 覆盖契约 D9：两分支各提出中断 → Suspension 含
// 2 个待答；一次 Resume 双答 → 两路继续、Join 照常汇聚。对丢失并发中断之一的实现判红。
// 清单次序不断言（契约 observabilityLimit 第 2 条），只断言集合形状。
func TestP5G7_D9_ParallelInterruptsThenJoin(t *testing.T) {
	branch := func(id string) graph.Node {
		return graph.NodeFunc(id, func(ctx context.Context, in any) (any, error) {
			if v, ok := graph.InterruptResponse(ctx, "i-"+id); ok {
				return id + ":" + v.(string), nil
			}
			return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-" + id, Message: "branch " + id})
		})
	}
	var joined map[string]any
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
		AddNode(branch("b")).
		AddNode(branch("c")).
		AddNode(graph.NodeFunc("d", func(_ context.Context, in any) (any, error) { joined = in.(map[string]any); return "joined", nil })).
		AddEdge("a", "b").AddEdge("a", "c").
		AddJoin([]string{"b", "c"}, "d").
		SetEntry("a").SetOutput("d")

	_, err := g.Run(context.Background(), "go")
	var susp *graph.Suspension
	if !errors.Is(err, graph.ErrSuspended) || !errors.As(err, &susp) {
		t.Fatalf("并行双中断 = %v, want 一次挂起携带两个待答", err)
	}
	if len(susp.Interrupts) != 2 {
		t.Fatalf("待答数 = %d, want 2: %+v", len(susp.Interrupts), susp.Interrupts)
	}
	ids := map[string]bool{}
	nodes := map[string]bool{}
	for _, w := range susp.Interrupts {
		ids[w.Interrupt.InterruptID] = true
		nodes[w.Node] = true
	}
	if !ids["i-b"] || !ids["i-c"] {
		t.Errorf("待答 ID 集 = %v, want {i-b, i-c}（并发中断一个都不丢）", ids)
	}
	if !nodes["b"] || !nodes["c"] {
		t.Errorf("等待节点集 = %v, want {b, c}", nodes)
	}
	res, err := g.Resume(context.Background(), map[string]any{"i-b": "yes", "i-c": "yes"})
	if err != nil {
		t.Fatalf("双答 Resume 失败: %v", err)
	}
	if got, want := res.Output(), "joined"; got != want {
		t.Fatalf("Output = %v, want %v", got, want)
	}
	if joined["b"] != "b:yes" || joined["c"] != "c:yes" {
		t.Errorf("Join 聚合输入 = %v, want 含 b:yes 与 c:yes（两路继续、Join 完整）", joined)
	}
}

// TestP5G7_D10_ResuspensionChain 覆盖契约 D10：恢复后的执行可再次提出中断（不同 ID）
// → 新 Suspension；可继续恢复至完成。对第二次挂起当错误的实现判红。
func TestP5G7_D10_ResuspensionChain(t *testing.T) {
	calls := p5g7Counter()
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
		AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
			*calls++
			switch *calls {
			case 1:
				return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-1"})
			case 2:
				return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-2"})
			default:
				return "done", nil
			}
		})).
		AddEdge("a", "b").
		SetEntry("a").SetOutput("b")

	_, err := g.Run(context.Background(), "go")
	if !errors.Is(err, graph.ErrSuspended) {
		t.Fatalf("第一次挂起 = %v", err)
	}
	_, err = g.Resume(context.Background(), map[string]any{"i-1": "r1"})
	var susp *graph.Suspension
	if !errors.Is(err, graph.ErrSuspended) || !errors.As(err, &susp) {
		t.Fatalf("恢复后的再挂起 = %v, want 新的 *Suspension（重悬链）", err)
	}
	if len(susp.Interrupts) != 1 || susp.Interrupts[0].Interrupt.InterruptID != "i-2" {
		t.Fatalf("新挂起的待答 = %+v, want 恰 i-2", susp.Interrupts)
	}
	res, err := g.Resume(context.Background(), map[string]any{"i-2": "r2"})
	if err != nil {
		t.Fatalf("第二次恢复失败: %v", err)
	}
	if got, want := res.Output(), "done"; got != want {
		t.Errorf("Output = %v, want %v", got, want)
	}
	if *calls != 3 {
		t.Errorf("节点体执行次数 = %d, want 3", *calls)
	}
}

// TestP5G7_D11_InvalidInterruptIDs 覆盖契约 D11：空 InterruptID → 运行期点名错误（不是
// 挂起、不是 panic）；并发重复 InterruptID → 运行期点名两个节点名。对静默接受空/重 ID
// 的实现判红。
func TestP5G7_D11_InvalidInterruptIDs(t *testing.T) {
	t.Run("empty-id", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
			AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
				return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: ""})
			})).
			AddEdge("a", "b").
			SetEntry("a").SetOutput("b")
		res, err := g.Run(context.Background(), "go")
		if res != nil {
			t.Fatalf("空 ID 时仍返回 Result %v", res)
		}
		if err == nil || errors.Is(err, graph.ErrSuspended) {
			t.Fatalf("空 InterruptID 的 Run = %v, want 运行期点名错误（不是挂起）", err)
		}
		if !strings.Contains(err.Error(), "b") {
			t.Errorf("错误文案 = %v, want 点名节点 b", err)
		}
	})
	t.Run("duplicate-id", func(t *testing.T) {
		interrupt := func(_ context.Context, _ any) (any, error) {
			return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "same-id"})
		}
		g := graph.New().
			AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
			AddNode(graph.NodeFunc("b", interrupt)).
			AddNode(graph.NodeFunc("c", interrupt)).
			AddEdge("a", "b").AddEdge("a", "c").
			SetEntry("a").SetOutput("a")
		res, err := g.Run(context.Background(), "go")
		if res != nil {
			t.Fatalf("重复 ID 时仍返回 Result %v", res)
		}
		if err == nil || errors.Is(err, graph.ErrSuspended) {
			t.Fatalf("并发重复 InterruptID 的 Run = %v, want 运行期点名错误（不是挂起）", err)
		}
		if !strings.Contains(err.Error(), "b") || !strings.Contains(err.Error(), "c") {
			t.Errorf("错误文案 = %v, want 同时点名两个节点名 b 与 c", err)
		}
	})
}

// TestP5G7_D12_StepBudgetContinuesAcrossResume 覆盖契约 D12：步数预算跨恢复累计——
// 一次逻辑运行（Run + Resume）共享一个 stepLimit；恢复段撞阀 → ErrStepLimitExceeded。
// 对每次 Resume 重置预算的实现判红。
func TestP5G7_D12_StepBudgetContinuesAcrossResume(t *testing.T) {
	calls := p5g7Counter()
	g := graph.New(graph.WithStepLimit(3)).
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
		AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
			*calls++
			if *calls == 1 {
				return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-1"})
			}
			return "b#1", nil
		})).
		AddEdge("a", "b").
		AddConditional("b", "a", func(_ any) bool { return true }).
		SetEntry("a").SetOutput("a")

	// Run 段：a 一步、b 一步（中断）→ 挂起，预算已花 2/3。
	if _, err := g.Run(context.Background(), "go"); !errors.Is(err, graph.ErrSuspended) {
		t.Fatalf("首跑未挂起: %v", err)
	}
	// Resume 段：b 重跑是第 3 步，它激活的 a 是第 4 步 → 撞同一个预算阀。
	_, err := g.Resume(context.Background(), map[string]any{"i-1": "go"})
	if !errors.Is(err, graph.ErrStepLimitExceeded) {
		t.Fatalf("恢复段 = %v, want errors.Is(ErrStepLimitExceeded)（预算跨恢复累计）", err)
	}
	if errors.Is(err, graph.ErrSuspended) {
		t.Errorf("撞阀不是挂起：错误不应可 Is 到 ErrSuspended")
	}
	if *calls != 2 {
		t.Errorf("b 执行次数 = %d, want 2（重跑发生，撞阀发生在其后的 a 派发前）", *calls)
	}
}

// TestP5G7_D13_SinkFailureAndExitFlush 覆盖契约 D13：挂起时 sink 失败 → 返回的错误同时
// errors.Is 到 ErrSuspended 与 sink 错误（不互掩）；Exit 档挂起照常冲刷（条目完整）。
// 对吞 sink 错误或吞挂起的实现判红。
func TestP5G7_D13_SinkFailureAndExitFlush(t *testing.T) {
	t.Run("sync-sink-failure-both-visible", func(t *testing.T) {
		sink := &p5g7Sink{failOn: func(cp graph.Checkpoint) bool { return cp.Kind == graph.EntryInterrupt }}
		g := graph.New(graph.WithDurability(graph.DurabilitySync), graph.WithCheckpointer(sink)).
			AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
			AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
				return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-1"})
			})).
			AddEdge("a", "b").
			SetEntry("a").SetOutput("b")
		res, err := g.Run(context.Background(), "go")
		if res != nil {
			t.Fatalf("挂起时仍返回 Result %v", res)
		}
		if !errors.Is(err, graph.ErrSuspended) {
			t.Fatalf("错误 = %v, want errors.Is(ErrSuspended)（sink 失败不吞挂起）", err)
		}
		if !errors.Is(err, errP5G7Sink) {
			t.Fatalf("错误 = %v, want errors.Is(errP5G7Sink)（挂起不吞 sink 错误）", err)
		}
		var susp *graph.Suspension
		if !errors.As(err, &susp) || len(susp.Interrupts) != 1 {
			t.Fatalf("errors.As(*Suspension) 不成立或待答丢失: %v", err)
		}
	})
	t.Run("exit-flush-on-suspension", func(t *testing.T) {
		sink := &p5g7Sink{}
		g := graph.New(graph.WithDurability(graph.DurabilityExit), graph.WithCheckpointer(sink)).
			AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "a#1", nil })).
			AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
				return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-1"})
			})).
			AddEdge("a", "b").
			SetEntry("a").SetOutput("b")
		if _, err := g.Run(context.Background(), "go"); !errors.Is(err, graph.ErrSuspended) {
			t.Fatalf("首跑未挂起: %v", err)
		}
		entries := sink.snapshot()
		if len(entries) != 2 {
			t.Fatalf("Exit 档挂起后 sink 收到 %d 条条目, want 2（挂起照常冲刷）: %+v", len(entries), entries)
		}
		if entries[0].Node != "a" || entries[1].Kind != graph.EntryInterrupt {
			t.Errorf("Exit 条目 = %+v, want [完成 a, 中断 b] 完整", entries)
		}
	})
}

// TestP5G7_D14_NonInterruptUnchanged 覆盖契约 D14（反向对照）：不含 RequestInterrupt 的
// 图在带缝引擎上行为同今天——正常交 Result、Resume 无处可恢复。实现前后都绿。
func TestP5G7_D14_NonInterruptUnchanged(t *testing.T) {
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(_ context.Context, in any) (any, error) { return in.(string) + "->a", nil })).
		AddNode(graph.NodeFunc("b", func(_ context.Context, in any) (any, error) { return in.(string) + "->b", nil })).
		AddEdge("a", "b").
		SetEntry("a").SetOutput("b")
	res, err := g.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("普通图执行失败: %v", err)
	}
	if got, want := res.Output(), "go->a->b"; got != want {
		t.Errorf("Output = %v, want %v", got, want)
	}
	if got, want := res.Completed(), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Completed = %v, want %v", got, want)
	}
	if _, err := g.Resume(context.Background(), nil); !errors.Is(err, graph.ErrNothingToResume) {
		t.Errorf("普通图上的 Resume = %v, want ErrNothingToResume", err)
	}
}

// TestP5G7_D15_ResponseAccessor 覆盖契约 D15：重跑的节点能经引擎提供的公共取值器
// InterruptResponse 拿到自己中断的响应（不靠闭包外带）；普通 Run 的 ctx 恒查不到值。
// 对只能靠夹具闭包传响应的实现判红（公共面缺取值器）。
func TestP5G7_D15_ResponseAccessor(t *testing.T) {
	var plainRunSaw bool
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(ctx context.Context, _ any) (any, error) {
			_, ok := graph.InterruptResponse(ctx, "i-1")
			plainRunSaw = ok
			return "a#1", nil
		})).
		AddNode(graph.NodeFunc("b", func(ctx context.Context, _ any) (any, error) {
			v, ok := graph.InterruptResponse(ctx, "i-1")
			if !ok {
				return nil, graph.RequestInterrupt(graph.Interrupt{InterruptID: "i-1"})
			}
			return fmt.Sprintf("got %v", v), nil
		})).
		AddEdge("a", "b").
		SetEntry("a").SetOutput("b")

	if _, err := g.Run(context.Background(), "go"); !errors.Is(err, graph.ErrSuspended) {
		t.Fatalf("首跑未挂起: %v", err)
	}
	if plainRunSaw {
		t.Errorf("普通 Run 的 ctx 查到了响应, want 恒查不到")
	}
	res, err := g.Resume(context.Background(), map[string]any{"i-1": "the-answer"})
	if err != nil {
		t.Fatalf("Resume 失败: %v", err)
	}
	if got, want := res.Output(), "got the-answer"; got != want {
		t.Errorf("Output = %v, want %v（重跑体经公共取值器拿到响应）", got, want)
	}
}
