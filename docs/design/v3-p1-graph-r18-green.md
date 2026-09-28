# P1 切片 18 · GREEN 实现记录（票面 B11 `graph.Typed` 的可归因类型不符）

工作项 `work-p1-graph-r18-typed`，契约 `docs/design/v3-test-scope-p1-graph-slice18.json`，
决定记录 `docs/design/v3-testability-decision-p1r18.json`（behavior-delta，红候选行 D2）。
RED 观察见 `docs/design/v3-red-observation-p1r18.md` §3–§4。

绑定场景命令逐字节为 `go test ./pkg/hno/graph -run TestP1R18_ -count=1 -race -v`（cwd 仓库根）：
RED `receipt:3f831afa-731c-4021-b4db-1f0e4efeb5dd`（exit 1）、GREEN
`receipt:8674cf3f-6981-40de-b2dd-bd5297c07b22`（exit 0，stdoutSha256=`25e12da6b2094c0327d6cd58b46a09a5…`）。
GREEN 途中先取过一次回执 `receipt:9d40005f-6173-4fc7-8643-0b92848cf1ac`（exit 0），其后有一处**仅注释**的更正（把 nil 渲染的说法改成实测的 `%!s(<nil>)`）；注释不参与编译输出，但为了让回执逐字节对应最终字节，本阶段以 `8674cf3f` 为准，`9d40005f` 只作为过程记录保留，不再作为阶段证据。

## 1. 最小实现：只改 `Typed` 的错误构造

新增导出面为零；`scheduler.go` 本片零改动（哈希逐字不变）；改动全部落在
`pkg/hno/graph/graph.go` 的 `Typed` 与其错误构造，外加一个未导出的 got 槽位助手。

有边界差异（`/tmp/p1-r18/baseline-graph.go` 由当前字节反向还原：只把 `Typed` 那段换回旧的
`errors.New` 形状并去掉 `reflect` 导入，`git hash-object --no-filters` 得 `dca23943fd9cade4766ceeffe43b47477b815a61`，
与契约 baselineBytes 逐字节相同 ⇒ 下面这份 diff 的左侧就是取红时那份字节，不是 HEAD 里
切片 2 之前的桩；`git diff --stat` 对 HEAD 会给出 355 行，那是整段 P1 的累积，不是本片）：

```diff
--- /tmp/p1-r18/baseline-graph.go
+++ pkg/hno/graph/graph.go
@@ -11,6 +11,7 @@
 	"context"
 	"errors"
 	"fmt"
+	"reflect"
 	"sort"
 	"strings"
 )
@@ -488,12 +489,26 @@
 
 // Typed 把带具体类型的函数适配为 Node，提供编译期类型安全出口。
 // 输入实际类型与 TIn 不符时返回错误而不是 panic。
+//
+// 错误必须可归因（票面 :70「可判定」按 §11.1 先例精确化）：图上每个 Typed 节点都在同一处
+// 断言失败，常量文案无法告诉调用方坏的是哪一个节点、期望什么类型、实际收到什么类型。
+// nil 单独渲染成 "no value"：reflect.TypeOf(nil) 返回 nil Type，用 %s 打印它交出的是
+// 格式化噪声 `%!s(<nil>)`，而「什么都没收到」正是调用方需要读出来的那件事。
 func Typed[TIn, TOut any](name string, fn func(ctx context.Context, in TIn) (TOut, error)) Node {
+	expected := reflect.TypeOf((*TIn)(nil)).Elem()
 	return NodeFunc(name, func(ctx context.Context, in any) (any, error) {
 		typed, ok := in.(TIn)
 		if !ok {
-			return nil, errors.New("graph: input type mismatch")
+			return nil, fmt.Errorf("graph: node %q: input type mismatch: expected %s, got %s", name, expected, actualTypeLabel(in))
 		}
 		return fn(ctx, typed)
 	})
 }
+
+// actualTypeLabel 描述调用方实际交来的值，用于类型不符错误的 got 槽位。
+func actualTypeLabel(in any) string {
+	if in == nil {
+		return "no value"
+	}
+	return reflect.TypeOf(in).String()
+}
```

三点实现取舍，都是判据要求而非个人偏好：

1. **节点名走 `%q`**：引号给出「完整名字边界」，这是 D4 能反向断言「不含 `"a"`」的前提；
   写成 `node a: …` 时单字母名会被任何含该字母的文案顶包。§11.1 的先例（join edge 报
   `"merge"`）就是同一形状。
2. **`expected` 在包装期算一次**：`reflect.TypeOf((*TIn)(nil)).Elem()` 对接口 `TIn` 同样给出
   类型名（`io.Reader`），不需要节点作者额外传类型描述器，也就不新增导出面（R18-Q1 未裁决）。
3. **nil 单独渲染 `no value`**：`reflect.TypeOf(nil)` 返回 nil `Type`，而本实现用 `%s` 打印它，
   实测交出 `got %!s(<nil>)`（`/tmp/p1-r18/nilcheck` 三条 `fmt.Sprintf` 逐字：`%s` → `%!s(<nil>)`、
   `%v` → `<nil>`、裸 `%s` 单独打印 → `%!s(<nil>)`）。那是格式化噪声而不是「什么都没收到」这句话。
   同一目录的另一条测量把四种渲染逐个喂给 D5 的 got 判据正则
   （`(?i)got[^a-z]*(no value|<nil>|\bnil\b)`）：`got no value` / `got <nil>` / `got nil` 都 match=true，
   唯独 `got %!s(<nil>)` match=false —— 标签与无值字样之间只容非小写字母，`%!s(` 里的那个 `s`
   把这段距离打断。所以 `no value` 这一支是被判据逼出来的，不是顺手加的糖。

没有加的东西：导出哨兵、`As` 目标、构建期跨节点类型连线校验（票面 §3 已划到 v3.1）、
`Validate()` 对 nil fn 的拒绝（R18-Q2）、任何 scheduler 侧改动。

## 2. 交出文本的前后对照（探针 10 场景，同一条 `go run .`，字节为当前实现）

| 场景 | 修复前（契约 T-2） | 修复后（实测） |
|---|---|---|
| 入口即不匹配 | `graph: input type mismatch` | `graph: node "needstring": input type mismatch: expected string, got int` |
| 下游不匹配（seed→a→b） | `graph: input type mismatch` | `graph: node "b": input type mismatch: expected int, got string` |
| nil 进具体 `TIn=string` | `graph: input type mismatch` | `graph: node "needstring": input type mismatch: expected string, got no value` |
| nil 进接口 `TIn=io.Reader` | `graph: input type mismatch` | `graph: node "needreader": input type mismatch: expected io.Reader, got no value` |
| 数组进切片 `[3]string → []string` | `graph: input type mismatch` | `graph: node "agg": input type mismatch: expected []string, got [3]string` |
| fn 自己的哨兵（D6） | `probe: upstream sentinel` | 逐字不变，`isSentinel=true`，文本不含 expected/got 标签 |
| nil fn 进 Typed（R18-Q2） | 由 panic 屏障交出 `graph: node "nilfn" panicked while running: …` | 逐字不变（本片不碰） |

最后两行是 GREEN 的「不得改坏」面：类型语义只发生在 `!ok` 分支，业务错误与 panic 屏障
两条路径的错误构造都未被触及。

## 3. 测量

| 项 | 命令 / 方法 | 结果 |
|---|---|---|
| 绑定场景 GREEN | 绑定命令 + receipt | `receipt:8674cf3f-6981-40de-b2dd-bd5297c07b22`，exit 0，7/7 PASS（含 9 个子用例） |
| 重复确定性 | `go test ./pkg/hno/graph -run TestP1R18_ -count=20 -race` | `ok … 1.458s`，0 FAIL，0 DATA RACE |
| 包级回归 | `go test ./pkg/hno/graph -race -count=1` | `ok … 2.422s`（最终字节复跑），0 DATA RACE，切片 1–17 全部行未减、未放宽 |
| 静态 | `gofmt -l pkg/hno/graph/` / `go vet ./pkg/hno/graph/` | 无输出 / 通过（最终字节复跑） |
| 零锁判据（票面 :71 B12） | `/tmp/p1-b9/struct.sh` | `sync_dot_graph=0 sync_dot_scheduler=0` |
| 导出面（契约 D8 哨兵条款） | `go doc -all ./pkg/hno/graph \| grep -cE '^(func\|type\|var\|const) '` | `24`（与基线相同，未新增） |
| 非测试 LOC（票面 §9 上限 1500） | `/tmp/p1-b9/struct.sh` | `loc_nonblank_nontest=663`（基线 649，本片 +14 ≤ 60） |
| 取消咨询点 / panic 屏障 | 同上 | `ctx_checks_in_scheduler=4`、`recover_scheduler=1`（scheduler 零改动，逐字与基线相同） |
| 测试函数总数 | 同上 | `testfunc_total=62`（基线 55，本片 +7 = D1–D7） |
| `Typed` 覆盖率（契约 D8） | `go tool cover -func=/tmp/p1-r18/cover-green.out` | `Typed 100.0%`（基线 0.0%）、`actualTypeLabel 100.0%`、`total 99.1%`（基线 96.6%） |
| 产品字节 | `git hash-object --no-filters` | graph.go `a8e46a3a2f0baaa699ae60f26498c6f5c1c851cb`（本片唯一改动文件）、scheduler.go `0ce6c3b3…` 逐字不变、测试文件 `10bce61b…`；反向还原的基线字节实测 `dca23943fd9cade4766ceeffe43b47477b815a61` = 契约 baselineBytes |

## 4. 判据逐行归属（GREEN 之后）

D1 `TypedCarriesConcreteInAndOutTypes`、D2 `TypeMismatchNamesTheOffendingNode`、
D3 `TypeMismatchDistinguishesExpectedFromActual`（两镜像形状）、
D4 `TypeMismatchNamesTheFailingNodeNotJustAnyNode`、D5 `NilInputIsAttributableTypeMismatch`
（两 nil 形状）、D6 `NodeErrorPassesThroughUnchanged`、D7 `TypeMismatchStaysDistinctFromGraphSentinels`
（三子用例）。D8 是命令背书行，见 §3 表格，不写成「测试读自身源码」的自指断言。

## 5. 未闭环项（留到 REFACTOR 与 review，不在 GREEN 里静默收口）

1. 契约 completionCriteria 第 3 条要求的**四条变异**尚未运行：(a) 节点名写死为入口节点 →
   应 D4 红、D2 绿；(b) 只报 expected 不报 got → 应 D3 红；(c) got 槽位打印
   `reflect.TypeOf(in)`（nil 时崩或空）→ 应 D5 红；(d) 把类型不符包成 `context.Canceled` →
   应 D7 红。GREEN 只主张「文本形状已满足判据」，不主张牙齿已被证伪。
2. D4 的反向断言（不含 `"a"`/`"seed"`）在 RED 时未真正执行到（基线连 `"b"` 都没有），
   它是否有效只能由上面 (a) 那条变异回答。
3. `golangci-lint` 仍未运行；`struct.sh` 仍在 `/tmp/p1-b9/`（S17-STD-5 债务，本片继续引用同一脚本）。
4. 回写债务（契约 `writeBackOwed`）：票面 :70 的「可判定」需按本片精确化为「可归因（点名节点 +
   expected/got 两槽位 + nil 渲染）」；母约 §3.4 那句「跨节点值以 `any` 流动」应补上
   「类型不符的归因由包装层给出，调度器不参与」。R18-Q1（是否新增导出哨兵）与 R18-Q2
   （nil fn 是否升级为构建期拒绝）仍是留给负责人的未裁决项。
