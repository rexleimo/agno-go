package graph_test

// 本文件是 P1 票面 B11（`graph.Typed` 泛型包装）这一片的行为族，契约见
// docs/design/v3-test-scope-p1-graph-slice18.json（验收行 D1–D7 与本文件的 Test 一一对应，
// D8 是命令背书的结构性护栏行，刻意不写成「测试读自身源码」的自指断言）。
//
// 票面 :70 的字面要求有两半：「接到 TIn、返回 TOut」与「对错误类型输入返回可判定错误
// 而非 panic」。第二半的「可判定」在本片按 §11.1 先例精确化为**可归因**：基线字节上
// graph.go:491 的 Typed 对四种不同成因（入口即不匹配、下游不匹配、nil 输入、切片进接口）
// 交出逐字相同的 `graph: input type mismatch`（契约 T-2），调用方在多节点图上无法知道是
// 哪一个节点坏了，也无法知道期望与实际各是什么类型——这一条按票面判据不合格。
//
// 第一半的「编译期类型安全」是调用方编译期事实，无法在本包内复现为运行时可观察的差异，
// 因此 D1 只钉运行后 TIn/TOut 确实按具体类型流动（green-at-red，teethMutant 见契约）。
//
// 观察面只用 graph 的导出 API（New/AddNode/AddEdge/SetEntry/SetOutput/Run、Result 的
// Output/Value/Completed）与 errors.Is。不读 cfg/plan/scheduler 私有字段，不观察 queue
// 长度或 goroutine 数（票面 :40 明令禁止）。类型不符是确定性事件，本片所有夹具都与调度
// 时序无关：文件里没有一处 time.Sleep 参与定序，也不需要 channel 握手（票面 :75）。
//
// 反作弊形态（契约 forbiddenShortcuts 第 1、2 条）：不得把期望值改成基线文案
// `input type mismatch`，也不得放宽成「文本里出现字母 a」——D4 要求引号包裹的完整名字
// 边界，并反向断言不含 "a"/"seed"。

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// 类型标签的判据只扫描「去掉引号包裹片段之后的错误文本」。
//
// 原因不是洁癖：夹具里的节点名本身就带类型词（"needstring" 含 string、"needint" 含 int），
// 直接在整段文本上匹配会让「点名节点」与「报出类型」两件事互相顶包——把期望类型写成
// 节点名也能蒙过正则。剥离引号片段后，标签判据只可能由真正的类型描述满足；引号片段本身
// 由 D2/D4/D5 的点名断言单独负责，两面各管一段，互不替补。
var (
	p1r18QuotedSpan = regexp.MustCompile(`"[^"]*"`)

	// 只要求两个标签与两个类型词按顺序出现，不钉具体措辞：实现可自由选择文案。
	p1r18ExpStringGotInt = regexp.MustCompile(`(?i)expected[^,]*string.*got[^,]*int`)
	p1r18ExpIntGotString = regexp.MustCompile(`(?i)expected[^,]*int.*got[^,]*string`)
	// 同词充数：两个槽位写成同一个类型词（expected string, got string）即判红。
	p1r18SameWordString = regexp.MustCompile(`(?i)expected.*string.*got.*string`)
	p1r18SameWordInt    = regexp.MustCompile(`(?i)expected.*int.*got.*int`)
	// got 槽位表示「无值」的三种可接受渲染（契约 D5）。
	p1r18GotNoValue = regexp.MustCompile(`(?i)got[^a-z]*(no value|<nil>|\bnil\b)`)
)

// p1r18Labels 返回剥离引号片段之后的错误文本，供类型标签判据使用。
func p1r18Labels(text string) string {
	return p1r18QuotedSpan.ReplaceAllString(text, "")
}

// TestP1R18_TypedCarriesConcreteInAndOutTypes 覆盖契约 D1（票面 :70 前半句）。
//
// 场景：Typed[string]int("len") 的下游是一个只能按 int 读出值的 NodeFunc("show")。
// 期望：TIn 以 string 进入、TOut 以 int 流出——下游收到的是具体 int 而不是 any 里的
// 其它形状，Result.Value("len") 也是 int，Output() 由下游按 int 拼出。
// 这是 green-at-red 护栏行：基线已成立，但修复（改错误构造、改断言分支）一旦把 TOut
// 丢弃或把 in 原样透传，本行判红。
func TestP1R18_TypedCarriesConcreteInAndOutTypes(t *testing.T) {
	var downstreamKind string

	lenNode := graph.Typed("len", func(_ context.Context, s string) (int, error) {
		return len(s), nil
	})
	showNode := graph.NodeFunc("show", func(_ context.Context, in any) (any, error) {
		n, ok := in.(int)
		if !ok {
			return nil, fmt.Errorf("show 收到的不是具体 int，而是 %T", in)
		}
		downstreamKind = fmt.Sprintf("%T", in)
		return fmt.Sprintf("n=%d typed=%s", n, downstreamKind), nil
	})

	g := graph.New().
		AddNode(lenNode).
		AddNode(showNode).
		AddEdge("len", "show").
		SetEntry("len").
		SetOutput("show")

	res, err := g.Run(context.Background(), "abcd")
	if err != nil {
		t.Fatalf("Run(ctx, \"abcd\") 返回错误, want nil: %v", err)
	}
	if got, want := res.Output(), "n=4 typed=int"; got != want {
		t.Errorf("Result.Output() = %v, want %v（TOut 必须以具体类型流到下游）", got, want)
	}
	if got, ok := res.Value("len"); !ok || got != 4 {
		t.Errorf("Result.Value(\"len\") = %#v, %v; want int(4), true", got, ok)
	}
	if got, want := res.Completed(), []string{"len", "show"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Result.Completed() = %v, want %v", got, want)
	}
}

// TestP1R18_TypeMismatchNamesTheOffendingNode 覆盖契约 D2，本片的红候选行。
//
// 场景：单节点图，唯一节点是 Typed[string]int("needstring")，Run 交出 int(42)。
// 期望：错误文本包含以引号包裹的 offending 节点名 "needstring"；Result 仍为 nil、
// 进程不 panic（这两段反向边界基线已成立，修复不得改坏）。
func TestP1R18_TypeMismatchNamesTheOffendingNode(t *testing.T) {
	g := graph.New().
		AddNode(graph.Typed("needstring", func(_ context.Context, s string) (int, error) {
			return len(s), nil
		})).
		SetEntry("needstring").
		SetOutput("needstring")

	res, err := g.Run(context.Background(), 42)
	if err == nil {
		t.Fatalf("Run(ctx, 42) 传入与 TIn=string 不符的 int 却未返回错误, res=%v", res)
	}
	if res != nil {
		t.Errorf("类型不符时仍交出非 nil Result (%v), want nil（票面 :70「可判定错误而非 panic」）", res)
	}
	if got := err.Error(); !strings.Contains(got, `"needstring"`) {
		t.Errorf("Run 的错误文本 %q 未包含以引号包裹的 offending 节点名 \"needstring\"；"+
			"调用方在多节点图上无法定位失败节点（契约 D2 可归因判据）", got)
	}
}

// TestP1R18_TypeMismatchDistinguishesExpectedFromActual 覆盖契约 D3。
//
// 两个镜像形状（string 收 int、int 收 string）：把两个槽位写成同一个词、或整段只报
// 期望类型而不报实际类型，都至少在其中一形状上判红。
func TestP1R18_TypeMismatchDistinguishesExpectedFromActual(t *testing.T) {
	tests := []struct {
		name  string
		node  graph.Node
		in    any
		match *regexp.Regexp
		same  *regexp.Regexp
	}{
		{
			name:  "want-string-got-int",
			node:  graph.Typed("needstring", func(_ context.Context, s string) (int, error) { return len(s), nil }),
			in:    42,
			match: p1r18ExpStringGotInt,
			same:  p1r18SameWordString,
		},
		{
			name:  "want-int-got-string",
			node:  graph.Typed("needint", func(_ context.Context, n int) (int, error) { return n * 2, nil }),
			in:    "not-an-int",
			match: p1r18ExpIntGotString,
			same:  p1r18SameWordInt,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.New().
				AddNode(tc.node).
				SetEntry(tc.node.Name()).
				SetOutput(tc.node.Name())

			res, err := g.Run(context.Background(), tc.in)
			if err == nil {
				t.Fatalf("Run(%#v) 与 TIn 不符却未返回错误, res=%v", tc.in, res)
			}
			if res != nil {
				t.Errorf("类型不符时仍交出非 nil Result (%v), want nil", res)
			}
			labels := p1r18Labels(err.Error())
			if loc := tc.match.FindStringIndex(labels); loc == nil {
				t.Errorf("Run 的错误文本 %q（剥离引号片段后 %q）未同时给出期望类型与实际类型，"+
					"要求形状：%s（契约 D3：两个槽位必须分别可判定）", err.Error(), labels, tc.match.String())
			}
			if tc.same.MatchString(labels) {
				t.Errorf("Run 的错误文本 %q（剥离引号片段后 %q）把 expected 与 got 写成同一个类型词，"+
					"两个槽位失去区分（契约 D3 反作弊）", err.Error(), labels)
			}
		})
	}
}

// TestP1R18_TypeMismatchNamesTheFailingNodeNotJustAnyNode 覆盖契约 D4。
//
// 三节点链 seed(any→string) → a=Typed[string]string → b=Typed[int]int：a 正常跑完，
// 不匹配发生在 b。断言点名 "b" 且不含 "a"、不含 "seed"。D2 的单节点夹具无法区分
// 「点名失败节点」与「点名唯一一个节点」，这一混淆只能由本行排除。
func TestP1R18_TypeMismatchNamesTheFailingNodeNotJustAnyNode(t *testing.T) {
	g := graph.New().
		AddNode(graph.NodeFunc("seed", func(_ context.Context, _ any) (any, error) { return "a string", nil })).
		AddNode(graph.Typed("a", func(_ context.Context, s string) (string, error) { return s + "!", nil })).
		AddNode(graph.Typed("b", func(_ context.Context, n int) (int, error) { return n * 2, nil })).
		AddEdge("seed", "a").
		AddEdge("a", "b").
		SetEntry("seed").
		SetOutput("b")

	res, err := g.Run(context.Background(), nil)
	if err == nil {
		t.Fatalf("a 交出 string 而 b 要求 int，Run 却未返回错误, res=%v", res)
	}
	if res != nil {
		t.Errorf("类型不符时仍交出非 nil Result (%v), want nil", res)
	}
	text := err.Error()
	if !strings.Contains(text, `"b"`) {
		t.Errorf("Run 的错误文本 %q 未点名真正失败的节点 \"b\"（契约 D4：不得写死成入口节点或第一个 Typed 节点）", text)
	}
	if strings.Contains(text, `"a"`) {
		t.Errorf("Run 的错误文本 %q 点名了正常跑完的节点 \"a\"，归因错位", text)
	}
	if strings.Contains(text, `"seed"`) {
		t.Errorf("Run 的错误文本 %q 点名了正常跑完的节点 \"seed\"，归因错位", text)
	}
}

// TestP1R18_NilInputIsAttributableTypeMismatch 覆盖契约 D5。
//
// 两种 nil 形状都走类型不符分支（T-3 实测：Go 的接口断言对 nil 一律失败，
// 「接口 TIn 收到 nil 合法」的直觉不成立）：具体 TIn=string 与接口 TIn=io.Reader。
// 除点名之外还钉 got 槽位必须渲染成表示「无值」的字样，不得打印成某个具体类型名，
// 也不得因 reflect.TypeOf(nil) 而崩。
func TestP1R18_NilInputIsAttributableTypeMismatch(t *testing.T) {
	tests := []struct {
		name string
		node graph.Node
		want string
	}{
		{
			name: "concrete-TIn",
			node: graph.Typed("needstring", func(_ context.Context, s string) (int, error) { return len(s), nil }),
			want: `"needstring"`,
		},
		{
			name: "interface-TIn",
			node: graph.Typed("needreader", func(_ context.Context, r io.Reader) (string, error) {
				return fmt.Sprintf("%T", r), nil
			}),
			want: `"needreader"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.New().
				AddNode(tc.node).
				SetEntry(tc.node.Name()).
				SetOutput(tc.node.Name())

			res, err := g.Run(context.Background(), nil)
			if err == nil {
				t.Fatalf("Run(ctx, nil) 与 TIn 不符却未返回错误, res=%v", res)
			}
			if res != nil {
				t.Errorf("nil 输入的类型不符仍交出非 nil Result (%v), want nil", res)
			}
			text := err.Error()
			if !strings.Contains(text, tc.want) {
				t.Errorf("Run 的错误文本 %q 未包含以引号包裹的 offending 节点名 %s", text, tc.want)
			}
			if !p1r18GotNoValue.MatchString(p1r18Labels(text)) {
				t.Errorf("Run 的错误文本 %q 的 got 槽位未渲染成表示「无值」的字样（接受 no value | <nil> | nil）", text)
			}
		})
	}
}

// TestP1R18_NodeErrorPassesThroughUnchanged 覆盖契约 D6（green-at-red 护栏行）。
//
// Typed 内 fn 自己返回的哨兵必须逐字交出、可被 errors.Is 命中，且不得被类型语义污染：
// GREEN 若把两条错误构造合并（例如一律包成类型不符），本行判红而 D2 仍可绿。
func TestP1R18_NodeErrorPassesThroughUnchanged(t *testing.T) {
	sentinel := errors.New("p1r18: store connection reset by peer")

	g := graph.New().
		AddNode(graph.Typed("fail", func(_ context.Context, _ string) (int, error) {
			return 0, sentinel
		})).
		SetEntry("fail").
		SetOutput("fail")

	res, err := g.Run(context.Background(), "abcd")
	if !errors.Is(err, sentinel) {
		t.Fatalf("errors.Is(err, sentinel) = false, want true；Run 交出的错误 = %v", err)
	}
	if res != nil {
		t.Errorf("fn 返回业务错误时仍交出非 nil Result (%v), want nil", res)
	}
	text := err.Error()
	if strings.Contains(text, "expected") || strings.Contains(text, "got") {
		t.Errorf("节点业务错误被改写成类型不符的形状：%q（契约 D6：不得与类型分支共用错误构造）", text)
	}
}

// TestP1R18_TypeMismatchStaysDistinctFromGraphSentinels 覆盖契约 D7（green-at-red 护栏行）。
//
// 三条路径各自独立：类型不符本身、刻意把步数预算压到 1 的同型夹具、以及一次正常的
// 未取消收敛。前两条必须既不是 ErrStepLimitExceeded 也不是 context.Canceled（承接票面
// §13.1 反向牙齿的同族写坏法：把类型不符包成取消或包成预算结论）；第三条必须正常收敛。
func TestP1R18_TypeMismatchStaysDistinctFromGraphSentinels(t *testing.T) {
	needstring := func() graph.Node {
		return graph.Typed("needstring", func(_ context.Context, s string) (int, error) { return len(s), nil })
	}
	single := func(opts ...graph.Option) *graph.Graph {
		return graph.New(opts...).
			AddNode(needstring()).
			SetEntry("needstring").
			SetOutput("needstring")
	}

	t.Run("plain-mismatch", func(t *testing.T) {
		_, err := single().Run(context.Background(), 42)
		if err == nil {
			t.Fatalf("Run 未返回错误, want 类型不符")
		}
		if errors.Is(err, graph.ErrStepLimitExceeded) {
			t.Errorf("类型不符被报成 ErrStepLimitExceeded: %v", err)
		}
		if errors.Is(err, context.Canceled) {
			t.Errorf("类型不符被归一化成 context.Canceled: %v（票面 §13 的 B9 语义与本行互斥）", err)
		}
	})

	t.Run("mismatch-under-tight-budget", func(t *testing.T) {
		_, err := single(graph.WithStepLimit(1)).Run(context.Background(), 42)
		if err == nil {
			t.Fatalf("Run 未返回错误, want 类型不符")
		}
		if errors.Is(err, graph.ErrStepLimitExceeded) {
			t.Errorf("步数预算顶掉了类型不符的结论：%v（期望错误点名类型不符，而不是超限）", err)
		}
		if errors.Is(err, context.Canceled) {
			t.Errorf("类型不符被归一化成 context.Canceled: %v", err)
		}
	})

	t.Run("uncanceled-graph-still-converges", func(t *testing.T) {
		g := graph.New().
			AddNode(needstring()).
			AddNode(graph.NodeFunc("show", func(_ context.Context, in any) (any, error) {
				return fmt.Sprintf("n=%v", in), nil
			})).
			AddEdge("needstring", "show").
			SetEntry("needstring").
			SetOutput("show")

		res, err := g.Run(context.Background(), "abcde")
		if err != nil {
			t.Fatalf("未取消、未超限的图未收敛: %v", err)
		}
		if got, want := res.Output(), "n=5"; got != want {
			t.Errorf("Result.Output() = %v, want %v", got, want)
		}
	})
}
