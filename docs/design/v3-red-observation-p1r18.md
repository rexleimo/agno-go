# P1 切片 18 · 设计阶段观察记录（B11 `graph.Typed`）

- 工作项：`work-p1-graph-r18-typed`
- activation：`c2bb91c2-f18b-4058-86a7-558972700eda`（capability `software.testing.design`，stage `design-tests`）
- 契约：`docs/design/v3-test-scope-p1-graph-slice18.json`（验收行 D1–D8）
- 本片绑定场景命令（后续每个 receipt 逐字节只用这一条，cwd `/Users/rex/codes/agno-go`）：

  ```
  go test ./pkg/hno/graph -run TestP1R18_ -count=1 -race -v
  ```

- 探针模块：`/tmp/p1-r18/probe`（`replace github.com/rexleimo/agno-go => /Users/rex/codes/agno-go`，
  只用导出 API，等价于外部测试包的观察面）。产物留在仓库外，未在工作树里留任何临时文件。

## 1. 现场观察（逐字，`receipt:c7b7fa29-164e-46c2-ab1d-1246b304304d` exit 0）

| # | 场景 | 观察到的 |
|---|---|---|
| 1 | `Typed[string,int]("len")` → `show`，`Run(ctx,"abcd")` | `err=<nil>`，`out=n=4 typed=int`，`completed=[len show]` ⇒ **TIn 接到、TOut 交出，两条都成立** |
| 2 | 入口即不匹配：`Typed("needstring", string→int)`，`Run(ctx,42)` | `err="graph: input type mismatch"`，`result=<nil>`，无 panic |
| 3 | 下游不匹配：`seed(any→string) → a(string→string) → b(int→int)` | `err="graph: input type mismatch"` —— **与场景 2 逐字相同，点名不出是 b** |
| 4 | fn 自交哨兵 | `err=probe: upstream sentinel`，`errors.Is=true`，`resultNil=true` ⇒ 逐字交出现已成立 |
| 5 | `Run(ctx, nil)` 进具体 `TIn` | `err="graph: input type mismatch"`，无 panic |
| 6 | `Run(ctx, nil)` 进接口 `TIn`（`io.Reader`） | 同样是 `input type mismatch` ⇒ **Go 的断言语义**：nil 接口值对所有断言都失败，「接口形状下 nil 合法」这一一直觉在本实现里不成立（D5 因此钉当前形状，不钉直觉） |
| 7 | `TOut` 为零值 `int(0)` | 流到下游 `got=0 kind=int`，`Value("zero")` 命中 ⇒ 零值输出没被当成缺失 |
| 8 | `Typed[int,int]("nilfn", nil)` | `err=graph: node "nilfn" panicked while running: runtime error: invalid memory address or nil pointer dereference` ⇒ 崩溃由切片 14 的 panic 屏障兜住，但结论形状是「另一条 goroutine 崩了」而不是「构造期就非法」 |
| 9 | `Typed[[]string,map[string]int]` 正确/错误（`[3]string`） | 正确：`out=map[n:3]`；错误：又是同一句 `input type mismatch`（数组与切片的区别被抹平） |
| 10 | 哨兵隔离 | `errors.Is(err, ErrStepLimitExceeded)=false`、`errors.Is(err, context.Canceled)=false` |

**四条不同的成因（2/3/5/9b）交出逐字相同、不带任何身份信息的错误串**，且 `Result` 为 `nil`，
所以调用方手上没有任何其它公共面可以判断是哪个节点、期望什么、收到了什么。
票面 §4:70 的「可判定错误」这一半因此**未成立**；「而非 panic」那一半已由切片 14 的屏障成立。

## 2. 覆盖与结构事实

| 判据 | 命令 | 结果 | receipt |
|---|---|---|---|
| `Typed` 的测试覆盖 | `go test ./pkg/hno/graph -count=1 -coverprofile=… && go tool cover -func=… \| grep Typed` | `graph.go:491: Typed 0.0%`，`total: 96.6%` | `receipt:27e54800-6ea7-4636-b9f3-4d0271f772c3` |
| 全仓是否有人测过它 | `grep -rn "Typed" pkg/hno/graph/` | 只命中 `graph.go:489,491` 两处（定义与注释），任何 `*_test.go` 零命中 | 同上命令族 |
| 零锁 | `sh /tmp/p1-b9/struct.sh` | `sync_dot_graph=0 sync_dot_scheduler=0 recover_scheduler=1 ctx_checks_in_scheduler=4 loc_nonblank_nontest=649 testfunc_total=55` | 切片 17 已登记，本片复用于 D8 基线 |
| 导出面 | `go doc -all ./pkg/hno/graph \| grep -cE '^(func\|type\|var\|const) '` | `exportedSymbols=24`（含 `Typed` 本身） | 切片 17 review 首次量化 |
| 基线字节 | `git hash-object --no-filters` | `graph.go dca23943…`（与切片 17 的基线**相同**，即切片 17 对 graph.go 零改动）、`scheduler.go 0ce6c3b3…`、七份测试文件逐条见契约 `baselineBytes` | — |

## 3. 设计判断（为什么本片的目标是「可归因」而不是别的）

1. **「编译期类型安全」不可作为本片判据。** `Typed[TIn,TOut]` 的编译期保证由泛型签名直接给出，
   运行期测试无法观察它，也不该假装能观察（见契约 `observabilityLimit`）。
2. **跨节点连线是 `any` 流，所以类型正确性在运行期才见分晓。** 这正是吸收 adk「构建期校验」时
   Go 泛型给不了的那一块；产品能补的最小一块是：**出错时必须把结论归因到节点与两个类型**。
   不补，调用方在 20 节点的图里拿到的是一个字符串，只能靠复现定位。
3. **构建期类型连线校验不是本片目标**：票面 §3 已把 `InputSchema`/`OutputSchema` 校验划到 v3.1，
   本片不得借 B11 顺带实现它。
4. **「可判定」的判据形式沿用票面自己的先例**：§11.1 已确立「错误文本包含 offending 节点名
   （可定位，不要求新导出哨兵）」。因此本片**不新增导出哨兵**（那会改动票面 §1 的公共面，属测试范围变更，
   须由 rex-harness 另行授权）。是否要给机器一个 `errors.As` 目标，登记为 `R18-Q1` 挂人类裁决。
5. **nil fn（场景 8）留在本片之外**：它今天由 panic 屏障兜住，不崩进程；要不要升级成构建期拒绝，
   登记为 `R18-Q2`。本片不为此改 `Validate()`。

## 3. 取红后的现场观察（绑定命令，`receipt:3f831afa-731c-4021-b4db-1f0e4efeb5dd` exit 1）

场景命令逐字节为 `go test ./pkg/hno/graph -run TestP1R18_ -count=1 -race -v`（cwd 仓库根），
D1–D7 的逐行归属全部取自这条命令的 `-v` 输出。取红时产品字节仍是契约 baselineBytes
（graph.go `dca23943…`、scheduler.go `0ce6c3b3…`，实测一致），本片新增字节只有
p1r18_typed_test.go `10bce61bd70f2b78a968d585cc81919e8fe68293`。

| 契约行 | Test | 结果 | 判红/判绿的关键输出（逐字） |
|---|---|---|---|
| D1 | `TestP1R18_TypedCarriesConcreteInAndOutTypes` | PASS（green-at-red） | `--- PASS` —— TOut 以具体 int 流到下游、`Value("len")==4`、`Completed()==[len show]` |
| D2 | `TestP1R18_TypeMismatchNamesTheOffendingNode` | **FAIL**（:127） | `Run 的错误文本 "graph: input type mismatch" 未包含以引号包裹的 offending 节点名 "needstring"` |
| D3 | `TestP1R18_TypeMismatchDistinguishesExpectedFromActual` | **FAIL**（:176，两个子用例各一次） | `未同时给出期望类型与实际类型，要求形状：(?i)expected[^,]*string.*got[^,]*int`（镜像形状为 `(?i)expected[^,]*int.*got[^,]*string`） |
| D4 | `TestP1R18_TypeMismatchNamesTheFailingNodeNotJustAnyNode` | **FAIL**（:211） | `未点名真正失败的节点 "b"`（同时未报 `"a"`/`"seed"` 的错位断言，本轮因名缺失而未触发） |
| D5 | `TestP1R18_NilInputIsAttributableTypeMismatch` | **FAIL**（:263 与 :266，两个 nil 形状各两行） | `未包含以引号包裹的 offending 节点名 "needstring"/"needreader"` + `got 槽位未渲染成表示「无值」的字样` |
| D6 | `TestP1R18_NodeErrorPassesThroughUnchanged` | PASS（green-at-red） | `--- PASS` —— fn 的哨兵仍逐字交出且文本不含类型标签 |
| D7 | `TestP1R18_TypeMismatchStaysDistinctFromGraphSentinels` | PASS（green-at-red，3 子用例） | `--- PASS`（plain / tight-budget / uncancelled） |

顶部汇总：`FAIL github.com/rexleimo/agno-go/pkg/hno/graph 1.251s`，4 个 Test 判红、3 个判绿，
无 DATA RACE、无 panic 泄漏（进程活着交出错误这一点本身就是票面 :70 后半句的现状证据，
本片不把它伪装成 RED）。

## 4. 失败原因（与目标行为一致，不是基础设施失败）

四行判红的原因同一个：**类型不符的错误不可归因**。`pkg/hno/graph/graph.go:491` 的 `Typed`
在 `in.(TIn)` 失败时于 `:495` 执行 `return nil, errors.New("graph: input type mismatch")`——
无节点名、无 expected/got 两个类型槽位、无包装、也不是导出哨兵，因此入口即不匹配（D2）、
下游不匹配（D4）、nil 输入（D5，两种 TIn 形状）、以及「期望类型为何/实收类型为何」（D3）
四种成因交出逐字相同的串。测试程序正常编译、夹具正常构建（`Validate()` 未被拒）、同包切片
1–17 的全部 Test 在同一条命令之外仍绿（`go test ./pkg/hno/graph -run "TestP1G_|TestP1R1[3-7]" -count=1 -race`
→ `ok … 2.636s`），所以判红只发生在 D2–D5 声明的公共错误语义上。

确定性：`-count=5 -race` 复跑得 40 条 `--- FAIL` 行（= 5 × (4 个 Test + 4 个子用例)，逐轮同集），
无竞态行、无需按命中率折算取样轮数（与切片 17 的 D2 相反，本片不需要 race 窗口夹具）。

判据形式说明（防「用实现反推测试」）：类型标签的两条正则只扫描**剥离引号包裹片段之后**的
错误文本。原因是夹具的节点名自带类型词（`"needstring"` 含 string、`"needint"` 含 int），
在整段文本上匹配会让「点名节点」与「报出类型」互相顶包；剥离后 D3 只可能由真正的类型描述
满足，而点名由 D2/D4/D5 的引号断言独立负责，两面各管一段、互不替补。这不是放宽期望：
基线文本在剥离前后逐字相同（无引号片段），判红行 :176 的输出里两段一并抄出可核对。

未闭环项（如实登记，不推进 GREEN 之外的结论）：

1. `golangci-lint` 仍未运行（本机 `make lint` 依赖的工具链未确认）；`gofmt -l` 无输出、`go vet` 通过。
2. D4 的「归因错位」反向断言（不含 `"a"`、不含 `"seed"`）本轮尚未真正执行到——基线连 `"b"` 都没有。
   它是否真能挡住「把节点名写死成入口节点」这一形态，留到 REFACTOR 的变异矩阵里证伪一次
   （契约 completionCriteria 第 3 条 (a)）。
3. 结构判据执行体 `struct.sh` 仍在 `/tmp/p1-b9/` 下（S17-STD-5 的既有债务），本片 D8 沿用同一脚本，
   机器一清即不可重放。
