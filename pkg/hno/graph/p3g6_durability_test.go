package graph_test

// 本文件是切片 24（母约 §7 G6：Durability 持久化档位）的行为族，对应契约
// docs/design/v3-test-scope-p1-graph-slice24.json（D1–D9）。与既有各片同构：
// 单独成文件让本片的行为族与文件边界对齐，便于按切片审差分。
//
// 观察面守契约 forbiddenObservations：提交流的唯一证据是夹具 sink 收到的条目
// 与 Run 的公共返回值；scheduler 内部字段（seq/commits/exitQueue/flushDone）、
// channel 容量、goroutine 数、time.Sleep 定序都不是本族的证据。Async 档不对
// mid-run 可见性作任何断言（契约 observabilityLimit 第 1 条），由 D3 的正向行
// 与 D5 的落齐行共同夹住。
//
// 夹具自备内存 Checkpointer：互斥锁属夹具不属引擎——Sync/Exit 的 Append 在
// 消费者 goroutine、Async 在冲刷 goroutine、节点体采样在生产者 goroutine，
// 夹具自己保证并发安全（契约 allowedTestSeam.add）。

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// errP3G6Sink 是 D7 用的夹具错误：sink 拒绝提交时交出的可归因错误。
var errP3G6Sink = errors.New("p3g6 fixture: sink refused")

// memSink 收集提交条目；failFirst 让前 N 次 Append 返回夹具错误（D7）。
type memSink struct {
	mu        sync.Mutex
	entries   []graph.Checkpoint
	calls     int
	failFirst int
}

func (m *memSink) Append(_ context.Context, cp graph.Checkpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.calls <= m.failFirst {
		return errP3G6Sink
	}
	m.entries = append(m.entries, cp)
	return nil
}

func (m *memSink) snapshot() []graph.Checkpoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]graph.Checkpoint, len(m.entries))
	copy(out, m.entries)
	return out
}

func (m *memSink) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}

func (m *memSink) callsMade() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// sinkSamples 记录每个节点体启动时 sink 已见条目数。它是 D2/D3（下一步开始前
// 已落）与 D4（运行中零条目）的公共观察面：只读夹具 sink 的条目计数，不触碰
// 引擎内部状态。
type sinkSamples struct {
	mu sync.Mutex
	at map[string]int
}

func newSinkSamples() *sinkSamples { return &sinkSamples{at: map[string]int{}} }

func (s *sinkSamples) record(name string, n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.at[name] = n
}

func (s *sinkSamples) get(name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.at[name]
}

// p3g6Linear 构造 a→b→out 线性图：每个节点返回 "名字#1"，节点体启动时采样 sink
// 已见条目数。sink 非 nil 时挂 WithCheckpointer（放在 opts 之前，档位 Option 由
// 调用方追加）；samples 非 nil 时记录采样。
func p3g6Linear(sink *memSink, samples *sinkSamples, opts ...graph.Option) *graph.Graph {
	node := func(name string) graph.Node {
		return graph.NodeFunc(name, func(_ context.Context, in any) (any, error) {
			if samples != nil && sink != nil {
				samples.record(name, sink.count())
			}
			return name + "#1", nil
		})
	}
	if sink != nil {
		opts = append([]graph.Option{graph.WithCheckpointer(sink)}, opts...)
	}
	return graph.New(opts...).
		AddNode(node("a")).
		AddNode(node("b")).
		AddNode(node("out")).
		AddEdge("a", "b").
		AddEdge("b", "out").
		SetEntry("a").
		SetOutput("out")
}

// wantLinear 断言线性三步图收敛后的条目形状：恰 3 条、Seq 从 1 连续递增、
// Node 按完成次序 a,b,out。
func wantLinear(t *testing.T, entries []graph.Checkpoint) {
	t.Helper()
	if len(entries) != 3 {
		t.Fatalf("sink 收到 %d 条条目, want 恰 3 条（每完成节点恰一条）: %v", len(entries), entries)
	}
	wantNodes := []string{"a", "b", "out"}
	for i, cp := range entries {
		if cp.Seq != i+1 {
			t.Errorf("第 %d 条条目 Seq = %d, want %d（Seq 由 1 连续递增）", i, cp.Seq, i+1)
		}
		if cp.Node != wantNodes[i] {
			t.Errorf("第 %d 条条目 Node = %q, want %q", i, cp.Node, wantNodes[i])
		}
	}
}

// TestP3G6_D1_ConstantsOrderAndSymbols 覆盖契约 D1：三常量存在、iota 序
// Sync<Async<Exit、零值即 DurabilitySync（默认档=最安全档由零值直接给出）；
// WithDurability/WithCheckpointer/Checkpointer/Checkpoint 四个符号存在——
// 符号存在性同时由本文件整体编译钉住（实现前是本片唯一允许的编译红）。
func TestP3G6_D1_ConstantsOrderAndSymbols(t *testing.T) {
	if !(graph.DurabilitySync < graph.DurabilityAsync && graph.DurabilityAsync < graph.DurabilityExit) {
		t.Fatalf("Durability 常量 = %d,%d,%d, want Sync<Async<Exit 的 iota 序",
			graph.DurabilitySync, graph.DurabilityAsync, graph.DurabilityExit)
	}
	if graph.Durability(0) != graph.DurabilitySync {
		t.Fatalf("Durability(0) = %v, want DurabilitySync（默认档由零值直接给出，不新增 Validate 行）", graph.Durability(0))
	}
	var _ graph.Checkpointer = (*memSink)(nil)
	cp := graph.Checkpoint{Seq: 1, Node: "a", Output: "x"}
	if cp.Seq != 1 || cp.Node != "a" || cp.Output != "x" {
		t.Fatalf("Checkpoint 三字段读写不通: %+v", cp)
	}
	_ = graph.WithDurability(graph.DurabilityExit)
	_ = graph.WithCheckpointer(nil)
}

// TestP3G6_D2_DefaultTierCommitsLikeSync 覆盖契约 D2：不声明 WithDurability、只挂
// sink 的线性图，默认档行为 = Sync——后继节点体执行时 sink 已见前驱条目，Run 返回后
// 条目序 1:a,2:b,3:out。默认档语义是行为钉，不读私有 config。
func TestP3G6_D2_DefaultTierCommitsLikeSync(t *testing.T) {
	sink := &memSink{}
	samples := newSinkSamples()
	g := p3g6Linear(sink, samples)
	res, err := g.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("默认档线性图执行失败: %v", err)
	}
	if got := samples.get("b"); got < 1 {
		t.Fatalf("后继节点 b 的节点体采样 = %d, want ≥1：默认档必须像 Sync 一样在下一步开始前落盘", got)
	}
	wantLinear(t, sink.snapshot())
	if got, want := res.Output(), "out#1"; got != want {
		t.Errorf("Result.Output() = %v, want %v", got, want)
	}
}

// TestP3G6_D3_SyncTierCommitsBeforeNextStep 覆盖契约 D3：Sync 档下每个节点完成后、
// 下一步开始前条目已落——b 体采样已见 a（≥1）、out 体采样已见 a,b（≥2）；返回后
// 恰三条、Seq=1,2,3。单链上的采样是确定性的：提交发生在消费者派发后继之前。
func TestP3G6_D3_SyncTierCommitsBeforeNextStep(t *testing.T) {
	sink := &memSink{}
	samples := newSinkSamples()
	g := p3g6Linear(sink, samples, graph.WithDurability(graph.DurabilitySync))
	if _, err := g.Run(context.Background(), "go"); err != nil {
		t.Fatalf("Sync 档线性图执行失败: %v", err)
	}
	if got := samples.get("b"); got < 1 {
		t.Fatalf("b 体采样 = %d, want ≥1：Sync 的提交点在 complete 之后、下一轮 dispatch 之前", got)
	}
	if got := samples.get("out"); got < 2 {
		t.Errorf("out 体采样 = %d, want ≥2：单链上每一步开始前，全部前驱条目都已落", got)
	}
	wantLinear(t, sink.snapshot())
}

// TestP3G6_D4_ExitTierHoldsUntilRunExits 覆盖契约 D4：Exit 档运行中 sink 零条目
// （每个节点体采样皆空，尽管前驱已完成），Run 返回前全部条目一次落齐（3 条、Seq 序
// 不乱）。两个方向各自可判红：运行中漏条目（采样非零）与退出不落齐（条目缺失）。
func TestP3G6_D4_ExitTierHoldsUntilRunExits(t *testing.T) {
	sink := &memSink{}
	samples := newSinkSamples()
	g := p3g6Linear(sink, samples, graph.WithDurability(graph.DurabilityExit))
	if _, err := g.Run(context.Background(), "go"); err != nil {
		t.Fatalf("Exit 档线性图执行失败: %v", err)
	}
	for _, name := range []string{"a", "b", "out"} {
		if got := samples.get(name); got != 0 {
			t.Errorf("节点 %s 体采样 = %d, want 0：Exit 档运行中 sink 不得见到任何条目", name, got)
		}
	}
	wantLinear(t, sink.snapshot())
}

// TestP3G6_D5_AsyncTierFlushedBeforeReturn 覆盖契约 D5：Async 档 Run 返回前全部条目
// 落齐（不丢）、Seq 序不乱（不乱）。mid-run 可见性不作断言——「不作承诺」这个负命题
// 没有不脆的断言形状，由 D3 的正向行与本行的落齐行共同夹住。
func TestP3G6_D5_AsyncTierFlushedBeforeReturn(t *testing.T) {
	sink := &memSink{}
	g := p3g6Linear(sink, nil, graph.WithDurability(graph.DurabilityAsync))
	res, err := g.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("Async 档线性图执行失败: %v", err)
	}
	entries := sink.snapshot()
	wantLinear(t, entries)
	gotNodes := make([]string, 0, len(entries))
	for _, cp := range entries {
		gotNodes = append(gotNodes, cp.Node)
	}
	if !reflect.DeepEqual(gotNodes, res.Completed()) {
		t.Errorf("条目 Node 序 = %v, want 与 Result.Completed() = %v 一致", gotNodes, res.Completed())
	}
}

// TestP3G6_D6_EntriesReconcileWithResult 覆盖契约 D6：条目流与 Result 对账（三档共用）
// ——每完成节点恰一条、Seq 从 1 连续无空洞、Node 集合 == Result.Completed()、
// 各条目 Output == Result.Value(该节点)。
func TestP3G6_D6_EntriesReconcileWithResult(t *testing.T) {
	for _, tc := range []struct {
		name string
		tier graph.Durability
	}{
		{"sync", graph.DurabilitySync},
		{"async", graph.DurabilityAsync},
		{"exit", graph.DurabilityExit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sink := &memSink{}
			g := p3g6Linear(sink, nil, graph.WithDurability(tc.tier))
			res, err := g.Run(context.Background(), "go")
			if err != nil {
				t.Fatalf("%s 档线性图执行失败: %v", tc.name, err)
			}
			entries := sink.snapshot()
			if len(entries) != len(res.Completed()) {
				t.Fatalf("条目数 = %d, want len(Completed()) = %d（每完成节点恰一条）", len(entries), len(res.Completed()))
			}
			seen := map[string]bool{}
			for i, cp := range entries {
				if cp.Seq != i+1 {
					t.Errorf("第 %d 条 Seq = %d, want %d（从 1 连续递增无空洞）", i, cp.Seq, i+1)
				}
				if seen[cp.Node] {
					t.Errorf("节点 %q 收到多条条目, want 恰一条", cp.Node)
				}
				seen[cp.Node] = true
				v, ok := res.Value(cp.Node)
				if !ok {
					t.Errorf("条目指向节点 %q, 但 Result.Value 读不到它", cp.Node)
				} else if cp.Output != v {
					t.Errorf("节点 %q 的条目 Output = %v, want 与 Result.Value 同值 %v", cp.Node, cp.Output, v)
				}
			}
			for _, name := range res.Completed() {
				if !seen[name] {
					t.Errorf("已完成节点 %q 没有对应条目", name)
				}
			}
		})
	}
}

// TestP3G6_D7_SinkFailureFailClosed 覆盖契约 D7：sink 失败 fail-closed——Sync 档首次
// Append 返回夹具错误 → Run 交出可 errors.Is 到该错误的非 nil 错误、res=nil；Exit 档
// 在退出落齐时同样上交不被吞。对吞错/只打日志/降级成 err==nil 成功的实现判红。
func TestP3G6_D7_SinkFailureFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name string
		tier graph.Durability
	}{
		{"sync", graph.DurabilitySync},
		{"exit", graph.DurabilityExit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sink := &memSink{failFirst: 1}
			g := p3g6Linear(sink, nil, graph.WithDurability(tc.tier))
			res, err := g.Run(context.Background(), "go")
			if !errors.Is(err, errP3G6Sink) {
				t.Fatalf("%s 档 sink 失败后 Run 的错误 = %v, want errors.Is 到夹具错误（fail-closed）", tc.name, err)
			}
			if res != nil {
				t.Errorf("%s 档 sink 失败后仍返回 Result %v，调用方会把未持久化的执行当成结论", tc.name, res)
			}
			if got := sink.callsMade(); got < 1 {
				t.Errorf("%s 档 sink 被调用 %d 次, want ≥1（sink 根本没被调用说明提交流缺失）", tc.name, got)
			}
		})
	}
}

// TestP3G6_D8_NilCheckpointerNoOp 覆盖契约 D8（反向锚）：声明 WithDurability 三档而
// 未挂 sink → 照常跑完、无 panic、无条目；引擎不存在全局默认 sink，「何时落」由档位
// 回答，「落不落」由 sink 的有无回答。实现前后都绿。
func TestP3G6_D8_NilCheckpointerNoOp(t *testing.T) {
	for _, tc := range []struct {
		name string
		tier graph.Durability
	}{
		{"sync-no-sink", graph.DurabilitySync},
		{"async-no-sink", graph.DurabilityAsync},
		{"exit-no-sink", graph.DurabilityExit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := p3g6Linear(nil, nil, graph.WithDurability(tc.tier))
			res, err := g.Run(context.Background(), "go")
			if err != nil {
				t.Fatalf("声明档位而未挂 sink 的图被拒绝或失败: %v", err)
			}
			if got, want := res.Output(), "out#1"; got != want {
				t.Errorf("Result.Output() = %v, want %v", got, want)
			}
			if got, want := res.Completed(), []string{"a", "b", "out"}; !reflect.DeepEqual(got, want) {
				t.Errorf("Completed() = %v, want %v", got, want)
			}
		})
	}
}

// TestP3G6_D9_StepLimitKeepsCommittedEntries 覆盖契约 D9：WithStepLimit(1) 的线性
// 两步图 + Sync → Run 交出可 errors.Is 到 ErrStepLimitExceeded 的错误，且 sink 恰含
// entry(a)（Seq=1）——已提交条目不回滚，未运行的 b 不补条目。安全阀与提交流共存。
func TestP3G6_D9_StepLimitKeepsCommittedEntries(t *testing.T) {
	sink := &memSink{}
	g := graph.New(
		graph.WithStepLimit(1),
		graph.WithDurability(graph.DurabilitySync),
		graph.WithCheckpointer(sink),
	).
		AddNode(graph.NodeFunc("a", func(_ context.Context, in any) (any, error) { return "a#1", nil })).
		AddNode(graph.NodeFunc("b", func(_ context.Context, in any) (any, error) { return "b#1", nil })).
		AddEdge("a", "b").
		SetEntry("a").SetOutput("b")

	res, err := g.Run(context.Background(), "go")
	if !errors.Is(err, graph.ErrStepLimitExceeded) {
		t.Fatalf("Run 的错误 = %v, want errors.Is(err, graph.ErrStepLimitExceeded)", err)
	}
	if res != nil {
		t.Errorf("撞阀后仍返回 Result %v", res)
	}
	entries := sink.snapshot()
	if len(entries) != 1 || entries[0].Seq != 1 || entries[0].Node != "a" {
		t.Errorf("撞阀后 sink 条目 = %+v, want 恰 [1:a]（已提交条目不回滚、未运行节点不补条目）", entries)
	}
}
