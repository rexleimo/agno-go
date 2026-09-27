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
