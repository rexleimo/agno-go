package graph_test

// 本文件是 P1 票面 B10（节点 panic 转错误）的行为族，与 graph_test.go 同包、复用其
// quotedNames/occurrences/looping/noCount 夹具。票面 §5 允许「新增 pkg/hno/graph 下的
// 同包测试文件」，单独成文件的目的是让这一片的行为族与文件边界对齐，便于按切片审差分。
//
// 观察面守本切片契约 allowedTestSeam：只用 graph 的公共入口搭建，只用调用方拿得到的
// 东西作证据（Run 的两个返回值、自己提供的节点体与计数器）。scheduler 字段、queue
// 长度、running 计数、goroutine 数都不是本族的证据（票面 §1 禁观察面）。
//
// 一条方法论事实：在当前实现下任一 panic 测试都会终止测试二进制，因此每行 RED 必须用
// -run 精确点到本行才能取得可归属的非零回执；包级命令在同一实现下会连同后续测试一起
// 被带走，那正是 T1 要钉住的「测试不中断」子句本身。

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// panicReturnGrace 是「Run 必须自己回来」这条断言的兜底时限，不是定序手段：节点侧的
// 先后一律由 channel gate 握手，时限只用来把「永不返回」从挂死变成可判定的失败。
const panicReturnGrace = 2 * time.Second

// runWithinGrace 在独立 goroutine 里执行 Run，并把「未返回」本身变成测试失败而不是
// 让 go test 的总超时来收拾现场。
func runWithinGrace(t *testing.T, g *graph.Graph, in any) (*graph.Result, error) {
	t.Helper()
	type outcome struct {
		res *graph.Result
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, err := g.Run(context.Background(), in)
		done <- outcome{res, err}
	}()
	select {
	case o := <-done:
		return o.res, o.err
	case <-time.After(panicReturnGrace):
		t.Fatalf("Run 在 %s 内没有返回：panic 被吞掉又没有交给唯一消费者时，running 永不归零，调用方看到的是永久悬挂", panicReturnGrace)
		return nil, nil
	}
}

// panicking 返回一个「立即以 v 崩溃」的节点体。v 由调用方在测试 goroutine 里准备好，
// 因此 panic 的值形态是这一行显式选定的，而不是节点体碰巧抛出的。
func panicking(v any) func(context.Context, any) (any, error) {
	return func(context.Context, any) (any, error) { panic(v) }
}

// TestP1R14_NodePanicBecomesRunError 覆盖契约 T1：节点体内 panic("boom") 时，调用方
// 必须从 Run 拿到一条非 nil error，而不是看着测试二进制被带走。
//
// 这条断言的失败形态本身就是被测行为：票面 B10 写的是「Run 返回非 nil error，进程不崩，
// 测试不中断」，三者是同一件事的三个观察面。
func TestP1R14_NodePanicBecomesRunError(t *testing.T) {
	g := graph.New().
		AddNode(graph.NodeFunc("a", panicking("boom"))).
		SetEntry("a").SetOutput("a")

	res, err := g.Run(context.Background(), "seed")
	if err == nil {
		t.Fatalf("节点 panic 之后 Run 交出 nil error，调用方拿不到任何失败信号（Result=%v）", res)
	}
	if res != nil {
		t.Errorf("panic 之后仍交出 Result %v，与「失败不半交出结果」矛盾", res)
	}
}

// TestP1R14_NonStringPanicValuesAlsoBecomeErrors 覆盖契约 T2：转换不挑 panic 的值形态。
//
// 表里的 error 值与下面 runtime-error 一行才是真实宿主最常见的形状——provider 里向 nil map
// 赋值或一次 nil 解引用抛出的是 runtime.Error，业务代码 panic 的常是一个 error 值。
// 只把 panic(string) 转成错误的实现会在「最少见的那种写法」上绿、在最常见的那种上带走进程。
//
// 断言只钉「Run 交出非 nil error」这一条规格要求：母约 §3.3:182 没规定恢复后是否保留原
// panic 值的 error 身份，本片不为此新增导出面，也不把未规定的形态写进断言。
func TestP1R14_NonStringPanicValuesAlsoBecomeErrors(t *testing.T) {
	cases := []struct {
		name        string
		panicValue  any
		inGraphBody bool // runtime error 必须在节点体内现场发生，不能由表预置
	}{
		{name: "string", panicValue: "boom-string"},
		{name: "error-value", panicValue: errors.New("boom-error")},
		{name: "runtime-error-nil-map", inGraphBody: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := panicking(tc.panicValue)
			if tc.inGraphBody {
				body = func(_ context.Context, in any) (any, error) {
					var writes map[string]string
					writes["boom"] = "boom"
					return in, nil
				}
			}
			g := graph.New().
				AddNode(graph.NodeFunc("a", body)).
				SetEntry("a").SetOutput("a")

			res, err := g.Run(context.Background(), "seed")
			if err == nil {
				t.Fatalf("节点以 %v 形态崩溃后 Run 交出 nil error（Result=%v）", tc.name, res)
			}
			if res != nil {
				t.Errorf("%v 形态崩溃后仍交出 Result %v", tc.name, res)
			}
		})
	}
}

// TestP1R14_PanicErrorNamesThePanickingNode 覆盖契约 T3（S14-AMEND-1）：错误必须点名
// 崩掉的那个节点，而且不得把无辜节点说成现场。
//
// 用 quotedNames 而不是 strings.Contains，沿用切片 12 已确立的名字牙齿判据：点错了名字、
// 把全部在途节点拼进同一句话、或只留一句通用文案，都会被引号集合的内容与数量抓到。
func TestP1R14_PanicErrorNamesThePanickingNode(t *testing.T) {
	g := graph.New().
		AddNode(graph.NodeFunc("entry", looping(noCount))).
		AddNode(graph.NodeFunc("ok", looping(noCount))).
		AddNode(graph.NodeFunc("bad", panicking("named-boom"))).
		AddEdge("entry", "ok").
		AddEdge("entry", "bad").
		SetEntry("entry").SetOutput("ok")

	_, err := runWithinGrace(t, g, "seed")
	if err == nil {
		t.Fatal("节点 panic 后 Run 没有返回错误，无法判断它点了谁")
	}
	names := quotedNames(err.Error())
	if occurrences(names, "bad") != 1 {
		t.Errorf("错误文案点名为 %v，want 恰好一次 \"bad\"：不点名崩溃节点的错误在扇出图里等于没点名", names)
	}
	for _, innocent := range []string{"ok", "entry"} {
		if occurrences(names, innocent) != 0 {
			t.Errorf("错误文案把无辜节点 %q 写进了现场（全量=%v）：恢复必须只归属真正崩溃的那次激活", innocent, names)
		}
	}
	if !strings.Contains(err.Error(), "panic") {
		t.Errorf("错误文案 %q 未表明这是一次 panic 而不是一次普通节点失败，调用方会按业务错误去重试", err)
	}
}

// TestP1R14_PanicWhileSiblingInFlightStillReturns 覆盖契约 T4：一条分支崩掉时 Run 必须
// 自己回来，不能把调用方留在等待里。
//
// 兄弟节点用 channel gate 真实占住在途位置（不用 sleep 定序，票面 §4.1），因此
// 「recover 了但忘了向唯一消费者投递」的实现会在这里悬挂到兜底时限。
func TestP1R14_PanicWhileSiblingInFlightStillReturns(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	g := graph.New().
		AddNode(graph.NodeFunc("entry", looping(noCount))).
		AddNode(graph.NodeFunc("slow", func(ctx context.Context, in any) (any, error) {
			select {
			case <-release:
				return in, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		})).
		AddNode(graph.NodeFunc("bad", panicking("in-flight-boom"))).
		AddEdge("entry", "slow").
		AddEdge("entry", "bad").
		SetEntry("entry").SetOutput("slow")

	res, err := runWithinGrace(t, g, "seed")
	if err == nil {
		t.Fatalf("在途兄弟未结束时发生 panic，Run 却交出 nil error（Result=%v）", res)
	}
	if res != nil {
		t.Errorf("panic 后仍交出 Result %v", res)
	}
}

// TestP1R14_SecondRunStillWorksAfterAPanic 覆盖契约 T5：屏障属于每次 Run，不是一次性
// 消耗品，也不把图弄脏。
//
// 节点第一次激活崩溃、第二次正常返回，这是调用方最常见的补救形状（修好上游再跑一轮）。
// 把恢复状态挂在 *Graph 上、或让第一次的 panic 短路后续消费者的实现，会让第二次 Run
// 立即失败或悬挂，而不是交出正确输出。
func TestP1R14_SecondRunStillWorksAfterAPanic(t *testing.T) {
	var activations atomic.Int64
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(_ context.Context, in any) (any, error) {
			if activations.Add(1) == 1 {
				panic("first-run-boom")
			}
			return in, nil
		})).
		SetEntry("a").SetOutput("a")

	if _, err := runWithinGrace(t, g, "first"); err == nil {
		t.Fatal("第一次 Run 没有把节点 panic 转成错误，T5 的前提不成立")
	}

	res, err := runWithinGrace(t, g, "second")
	if err != nil {
		t.Fatalf("第二次 Run 被第一次的 panic 污染: %v", err)
	}
	if got, want := res.Output(), "second"; got != want {
		t.Errorf("第二次 Run 的 Output() = %v, want %v", got, want)
	}
	if got, want := strings.Join(res.Completed(), ","), "a"; got != want {
		t.Errorf("第二次 Run 的 Completed() = %q, want %q", got, want)
	}
}

// TestP1R14_NodeDefersStillRunWhenItPanics 覆盖契约 T8：引擎的 recover 站在节点帧之外，
// 所以节点自己的 defer 必须先跑完（资源清理是节点作者的职责，不是被屏障吞掉的东西）。
//
// 计数器由测试自己提供，属调用方可观察面；恢复点若落在节点帧之内，这里读到 0。
func TestP1R14_NodeDefersStillRunWhenItPanics(t *testing.T) {
	var cleaned atomic.Int64
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(context.Context, any) (any, error) {
			defer cleaned.Add(1)
			panic("defer-boom")
		})).
		SetEntry("a").SetOutput("a")

	if _, err := runWithinGrace(t, g, "seed"); err == nil {
		t.Fatal("节点 panic 后 Run 交出 nil error，defer 断言失去前提")
	}
	if got, want := cleaned.Load(), int64(1); got != want {
		t.Errorf("节点 defer 执行次数 = %d, want %d：panic 展开时节点自己的清理没有跑完", got, want)
	}
}

// TestP1R14_NodeReturnedErrorKeepsItsIdentity 是契约 T6 的对照行：新增的 panic 屏障不得
// 改动「节点自己 return error」的既有语义——身份保持、不被重新包装、不被写成 panic 文案。
//
// 本行在当前字节下就应通过（对照行的定义），它约束的是实现阶段不许越界。
func TestP1R14_NodeReturnedErrorKeepsItsIdentity(t *testing.T) {
	sentinel := errors.New("provider said no")
	g := graph.New().
		AddNode(graph.NodeFunc("a", func(context.Context, any) (any, error) { return nil, sentinel })).
		SetEntry("a").SetOutput("a")

	res, err := runWithinGrace(t, g, "seed")
	if !errors.Is(err, sentinel) {
		t.Errorf("节点自身返回的错误被改写成 %v, want errors.Is(err, %v)", err, sentinel)
	}
	if res != nil {
		t.Errorf("节点失败后仍交出 Result %v", res)
	}
	if strings.Contains(err.Error(), "panic") {
		t.Errorf("普通节点错误被并入 panic 文案 %q，调用方会误判成引擎级故障", err)
	}
}

// TestP1R14_HappyPathUnaffected 是契约 T7 的对照行：无 panic 的两节点图照旧收敛，
// Completed() 与 Output() 一字不变。安全屏障不得改变正常路径的任何可观察事实。
func TestP1R14_HappyPathUnaffected(t *testing.T) {
	g := graph.New().
		AddNode(graph.NodeFunc("a", looping(noCount))).
		AddNode(graph.NodeFunc("b", looping(noCount))).
		AddEdge("a", "b").
		SetEntry("a").SetOutput("b")

	res, err := runWithinGrace(t, g, "seed")
	if err != nil {
		t.Fatalf("加了 panic 屏障之后正常图失败: %v", err)
	}
	if got, want := res.Output(), "seed"; got != want {
		t.Errorf("Output() = %v, want %v", got, want)
	}
	if got, want := strings.Join(res.Completed(), ","), "a,b"; got != want {
		t.Errorf("Completed() = %q, want %q", got, want)
	}
}
