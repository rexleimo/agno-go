package graph_test

// 本文件是切片 15b 的行为族，契约见 docs/design/v3-test-scope-p1-graph-slice15b.json。
// 观察面只用 graph 的公共入口（New/AddNode/AddEdge/AddConditional/SetEntry/SetOutput/
// Validate/Warnings/Run + Result 的读方法），不读 cfg、不调 cycleOf/firstCycleOf，也不对
// 整条文案做快照式全文相等断言——本片要改的正是措辞本身（契约 allowedTestSeam 禁观察面）。
//
// 夹具与 p1r15_warnings_test.go 共用（conditionalCycleGraph / stepLimitWarnings），那个文件
// 在本片字节不变；这里新增的行只往「文案说了什么」这一面加约束，不重开切片 15 的判据。

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

const (
	// maxNamedNodes/warningSizeBound 是契约 U4 的两条界。界值不是审美：点名要够到「哪条环」
	// （环的头尾），字节上界要保证 Warnings() 作为「启动时打印一次」的天然用法在大图上仍然
	// 可读。改动前实测 400 节点的环 = 整条列出、4000+ 字节，20000 节点的环 = 260134 字节。
	maxNamedNodes    = 6
	warningSizeBound = 1024

	// longCycleNodes 是 U4/U6 场景的环长：够长到把「整条列出」和「有界列出」分开，
	// 又不至于让本行变成压力测试。
	longCycleNodes = 400

	// defaultStepLimitNumber 是母约 §7 与票面 :27 定的默认预算。测试不读私有常量，
	// 而是在文案里找这个独立数字——它是调用方没选过的数，正是提示要交出去的东西。
	defaultStepLimitNumber = 1000
)

var (
	// quotedNodeNames 认出文案里被点名的节点：%q 是环渲染唯一的节点输出方式。
	quotedNodeNames = regexp.MustCompile(`"[^"]*"`)
	// digitRuns 用来判「某个数是否被作为独立数字说出来」，而不是钉死它写的是 "400 nodes" 还是「长度 400」。
	digitRuns = regexp.MustCompile(`\d+`)
	// runMethodName 判文案有没有拿某一次 Run 的结果说话。导出的方法名是执行预测的签名——
	// 契约 U2/U3 的 observable 就是「不含 Run」，而 \b 让 "running"、"runtime" 这类词不误伤。
	runMethodName = regexp.MustCompile(`\bRun\b`)
)

// longConditionalCycle 搭一张 n 个节点的条件环：entry → n0000 → … → n(n-1)，闭合那条是
// 条件边，所以 §3.5 第 7 项放行（环上有一条边可被跳过），而环确实有 n 个节点。
//
// 命名用四位补零：n0000…n0399 的数字串里没有一条等于 400，也没有一条等于 1000，因此
// 「文案里出现了独立数字 400 / 1000」只能是在说环长或预算，不是节点名里的巧合。
func longConditionalCycle(n int) *graph.Graph {
	noop := func(_ context.Context, in any) (any, error) { return in, nil }
	never := func(any) bool { return false }

	first := longCycleName(0)
	g := graph.New().
		AddNode(graph.NodeFunc("entry", noop)).
		AddNode(graph.NodeFunc(first, noop)).
		AddEdge("entry", first).
		SetEntry("entry").SetOutput("entry")

	prev := first
	for i := 1; i < n; i++ {
		name := longCycleName(i)
		g.AddNode(graph.NodeFunc(name, noop))
		g.AddEdge(prev, name)
		prev = name
	}
	return g.AddConditional(prev, first, never)
}

// longCycleName 与 longConditionalCycle 共用同一套命名，避免夹具与「声明过的节点集合」各写一遍。
func longCycleName(i int) string { return fmt.Sprintf("n%04d", i) }

// declaredCycleNodes 交出夹具声明的节点名集合——测试自己知道声明了什么，所以拿它核对文案
// 是公共可观察判据，不是读私有字段（契约 U6）。
func declaredCycleNodes(n int) map[string]bool {
	declared := map[string]bool{"entry": true}
	for i := 0; i < n; i++ {
		declared[longCycleName(i)] = true
	}
	return declared
}

// onlyWarningOfThisFamily 取本族唯一在管的那条告警：切片 15 的判据 8 在当前图上只产出一条，
// 多条告警并存时的条数与排序属未裁决面（S15-REFACTOR-1），因此这里按「至少一条、取第一条」
// 观察，不把条数钉死。
func onlyWarningOfThisFamily(t *testing.T, g *graph.Graph) string {
	t.Helper()
	hits := stepLimitWarnings(g.Warnings())
	if len(hits) == 0 {
		t.Fatal("Warnings() 没有交出「步数上限 + 环」这一条告警：告警缺席时本行会退化成空断言")
	}
	return hits[0]
}

// statesNumber 判某个整数是否作为独立数字出现在文案里。
func statesNumber(msg string, want int) bool {
	for _, run := range digitRuns.FindAllString(msg, -1) {
		if n, err := strconv.Atoi(run); err == nil && n == want {
			return true
		}
	}
	return false
}

// TestP1R15B_WarningNamesTheKnobAndItsDefaultNumber 覆盖契约 U1（票面 :27 + :66 B7）。
//
// 今天这两半「碰巧」都在文案里，但没有任何一行在管（契约 M-3：测试侧对默认预算的断言只有
// 「未显式设置时撞上限」那族）。不加这一行，下一次改文案把数字换成「默认预算」四个字，
// 调用方照样读不到他能改什么、上限是多少。
func TestP1R15B_WarningNamesTheKnobAndItsDefaultNumber(t *testing.T) {
	msg := onlyWarningOfThisFamily(t, conditionalCycleGraph())

	if !strings.Contains(msg, "WithStepLimit") {
		t.Errorf("告警 %q 没点名旋钮 WithStepLimit：只说「有环没预算」不给调用方可拧的开关名，等于让人自己去翻文档", msg)
	}
	if !statesNumber(msg, defaultStepLimitNumber) {
		t.Errorf("告警 %q 没把默认预算 %d 作为独立数字交出来：那是调用方没选过的数，不写出来他不知道不设置会停在哪一步", msg, defaultStepLimitNumber)
	}
}

// TestP1R15B_WarningMakesNoClaimAboutRun 覆盖契约 U2（审查 blocking S15-SPEC-1）。
//
// 这句不是风格洁癖：改动前文案写的是「so Run stops only when the default stepLimit=1000 is
// reached」，而同族切片 15 的 T4 实测这张图 Completed()=[entry,ping]、Run 返回 nil —— 它停在
// 自己的收敛上，根本没撞上 1000。仓库对同一类问题早有判红口径（graph_test.go:1238 把「不会
// 跑的东西说成运行时风险」当缺陷），本行只是把那条口径从 Validate 的致命文案延伸到提示文案。
func TestP1R15B_WarningMakesNoClaimAboutRun(t *testing.T) {
	msg := onlyWarningOfThisFamily(t, conditionalCycleGraph())

	if loc := runMethodName.FindStringIndex(msg); loc != nil {
		t.Errorf("告警 %q 在 %d 处提到 Run：提示只能对构建期可见的事实说话（有环、没设预算）。Run 这个词一旦出现在文案里就是对某次执行结果的断言，而 Warnings() 没有任何依据支持它", msg, loc[0])
	}
}

// TestP1R15B_InvalidGraphWarningAlsoMakesNoRunClaim 覆盖契约 U3。
//
// 非法图侧今天实测就是「只在合法图上清洗、非法图分支保留原句」的形态：Warnings() 不查图是否
// 合法，所以无条件环图上照样交出同一句 Run 预测——而那张图 Run 一步都不会跑（scheduler.Run
// 先 Validate）。本行刻意对「非法图上该不该出提示」保持中立（R15-UNADJ-1 未裁决）：提示为空
// 通过，非空则每条都不得预测执行。
func TestP1R15B_InvalidGraphWarningAlsoMakesNoRunClaim(t *testing.T) {
	noop := func(_ context.Context, in any) (any, error) { return in, nil }
	invalid := graph.New().
		AddNode(graph.NodeFunc("a", noop)).
		AddNode(graph.NodeFunc("b", noop)).
		AddEdge("a", "b").AddEdge("b", "a").
		SetEntry("a").SetOutput("a")

	if err := invalid.Validate(); err == nil {
		t.Fatal("无条件环被 Validate() 放行：判据 7 的致命语义回归了，本行的前提（这是一张非法图）不成立")
	}
	for i, msg := range stepLimitWarnings(invalid.Warnings()) {
		if loc := runMethodName.FindStringIndex(msg); loc != nil {
			t.Errorf("非法图上第 %d 条告警 %q 在 %d 处提到 Run：同一张图 Run 一步都不跑，把这句留在非法图分支上就是「合法图改了、非法图没改」的那半个缺陷", i+1, msg, loc[0])
		}
	}
}

// TestP1R15B_LongCycleWarningIsBoundedAndStillLocatable 覆盖契约 U4（审查 blocking S15-STD-5）。
//
// 三条断言是一件事的两面：文案不能随环长无限膨胀（点名数、字节数各有上界），但收缩之后调用方
// 仍要知道「哪条环」「有多长」。只做到前者是把诊断信息扔掉，只做到后者是换个方向继续膨胀。
func TestP1R15B_LongCycleWarningIsBoundedAndStillLocatable(t *testing.T) {
	g := longConditionalCycle(longCycleNodes)
	if err := g.Validate(); err != nil {
		t.Fatalf("长条件环被 Validate() 拒绝（%v）：本行前提是这张图合法可跑，前提变了就不是文案的问题", err)
	}
	msg := onlyWarningOfThisFamily(t, g)

	if names := quotedNodeNames.FindAllString(msg, -1); len(names) > maxNamedNodes {
		t.Errorf("文案点名 %d 个节点，上界 %d：整条环抄进错误信息里，%d 节点的环就要读 %d 个名字，调用方只会跳过整段（本轮实测文案 %d 字节）",
			len(names), maxNamedNodes, longCycleNodes, len(names), len(msg))
	}
	if len(msg) >= warningSizeBound {
		t.Errorf("告警文案长 %d 字节，未过 %d 的上界：长度跟着环长线性增长，20000 节点的环实测 260134 字节",
			len(msg), warningSizeBound)
	}
	if !statesNumber(msg, longCycleNodes) {
		t.Errorf("文案 %s 里没有把环长 %d 作为独立数字说出来：截断之后调用方看不出这环有多大，也就没法判断默认预算够不够",
			truncateForEvidence(msg, 320), longCycleNodes)
	}
}

// TestP1R15B_ShortCycleStaysFullyRendered 覆盖契约 U5。
//
// 加界的动机不能变成新的不可定位：2 节点环是实务里最常见的那条（ping⇄pong），它的整条引用
// 本来只占十几字节，界对它必须完全透明——包括收尾重复的那个节点名，否则「回到哪儿」就丢了。
func TestP1R15B_ShortCycleStaysFullyRendered(t *testing.T) {
	msg := onlyWarningOfThisFamily(t, conditionalCycleGraph())

	const wantRef = `"ping" -> "pong" -> "ping"`
	if !strings.Contains(msg, wantRef) {
		t.Errorf("2 节点环的引用没有逐字交出 %s，实际文案 %q：界把短环也截了，收尾节点丢失后调用方看不出环回到哪里", wantRef, truncateForEvidence(msg, 320))
	}
}

// TestP1R15B_NamedNodesComeFromTheGraph 覆盖契约 U6，顺带在有界渲染路径上关掉切片 15 挂号的
// S15-REFACTOR-2：当年的存活变异 (i) 把环名换成常量 "a" -> "b" -> "a"，全部 7 行都还是绿的。
//
// 两种形状都要查：短环看「点名的就是 ping/pong」，长环看「截断后剩下的点名仍来自声明集合」——
// 只在短环上查会放过常量（ping/pong 之外无信息），只在长环上查会放过「界之后干脆写死一个样例名」。
func TestP1R15B_NamedNodesComeFromTheGraph(t *testing.T) {
	long := longConditionalCycle(longCycleNodes)
	longMsg := onlyWarningOfThisFamily(t, long)
	declaredLong := declaredCycleNodes(longCycleNodes)
	for _, name := range quotedNodeNames.FindAllString(longMsg, -1) {
		if !declaredLong[strings.Trim(name, `"`)] {
			t.Errorf("长环告警点名 %s，但它不在测试声明过的节点集合里：环名是从别处抄来的常量，调用方按它去图里找节点会一无所获", name)
		}
	}

	shortMsg := onlyWarningOfThisFamily(t, conditionalCycleGraph())
	declaredShort := map[string]bool{"entry": true, "ping": true, "pong": true}
	for _, name := range quotedNodeNames.FindAllString(shortMsg, -1) {
		if !declaredShort[strings.Trim(name, `"`)] {
			t.Errorf("短环告警点名 %s，但它不在 entry/ping/pong 里：同上，写死的样例名在这里同样是存活形态", name)
		}
	}
}

// TestP1R15B_JudgementSevenMessageUnchanged 是契约 U7 的护栏行，不新增判据。
//
// 为什么要在这里重复一遍 graph_test.go 已经覆盖的文案：本片动的渲染器是判据 7 与判据 8 共用的
// （firstCycleOf → 环文案）。契约 M-4 量出既有 B5 族的环长全部 ≤2，因此「保留前 6 个点名」的界
// 不该碰到它们的锚点；这条行就是那个推理的落地检查，界一旦被实现成「只留第一个节点」或整条省略，
// 判据 7 的致命文案会先在这里变红，而不是让 B5 族在自己的文件里静默漂移。
func TestP1R15B_JudgementSevenMessageUnchanged(t *testing.T) {
	noop := func(_ context.Context, in any) (any, error) { return in, nil }
	g := graph.New().
		AddNode(graph.NodeFunc("a", noop)).
		AddNode(graph.NodeFunc("b", noop)).
		AddEdge("a", "b").AddEdge("b", "a").
		SetEntry("a").SetOutput("a")

	err := g.Validate()
	if err == nil {
		t.Fatal("无条件环被 Validate() 放行：判据 7 回归")
	}
	for _, want := range []string{"graph: unconditional cycle", `"a"`, `"b"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("判据 7 的致命文案 %q 未含 %s：本片的渲染界吃掉了共享文案（切片 10 的诊断被抢走）", err, want)
		}
	}
}

// truncateForEvidence 只用于把失败输出缩进可读范围，不参与任何判据。
func truncateForEvidence(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…(截断)"
}
