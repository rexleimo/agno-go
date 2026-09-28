package graph_test

import (
	"context"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// TestP1G_LinearDagRunsChain 覆盖契约 §4 B1 与票面判据 1、5 的 DAG 类。
// 观察面只用 graph 的导出 API：New/AddNode/AddEdge/SetEntry/SetOutput/Run
// 与 Result 的 Output/Value/Completed。
func TestP1G_LinearDagRunsChain(t *testing.T) {
	var trace []string

	entry := graph.NodeFunc("entry", func(_ context.Context, in any) (any, error) {
		trace = append(trace, "entry")
		return "seed:" + in.(string), nil
	})
	out := graph.NodeFunc("out", func(_ context.Context, in any) (any, error) {
		trace = append(trace, "out")
		return "final<-" + in.(string), nil
	})

	g := graph.New().
		AddNode(entry).
		AddNode(out).
		AddEdge("entry", "out").
		SetEntry("entry").
		SetOutput("out")

	res, err := g.Run(context.Background(), "x")
	if err != nil {
		t.Fatalf("Run 未产出结果: %v", err)
	}

	if got, want := res.Output(), "final<-seed:x"; got != want {
		t.Errorf("Result.Output() = %v, want %v", got, want)
	}
	if got, ok := res.Value("entry"); !ok || got != "seed:x" {
		t.Errorf("Result.Value(\"entry\") = %v, %v; want \"seed:x\", true", got, ok)
	}
	if got, want := res.Completed(), []string{"entry", "out"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Result.Completed() = %v, want %v", got, want)
	}
	if got, want := trace, []string{"entry", "out"}; !reflect.DeepEqual(got, want) {
		t.Errorf("节点执行轨迹 = %v, want %v（线性链为因果关系，可作全序断言）", got, want)
	}
}

// TestP1G_UnsupportedEdgeDeclarationsAreRejected 覆盖契约 §12.1 R1：
// 声明了引擎尚未路由的边类型时，必须构建期被拒，而不是静默产出空 Result。
// 观察面：Validate / Run 的返回值；断言不得依赖调度器内部字段。
// 切片 8 起把定位从「节点名子串」升级为「种类标签 + 不可路由原因词 + 目标节点名」：
// 此前把原因短语整段删掉仍全绿，理由与实测见 cycle7 审查 C7-SPEC-4。
func TestP1G_UnsupportedEdgeDeclarationsAreRejected(t *testing.T) {
	identity := func(_ context.Context, in any) (any, error) { return in, nil }
	// 表内只列「本周期仍未被路由」的边类型。条件边与 Default 将在 R2 被真实路由，
	// 若在此断言它们被拒绝，就会与 R2 的期望互相拆台（R1 的拒绝集合随实现收缩）。
	// wantPhrases 同时钉「graph: + 该种类完整标签」「不可路由原因词」与目标节点名。
	// 只钉节点名时，删掉原因短语整段仍全绿（cycle7 C7-SPEC-4 实测 receipt:90f223ce 面）；
	// 早先用单字母 "j" 作定位名时，"join edge" 一词本身就能满足断言，定位性形同虚设。
	//
	// 切片 19 起 Join 已被路由，「join edge … not routable」那句随之从公共面消失（票面 :168
	// 的「被拒绝的集合会缩小」）。这一格因此改为「Join 声明本身凑不出屏障」——它同样是
	// 引擎无法照办、却也不许静默放行的声明，只是原因词换成了前驱数不足。
	tests := []struct {
		name        string
		declare     func(*graph.Graph) *graph.Graph
		wantPhrases []string
	}{
		{
			name: "join-with-one-predecessor",
			declare: func(g *graph.Graph) *graph.Graph {
				return g.AddJoin([]string{"alpha"}, "merge")
			},
			wantPhrases: []string{"graph: join target", "at least 2", `"merge"`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := tc.declare(graph.New().
				AddNode(graph.NodeFunc("alpha", identity)).
				AddNode(graph.NodeFunc("beta", identity)).
				AddNode(graph.NodeFunc("merge", identity)).
				SetEntry("alpha").
				SetOutput("beta"))

			err := g.Validate()
			if err == nil {
				t.Fatal("声明了引擎未路由的边，Validate 却判定合法")
			}
			for _, want := range tc.wantPhrases {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("错误 %q 未含 %s，无法把缺陷定位到这条未路由的边声明", err, want)
				}
			}

			res, runErr := g.Run(context.Background(), "x")
			if runErr == nil {
				t.Fatalf("非法图仍可执行并返回 Output=%v Completed=%v", res.Output(), res.Completed())
			}
		})
	}

	t.Run("只使用已实现边类型的图仍然合法", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("a", identity)).
			AddNode(graph.NodeFunc("b", identity)).
			AddEdge("a", "b").
			SetEntry("a").
			SetOutput("b")

		if err := g.Validate(); err != nil {
			t.Fatalf("合法图被拒: %v", err)
		}
		res, err := g.Run(context.Background(), "seed")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if got, want := res.Output(), "seed"; got != want {
			t.Errorf("Output() = %v, want %v", got, want)
		}
	})
}

// TestP1G_ConditionalAndDefaultRouting 覆盖契约 slice4 的 S1–S4 与对照行 C1：
// §3.4 规则 1/2/4 的路由语义只经 Completed / Output / Value 三面观察。
func TestP1G_ConditionalAndDefaultRouting(t *testing.T) {
	const entryOut = "a-out"

	// src 是入口节点，固定产出 entryOut；sink 将「自己收到什么」原样标注后返回，
	// 于是 Output() 同时证明「哪个节点被执行」与「它收到的上游值」。
	src := graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return entryOut, nil })
	sink := func(name string) graph.Node {
		return graph.NodeFunc(name, func(_ context.Context, in any) (any, error) {
			return name + "<-" + in.(string), nil
		})
	}
	yes := graph.Predicate(func(out any) bool { return true })
	no := graph.Predicate(func(out any) bool { return false })

	tests := []struct {
		name          string
		nodes         []string
		build         func(*graph.Graph) *graph.Graph
		wantCompleted []string
		wantOutput    string
		wantRanNever  []string
	}{
		{
			// S1：条件边命中 ⇒ 兜底不得触发。
			name:  "predicate-hit-suppresses-default",
			nodes: []string{"b", "c"},
			build: func(g *graph.Graph) *graph.Graph {
				return g.
					AddConditional("a", "b", yes).
					AddDefault("a", "c").
					SetOutput("b")
			},
			wantCompleted: []string{"a", "b"},
			wantOutput:    "b<-" + entryOut,
			wantRanNever:  []string{"c"},
		},
		{
			// S2：谓词不命中且没有其它具体边 ⇒ 兜底触发。
			name:  "predicate-miss-falls-back-to-default",
			nodes: []string{"b", "c"},
			build: func(g *graph.Graph) *graph.Graph {
				return g.
					AddConditional("a", "b", no).
					AddDefault("a", "c").
					SetOutput("c")
			},
			wantCompleted: []string{"a", "c"},
			wantOutput:    "c<-" + entryOut,
			wantRanNever:  []string{"b"},
		},
		{
			// S3：多条条件边同时命中 ⇒ 扇出，不互斥。
			name:  "multiple-predicates-all-fire",
			nodes: []string{"b", "d"},
			build: func(g *graph.Graph) *graph.Graph {
				return g.
					AddConditional("a", "b", yes).
					AddConditional("a", "d", yes).
					SetOutput("d")
			},
			wantCompleted: []string{"a", "b", "d"},
			wantOutput:    "d<-" + entryOut,
		},
		{
			// S4：已有无条件边命中 ⇒ 未命中的条件边与兜底都不触发。
			name:  "unconditional-edge-suppresses-default",
			nodes: []string{"b", "c", "x"},
			build: func(g *graph.Graph) *graph.Graph {
				return g.
					AddEdge("a", "b").
					AddConditional("a", "x", no).
					AddDefault("a", "c").
					SetOutput("b")
			},
			wantCompleted: []string{"a", "b"},
			wantOutput:    "b<-" + entryOut,
			wantRanNever:  []string{"c", "x"},
		},
		{
			// C1（§3.4 规则 1 的对照）：无条件边全部激活。
			name:  "unconditional-fanout",
			nodes: []string{"b", "c"},
			build: func(g *graph.Graph) *graph.Graph {
				return g.
					AddEdge("a", "b").
					AddEdge("a", "c").
					SetOutput("c")
			},
			wantCompleted: []string{"a", "b", "c"},
			wantOutput:    "c<-" + entryOut,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.New().AddNode(src)
			for _, n := range tc.nodes {
				g = g.AddNode(sink(n))
			}
			g = tc.build(g).SetEntry("a")

			if err := g.Validate(); err != nil {
				t.Fatalf("按 §3.4 合法的路由图被 Validate 拒绝: %v", err)
			}
			res, err := g.Run(context.Background(), "seed")
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got, want := res.Completed(), tc.wantCompleted; !reflect.DeepEqual(got, want) {
				t.Errorf("Completed() = %v, want %v", got, want)
			}
			if got, want := res.Output(), tc.wantOutput; got != want {
				t.Errorf("Output() = %v, want %v", got, want)
			}
			for _, n := range tc.wantRanNever {
				if _, ok := res.Value(n); ok {
					t.Errorf("节点 %q 不应被执行，却留下了输出值", n)
				}
			}
		})
	}

	// S5：谓词实参必须是源节点本次输出，而不是图入口输入。
	t.Run("predicate-receives-source-output", func(t *testing.T) {
		var seen []any
		g := graph.New().
			AddNode(src).
			AddNode(sink("b")).
			AddConditional("a", "b", func(out any) bool {
				seen = append(seen, out)
				return true
			}).
			SetEntry("a").
			SetOutput("b")

		res, err := g.Run(context.Background(), "seed")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if len(seen) != 1 {
			t.Fatalf("谓词调用次数 = %d, want 1（每个源节点输出判定一次）", len(seen))
		}
		got, ok := res.Value("a")
		if !ok {
			t.Fatalf("Result.Value(\"a\") 缺失，无法比对谓词实参")
		}
		if seen[0] != got {
			t.Errorf("谓词实参 = %v, want 源节点输出 %v", seen[0], got)
		}
		if seen[0] == "seed" {
			t.Errorf("谓词收到的是图入口输入 %q，而不是源节点输出", seen[0])
		}
	})
}

// TestP1G_ValidateRejectsDanglingEdge 覆盖契约 §11.1（切片 2）与票面判据 3、5。
// 观察面：Validate 的返回值与错误可定位性、Run 对非法图的拒绝（不得崩溃）。
func TestP1G_ValidateRejectsDanglingEdge(t *testing.T) {
	identity := func(_ context.Context, in any) (any, error) { return in, nil }

	t.Run("悬空边被拒且可定位", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("a", identity)).
			AddEdge("a", "ghost").
			SetEntry("a").
			SetOutput("a")

		err := g.Validate()
		if err == nil {
			t.Fatal("出边指向不存在的节点，Validate 却判定合法")
		}
		if !strings.Contains(err.Error(), "ghost") {
			t.Errorf("Validate 错误未指出问题节点 %q: %v", "ghost", err)
		}

		res, runErr := g.Run(context.Background(), "x")
		if runErr == nil {
			t.Fatalf("非法图仍可执行，且返回了结果 %v", res)
		}
	})

	t.Run("合法图通过校验并可执行", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("a", identity)).
			AddNode(graph.NodeFunc("b", identity)).
			AddEdge("a", "b").
			SetEntry("a").
			SetOutput("b")

		if err := g.Validate(); err != nil {
			t.Fatalf("合法图被拒: %v", err)
		}
		res, err := g.Run(context.Background(), "seed")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if got, want := res.Output(), "seed"; got != want {
			t.Errorf("Output() = %v, want %v", got, want)
		}
	})
}

// TestP1G_ValidateRejectsConditionalWithoutPredicate 覆盖契约 slice5 的 S1、S2 与
// S3 反向对照（cycle4 审查 C4-SPEC-2）。
// 观察面：AddConditional 缺失谓词时 Validate 的返回值与可定位性、Run 的 err 与 Result，
// 以及合法谓词（含永不命中的谓词）不受新校验牵连。
func TestP1G_ValidateRejectsConditionalWithoutPredicate(t *testing.T) {
	identity := func(_ context.Context, in any) (any, error) { return in, nil }
	nilPredicateGraph := func() *graph.Graph {
		return graph.New().
			AddNode(graph.NodeFunc("a", identity)).
			AddNode(graph.NodeFunc("b", identity)).
			AddConditional("a", "b", nil).
			SetEntry("a").
			SetOutput("b")
	}

	t.Run("缺失谓词的条件边在构建期被拒且可定位", func(t *testing.T) {
		err := nilPredicateGraph().Validate()
		if err == nil {
			t.Fatal("Validate() = <nil>，缺失谓词的条件边被放行")
		}
		// 钉「种类 + 原因 + 两端节点名」，不钉整句：只断言节点名会让任何提到 a/b 的
		// 拒绝文案都通过（契约 slice5 S1 原文要求同时指名边种类）。
		for _, want := range []string{"graph: conditional edge", "predicate", `"a"`, `"b"`} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("错误 %q 未含 %s，无法把缺陷定位到这条条件边声明", err, want)
			}
		}
	})

	t.Run("同一非法声明经 Run 传达为错误且不返回空结果", func(t *testing.T) {
		res, err := nilPredicateGraph().Run(context.Background(), "seed")
		if err == nil {
			t.Fatalf("Run() 的 err = <nil>，Completed() = %v、Output() = %v：调用方拿到一个没有输出的空结果",
				res.Completed(), res.Output())
		}
		if res != nil {
			t.Errorf("非法图被拒时仍返回 Result，Completed() = %v", res.Completed())
		}
		for _, want := range []string{"graph: conditional edge", "predicate", `"a"`, `"b"`} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("Run 的错误 %q 未含 %s，无法把缺陷定位到这条条件边声明", err, want)
			}
		}
	})

	t.Run("永不命中的合法谓词不得被误判为非法", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("a", identity)).
			AddNode(graph.NodeFunc("b", identity)).
			AddNode(graph.NodeFunc("c", identity)).
			AddConditional("a", "b", func(any) bool { return false }).
			AddDefault("a", "c").
			SetEntry("a").
			SetOutput("c")

		if err := g.Validate(); err != nil {
			t.Fatalf("Validate() = %v，合法声明被误拒", err)
		}
		res, err := g.Run(context.Background(), "seed")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if got, want := res.Completed(), []string{"a", "c"}; !reflect.DeepEqual(got, want) {
			t.Errorf("Completed() = %v, want %v", got, want)
		}
	})
}

// TestP1G_ValidateRejectsDanglingEndpointForEveryEdgeKind 覆盖契约 slice6 的 S5
// （cycle5 审查 C5-SPEC-4）与 slice5 的 S4（cycle4 审查 C4-STD-2 (1)），并按 slice7
// 的 T1 把种类钉住改为锚定形式。
// 观察面：四种边种类的悬空端点各自成行，且错误必须同时钉住「graph: + 该种类完整标签」
// 前缀、缺失的那一端与 does not exist 原因词。锚定不是洁癖：裸子串 "conditional" 是
// "unconditional edge" 的子串，因此只查 kind 时把条件边的文案换成无条件边的文案仍全绿
// （cycle6 receipt:422ce345 实测），四行彼此不可区分。
func TestP1G_ValidateRejectsDanglingEndpointForEveryEdgeKind(t *testing.T) {
	identity := func(_ context.Context, in any) (any, error) { return in, nil }
	yes := func(any) bool { return true }

	cases := []struct {
		name    string
		declare func(g *graph.Graph) *graph.Graph
		kind    string // 错误必须指明的边种类
		side    string // 错误必须指明缺失的是哪一端
	}{
		{"unconditional-to-missing", func(g *graph.Graph) *graph.Graph { return g.AddEdge("a", "ghost") }, "unconditional", `to node "ghost"`},
		{"unconditional-from-missing", func(g *graph.Graph) *graph.Graph { return g.AddEdge("ghost", "a") }, "unconditional", `from node "ghost"`},
		{"conditional-to-missing", func(g *graph.Graph) *graph.Graph { return g.AddConditional("a", "ghost", yes) }, "conditional", `to node "ghost"`},
		{"conditional-from-missing", func(g *graph.Graph) *graph.Graph { return g.AddConditional("ghost", "a", yes) }, "conditional", `from node "ghost"`},
		{"default-to-missing", func(g *graph.Graph) *graph.Graph { return g.AddDefault("a", "ghost") }, "default", `to node "ghost"`},
		{"default-from-missing", func(g *graph.Graph) *graph.Graph { return g.AddDefault("ghost", "a") }, "default", `from node "ghost"`},
		{"join-to-missing", func(g *graph.Graph) *graph.Graph { return g.AddJoin([]string{"a"}, "ghost") }, "join", `to node "ghost"`},
		{"join-from-missing", func(g *graph.Graph) *graph.Graph { return g.AddJoin([]string{"ghost"}, "b") }, "join", `from node "ghost"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.New().
				AddNode(graph.NodeFunc("a", identity)).
				AddNode(graph.NodeFunc("b", identity)).
				AddEdge("a", "b")
			g = tc.declare(g).SetEntry("a").SetOutput("b")

			// 每张图只含「某一类边的端点缺失」这一种缺陷：join 行的端点检查先于
			// not-routable 检查触发，因此本行断言的是端点校验，不是 Join 路由。
			if err := g.Validate(); err == nil {
				t.Fatalf("Validate() = <nil>，%s 的悬空端点被放行", tc.name)
			} else {
				for _, want := range []string{"graph: " + tc.kind + " edge", tc.side, "does not exist"} {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("错误 %q 未含 %s，无法把缺陷定位到这条声明", err, want)
					}
				}
			}
			res, runErr := g.Run(context.Background(), "seed")
			if runErr == nil {
				t.Error("含悬空端点的图仍可执行")
			}
			if res != nil {
				t.Errorf("含悬空端点的图被拒时仍返回 Result %v，调用方无从分辨成功与失败", res)
			}
		})
	}
}

// TestP1G_ValidateRejectsMissingOrUnknownEntry 覆盖母约 §3.5 必检项 3「无入口 / 入口
// 不存在」与母票 §4 B6、§6 判据 3（cycle7 审查 C7-SPEC-4）。
// 观察面：Validate 与 Run 两个公共入口都拒绝、错误钉住「entry node + 被指名的入口名 +
// 原因词」、且拒绝时不返回 Result。缺这条覆盖的代价是实测出来的：删除 graph.go 的入口
// 分支后既有 35 个断言全绿（receipt:b16b68fd-220c-4f3e-9679-95f111774bdf），而导出 API
// 把一次干净的构建期拒绝换成杀死宿主的 SIGSEGV（panic 帧 scheduler.go:110）。
func TestP1G_ValidateRejectsMissingOrUnknownEntry(t *testing.T) {
	identity := func(_ context.Context, in any) (any, error) { return in, nil }
	// skeleton 是一张除入口声明外完全合法的图：只有一条缺陷，判红不会归错原因。
	skeleton := func() *graph.Graph {
		g := graph.New().
			AddNode(graph.NodeFunc("a", identity)).
			AddNode(graph.NodeFunc("b", identity)).
			AddEdge("a", "b")
		return g.SetOutput("b")
	}

	tests := []struct {
		name     string
		build    func() *graph.Graph
		wantName string // 错误必须指明的入口名
	}{
		{"never-declared", skeleton, `""`},
		{"not-a-node", func() *graph.Graph { return skeleton().SetEntry("ghost") }, `"ghost"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := tc.build()

			err := g.Validate()
			if err == nil {
				t.Fatal("入口不是已注册节点，Validate 却判定合法")
			}
			for _, want := range []string{"graph: entry node", tc.wantName, "does not exist"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("错误 %q 未含 %s，无法把缺陷定位到入口声明", err, want)
				}
			}

			res, runErr := g.Run(context.Background(), "seed")
			if runErr == nil {
				t.Fatalf("缺入口的图仍可执行: Output=%v Completed=%v", res.Output(), res.Completed())
			}
			if res != nil {
				t.Errorf("Run 拒绝时仍返回 Result %v，调用方无从分辨成功与失败", res)
			}
		})
	}
}

// TestP1G_ValidateRejectsMissingOrUnknownOutput 覆盖契约 slice9 的 A1–A4。
// 观察面：Validate / Run / Result 三个公共面。A1/A2 钉「输出声明本身不合法必须构建期
// 被拒」，A3/A4 是反向对照，钉住新校验不得越界成「按输出值判非法」——合法节点返回 nil
// 与声明不合法是两件事，前者必须照常成功。
func TestP1G_ValidateRejectsMissingOrUnknownOutput(t *testing.T) {
	identity := func(_ context.Context, in any) (any, error) { return in, nil }

	// skeleton 是一张除输出声明外完全合法的图：只有一条缺陷，判红不会归错原因。
	skeleton := func() *graph.Graph {
		g := graph.New().
			AddNode(graph.NodeFunc("a", identity)).
			AddNode(graph.NodeFunc("b", identity)).
			AddEdge("a", "b")
		return g.SetEntry("a")
	}

	tests := []struct {
		name     string
		build    func() *graph.Graph
		wantName string // 错误必须指明的输出节点名
	}{
		{"never-declared", skeleton, `""`},
		{"not-a-node", func() *graph.Graph { return skeleton().SetOutput("ghost") }, `"ghost"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := tc.build()

			err := g.Validate()
			if err == nil {
				t.Fatal("输出不是已注册节点，Validate 却判定合法")
			}
			for _, want := range []string{"graph: output node", tc.wantName, "does not exist"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("错误 %q 未含 %s，无法把缺陷定位到输出声明", err, want)
				}
			}

			res, runErr := g.Run(context.Background(), "seed")
			if runErr == nil {
				t.Fatalf("缺合法输出的图仍可执行: Output=%v Completed=%v", res.Output(), res.Completed())
			}
			if res != nil {
				t.Errorf("Run 拒绝时仍返回 Result %v，调用方无从分辨成功与失败", res)
			}
		})
	}

	t.Run("对照：合法输出节点照常解析", func(t *testing.T) {
		g := skeleton().SetOutput("b")
		if err := g.Validate(); err != nil {
			t.Fatalf("合法输出声明被误拒: %v", err)
		}
		res, err := g.Run(context.Background(), "seed")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if want := any("seed"); res.Output() != want {
			t.Errorf("Result.Output() = %v, want %v", res.Output(), want)
		}
	})

	t.Run("对照：合法输出节点返回 nil 不是非法声明", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("a", identity)).
			AddNode(graph.NodeFunc("n", func(_ context.Context, _ any) (any, error) { return nil, nil })).
			AddEdge("a", "n").
			SetEntry("a").
			SetOutput("n")

		if err := g.Validate(); err != nil {
			t.Fatalf("输出节点合法、只是返回 nil，Validate 却拒绝: %v", err)
		}
		res, err := g.Run(context.Background(), "seed")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if res.Output() != nil {
			t.Errorf("Result.Output() = %v, want <nil>", res.Output())
		}
		// Value 的第二个返回值才是「这个 nil 来自合法完成节点」的证据：只比 Output()
		// 时，A1/A2 那种「没人给它赋值」的 nil 与这里的 nil 在公共面上完全同形。
		if v, ok := res.Value("n"); !ok || v != nil {
			t.Errorf("Result.Value(\"n\") = %v, %v; want <nil>, true", v, ok)
		}
	})
}

// cycleLimit 是给「可能永不返回的 Run」设的期限。无条件环今天在构建期完全放行，
// 一次误加的等待会把测试变成挂死（cycle-9 R9-SPEC-3 的探针就是这样挂了约 15 分钟）。
const cycleLimit = time.Second

// runWithinLimit 把 Run 放进 goroutine 并在 cycleLimit 内等它。期限不是便利，而是本
// 测试族的一条断言前提：期限到就 cancel 并判红，于是「不自退出」表现为可定位的失败
// 输出而不是挂死，聚焦命令因此始终能给出退出码。
func runWithinLimit(t *testing.T, g *graph.Graph) (*graph.Result, error) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type outcome struct {
		res *graph.Result
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, err := g.Run(ctx, "seed")
		done <- outcome{res, err}
	}()

	select {
	case o := <-done:
		return o.res, o.err
	case <-time.After(cycleLimit):
		t.Fatalf("Run 未在 %v 内返回：不可终止的图必须构建期被拒，而不是把调用方拖进不自退出的执行", cycleLimit)
		return nil, nil
	}
}

// TestP1G_ValidateRejectsUnconditionalCycle 覆盖契约 slice10 的 A1–A4（不可断开的环必
// 须构建期被拒且可定位）与 A5（合法环不得被误杀）。
// 「无条件环」按契约 AR1 取「不可断开」判据：环上每条边一旦其起点被激活就必定被走 ——
// AddEdge，或该起点没有任何条件边替代的 AddDefault。条件可断开的环仍然合法。
// 观察面只用公共入口：Validate / Run / Result，以及节点体内那个只用于「拒绝时零副
// 作用」的执行计数。
func TestP1G_ValidateRejectsUnconditionalCycle(t *testing.T) {
	type illegal struct {
		name      string
		build     func(count func()) *graph.Graph
		wantNodes []string // 错误必须点名的成环节点
	}

	illegals := []illegal{
		{
			name: "self-loop",
			build: func(count func()) *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("a", looping(count))).
					AddEdge("a", "a").
					SetEntry("a").SetOutput("a")
			},
			wantNodes: []string{`"a"`},
		},
		{
			name: "two-node",
			build: func(count func()) *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("a", looping(count))).
					AddNode(graph.NodeFunc("b", looping(count))).
					AddEdge("a", "b").AddEdge("b", "a").
					SetEntry("a").SetOutput("b")
			},
			wantNodes: []string{`"a"`, `"b"`},
		},
		{
			name: "three-node",
			build: func(count func()) *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("a", looping(count))).
					AddNode(graph.NodeFunc("b", looping(count))).
					AddNode(graph.NodeFunc("c", looping(count))).
					AddEdge("a", "b").AddEdge("b", "c").AddEdge("c", "a").
					SetEntry("a").SetOutput("c")
			},
			wantNodes: []string{`"a"`, `"b"`, `"c"`},
		},
		{
			// 恒真条件边挂在环旁边不构成出口：调度器是扇出，环与出口会同时跑。
			name: "with-always-true-exit",
			build: func(count func()) *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("a", looping(count))).
					AddNode(graph.NodeFunc("b", looping(count))).
					AddNode(graph.NodeFunc("exit", looping(count))).
					AddEdge("a", "b").AddEdge("b", "a").
					AddConditional("b", "exit", func(any) bool { return true }).
					SetEntry("a").SetOutput("exit")
			},
			wantNodes: []string{`"a"`, `"b"`},
		},
		{
			// 兜底边在没有条件替代时必定被走，因此它构成的环同样是死循环。
			name: "default-edge-without-alternative",
			build: func(count func()) *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("a", looping(count))).
					AddNode(graph.NodeFunc("b", looping(count))).
					AddEdge("a", "b").AddDefault("b", "a").
					SetEntry("a").SetOutput("b")
			},
			wantNodes: []string{`"a"`, `"b"`},
		},
	}

	for _, tc := range illegals {
		t.Run(tc.name, func(t *testing.T) {
			var executions atomic.Int64
			g := tc.build(func() { executions.Add(1) })

			// 两个入口必须给出同一份可定位清单，锚点因此只构造一次。
			anchors := append([]string{"graph: unconditional cycle"}, tc.wantNodes...)

			err := g.Validate()
			if err == nil {
				t.Fatal("不可断开的环被 Validate 判为合法，调用方只能拿到一个永不返回的 Run")
			}
			for _, want := range anchors {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("错误 %q 未含 %s，无法把缺陷定位到这个环", err, want)
				}
			}

			res, runErr := runWithinLimit(t, g)
			if runErr == nil {
				t.Fatalf("不可断开的环仍可执行: Completed=%v", res.Completed())
			}
			if res != nil {
				t.Errorf("Run 拒绝时仍返回 Result %v，调用方无从分辨成功与失败", res)
			}
			// 拒绝必须发生在任何节点副作用之前（join 回边今天已做到，见 slice10 基线）。
			if got := executions.Load(); got != 0 {
				t.Errorf("Validate 拒绝前已执行 %d 次节点，副作用发生在校验之前", got)
			}
			for _, want := range anchors {
				if !strings.Contains(runErr.Error(), want) {
					t.Errorf("Run 的错误 %q 未含 %s，两个入口给出的定位不一致", runErr, want)
				}
			}
		})
	}

	// A5 反向对照：本轮新增的拒绝只能砍掉不可终止的形状。下面三行全部是 slice10 基线
	// 里实测「今天就能结束」的图，若实现把「任何环」都判非法，前两行立刻红。
	t.Run("对照：谓词会转假的条件环合法且能结束", func(t *testing.T) {
		trials := 0
		g := graph.New().
			AddNode(graph.NodeFunc("a", looping(noCount))).
			AddNode(graph.NodeFunc("b", looping(noCount))).
			AddNode(graph.NodeFunc("end", looping(noCount))).
			AddEdge("a", "b").
			AddConditional("b", "a", func(any) bool { trials++; return trials < 2 }).
			AddConditional("b", "end", func(any) bool { return trials >= 2 }).
			SetEntry("a").SetOutput("end")

		if err := g.Validate(); err != nil {
			t.Fatalf("可断开的条件环被误拒: %v", err)
		}
		res, err := runWithinLimit(t, g)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		// 精确等值而非「包含」：重访节点必须留在 Completed 里，去重或只跑一遍都会红。
		if got, want := res.Completed(), []string{"a", "a", "b", "b", "end"}; !reflect.DeepEqual(got, want) {
			t.Errorf("Result.Completed() = %v, want %v", got, want)
		}
		// 节点都是透传体，所以输出节点拿到的仍是入口值：这条钉的是「值穿过重访之后
		// 没有丢」，不是节点名。
		if got, want := res.Output(), "seed"; got != want {
			t.Errorf("Result.Output() = %v, want %v", got, want)
		}
	})

	t.Run("对照：有条件替代的兜底回环合法且能结束", func(t *testing.T) {
		trials := 0
		g := graph.New().
			AddNode(graph.NodeFunc("a", looping(noCount))).
			AddNode(graph.NodeFunc("b", looping(noCount))).
			AddNode(graph.NodeFunc("end", looping(noCount))).
			AddEdge("a", "b").
			AddConditional("b", "end", func(any) bool { trials++; return trials >= 2 }).
			AddDefault("b", "a").
			SetEntry("a").SetOutput("end")

		if err := g.Validate(); err != nil {
			t.Fatalf("带条件替代的兜底回环被误拒: %v", err)
		}
		res, err := runWithinLimit(t, g)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if got, want := res.Completed(), []string{"a", "a", "b", "b", "end"}; !reflect.DeepEqual(got, want) {
			t.Errorf("Result.Completed() = %v, want %v", got, want)
		}
	})

	t.Run("对照：无环图不受影响", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("a", looping(noCount))).
			AddNode(graph.NodeFunc("b", looping(noCount))).
			AddEdge("a", "b").
			SetEntry("a").SetOutput("b")

		if err := g.Validate(); err != nil {
			t.Fatalf("无环图被误拒: %v", err)
		}
		res, err := runWithinLimit(t, g)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if got, want := res.Completed(), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
			t.Errorf("Result.Completed() = %v, want %v", got, want)
		}
	})
}

// noCount 是「不计数」的那份节点体：对照行只关心放行与执行顺序，没有副作用时序可钉，
// 传它而不是 looping(nil)，免得读代码的人以为那条行也在查拒绝时机。
func noCount() {}

// looping 返回一个把「自己被执行了一次」记进 count 的最小节点体。计数只服务于一条用
// 户可观察事实：拒绝必须发生在副作用之前。
func looping(count func()) func(context.Context, any) (any, error) {
	return func(_ context.Context, in any) (any, error) {
		count()
		return in, nil
	}
}

// quotedNames 按出现顺序返回文案里被成对双引号包住的内容。
//
// 用它而不是 strings.Contains 是因为「点了哪个名字」与「一共点了几个名字」都是这条
// 拒绝的用户可观察面：子串匹配无法发现「把全部不可达名字拼进同一条消息」，而引号段的
// 数量与内容可以。只解析公共 error.Error() 的字符串，不碰任何私有状态。
func quotedNames(msg string) []string {
	var names []string
	for {
		open := strings.IndexByte(msg, '"')
		if open < 0 {
			return names
		}
		msg = msg[open+1:]
		end := strings.IndexByte(msg, '"')
		if end < 0 {
			return names
		}
		names = append(names, msg[:end])
		msg = msg[end+1:]
	}
}

// occurrences 返回 name 在 names 里出现的次数。
func occurrences(names []string, name string) int {
	count := 0
	for _, n := range names {
		if n == name {
			count++
		}
	}
	return count
}

// TestP1G_RunIsolatesTopologyFromConcurrentMutation 覆盖契约 slice6 的 S1–S3 与
// S4 反向对照（cycle5 审查 C5-STD-5：运行中改图可把宿主打成不可 recover 的崩溃），
// 并按 slice7 的 T2–T5 补齐被证明没有检测力的两面：运行中覆盖同名节点、运行中改写
// 输出节点名，以及参照 Run 自身的绝对 Output 期望。
// 观察面：同一次测试运行内，「入口节点在自己函数体内改图的 Run」与「不改图的参照
// Run」必须在 err、Completed()、Output() 三面上完全一致，且进程要活着走完断言。
// 对照值取自实跑的参照图，不写死实现现状；参照另带绝对期望，以免同向退化看不出来。
func TestP1G_RunIsolatesTopologyFromConcurrentMutation(t *testing.T) {
	identity := func(_ context.Context, in any) (any, error) { return in, nil }
	// d 透传输入会让「输出来源被换掉」在 Output 面上看不出来，所以给一个可区分的值。
	dOut := func(_ context.Context, _ any) (any, error) { return "D-OUT", nil }

	// newGraph 搭出 a→b（输出 b）的合法骨架，d 是已注册的备用靶节点。
	// d 必须挂一条运行期永不命中的条件边：切片 11 之后，注册了却从入口不可达的节点会被
	// Validate 拒绝（§3.5 第 4 项），而本测试要的正是「静态可达、但参照 Run 的 Completed
	// 仍是 [a b]」——谓词恒假同时满足这两面，下面所有断言与绝对期望一字未改。
	// mutate 非空时，由入口节点 a 在执行期间对同一 builder 施加结构性修改。
	newGraph := func(mutate func(g *graph.Graph)) *graph.Graph {
		var g *graph.Graph
		entry := identity
		if mutate != nil {
			entry = func(_ context.Context, in any) (any, error) {
				mutate(g)
				return in, nil
			}
		}
		g = graph.New().
			AddNode(graph.NodeFunc("a", entry)).
			AddNode(graph.NodeFunc("b", identity)).
			AddNode(graph.NodeFunc("d", dOut)).
			AddEdge("a", "b").
			AddConditional("b", "d", func(any) bool { return false }).
			SetEntry("a").
			SetOutput("b")
		return g
	}

	mustRun := func(t *testing.T, g *graph.Graph) *graph.Result {
		t.Helper()
		if err := g.Validate(); err != nil {
			t.Fatalf("修改之前的图本身合法，Validate 却拒绝: %v", err)
		}
		res, err := g.Run(context.Background(), "seed")
		if err != nil {
			t.Fatalf("Run 返回错误 %v，而本次执行所依据的拓扑并未改变", err)
		}
		if res == nil {
			t.Fatal("Run 返回 err == nil 却没有 Result")
		}
		return res
	}

	reference := mustRun(t, newGraph(nil))
	if got, want := reference.Completed(), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("参照图自身就不满足切片 1–4 的既有语义: Completed() = %v, want %v", got, want)
	}
	// 参照必须同时带绝对期望：只与实跑参照值相比时，任何让参照与受试同向退化的缺陷
	// （例如 output 捕获整体失效）在该面永不可见（cycle6 receipt:cf8c0d58）。
	if got, want := reference.Output(), "seed"; got != want {
		t.Fatalf("参照图自身就不满足切片 1–4 的既有语义: Output() = %v, want %v", got, want)
	}

	assertSameAsReference := func(t *testing.T, label string, got *graph.Result) {
		t.Helper()
		if !reflect.DeepEqual(got.Completed(), reference.Completed()) {
			t.Errorf("%s: Completed() = %v，未被修改的参照图为 %v —— 运行中的 builder 修改推翻了构建期已验证的拓扑",
				label, got.Completed(), reference.Completed())
		}
		if !reflect.DeepEqual(got.Output(), reference.Output()) {
			t.Errorf("%s: Output() = %v，参照图为 %v", label, got.Output(), reference.Output())
		}
	}

	t.Run("运行中追加悬空边不得让执行崩溃", func(t *testing.T) {
		assertSameAsReference(t, "悬空边", mustRun(t, newGraph(func(g *graph.Graph) {
			g.AddEdge("a", "ghost")
		})))
	})

	t.Run("运行中追加缺失谓词的条件边不得让执行崩溃", func(t *testing.T) {
		// 靶点选 d 而不是已有的 b：同靶时「被读到后静默跳过」与「根本没读到」在
		// Completed 面上不可区分；换到新靶后，「条件成立即路由」这类把 nil 谓词当
		// 真的实现会让 Completed 多出 d 而判红（slice7 T2）。
		assertSameAsReference(t, "缺失谓词的新条件边", mustRun(t, newGraph(func(g *graph.Graph) {
			g.AddConditional("a", "d", nil)
		})))
	})

	t.Run("运行中覆盖同名节点不得改变本次输出来源", func(t *testing.T) {
		assertSameAsReference(t, "运行中被覆盖的节点 b", mustRun(t, newGraph(func(g *graph.Graph) {
			g.AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
				return "OVERRIDE-B", nil
			}))
		})))
	})

	t.Run("运行中改写输出节点名不得改变本次输出", func(t *testing.T) {
		assertSameAsReference(t, "运行中被改写的输出节点", mustRun(t, newGraph(func(g *graph.Graph) {
			g.SetOutput("d")
		})))
	})

	t.Run("运行中追加合法边不得改变本次路由", func(t *testing.T) {
		assertSameAsReference(t, "合法新边", mustRun(t, newGraph(func(g *graph.Graph) {
			g.AddEdge("a", "d")
		})))
	})

	t.Run("对照：捕获的范围是一次 Run，不是图的终身状态", func(t *testing.T) {
		g := newGraph(nil)
		mustRun(t, g)
		g.AddEdge("b", "d").SetOutput("d")
		res := mustRun(t, g)
		if want := []string{"a", "b", "d"}; !reflect.DeepEqual(res.Completed(), want) {
			t.Errorf("Run 返回之后再声明的边未生效: Completed() = %v, want %v", res.Completed(), want)
		}
		// 输出解析必须落在「本次捕获的 output 节点自身的返回值」上。只查 Completed 时，
		// 把输出错配到别的完成节点或错配到输入都看不出来；d 返回可区分的 D-OUT，
		// 正是为了让这一面拿到绝对期望（cycle7 审查 C7-SPEC-3：该值此前无人断言）。
		if want := any("D-OUT"); res.Output() != want {
			t.Errorf("下一次 Run 的输出未解析到新输出节点: Output() = %v, want %v", res.Output(), want)
		}
	})

	// 上面对照只覆盖 AddEdge/SetOutput 两条腿；这里补节点腿，使「运行中覆盖同名节点」
	// 的用例不能靠『把图终身冻结』这种实现蒙过去 —— 终身冻结会让新注册节点永不可见。
	t.Run("对照：Run 之后新注册的节点对下一次 Run 可见", func(t *testing.T) {
		g := newGraph(nil)
		mustRun(t, g)
		g.AddNode(graph.NodeFunc("e", func(_ context.Context, _ any) (any, error) {
			return "E-OUT", nil
		}))
		g.AddEdge("b", "e")
		got := mustRun(t, g)
		if want := []string{"a", "b", "e"}; !reflect.DeepEqual(got.Completed(), want) {
			t.Errorf("Run 之后注册的节点未参与下一次执行: Completed() = %v, want %v",
				got.Completed(), want)
		}
		// 新节点接在输出节点之后，不得把输出从 b 挪走：这一面用绝对期望钉住，
		// 而不是回头比对 reference —— reference 是函数开头那次已完成 Run 的不可变
		// Result，改的是另一个图对象，那样的断言按构造不可能为假（cycle7 C7-STD-1）。
		if want := any("seed"); got.Output() != want {
			t.Errorf("新注册的节点改变了输出来源: Output() = %v, want %v", got.Output(), want)
		}
	})
}

// TestP1G_ValidateRejectsUnreachableNodes 覆盖契约 slice11 的 A1–A7。
// 观察面仍是 Validate / Run / Result 三个公共入口。基线实测（/tmp/p1-r11-probe）里
// 「注册了却从入口轮不到」的节点全部静默放行，其中 B2 那一张甚至交出 err==nil 而
// Output()==nil —— 正是切片 9 已经承诺不再出现的形状，只是那时判的是「输出节点不
// 存在」，这一轮补的是「输出节点存在但永远轮不到」。
//
// A3 与 A5 是反向对照，钉住「可达」必须按声明的边算而不是按运行期是否真的执行过算；
// A6 钉住两个检查同时成立时报哪一个，因为不钉它的话插入次序只是实现者的自由。
//
// 切片 12 只做牙齿加固，不改这条行为：引号段计数把「一次只点一个字典序最小的名字」
// 从文案里的一个子串事实变成可判红的事实（加固前「拼进全部不可达名字」的变异能穿过
// 全部断言），同实例重复校验与 Validate/Run 整串相等补齐取样面与两入口一致性。
func TestP1G_ValidateRejectsUnreachableNodes(t *testing.T) {
	passthrough := func(_ context.Context, in any) (any, error) { return in, nil }

	tests := []struct {
		name      string
		build     func(count func()) *graph.Graph
		wantNodes []string
		// entry 是该行的入口名，wantName 是期望被唯一点名的不可达节点，
		// alsoUnreachable 是同样不可达但期望一个都不出现在文案里的名字。
		entry           string
		wantName        string
		alsoUnreachable []string
	}{
		{
			// B1：注册了但没有任何边指向它。
			name: "orphan-registered",
			build: func(count func()) *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("entry", looping(count))).
					AddNode(graph.NodeFunc("orphan", looping(count))).
					SetEntry("entry").SetOutput("entry")
			},
			wantNodes: []string{`"orphan"`},
			entry:     "entry",
			wantName:  "orphan",
		},
		{
			// B4：一整条接不上主线的链，两个节点都不可达。
			name: "unreachable-chain",
			build: func(count func()) *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("a", looping(count))).
					AddNode(graph.NodeFunc("c", looping(count))).
					AddNode(graph.NodeFunc("d", looping(count))).
					AddEdge("c", "d").
					SetEntry("a").SetOutput("a")
			},
			wantNodes:       []string{`"c"`},
			entry:           "a",
			wantName:        "c",
			alsoUnreachable: []string{"d"},
		},
		{
			// 多个不可达节点时报哪一个必须是可预期的：字典序第一个。
			name: "unreachable-many",
			build: func(count func()) *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("a", looping(count))).
					AddNode(graph.NodeFunc("beta", looping(count))).
					AddNode(graph.NodeFunc("alpha", looping(count))).
					AddNode(graph.NodeFunc("gamma", looping(count))).
					AddNode(graph.NodeFunc("delta", looping(count))).
					AddEdge("beta", "alpha").
					AddEdge("delta", "gamma").
					SetEntry("a").SetOutput("a")
			},
			wantNodes:       []string{`"alpha"`},
			entry:           "a",
			wantName:        "alpha",
			alsoUnreachable: []string{"beta", "gamma", "delta"},
		},
		{
			// 契约 T4：可达只按正向声明边算。sink 与入口之间确实有边相连，但方向是
			// sink→入口，从入口出发走不到它。按「相连即算可达」实现的 BFS 会放行这张图，
			// 而 unreachable-chain 那一行无论方向如何都不可达，看不见这个差异。
			name: "reverse-edge-connected",
			build: func(count func()) *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("entry", looping(count))).
					AddNode(graph.NodeFunc("sink", looping(count))).
					AddEdge("sink", "entry").
					SetEntry("entry").SetOutput("entry")
			},
			wantNodes: []string{`"sink"`},
			entry:     "entry",
			wantName:  "sink",
		},
		{
			// B2：被声明为输出的节点自己不可达。
			name: "unreachable-output",
			build: func(count func()) *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("entry", looping(count))).
					AddNode(graph.NodeFunc("out", looping(count))).
					SetEntry("entry").SetOutput("out")
			},
			wantNodes: []string{`"out"`},
			entry:     "entry",
			wantName:  "out",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var executions atomic.Int64
			g := tc.build(func() { executions.Add(1) })

			anchors := append([]string{"graph: unreachable node"}, tc.wantNodes...)

			err := g.Validate()
			if err == nil {
				t.Fatal("从入口不可达的节点被 Validate 判为合法，调用方拿到的是少跑了节点的 Result")
			}
			// 同一条缺陷重复校验必须给出同一份文案。不可达集合是从 map 里遍历出来的，
			// 不钉这一条就等于把「报哪个名字」交给运行期随机（变异矩阵里唯一存活的那个
			// 变异正是「取 map 遍历到的第一个」，见 slice11 GREEN 账本）。
			first := err.Error()
			for i := 1; i <= 16; i++ {
				again := tc.build(noCount).Validate()
				if again == nil {
					t.Fatalf("第 %d 次重复校验放行了同一张图", i)
				}
				if again.Error() != first {
					t.Fatalf("第 %d 次重复校验给出不同文案: %q vs %q", i, again.Error(), first)
				}
			}
			// 同一个图对象自身也要给出同一份文案：调用方拿到 g 之后完全可能再问一次，
			// 取样面只覆盖新建实例的话，实例内状态漂移就看不见（slice11 审查 S11-STD-6）。
			for i := 1; i <= 3; i++ {
				same := g.Validate()
				if same == nil {
					t.Fatalf("同一实例第 %d 次校验放行了这张图", i)
				}
				if same.Error() != first {
					t.Fatalf("同一实例第 %d 次校验给出不同文案: %q vs %q", i, same.Error(), first)
				}
			}
			for _, want := range anchors {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("错误 %q 未含 %s，无法把缺陷定位到那条不可达声明", err, want)
				}
			}

			// 被点名的节点恰好一个：文案里的引号段只能是「那一个不可达节点加入口」。
			// 子串匹配看不见「把全部不可达名字拼进同一条消息」，这里按引号段计数并逐名
			// 比对（契约 T1；变异 MU9 在加固前正是从 anchors 的子串形状下存活的）。
			names := quotedNames(first)
			if len(names) != 2 {
				t.Errorf("拒绝文案 %q 点名了 %d 个节点, want 2（恰好一个不可达节点加入口）", first, len(names))
			}
			if got := occurrences(names, tc.wantName); got != 1 {
				t.Errorf("拒绝文案 %q 里 %q 出现 %d 次, want 1（应唯一点名字典序最小的不可达节点）", first, tc.wantName, got)
			}
			if got := occurrences(names, tc.entry); got != 1 {
				t.Errorf("拒绝文案 %q 里入口 %q 出现 %d 次, want 1（调用方要靠它看出从哪出发算的可达）", first, tc.entry, got)
			}
			for _, bad := range tc.alsoUnreachable {
				if got := occurrences(names, bad); got != 0 {
					t.Errorf("拒绝文案 %q 点名了 %q（出现 %d 次），同图其它不可达节点必须留给后续诊断，一次只报一个", first, bad, got)
				}
			}

			res, runErr := runWithinLimit(t, g)
			if runErr == nil {
				t.Fatalf("含不可达节点的图仍可执行: Output=%v Completed=%v", res.Output(), res.Completed())
			}
			if res != nil {
				t.Errorf("Run 拒绝时仍返回 Result %v，调用方无从分辨成功与失败", res)
			}
			if got := executions.Load(); got != 0 {
				t.Errorf("Validate 拒绝前已执行 %d 次节点，副作用发生在校验之前", got)
			}
			for _, want := range anchors {
				if !strings.Contains(runErr.Error(), want) {
					t.Errorf("Run 的错误 %q 未含 %s，两个入口给出的定位不一致", runErr, want)
				}
			}
			// 两个入口传达同一缺陷时必须给出同一条文案，而不是「都含那个名字」：调用方
			// 从 Run 拿到的错误也要能看出只点了一个名字（契约 T2 把子串比较升级为整串比较）。
			if runErr.Error() != first {
				t.Errorf("Run 的拒绝与 Validate 不是同一条文案: %q vs %q", runErr.Error(), first)
			}
		})
	}

	// A3：静态可达但运行期永不执行，必须照旧放行。「可达」一旦被实现成「跑过」，
	// 这一条立刻红，而它正是条件路由的全部意义。
	t.Run("对照：条件边运行期不命中也算可达", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("a", passthrough)).
			AddNode(graph.NodeFunc("b", passthrough)).
			AddNode(graph.NodeFunc("end", passthrough)).
			AddEdge("a", "end").
			AddConditional("a", "b", func(any) bool { return false }).
			SetEntry("a").SetOutput("end")

		if err := g.Validate(); err != nil {
			t.Fatalf("b 只是运行期没被执行，Validate 却拒绝: %v", err)
		}
		res, err := runWithinLimit(t, g)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if want := []string{"a", "end"}; !reflect.DeepEqual(res.Completed(), want) {
			t.Errorf("Result.Completed() = %v, want %v", res.Completed(), want)
		}
		if v, ok := res.Value("b"); ok || v != nil {
			t.Errorf("Result.Value(\"b\") = %v, %v; want <nil>, false —— b 声明过但没跑过，ok 必须为假，"+
				"而图本身仍要合法（这正是「可达」不等于「执行过」的那条界线）", v, ok)
		}
	})

	// A6 前半 + 后半：同一条不可达的环现在报可达性，可达的环文案一字不动。
	t.Run("对照：不可达的环报可达性，可达的环仍报环", func(t *testing.T) {
		unreachableCycle := graph.New().
			AddNode(graph.NodeFunc("a", passthrough)).
			AddNode(graph.NodeFunc("c", passthrough)).
			AddNode(graph.NodeFunc("d", passthrough)).
			AddEdge("c", "d").AddEdge("d", "c").
			SetEntry("a").SetOutput("a")

		err := unreachableCycle.Validate()
		if err == nil {
			t.Fatal("不可达的无条件环被放行")
		}
		if !strings.Contains(err.Error(), "graph: unreachable node") {
			t.Errorf("两个检查同时成立时报了 %q，未先报更根本的不可达事实", err)
		}
		if strings.Contains(err.Error(), "unconditional cycle") {
			t.Errorf("不可达的环报成了 %q，把永远不会跑的东西说成了运行时风险", err)
		}

		reachableCycle := graph.New().
			AddNode(graph.NodeFunc("a", passthrough)).
			AddNode(graph.NodeFunc("b", passthrough)).
			AddEdge("a", "b").AddEdge("b", "a").
			SetEntry("a").SetOutput("a")

		err = reachableCycle.Validate()
		if err == nil {
			t.Fatal("可达的无条件环被放行，切片 10 的拒绝回归了")
		}
		for _, want := range []string{"graph: unconditional cycle", `"a"`, `"b"`} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("可达环的错误 %q 未含 %s，切片 10 的诊断被本轮抢走", err, want)
			}
		}
	})

	// A5：新增检查不得误伤三类合法形状，Completed 用精确等值。
	t.Run("对照：单节点图/多路汇聚/条件环照旧", func(t *testing.T) {
		single := graph.New().AddNode(graph.NodeFunc("only", passthrough)).SetEntry("only").SetOutput("only")
		fanIn := graph.New().
			AddNode(graph.NodeFunc("a", passthrough)).
			AddNode(graph.NodeFunc("b", passthrough)).
			AddNode(graph.NodeFunc("c", passthrough)).
			AddEdge("a", "b").AddEdge("a", "c").AddEdge("b", "c").
			SetEntry("a").SetOutput("c")

		trials := 0
		condCycle := graph.New().
			AddNode(graph.NodeFunc("a", passthrough)).
			AddNode(graph.NodeFunc("b", passthrough)).
			AddNode(graph.NodeFunc("end", passthrough)).
			AddEdge("a", "b").
			AddConditional("b", "a", func(any) bool { trials++; return trials < 2 }).
			AddConditional("b", "end", func(any) bool { return trials >= 2 }).
			SetEntry("a").SetOutput("end")

		cases := []struct {
			name string
			g    *graph.Graph
			want []string
		}{
			{"单节点图", single, []string{"only"}},
			{"两条来路", fanIn, []string{"a", "b", "c", "c"}},
			{"谓词会转假的环", condCycle, []string{"a", "a", "b", "b", "end"}},
		}
		for _, tc := range cases {
			if err := tc.g.Validate(); err != nil {
				t.Fatalf("%s 被误拒: %v", tc.name, err)
			}
			res, err := runWithinLimit(t, tc.g)
			if err != nil {
				t.Fatalf("%s Run: %v", tc.name, err)
			}
			if !reflect.DeepEqual(res.Completed(), tc.want) {
				t.Errorf("%s 的 Completed() = %v, want %v", tc.name, res.Completed(), tc.want)
			}
		}
	})

	// A7：既有诊断不得被新检查抢走。每张图都同时含一条不可达节点和一个更前面的缺陷。
	t.Run("对照：既有拒绝次序不被抢走", func(t *testing.T) {
		cases := []struct {
			name       string
			build      func() *graph.Graph
			wantAnchor string
		}{
			{"悬空边", func() *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("a", passthrough)).
					AddNode(graph.NodeFunc("orphan", passthrough)).
					AddEdge("a", "ghost").
					SetEntry("a").SetOutput("a")
			}, `to node "ghost"`},
			{"入口不存在", func() *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("a", passthrough)).
					AddNode(graph.NodeFunc("orphan", passthrough)).
					SetEntry("ghost").SetOutput("a")
			}, `entry node "ghost"`},
			{"输出指向未注册节点", func() *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("a", passthrough)).
					AddNode(graph.NodeFunc("orphan", passthrough)).
					SetEntry("a").SetOutput("ghost")
			}, `output node "ghost"`},
			{"条件边缺谓词", func() *graph.Graph {
				return graph.New().
					AddNode(graph.NodeFunc("a", passthrough)).
					AddNode(graph.NodeFunc("orphan", passthrough)).
					AddConditional("a", "a", nil).
					SetEntry("a").SetOutput("a")
			}, "declares no predicate"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.build().Validate()
				if err == nil {
					t.Fatalf("含不可达节点的 %s 图被放行，既有诊断没跑到", tc.name)
				}
				if !strings.Contains(err.Error(), tc.wantAnchor) {
					t.Errorf("错误 %q 未含 %s：新增的可达性检查抢在了既有诊断前面", err, tc.wantAnchor)
				}
			})
		}
	})
}
