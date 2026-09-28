package graph_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// P2 第 1 片（母约 §4 G3：节点策略四合一）：
// 契约 docs/design/v3-test-scope-p1-graph-slice22.json 的 D1–D14 逐行一测。
//
// 观察面只有导出 API：WithRetry/WithTimeout/WithCache/WithTrace 挂载、Run 的返回值、
// 夹具自己的调用计数与内存 CacheStore、Hook 收到的 NodeEvent。重试次数等运行期事实
// 全部由夹具自身状态计数，不触碰引擎内部（票面 :40）。
//
// 时序判据按契约 observabilityLimit 只断言错误类型与次数，不断言真实耗时；
// 需要期限的节点都先检查 ctx.Deadline()，无期限时立即报错返回，保证 RED 阶段快速判红。

// p2g3Counted 构造一个自带调用计数（第 n 次调用的 n 作为 call 传入）的节点。
func p2g3Counted(name string, fn func(ctx context.Context, in any, call int) (any, error)) (graph.Node, *int) {
	calls := 0
	n := graph.NodeFunc(name, func(ctx context.Context, in any) (any, error) {
		calls++
		return fn(ctx, in, calls)
	})
	return n, &calls
}

// p2g3AlwaysRetry 是「除 deadline 外全部可重试」的 ShouldRetry。
func p2g3AlwaysRetry(err error) bool { return !errors.Is(err, context.DeadlineExceeded) }

// p2g3MemStore 是夹具自备的内存 CacheStore（TTL 语义由 Store 承载，本夹具不过期）。
type p2g3MemStore struct {
	mu sync.Mutex
	m  map[string]any
}

func newP2g3MemStore() *p2g3MemStore { return &p2g3MemStore{m: map[string]any{}} }

func (s *p2g3MemStore) GetAny(_ context.Context, k string) (any, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[k]
	return v, ok, nil
}

func (s *p2g3MemStore) SetAny(_ context.Context, k string, v any, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[k] = v
	return nil
}

// p2g3Collector 串行收集 trace 事件（Hook 只被消费者 goroutine 调用，仍加锁防测试自身竞争）。
type p2g3Collector struct {
	mu sync.Mutex
	ev []graph.NodeEvent
}

func (c *p2g3Collector) hook(ev graph.NodeEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ev = append(c.ev, ev)
}

func (c *p2g3Collector) snapshot() []graph.NodeEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]graph.NodeEvent(nil), c.ev...)
}

// TestP2G3_RetryRecoversUntilSuccess 观察契约 D1：前两次失败第三次成功、MaxAttempts=3
// 且 ShouldRetry 恒真 → 恰执行 3 次，Run 成功并交出最后一次的返回值。
func TestP2G3_RetryRecoversUntilSuccess(t *testing.T) {
	n, calls := p2g3Counted("flaky", func(_ context.Context, _ any, call int) (any, error) {
		if call < 3 {
			return nil, fmt.Errorf("transient %d", call)
		}
		return "recovered", nil
	})
	g := graph.New().AddNode(n, graph.WithRetry(graph.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		ShouldRetry:  p2g3AlwaysRetry,
	})).SetEntry("flaky").SetOutput("flaky")

	res, err := g.Run(context.Background(), nil)
	if *calls != 3 {
		t.Fatalf("节点执行了 %d 次，want 恰好 3 次（今天失败即停 calls=1，重试未生效）", *calls)
	}
	if err != nil {
		t.Fatalf("重试后 Run 仍失败（%v）", err)
	}
	if res == nil || res.Output() != "recovered" {
		t.Fatalf("Output()=%v（res nil=%v），want 最后一次尝试的返回值 recovered", res.Output(), res == nil)
	}
}

// TestP2G3_RetryZeroValueDoesNotRetry 观察契约 D2（反向对照）：不声明 WithRetry 的失败节点
// 恰执行 1 次、错误原样交出 —— 「不重试」是零值直接给出的语义。
func TestP2G3_RetryZeroValueDoesNotRetry(t *testing.T) {
	n, calls := p2g3Counted("plain", func(_ context.Context, _ any, _ int) (any, error) {
		return nil, errors.New("boom once")
	})
	g := graph.New().AddNode(n).SetEntry("plain").SetOutput("plain")

	_, err := g.Run(context.Background(), nil)
	if *calls != 1 {
		t.Fatalf("未声明重试的节点执行了 %d 次，want 恰好 1 次（默认不得重试）", *calls)
	}
	if err == nil || err.Error() != "boom once" {
		t.Fatalf("Run 交出 %v，want 节点错误原样透传", err)
	}
}

// TestP2G3_RetryExhaustedReturnsLastError 观察契约 D3：恒失败、MaxAttempts=2 →
// 恰执行 2 次，交出最后一次的错误（不是第一次的）。
func TestP2G3_RetryExhaustedReturnsLastError(t *testing.T) {
	n, calls := p2g3Counted("hopeless", func(_ context.Context, _ any, call int) (any, error) {
		return nil, fmt.Errorf("still broken %d", call)
	})
	g := graph.New().AddNode(n, graph.WithRetry(graph.RetryConfig{
		MaxAttempts:  2,
		InitialDelay: time.Millisecond,
		ShouldRetry:  p2g3AlwaysRetry,
	})).SetEntry("hopeless").SetOutput("hopeless")

	_, err := g.Run(context.Background(), nil)
	if *calls != 2 {
		t.Fatalf("节点执行了 %d 次，want 恰好 2 次（上限钳制失效或 off-by-one）", *calls)
	}
	if err == nil || !strings.Contains(err.Error(), "still broken 2") {
		t.Fatalf("Run 交出 %v，want 最后一次尝试的错误 still broken 2", err)
	}
}

// TestP2G3_ShouldRetryGatesAndNilMeansNever 观察契约 D4：ShouldRetry 对首错返回 false →
// 只执行 1 次；ShouldRetry 为 nil 时即便 MaxAttempts=3 也只执行 1 次（fail-closed，
// 引擎不替调用方分类可重试错误）。
func TestP2G3_ShouldRetryGatesAndNilMeansNever(t *testing.T) {
	t.Run("shouldretry 拒绝首错", func(t *testing.T) {
		n, calls := p2g3Counted("gated", func(_ context.Context, _ any, _ int) (any, error) {
			return nil, errors.New("deterministic")
		})
		g := graph.New().AddNode(n, graph.WithRetry(graph.RetryConfig{
			MaxAttempts: 3,
			ShouldRetry: func(error) bool { return false },
		})).SetEntry("gated").SetOutput("gated")
		_, err := g.Run(context.Background(), nil)
		if *calls != 1 {
			t.Fatalf("ShouldRetry 拒绝后仍执行了 %d 次，want 1", *calls)
		}
		if err == nil || err.Error() != "deterministic" {
			t.Fatalf("Run 交出 %v，want 原错误", err)
		}
	})
	t.Run("nil 等于从不重试", func(t *testing.T) {
		n, calls := p2g3Counted("nilpolicy", func(_ context.Context, _ any, _ int) (any, error) {
			return nil, errors.New("no classifier")
		})
		g := graph.New().AddNode(n, graph.WithRetry(graph.RetryConfig{MaxAttempts: 3})).
			SetEntry("nilpolicy").SetOutput("nilpolicy")
		_, err := g.Run(context.Background(), nil)
		if *calls != 1 {
			t.Fatalf("ShouldRetry 为 nil 时执行了 %d 次，want 1（nil 不得当成全部重试）", *calls)
		}
		if err == nil || err.Error() != "no classifier" {
			t.Fatalf("Run 交出 %v，want 原错误", err)
		}
	})
}

// TestP2G3_RetryDoesNotConsumeStepBudget 观察契约 D5：单节点图 stepLimit=1、重试 3 次
// 第三次成功 → Run 成功。重试的是「同一次激活的执行」，不是新激活；若每次尝试计一步，
// 第二步就会撞 ErrStepLimitExceeded。
func TestP2G3_RetryDoesNotConsumeStepBudget(t *testing.T) {
	n, calls := p2g3Counted("budgeted", func(_ context.Context, _ any, call int) (any, error) {
		if call < 3 {
			return nil, errors.New("transient")
		}
		return "ok", nil
	})
	g := graph.New(graph.WithStepLimit(1)).AddNode(n, graph.WithRetry(graph.RetryConfig{
		MaxAttempts: 3,
		ShouldRetry: p2g3AlwaysRetry,
	})).SetEntry("budgeted").SetOutput("budgeted")

	res, err := g.Run(context.Background(), nil)
	if *calls != 3 {
		t.Fatalf("节点执行了 %d 次，want 3（重试没有生效）", *calls)
	}
	if err != nil {
		t.Fatalf("步数预算为 1 的图上重试失败（%v）：重试疑似被计成了激活", err)
	}
	if res == nil || res.Output() != "ok" {
		t.Fatalf("Output()=%v，want ok", res.Output())
	}
}

// TestP2G3_TimeoutCancelsNodeWithDeadline 观察契约 D6：Timeout=20ms、无重试 →
// 节点收到带期限的 ctx 并交出 DeadlineExceeded，Run 交出可 errors.Is 到它的错误，恰执行 1 次。
// 不断言真实耗时（契约 observabilityLimit）。
func TestP2G3_TimeoutCancelsNodeWithDeadline(t *testing.T) {
	n, calls := p2g3Counted("slow", func(ctx context.Context, _ any, _ int) (any, error) {
		if _, ok := ctx.Deadline(); !ok {
			return nil, errors.New("no deadline reached the node")
		}
		<-ctx.Done()
		return nil, ctx.Err()
	})
	g := graph.New().AddNode(n, graph.WithTimeout(graph.TimeoutConfig{Timeout: 20 * time.Millisecond})).
		SetEntry("slow").SetOutput("slow")

	_, err := g.Run(context.Background(), nil)
	if *calls != 1 {
		t.Fatalf("节点执行了 %d 次，want 1（未声明重试）", *calls)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run 交出 %v，want 可归因到 context.DeadlineExceeded（今天 ctx 无期限，节点根本等不到取消）", err)
	}
}

// TestP2G3_NoTimeoutKeepsCallerContext 观察契约 D7（反向对照）：不声明 WithTimeout →
// 节点拿到的 ctx 与调用方同形（没有新增 deadline），照常执行。
func TestP2G3_NoTimeoutKeepsCallerContext(t *testing.T) {
	var sawDeadline bool
	n, _ := p2g3Counted("plain", func(ctx context.Context, _ any, _ int) (any, error) {
		_, ok := ctx.Deadline()
		sawDeadline = ok
		return "done", nil
	})
	g := graph.New().AddNode(n).SetEntry("plain").SetOutput("plain")

	res, err := g.Run(context.Background(), nil)
	if sawDeadline {
		t.Fatalf("未声明超时的节点拿到了 deadline：引擎不得给所有节点默认加期限")
	}
	if err != nil || res == nil || res.Output() != "done" {
		t.Fatalf("Run 失败（err=%v out=%v），照常执行不得被殃及", err, res.Output())
	}
}

// TestP2G3_PerAttemptDeadlineFreshEachTry 观察契约 D8：PerAttempt=true、Timeout=15ms、
// Retry MaxAttempts=2 且 ShouldRetry 接受 DeadlineExceeded → 两次尝试各自拿到新期限
// （夹具记录每次的 deadline 值，严格递增），恰执行 2 次。
func TestP2G3_PerAttemptDeadlineFreshEachTry(t *testing.T) {
	var mu sync.Mutex
	var deadlines []time.Time
	n, calls := p2g3Counted("slowretry", func(ctx context.Context, _ any, _ int) (any, error) {
		d, ok := ctx.Deadline()
		if !ok {
			return nil, errors.New("no deadline reached the node")
		}
		mu.Lock()
		deadlines = append(deadlines, d)
		mu.Unlock()
		<-ctx.Done()
		return nil, ctx.Err()
	})
	g := graph.New().AddNode(n,
		graph.WithTimeout(graph.TimeoutConfig{Timeout: 15 * time.Millisecond, PerAttempt: true}),
		graph.WithRetry(graph.RetryConfig{
			MaxAttempts:  2,
			InitialDelay: time.Millisecond,
			ShouldRetry:  func(err error) bool { return errors.Is(err, context.DeadlineExceeded) },
		})).SetEntry("slowretry").SetOutput("slowretry")

	_, err := g.Run(context.Background(), nil)
	if *calls != 2 {
		t.Fatalf("节点执行了 %d 次，want 恰好 2（PerAttempt 期限耗尽一次重试一次）", *calls)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run 交出 %v，want 最终仍是 DeadlineExceeded", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(deadlines) != 2 {
		t.Fatalf("夹具记录到 %d 个 deadline，want 2", len(deadlines))
	}
	if !deadlines[1].After(deadlines[0]) {
		t.Fatalf("第二次尝试的 deadline %v 未晚于第一次的 %v：期限疑似只挂了一次", deadlines[1], deadlines[0])
	}
}

// TestP2G3_CacheHitAcrossRuns 观察契约 D9：同输入两次 Run → 节点只真执行 1 次，
// 两次输出同值（默认键从输入值派生，KeyFunc 为 nil）。
func TestP2G3_CacheHitAcrossRuns(t *testing.T) {
	n, calls := p2g3Counted("costly", func(_ context.Context, in any, call int) (any, error) {
		return fmt.Sprintf("computed-%d", call), nil
	})
	g := graph.New().AddNode(n, graph.WithCache(graph.CacheConfig{Store: newP2g3MemStore()})).
		SetEntry("costly").SetOutput("costly")

	r1, err1 := g.Run(context.Background(), "same")
	r2, err2 := g.Run(context.Background(), "same")
	if *calls != 1 {
		t.Fatalf("节点执行了 %d 次，want 1（同键第二次 Run 应命中缓存；今天每次都真跑）", *calls)
	}
	if err1 != nil || err2 != nil {
		t.Fatalf("两次 Run 出错：%v / %v", err1, err2)
	}
	if r1.Output() != r2.Output() {
		t.Fatalf("两次输出不同（%v / %v），命中的应是同一份缓存结论", r1.Output(), r2.Output())
	}
}

// TestP2G3_CacheKeyPartitionsInputs 观察契约 D10：不同输入（不同键）→ 再真执行；
// 同键再次 Run 仍命中。
func TestP2G3_CacheKeyPartitionsInputs(t *testing.T) {
	n, calls := p2g3Counted("keyed", func(_ context.Context, in any, call int) (any, error) {
		return fmt.Sprintf("out-%v-%d", in, call), nil
	})
	g := graph.New().AddNode(n, graph.WithCache(graph.CacheConfig{Store: newP2g3MemStore()})).
		SetEntry("keyed").SetOutput("keyed")

	ctx := context.Background()
	r1, _ := g.Run(ctx, "a")
	_, _ = g.Run(ctx, "b")
	if *calls != 2 {
		t.Fatalf("两个不同键后节点执行了 %d 次，want 2（不同输入不得互相命中）", *calls)
	}
	r3, _ := g.Run(ctx, "a")
	if *calls != 2 {
		t.Fatalf("同键 a 的第三次 Run 又真执行了（calls=%d），want 命中缓存保持 2", *calls)
	}
	if r3.Output() != r1.Output() {
		t.Fatalf("同键的输出不一致（%v / %v）", r3.Output(), r1.Output())
	}
}

// TestP2G3_CacheNeverStoresFailures 观察契约 D11（fail-closed）：失败结果不得入库 ——
// 失败后同键 Run 仍真执行；成功一次之后同键 Run 才命中。
func TestP2G3_CacheNeverStoresFailures(t *testing.T) {
	n, calls := p2g3Counted("flakykey", func(_ context.Context, _ any, call int) (any, error) {
		if call <= 2 {
			return nil, fmt.Errorf("failing %d", call)
		}
		return "value-3", nil
	})
	g := graph.New().AddNode(n, graph.WithCache(graph.CacheConfig{Store: newP2g3MemStore()})).
		SetEntry("flakykey").SetOutput("flakykey")

	ctx := context.Background()
	if _, err := g.Run(ctx, nil); err == nil {
		t.Fatal("第一次 Run 应失败")
	}
	if _, err := g.Run(ctx, nil); err == nil {
		t.Fatal("失败被当成结论缓存了：第二次 Run 不该成功")
	}
	if *calls != 2 {
		t.Fatalf("失败后节点执行了 %d 次，want 2（失败绝不能顶替真执行）", *calls)
	}
	r3, err3 := g.Run(ctx, nil)
	if err3 != nil || r3.Output() != "value-3" {
		t.Fatalf("第三次 Run 应真执行并成功（calls=%d err=%v out=%v）", *calls, err3, r3.Output())
	}
	r4, err4 := g.Run(ctx, nil)
	if *calls != 3 {
		t.Fatalf("成功后同键 Run 又真执行了（calls=%d），want 命中缓存", *calls)
	}
	if err4 != nil || r4.Output() != "value-3" {
		t.Fatalf("缓存命中的输出应是 value-3（err=%v out=%v）", err4, r4.Output())
	}
}

// TestP2G3_TraceEmitsNodeEvents 观察契约 D12：Enabled=true → 每个节点完成时 Hook 恰收到
// 一条事件，含节点名、Attempt 与未脱敏的 In/Out。
func TestP2G3_TraceEmitsNodeEvents(t *testing.T) {
	col := &p2g3Collector{}
	g := graph.New().AddNode(
		graph.NodeFunc("src", func(_ context.Context, in any) (any, error) { return "payload", nil }),
		graph.WithTrace(graph.TraceConfig{Enabled: true, Hook: col.hook}),
	).SetEntry("src").SetOutput("src")

	if _, err := g.Run(context.Background(), "greeting"); err != nil {
		t.Fatalf("Run 失败（%v）", err)
	}
	ev := col.snapshot()
	if len(ev) != 1 {
		t.Fatalf("收到 %d 条事件，want 恰 1（今天导出面上没有任何 sink，事件数为 0）", len(ev))
	}
	if ev[0].Node != "src" {
		t.Errorf("事件节点名 %q，want src", ev[0].Node)
	}
	if ev[0].In != "greeting" || ev[0].Out != "payload" {
		t.Errorf("未脱敏时 In/Out 应可见（in=%v out=%v）", ev[0].In, ev[0].Out)
	}
	if ev[0].Attempt < 1 {
		t.Errorf("Attempt=%d，want >=1", ev[0].Attempt)
	}
}

// TestP2G3_TraceRedaction 观察契约 D13：RedactIn/RedactOut=true → 事件仍送达，
// 但 In/Out 均不可见（脱敏在发射侧完成）。
func TestP2G3_TraceRedaction(t *testing.T) {
	col := &p2g3Collector{}
	g := graph.New().AddNode(
		graph.NodeFunc("src", func(_ context.Context, in any) (any, error) { return "secret", nil }),
		graph.WithTrace(graph.TraceConfig{Enabled: true, RedactIn: true, RedactOut: true, Hook: col.hook}),
	).SetEntry("src").SetOutput("src")

	if _, err := g.Run(context.Background(), "classified"); err != nil {
		t.Fatalf("Run 失败（%v）", err)
	}
	ev := col.snapshot()
	if len(ev) != 1 {
		t.Fatalf("收到 %d 条事件，want 1（脱敏不得把事件本身弄丢）", len(ev))
	}
	if ev[0].In != nil || ev[0].Out != nil {
		t.Errorf("脱敏后 In/Out 应为 nil（in=%v out=%v）", ev[0].In, ev[0].Out)
	}
}

// TestP2G3_TraceDisabledOrNilHookStaysSilent 观察契约 D14（反向对照）：
// Enabled=false → Hook 零调用；Enabled=true 但 Hook 为 nil → Run 照常成功（引擎必须容忍空 sink）。
func TestP2G3_TraceDisabledOrNilHookStaysSilent(t *testing.T) {
	t.Run("enabled false 零事件", func(t *testing.T) {
		col := &p2g3Collector{}
		g := graph.New().AddNode(
			graph.NodeFunc("src", func(_ context.Context, in any) (any, error) { return "x", nil }),
			graph.WithTrace(graph.TraceConfig{Enabled: false, Hook: col.hook}),
		).SetEntry("src").SetOutput("src")
		if _, err := g.Run(context.Background(), nil); err != nil {
			t.Fatalf("Run 失败（%v）", err)
		}
		if ev := col.snapshot(); len(ev) != 0 {
			t.Fatalf("Enabled=false 仍发出 %d 条事件，want 0", len(ev))
		}
	})
	t.Run("nil hook 不炸", func(t *testing.T) {
		g := graph.New().AddNode(
			graph.NodeFunc("src", func(_ context.Context, in any) (any, error) { return "x", nil }),
			graph.WithTrace(graph.TraceConfig{Enabled: true}),
		).SetEntry("src").SetOutput("src")
		res, err := g.Run(context.Background(), nil)
		if err != nil || res == nil || res.Output() != "x" {
			t.Fatalf("空 sink 不得影响执行（err=%v out=%v）", err, res.Output())
		}
	})
}
