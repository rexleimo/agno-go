# P1 图引擎 — 测试差异审查（切片 1：B1 线性 DAG）

工作项：`work-p1-graph-engine` ｜ 契约：`docs/design/v3-test-scope-p1-graph.md`

## 1. 测试差异范围

本轮只**新增**一个测试文件，未修改、未删除、未跳过任何既有测试：

| 文件 | 状态 | blob |
|---|---|---|
| `pkg/hno/graph/graph_test.go` | 新增（`package graph_test`，外部测试包） | `f747ecd78aa29e3c9d473140b8a1074dc5b076d3` |
| `pkg/hno/graph/graph.go` | 新增（API + RED 夹具骨架 → GREEN 填充 builder） | `41848a97952c21368617997eb4f136f1d714502c` |
| `pkg/hno/graph/scheduler.go` | 新增（单消费者调度器） | `4627fc90598f7fd213a12dd4c061dbe30bcae33b1aff3cd72d07c785aa5ca9bb` |

## 2. 该测试是否仍约束契约行为（而非实现细节）

`TestP1G_LinearDagRunsChain` 的四条断言逐条对应契约 §4 B1，且全部走 §1 公共面：

| 断言 | 契约条款 | 是否可被实现细节替代 |
|---|---|---|
| `res.Output() == "final<-seed:x"` | B1 输出投影 + 前驱输出→后继输入 | 否，只经 `SetOutput`/`Run` |
| `res.Value("entry") == "seed:x"` | B1 入口收到 `Run` 的 `in` | 否 |
| `res.Completed() == ["entry","out"]`（字典序） | B1 完成集合 | 否，不依赖时序 |
| 节点自记录轨迹 `["entry","out"]` | B1 因果顺序（线性链允许全序） | 否 |

**未弱化项**：无 `t.Skip`、无放宽容差、无把期望改成实际输出、无断言 mock 被调用代替结果断言。测试不使用 `time.Sleep`、不引用任何未导出符号、不窥探 `queue`/`pending`/`running`。

## 3. 检错力（变异证据）

| 变异 | 位置 | 结果 | 回执 |
|---|---|---|---|
| MUTANT-1 路由丢弃前驱输出（后继输入改为 `nil`） | `scheduler.go` `successors` | exit 1：节点收到 nil，测试失败 | `receipt:22bbb684-40fe-4e28-9ae7-859637acd091` |

变异后按 sha256 逐字节还原：还原后 `4627fc90598f7fd213a12dd4c061dbe30bcae33b1aff3cd72d07c785aa5ca9bb` 与变异前基线一致。

## 4. REFACTOR 内容

`complete()` 内的出边扫描提取为 `successors(name, out)`——§3.4 路由核心的唯一落点，使条件边 / Join / Default 在后续切片里只需扩展一个函数，而不是再次改执行路径。行为不变，测试保持通过：`receipt:917a6fe2-8de1-4d20-acb4-b339e6b0ed54`（exit 0）。

## 5. 结构与规模判据（当前实测）

```
grep -c 'sync\.' pkg/hno/graph/graph.go pkg/hno/graph/scheduler.go  → 0 / 0   （B12，判据 2）
非测试 LOC：graph.go 164 + scheduler.go 104 = 268                                （B13，判据 6，上限 1500）
go test -race ./pkg/hno/graph/... -count=1 → ok
go vet ./pkg/hno/graph/... → 净；gofmt -l pkg/hno/graph → 无输出
邻近回归 go test ./pkg/hno/agent/... ./pkg/hno/runner/... ./internal/session/contract/... -count=1 → exit 0（receipt:9d440084-24d4-406e-bb7e-9d9099cbd626）
```

## 6. 契约覆盖缺口（切片 1 之后仍待做）

票面 6 条判据中，切片 1 只覆盖判据 1 的 DAG 部分与判据 6。仍为占位实现：`Validate()` 返回 `errNotImplemented`（判据 3 / B5、B6、B7）、`AddConditional`/`AddDefault`/`AddJoin` 为空体（B2、B3）、`WithMaxConcurrency`/`WithStepLimit` 未参与派发（B8、B4）、无 panic 恢复与取消归一化（B10、B9）、`Warnings()` 恒 nil（B7）。这些必须按契约 §8 的切片顺序各自先取得 RED。

## 7. 切片 2（构建期拒绝 + fail-closed 执行）

| 项 | 内容 |
|---|---|
| 测试差异 | 仅向 `graph_test.go` 追加 `TestP1G_ValidateRejectsDanglingEdge`（两个子用例：悬空边拒绝 + 合法图反向对照）。未修改切片 1 的断言，未删除、未跳过任何用例 |
| RED | `receipt:cdd74aec-d8fe-4456-958f-64db50973d9a` exit 1 —— 失败形态即被指控缺陷：Validate 返回夹具占位错误（不含 `ghost`），断言继续走到 Run，进程 SIGSEGV 于 `scheduler.go:83` |
| 基础设施失败排除 | 中途一次 `undefined: fmt` 的 build failed 回执（`receipt:c6cc3342-1d32-42ba-87ff-c19753e1edb4`）**未**被用作 RED 证据，按 `rex-tdd` RED 第 3 条判为不合法，仅在此登记为过程噪声 |
| GREEN | `receipt:898838fc-c1b2-40cc-9d37-b1234a8f115e` exit 0；`-race` 绿；邻近回归含 graph 包四包同跑 `receipt:676fa64f-9242-4c8e-8978-e15cc22c8867` exit 0 |
| 检错力 | MUTANT-2 删除 `Run` 的 `Validate` 前置守卫 → `receipt:0f7b1e8c-9291-4702-b605-50c3f498acb5` exit 1（重新出现 SIGSEGV）；还原后 sha256 `aafb3ae611d8cb11cd2bb33eb43e87e6b1bdcddfae7851b2c62801c2bdb3649e` 与变异前逐字节一致 |
| REFACTOR | 落实审查项 STD-4 与 Mysterious Name：`start`→`spawn`（并写明其占槽职责）、`dispatch`/`successors` 中的 `next` 分别改 `act`/`acts`。行为不变 → `receipt:0c2d86d4-780c-4494-b9e0-ebd3db341966` exit 0 |
| 结构判据 | `grep -c 'sync\.' pkg/hno/graph/*.go` → graph.go 0 / scheduler.go 0 / graph_test.go 0；非测试 LOC 293（上限 1500） |

**仍开放的阻断项（切片 2 未触及，按契约 §11.3 次序推进）**：SPEC-2（`AddConditional`/`AddDefault`/`AddJoin` 静默丢弃声明）、SPEC-3（`WithStepLimit` 默认 1000 未落实）、§3.5 其余 6 项校验与 B5 无条件环检测。

## 8. 切片 3（R1：未路由边声明的 fail-closed 拒绝）

闭包 C2-SPEC-1（审查轴 cycle-2 阻断项：已校验通过的图仍会静默跑出空 Result）。

| 项 | 内容 |
|---|---|
| 测试差异 | 仅向 `graph_test.go` 追加 `TestP1G_UnsupportedEdgeDeclarationsAreRejected`（表内一行 `join-only` + 反向对照子用例）。切片 1/2 的断言零修改，无删除、无跳过 |
| RED | `receipt:31abe8b8-843d-4b12-84c9-5699e8808a95` exit 1，失败点在 `graph_test.go:86`「声明了引擎未路由的边，Validate 却判定合法」；类型化结论见 `docs/design/v3-testability-decision-p1-graph-slice3.json` |
| GREEN | `receipt:749f5817-2a35-4656-83aa-9a5c6c58d2d0` exit 0（`go test ./pkg/hno/graph/... -run P1G -count=1`，测试夹具加固之后重取）；`-race` `receipt:3b22ce56-d6d4-49f1-be6e-44c4340de723` exit 0；`go build ./...` `receipt:fa30e346-500a-466f-8e49-8cb33eacb487` exit 0；邻近回归 `receipt:7d2d93e4-2d4b-440b-827b-a543fc0a12d8` exit 0 |
| 断言加固（同一切片内的自纠） | 探针发现原 `wantHit: "j"` 形同虚设：错误文案里的单词 `join` 自带字母 `j`，任何不指名节点的报错也能通过。已把节点名改为 `alpha`/`beta`/`merge`、`wantHit` 改为 `"merge"`。加固后重取 GREEN 回执（上表） |
| 无效证据登记 | 加固前的第一次变异回执 `receipt:5cfbd502-eaf0-4eff-a877-9653c07b629f` exit 1 实为 `declared and not used: d` 的 **build failed**，按 `rex-tdd` RED/GREEN 第 3 条判为不合法检错证据，已作废、不作为任何阶段的推进依据 |
| MUTANT-1 静默丢弃回归 | `declareUnrouted` 的 append 改为 `_ = unroutedDecl{...}` → `receipt:8fba84da-5d36-4421-a6aa-e47c715ff313` exit 1，`graph_test.go:88` 报「Validate 却判定合法」；即 C2-SPEC-1 缺陷本身可被杀死 |
| MUTANT-2 报错不可定位 | `%s from %q to %q` → `%s`（保留 `d.kind` 可编译）→ `receipt:9d114403-f91e-4847-99d9-27b46ae78ab5` exit 1，`graph_test.go:91` 报「Validate 错误未定位到 "merge": graph: join edge is declared but not routable by this engine」。该输出同时证明：若仍用旧的 `"j"`，此变异会存活 |
| MUTANT-3 一律拒绝 | `if len(g.unrouted) >= 0` + 无条件报错 → `receipt:1f55bbac-52b5-40f1-b1c0-a1ad09e9e276` exit 1，命中契约 §12.3 反例条款：`graph_test.go:110`「合法图被拒」+ 切片 1 DAG 用例 + 切片 2 合法对照子用例 |
| MUTANT-4 Run 不复查校验 | `scheduler.go:37` 的 `err != nil` → `err != nil && false` → `receipt:1851f1ab-207d-4f80-8066-58819c6bec63` exit 1，先由 `graph_test.go:96`「非法图仍可执行并返回 Output=\<nil\> Completed=[alpha]」这条行为断言判红，随后切片 2 用例复现 `scheduler.go:86` 的 nil 指针 SIGSEGV —— 即 SPEC-1 的崩溃守卫同样由 Run 前置校验承担 |
| 还原校验 | 每次变异后 `cp` 基线还原，`shasum -a 256` 与变异前逐字节一致（graph.go `2586bd0c…`、scheduler.go `f7f827b9…`） |
| REFACTOR | 去重：`unroutedDecl` 与 `declareUnrouted` 的注释重复陈述同一条迁移规则，只保留类型上的 fail-closed 说明；`AddJoin` 循环不再逐次重绑定 `g`；补回切片 2 测试函数被误删的观察面注释行；删除夹具中无用的 `gamma` 节点（它既不被断言，也会在 B6「不可达节点」校验落地后成为未来假失败源） |
| 重复代码的取舍 | 反向对照（a→b 合法图）在切片 2 与切片 3 的测试里各写一遍，是刻意选择：两条契约必须各自独立证明「拒绝一切」会被抓到，共享夹具会把两个切片的红绿信号耦合在一起 |
| 结构判据 | `gofmt -l pkg/hno/graph` 无输出；`go vet ./pkg/hno/graph/...` 净；`grep -c 'sync\.'` → graph.go 0 / scheduler.go 0；非测试 LOC 323（上限 1500）；`-race` `receipt:bf10618e-7b46-474e-a05b-3e99e82b8d07` exit 0 |

**本切片刻意不做**：条件边/Default/Join 的真实路由（R2、B2）、§3.5 其余校验项（B5/B6/B7）、`stepLimit` 默认 1000（B4）、并发上限（B8）、取消归一化（B9）、panic 转错误（B10）、`Typed` 分支测试（B11）。`AddConditional`/`AddDefault` 与 `AddJoin` 一样进入拒绝集合，属 fail-closed 的临时状态，R2 落地路由后从该集合收缩 —— 这也让 R2 的 RED 仍可复现（当前实现会把条件边判为非法）。

