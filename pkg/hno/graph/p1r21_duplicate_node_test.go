package graph_test

import (
	"context"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// 票面 B6 第 2 项（docs/design/v3-test-scope-p1-graph.md:65、母约 §3.5 第 2 项）：
// 重复节点名须在构建期被点名拒绝。契约
// docs/design/v3-test-scope-p1-graph-slice21.json 的 D1–D5 逐行一测。
//
// 观察面只有导出 API：New/AddNode/NodeFunc/SetEntry/SetOutput/Validate/Run 与
// Result.Output/Completed。票面 :40 禁止的内部面一行未碰。

// p1r21Static 返回一个与名字无关、只用来占位的节点实现。
func p1r21Static(name, out string) graph.Node {
	return graph.NodeFunc(name, func(_ context.Context, _ any) (any, error) { return out, nil })
}

// TestP1R21_DuplicateNodeNameIsRejectedLocatably 观察契约 D1：同名节点注册两次后，
// Validate 须返回非 nil，并且错误文本带引号边界点名那个名字（票面 §11.1 的可定位判据）。
// 图取契约 M1 形2 的原样（重复 entry + 入口即输出）：除重复名外其余声明全部成立，
// 错误只可能来自重复名检查本身 —— 若图上另有未可达的节点，既有的 unreachable 文案会
// 替这条断言顶名，牙齿就假了（首次 RED 实测踩中，见红文档 §1）。
func TestP1R21_DuplicateNodeNameIsRejectedLocatably(t *testing.T) {
	g := graph.New().
		AddNode(p1r21Static("entry", "E1")).
		AddNode(p1r21Static("entry", "E2")).
		SetEntry("entry").
		SetOutput("entry")

	err := g.Validate()
	if err == nil {
		t.Fatalf(`重复注册同名节点 "entry" 的图被 Validate 放行（返回 nil），构建期一句话都没有`)
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "graph: ") {
		t.Errorf(`错误文本 %q 未以 "graph: " 前缀开头，调用方无法判定归因来自本引擎`, msg)
	}
	if !strings.Contains(msg, `"entry"`) {
		t.Errorf(`错误文本 %q 未带引号边界点名被重复注册的名字 "entry"，不可定位`, msg)
	}
}

// TestP1R21_RunRefusesGraphWithDuplicateNodeName 观察契约 D2：Run 不得跑半张图，
// 更不得把后注册者的输出当成结论交出去。
func TestP1R21_RunRefusesGraphWithDuplicateNodeName(t *testing.T) {
	g := graph.New().
		AddNode(p1r21Static("entry", "E1")).
		AddNode(p1r21Static("entry", "E2")).
		SetEntry("entry").
		SetOutput("entry")

	validateErr := g.Validate()
	res, runErr := g.Run(context.Background(), nil)
	if validateErr == nil {
		t.Fatalf("Run 之前 Validate 返回 nil：重复节点名未在构建期被拒（Run err=%v）", runErr)
	}
	if res != nil {
		t.Errorf("带重复节点名的 Run 交出了 Result（Output()=%v），应交出 nil 而不是半张图的结论", res.Output())
	}
	if runErr == nil {
		t.Fatalf("Run 未拒绝带重复节点名的图，返回了 nil error")
	}
	if runErr.Error() != validateErr.Error() {
		t.Errorf("Run 交出的错误 %q 与 Validate 的 %q 不一致，调用方拿到两种说法", runErr.Error(), validateErr.Error())
	}
}

// TestP1R21_DistinctNamesStillValidateAndRun 观察契约 D3：判据只拒「重复的名字」，
// 不得顺手收紧成「不许注册」或「图一旦跑过就冻结」。全程不同名的合法图必须照常
// 通过构建期并跑出末端节点的返回值。
func TestP1R21_DistinctNamesStillValidateAndRun(t *testing.T) {
	g := graph.New().
		AddNode(p1r21Static("entry", "E")).
		AddNode(p1r21Static("mid", "M")).
		AddNode(p1r21Static("out", "O")).
		AddEdge("entry", "mid").
		AddEdge("mid", "out").
		SetEntry("entry").
		SetOutput("out")

	if err := g.Validate(); err != nil {
		t.Fatalf("全程不同名的合法图被 Validate 拒绝（%v）：判据被收紧到了名字之外", err)
	}
	res, err := g.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("合法图的 Run 失败（%v）：判据被收紧到了名字之外", err)
	}
	if res == nil || res.Output() != "O" {
		t.Fatalf("合法图 Run 的 Output()=%v（res nil=%v），末端节点应交出 O", res.Output(), res == nil)
	}
	completed := res.Completed()
	if len(completed) != 3 || completed[0] != "entry" || completed[1] != "mid" || completed[2] != "out" {
		t.Errorf("Completed()=%v，三个节点的形状应保持字典序全量", completed)
	}
}

// TestP1R21_NewNameAfterRunStaysVisible 观察契约 D4：Run 之后再注册**新名字**的节点
// 必须对下一次 Run 可见并参与执行 —— 既有担保「Run 之后新注册的节点对下一次 Run 可见」
// （契约 M1 形3 实测）不得因重复名检查退化成「终身冻结」。
func TestP1R21_NewNameAfterRunStaysVisible(t *testing.T) {
	g := graph.New().
		AddNode(p1r21Static("entry", "A")).
		SetEntry("entry").
		SetOutput("entry")

	res1, err1 := g.Run(context.Background(), nil)
	if err1 != nil {
		t.Fatalf("第一次 Run 失败（%v）：与名字判据无关的既有行为不得被殃及", err1)
	}
	if res1 == nil || res1.Output() != "A" {
		t.Fatalf("第一次 Run 的 Output()=%v（res nil=%v），want A", res1.Output(), res1 == nil)
	}

	g.AddNode(p1r21Static("b", "B")).
		AddEdge("entry", "b").
		SetOutput("b")

	res2, err2 := g.Run(context.Background(), nil)
	if err2 != nil {
		t.Fatalf("注册新名字 b 后的第二次 Run 被拒（%v）：检查不得退化成「注册过就冻结」", err2)
	}
	if res2 == nil || res2.Output() != "B" {
		t.Errorf("第二次 Run 的 Output()=%v（res nil=%v），新注册的 b 应参与执行并交出 B", res2.Output(), res2 == nil)
	}
}

// TestP1R21_MultipleDuplicateNamesStillNameOne 观察契约 D5：一张图里 b、c 各重复一次时，
// 错误必须点名其中之一并给出可判定的名字（引号边界）。不断点名的是哪一个、不断一次报几个 ——
// 点名次序依赖 map 遍历序或内部记账序，按票面 :40 不在公共面担保范围。
//
// 这张图除重复名外全部成立（端点齐全、全部可达、无环），因此错误只可能来自重复名检查本身；
// 否则「unreachable node "b"」之类的既有文案会替这条断言顶名，牙齿就假了。
func TestP1R21_MultipleDuplicateNamesStillNameOne(t *testing.T) {
	g := graph.New().
		AddNode(p1r21Static("entry", "E")).
		AddNode(p1r21Static("b", "B1")).
		AddNode(p1r21Static("b", "B2")).
		AddNode(p1r21Static("c", "C1")).
		AddNode(p1r21Static("c", "C2")).
		AddNode(p1r21Static("out", "O")).
		AddEdge("entry", "b").
		AddEdge("b", "c").
		AddEdge("c", "out").
		SetEntry("entry").
		SetOutput("out")

	err := g.Validate()
	if err == nil {
		t.Fatalf("b、c 各重复一次的图被 Validate 放行（返回 nil），重复名检查整段失效")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"b"`) && !strings.Contains(msg, `"c"`) {
		t.Errorf("错误文本 %q 既未点名 %q 也未点名 %q，调用方拿不到可判定的名字", msg, "b", "c")
	}
}
