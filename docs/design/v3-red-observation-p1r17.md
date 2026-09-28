# P1 切片 17 · RED 观察记录（B9 取消归一化）

- 工作项：`work-p1-graph-r17-cancel-normalization`
- activation：`eb82a84b-3f78-4e9b-846d-de9efd8296b0`（capability `software.testing.tdd`，stage `red`）
- 契约：`docs/design/v3-test-scope-p1-graph-slice17.json`（验收行 D1–D7）
- 类型化决定：`docs/design/v3-testability-decision-p1r17.json`（`behavior-delta`，redCandidateRow = D1）
- 绑定场景命令（本片每个 receipt 逐字节只用这一条，cwd `/Users/rex/codes/agno-go`）：

  ```
  go test ./pkg/hno/graph -run TestP1R17_ -count=1 -race -v
  ```

- 本阶段 RED receipt：**`receipt:8851e9e8-b2d2-4a21-955d-9ee09e96600a`，exitCode = 1**
  （stdout sha256 `bca87f6e4cb712320e44e0e1649f56e6df4963e5a8d42e880a09adab2bbf3481`）
- receipt 当时的字节：`pkg/hno/graph/p1r17_cancel_normalization_test.go` = `788b3d7c40cbe7b02aaa9a7068efe95d03b4a2c4`，
  `scheduler.go` = `f2d7fb4b10e9c133f9bbe8b566da40ea8e96d55e`（未改动），`graph.go` = `dca23943fd9cade4766ceeffe43b47477b815a61`（未改动）

## 1. 观察到的失败（逐行归属，取自同一条命令的一次抄录运行）

抄录运行 exit = 1，输出 `/tmp/p1-b9/r17-red-final.txt`；它与 receipt 那次不是同一次执行，因此时间行
（`0.02s` / `1.24s`）与 D2 命中轮次不同（receipt 那次 stdout 的 sha256 已列在上面），失败**行**与**内容**一致。

| 契约行 | Test 函数 | 结果 | 关键输出（逐字） |
|---|---|---|---|
| D1 | `TestP1R17_CancellationWinsOverStepLimitDuringDispatch` | **FAIL** | `:213` 「取消落在派发窗口内时 Run 的错误 = graph: step limit 100 reached before the graph converged: graph: step limit exceeded, want errors.Is(err, context.Canceled)（取消时刻已启动 2/2000 个后继，安全阀是在那之后才撞上的）」；`:217` 「取消被报成步数超限：… —— 预算是取消之后才被耗尽的，调用方要的是取消语义」 |
| D2 | `TestP1R17_CancellationIsNotDisguisedAsNodeBusinessErrorAcrossRounds` | **FAIL** | `:252` 「第 14 轮：取消被伪装成节点业务错误 p1r17: store connection reset by peer（取消时刻已启动 1/2000 个后继，业务错误只可能在 ctx.Done 之后产生）」，同一次执行另命中第 15、42、44 轮等，共 15/100 轮 |
| D3 | `TestP1R17_AlreadyCancelledContextIsReportedAsCancellation` | PASS（3 子用例） | green-at-red |
| D4 | `TestP1R17_BlockedNodeExitsAllNormalizeToCancellation` | PASS（3 种节点出口） | green-at-red |
| D5 | `TestP1R17_CancellationWhileQueueHasBacklog` | PASS（2 段） | green-at-red |
| D6 | `TestP1R17_UncancelledBusinessErrorStillSurfacesVerbatim` | PASS（2 段） | green-at-red，D1/D2 的反向牙齿 |
| D7 | —（命令背书的结构性护栏行，见 §4） | — | 契约里就写明它不承担新增行为 |

抄录运行汇总：`--- FAIL` 2 行、`--- PASS` 6 个 Test 级条目（含 D3–D6）、包内 `TestP1R17_` 族总耗时 1.672s、
`DATA RACE` 0 次。

## 2. 失败原因与缺失的目标行为是否一致

票面 `docs/design/v3-test-scope-p1-graph.md:68` 的 B9 要求：取消时 `Run` 必须交出 `context.Canceled`
语义，「不得伪装成节点业务错误」。D1/D2 两条判红恰好是这句话的两个反面：

- **D1（确定性）**：调用方 cancel 之后，消费者正忙在 `scheduler.go:119-133` 的 `dispatch` 循环里一次性
  放出那 2000 条激活，循环头只判步数预算、从不咨询 ctx，于是第 100 步撞上安全阀并把
  `ErrStepLimitExceeded` 交给调用方。观察到的错误正是契约预测的那一个。
- **D2（竞态）**：`scheduler.go:101-103` 的 `return nil, item.err` 拿到队列项后不看 ctx，
  两件事（项已在手上、ctx 已取消）同时成立时 Go 的 select 随机挑，挑中队列项就把「我取消了」
  说成节点业务失败。夹具里的节点只在 `<-ctx.Done()` 之后才交出 `errP1R17Business`，
  所以「它出现在 Run 的错误里」本身就等价于取消被伪装。

前提自证（契约 `vacuousPassGuard`）在同一处代码里完成，三项缺一即判「窗口没建立」而不是判错误语义：
`started < 2000`（图未收敛）、`started < 100`（预算尚未撞上）、以及非阻塞探 Run 的结果通道**仍无值**
（阀若撞上 Run 必然已结束）。实测 `-count=20` 下 D1 出现 20/20 次上述判红、0/20 次前提报错，
说明窗口每次都成立。

因此两条红都不是编译、夹具或环境失败：同一条命令下 `Validate()` 通过、D3–D6 八段护栏全部成立、
包内既有 49 个 Test 函数全部保持（§3），失败只落在 D1/D2 声明的公共错误语义上。

## 3. 逐轮复现与既有测试未受影响

| 测量 | 命令 | 结果 |
|---|---|---|
| D1 确定性红 | `go test ./pkg/hno/graph -run TestP1R17_CancellationWinsOverStepLimitDuringDispatch -count=20 -race` | 20/20 交出 `ErrStepLimitExceeded`；「窗口没建立/前提不成立」类报错 0/20 |
| D2 红色可复现性（30 轮时） | `go test ./pkg/hno/graph -run TestP1R17_CancellationIsNotDisguisedAsNodeBusinessErrorAcrossRounds -count=1 -race` × 20 | 整行判红 **17/20** 次执行；命中 35/600 轮（≈5.8%/轮）→ 样本量不足 |
| D2 红色可复现性（100 轮时） | 同上，轮数改为 100 | 整行判红 **20/20** 次执行；命中 166/2000 轮（≈8.3%/轮） |
| 包级不回归 | `go test -count=1 -race -v ./pkg/hno/graph/...` | 55 个 Test 函数：53 PASS / 2 FAIL，且 2 个 FAIL 全部是 `TestP1R17_` 的 D1、D2；既有 49 个 Test 无一失败；`DATA RACE` 0 次 |

D2 轮数由契约原设的 30 上调到 100 的理由与实测数字已写入测试文件注释与本记录：判据形式（跨轮 0 容忍）
未动，只增加样本量把红色从 17/20 提到 20/20；契约 `forbiddenShortcuts` 第 1 条禁止的是「减轮数直到碰巧全绿」，
与此方向相反。此项属 RED 阶段内部的夹具强度修正，不改写测试范围（`raceAssertionForm` 的形式与理由不变）。

## 4. 结构旁证（RED 时刻，非 receipt）

`sh /tmp/p1-b9/struct.sh`：

```
sync_dot_graph=0 sync_dot_scheduler=0 recover_scheduler=1 ctx_checks_in_scheduler=2 loc_nonblank_nontest=634 testfunc_total=55
```

- `testfunc_total` 49 → 55，与契约完成判据第 1 条（本片新增 6 个 Test 函数）一致；
- `ctx_checks_in_scheduler = 2` 是 M-2 登记的事实：两个咨询点都在 `consume` 的 select 里，
  也就是 D1/D2 两条红的共同根因。GREEN 之后该数必须增加，增加量记在 GREEN 证据里（契约完成判据第 4 条）；
- 零锁（`sync.` 计数 0）与单一 `recover(` 在 RED 时刻即为 0/0/1，D7 行由这条命令与 diff 审查共同背书。

## 5. 与契约的差异（如实登记，不静默改写）

`docs/design/v3-testability-decision-p1r17.json` 的 `contractDeltaFromMeasurement` 已记四项，均为**夹具形态**
修正而非判据修正：D5 原形态（复用 `p1r17RaceGraph` + cap=1 + failer）实测在取消之前必然死锁，改为
`p1r17CappedBacklogGraph`；D4 的 20ms sleep 定序改为「节点确实进入阻塞」的事件握手（票面 `:75`）；
D1/D2 原写法的 `Run` 在 cancel 之后才启动，窗口根本不存在，改为 `p1r17RunAsync` + 三项前提自证；
D2 轮数 30 → 100 并附实测命中率。契约的 M-4 漏检率数字（17.5%~22.5%）来自早期更宽松的探针夹具，
本夹具的实测为 5.8%~8.3%/轮，该项连同 `raceAssertionForm.reproducibilityDisclosure` 的回写仍在
`writeBackOwed` 里。
