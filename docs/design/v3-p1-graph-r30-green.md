# 切片 30 第一段 GREEN：R17 取消归一化夹具的负载加固

工作项 `work-p1-graph-r17-load-hardening` / 契约 `docs/design/v3-test-scope-p1-graph-slice30.json`（D1–D8）。
执行方式：编排方在主会话内直接实现（`activationId` 仍为 `pending`，与切片 24/26/28 同口径），
所有回执现取现贴（`node /tmp/rex-receipt.mjs -- <命令>`，命令逐字即契约 `verificationCommands` 的那几条）。
第二段的变异矩阵见 `docs/design/v3-p1-graph-r30-refactor.md`；两段之间测试文件零改动（§4 哈希自证）。

本片没有「实现前的 RED」：加固只动测试文件，**红来自回归注入**（契约 redProtocol 的分工说明）。
因此第一段的交付判据是「加固形在负载协议 + 串行门禁下全绿」，加上「同一负载协议下原始夹具仍能红」。

## 1. 串行门禁（全部 exit 0）

| 项 | 命令（逐字） | exit | receipt |
|---|---|---|---|
| 契约绑定场景命令 | `go test ./pkg/hno/graph -run 'TestP1R17_' -count=1 -race -v` | 0 | `receipt:ab8987aa-67df-4c2e-bf60-42334a420747` |
| 防碰巧绿 | `go test ./pkg/hno/graph -run 'TestP1R17_' -count=30 -race` | 0 | `receipt:92813c00-c02f-4f5e-8aa4-1e67efe4e4af` |
| 整包协议 第 1 轮 | `go test ./pkg/hno/graph -count=5 -race` | 0 | `receipt:5efe7f82-0b8e-4500-bfe9-039af1016944` |
| 整包协议 第 2 轮 | 同上 | 0 | `receipt:a2dac9f4-a3f6-47f9-93ff-85b37e7edf7e` |
| 整包协议 第 3 轮 | 同上 | 0 | `receipt:b3fa2e87-8bef-47f6-b42e-6294bf3afc65` |
| 邻居回归（禁全仓 `make test`） | `go build ./...` / `go vet ./pkg/hno/graph/...` / `go test ./pkg/hno/agent/... ./pkg/hno/runner/... -count=1` | 0 | `receipt:f6c0372e-5a5f-43aa-b679-c3470b00c70f` / `receipt:9fafcf2f-eeff-40db-97af-1d0e713de1b8` / `receipt:c05bc67c-7764-431f-8954-fd296ced4a78` |
| 格式 | `gofmt -l pkg/hno/graph` | 0（**stdout 为空**：`e3b0c442…b855` 即 `sha256("")`） | `receipt:ea6a33be-6baa-413c-90c2-1601d8f373ed` |

`-count=30` 那一条的 `-v` 复跑（同一命令、非计数门禁）读数：**180 条 `--- PASS` = 6 个 Test × 30，0 FAIL**，
末行 `ok github.com/rexleimo/agno-go/pkg/hno/graph 1.991s`；原始输出 `/tmp/r30/count30.log`。
**没有 DATA RACE 报告**（`-race` 下 3 轮整包 + 16 轮负载协议 + 10 条矩阵，全部无 `WARNING: DATA RACE`）。

## 2. 负载判据（D5）：加固形 16 轮 0 红，对照锚见 §3

协议已入库（`S17-STD-5`/`S18-STD-6`：判定脚本留在 /tmp 时别人无法复现结论）：

```bash
bash scripts/mutation/p1r17-load-rounds.sh 14 10 'TestP1R17_'   # receipt:f5337f72-cf86-495f-b0d1-ed922e64b50d exit=0
bash scripts/mutation/p1r17-load-rounds.sh 14 6  'TestP1R17_'   # receipt:94ad9f73-aa42-435b-9a70-cf3ad2f28fb9 exit=0
```

脚本逐条实现契约 D5 的口径：14 个 `yes > /dev/null` 压载（本机 8 核）+ 每轮**仍是契约那条
`go test ./pkg/hno/graph -run <族> -count=1 -race -v`**（不换预编译二进制，否则「轮次里执行了什么」
和契约对不上）+ 每轮把 `sysctl -n vm.loadavg`（轮前与轮后）与本轮原始文件路径写进同一份轮次账
+ `trap` 收尾杀压载 + 任一红即 exit 1。

| 轮 | exit | pass 行数 | loadavg 轮前 → 轮后 | 原始文件 |
|---|---|---|---|---|
| 1 | 0 | 17 | 3.17 → 3.17 | `/tmp/r17-load/round.20260928-185030.1.txt` |
| 2 | 0 | 17 | 3.17 → 7.80 | `…/round.20260928-185030.2.txt` |
| 3 | 0 | 17 | 7.80 → 12.70 | `…/round.20260928-185030.3.txt` |
| 4–5 | 0 | 17 | 12.70（平台期） | `…/round.20260928-185030.{4,5}.txt` |
| 6 | 0 | 17 | 12.70 → 14.97 | `…/round.20260928-185030.6.txt` |
| 7–8 | 0 | 17 | 14.97 | `…/round.20260928-185030.{7,8}.txt` |
| 9 | 0 | 17 | 14.97 → 21.38 | `…/round.20260928-185030.9.txt` |
| 10 | 0 | 17 | 21.38 | `…/round.20260928-185030.10.txt` |
| 追加 1–6 | 0 | 17 | 15.28 → 17.42 平台期 | 账本 `/tmp/r17-load/rounds.20260928-185117.txt` |

合计 **16/16 轮 0 红**（阈值 ≥10 由契约写死，只增不减）。每轮 17 条 pass 行 = 6 个 Test + 11 个子例。
**0 条前提红**：加固形的停靠点守卫（①–⑤）在 16 轮里没有一次开口——这正是「加固完成」的定义，
而不是「跑绿了」的定义。

## 3. 对照锚：同一份二进制里的 A/B（completionCriteria #2 要求的那一次）

原始夹具与加固行同名，仓库内不留双胞胎行，所以 A/B 在 /tmp 的仓库副本里做
（`/tmp/r30ab/copy`，HEAD 版夹具改名 `TestP1R17O_*` / `p1r17o*` 与加固行同包共存）：

```bash
bash /tmp/r30ab/ab.sh 6     # receipt:79cb40a1-ebbf-4853-91be-78636fc0d19e exit=0
                            # 账本 /tmp/r30ab/ab-rounds.20260928-190014.txt
```

同一段递增负载（loadavg 3.07 → 50.09）里交替跑 A/B 各 6 轮，只跑那三行负载敏感行：

| pair | A 原始夹具 | B 加固形 |
|---|---|---|
| 1 | exit 0 | exit 0 |
| 2 | exit 0 | exit 0 |
| 3 | **exit 1** — `TestP1R17O_CancellationWinsOverStepLimitDuringDispatch` | exit 0 |
| 4–6 | exit 0 | exit 0 |

A 侧那一红的原文（`/tmp/r30ab/ab.A.3.txt`，`p1r17o_ab_rows_test.go:211`）：

> 取消之前 Run 已经交出（err=graph: step limit 100 reached before the graph converged: graph: step limit
> exceeded, result=`<nil>`）：派发窗口没建立，本行前提不成立

**如实分型（契约 D8）**：这一红是 **前提红（守卫）**，不是判据红 —— 它证明的正是 S24-STD-2 的病根：
负载下**前提本身**建不起来（安全阀抢在测试取消之前撞上，`ran` 那个瞬时值读到 1/2000 时结论已经交出了）。
加固形在同一负载曲线、同一二进制里 0 红，且它的结论不再依赖任何瞬时读数。
本轮 A 侧 1/6 的红率低于 M1 的 11/30：M1 的定向三行协议每轮更长、且那 11 轮里含判据红
（账本 `/tmp/r17exp/{freshA,loadA,loadB,hardab}.*.txt`，见状态文档 §5 的 M1 登记）；
本片只把「同二进制 A/B 至少红一次、且加固行零红」这一条做成可复跑的对照，不冒充 M1 的样本量。

## 4. 判据不变量（D1–D4 的「逐字未动」是机器核对，不是叙述）

| 不变量 | 核对方式 | 结果 |
|---|---|---|
| 26 处 `t.Error*` 判据行零改动 | `git show HEAD:…\| grep t.Errorf` vs 现文件，逐行 `diff` | **26 行逐字全同**（`/tmp/r30/err_before.txt` vs `/tmp/r30/err_after.txt`，diff 空） |
| 生产字节零改动（D7） | `git hash-object --no-filters` 7 份非测试 `.go` | 与 `baselineBytes.anchors` 逐位全同（`receipt:dab7c96f-1d08-4474-9bfc-083052badd52`；`scheduler.go == dfd6c1ba1c2a9c35bf4ccb613ee986dd80a7c736`） |
| 导出面零新增 | `go doc -all ./pkg/hno/graph` 的 `^(func\|type\|var\|const)` 计数 | **64**（`receipt:dbfbef5c-14ee-4cff-8a3b-6970b10af65c`） |
| 其余 13 份测试文件逐字节不动 | `git diff --name-only HEAD -- 'pkg/hno/graph/*_test.go'` | 只列出 `p1r17_cancel_normalization_test.go` 一份（`allowedModifiedTestFiles: 1`） |
| `fixtureBefore` 四份哈希 | `git show HEAD:… \| git hash-object --stdin` | `cbc8b692…`/`624f1962…`/`bdeb8ef9…`/`a8a936ed…` 全部命中（即契约登记的就是 HEAD 态） |
| 四个常量取值 | `grep` | `p1r17Grace=10s`、`p1r17Succ=2000`、`p1r17StepBudget=100`、`p1r17RaceRounds=100` 未变 |
| 工作树只含允许文件 | `git status --porcelain` | 4 条：本测试文件 + `scripts/` 两份新脚本 + `yarn.lock`（宿主 churn，刻意不入库） |

测试文件最终字节：`88d3cf15e5a555eedb827d1221331dbac46816fd`（加固前 `cbc8b692a84ed5ce23b461a8ed40b9ba3870d386`），
796 行。

## 5. 加固形改了什么（只动「前提如何建立」）

三行负载敏感行换成**停靠点构造**：`WithCheckpointer(sink) + WithDurability(DurabilitySync)`，
让 sink 在指定那次提交上停住消费者（`durability.go:103` 的 Sync 调用点在 `complete` 之后、
下一轮 `dispatch` 之前，`scheduler.go:170-173`）。于是「取消时预算尚未耗尽」「失败项已在手上」
「派发因槽位上限停住」都成了**放行次序的算术后果**，不再是对瞬时值的抽样观测。
原 5 处派发窗口守卫一一映射为停靠点守卫（仍是前提红分型，另加一条「停靠到的不是那一次提交」）。
`p1r17RaceGraph` / `p1r17AwaitDispatchWindow` / 死代码 `p1r17BusinessAfterCancel` 随之删除。
`Append` 只当会合点用，不读引擎内部态、不断言持久化形状（`forbiddenShortcuts` 第 5 条）。

## 6. 入库前复跑（commit gate，2026-09-28）

上面 §3 那行「工作树只含允许文件 4 条」写于台账/裁决件落地之前。入库时点把范围与读数一并复采：

| 项 | 命令（cwd 仓库根，逐字） | 读数 |
| --- | --- | --- |
| R17 全行 | `go test ./pkg/hno/graph -run TestP1R17_ -count=1 -race` | `exit 0`，`receipt:3778f814-e43b-4d4a-8c73-97f11346c219` |
| 受影响四面 | `go test ./pkg/hno/graph ./pkg/hno/run ./pkg/hno/agent ./pkg/hno/runner -count=1 -race` | `exit 0`，`receipt:18d30fbb-cc3a-4f8c-8450-5447878aaf21` |
| 编译 / 静态 | `go build ./...`、`go vet ./pkg/hno/...`、`gofmt -l pkg/hno/graph scripts` | 三者均无输出 |
| 证据指针对账 | `node scripts/evidence-audit.mjs` | `NOWHERE 16`，全部在切片 30 之外的历史裁决件；本片新增件 0 条 |
| 回执存在性 | 7 份本片相关文档里 62 个唯一 uuid 逐个 `test -f .rex-harness/receipts/<id>.json` | MISSING 0 |

入库文件集（10 条，`yarn.lock` 刻意留在工作树不入库）：本测试文件、`scripts/mutation/p1r17-cancel-window.mjs`、
`scripts/mutation/p1r17-load-rounds.sh`、切片 30 契约、`v3-p1-graph-r30-green.md`、
`v3-p1-graph-r30-refactor.md`、`v3-p1-graph-r30-review-verdict.json`，以及 `v3-test-scope-p1-graph.md`、
`v3-p1-graph-status.md`、`v3-platform.md` 三份写回。
