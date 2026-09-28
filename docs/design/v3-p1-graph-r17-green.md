# P1 切片 17 · GREEN 实现差异（B9 取消归一化）

- 工作项：`work-p1-graph-r17-cancel-normalization`；activation `eb82a84b-3f78-4e9b-846d-de9efd8296b0`，stage `green`
- 契约：`docs/design/v3-test-scope-p1-graph-slice17.json`；RED 证据：`docs/design/v3-red-observation-p1r17.md`
  （`receipt:8851e9e8-b2d2-4a21-955d-9ee09e96600a`，exitCode 1）
- GREEN receipt：**`receipt:88e2073c-dc8d-43dc-a056-6585cb6658b8`，exitCode = 0**
  （绑定命令逐字节不变：`go test ./pkg/hno/graph -run TestP1R17_ -count=1 -race -v`，cwd `/Users/rex/codes/agno-go`；
  stdout sha256 `81ebf6969f92cf96457049aae57afdc296893b589b211823aceba29fbd5f3ef8`）
- 字节：`scheduler.go` `f2d7fb4b10e9c133f9bbe8b566da40ea8e96d55e` → `ee072e86331bfd986913c85f185d36b7b0554eeb`；
  测试文件保持 RED 时的 `788b3d7c40cbe7b02aaa9a7068efe95d03b4a2c4`（GREEN 阶段一字未动）；
  `graph.go` 保持 `dca23943fd9cade4766ceeffe43b47477b815a61`（本片零改动）

## 1. 有边界差异（唯一被改动的生产文件）

```diff
--- a/pkg/hno/graph/scheduler.go   (切片 16 收尾字节 f2d7fb4b)
+++ b/pkg/hno/graph/scheduler.go   (GREEN 之后 ee072e86)
@@ -98,6 +98,12 @@
 			return nil, ctx.Err()
 		case item := <-s.queue:
 			s.running--
+			// 先问取消，再决定手上这一项算不算结论：随机挑选让「调用方已经 cancel」与
+			// 「节点交出错误」这两件事可能同时成立，此时票面 :68 要的仍是取消语义。
+			// 顺序反过来就等于把调用方施加的取消说成节点业务失败。
+			if err := s.ctx.Err(); err != nil {
+				return nil, err
+			}
 			if item.err != nil {
 				return nil, item.err
 			}
@@ -116,8 +122,15 @@
 // 这里不需要任何锁——running/pending/steps 只由唯一消费者读写，而「停在这里」本身就是
 // 消费者让出队列的方式。判据写在循环头而不是循环之外：每个完成事件后都会重新进入这里，
 // 只在首轮判上限的实现会在第一个 completion 之后把整批一次放出。
+//
+// 取消判据写在预算判据之前，且同样写在循环头：消费者忙在这一个循环里时不会回到 select，
+// 于是这里是调用方施加的取消唯一可能被看见的地方。放在预算之后会交出一个调用方没做过的事
+// ——「图跑飞了」，而真实原因是它自己已经取消（票面 :68 的「不得伪装」）。
 func (s *scheduler) dispatch() error {
 	for len(s.pending) > 0 {
+		if err := s.ctx.Err(); err != nil {
+			return err
+		}
 		if s.steps >= s.stepLimit {
 			return fmt.Errorf("graph: step limit %d reached before the graph converged: %w", s.stepLimit, ErrStepLimitExceeded)
 		}
```

行为代码共 6 行（两个 `if err := s.ctx.Err(); err != nil` 卫语句），其余是各自「为什么写在这里、
为什么必须是这个顺序」的注释。没有新增导出标识符，没有新增文件，没有引入锁或原子量。

## 2. 为什么恰好是这两处（不是第三、第四处）

契约 M-2 量出：调度器全体的取消咨询点只有 2 个，且都落在 `consume` 的 select 里。两条红正是
那两个写点的直接后果，因此修复只把「交出结论之前必问一次取消」补到那两个写点上：

- `dispatch` 的循环头 → D1：消费者忙在这一批派发里时不会回到 select，这里是取消唯一可能被看见的
  地方；顺序必须在步数判据之前，否则交出的就是 `ErrStepLimitExceeded`（RED 实测 20/20）。
- `consume` 取到 queue 项、`s.running--` 之后 → D2：select 的随机挑选使「项已在手上」与「ctx 已取消」
  同时成立，先看 `item.err` 就把调用方的取消说成节点业务失败。

`running == 0` 的成功交出**刻意没有加守卫**：契约把「取消之后图仍合法收敛并交出 Result」登记为
0/300 未复现、因此未合同化的窗口（`notTouchedByThisSlice` 第 4 项）。为一条没有 RED、也没有验收行
要求的路径加代码，等于把实现范围建立在猜上；本片把不变量如实表述为
**「取消必须在结论交出之前被咨询到」**，而不是「任何与取消重叠的 Run 都必须失败」——
后者属 R17-UNADJ-3（真实节点错误与取消同时成立时的优先次序），仍挂人类裁决。

## 3. GREEN 实测（全部命令与退出状态）

| 判据 | 命令 | 结果 |
|---|---|---|
| 绑定场景命令逐字节通过 | `go test ./pkg/hno/graph -run TestP1R17_ -count=1 -race -v` | **exit 0**，`receipt:88e2073c-dc8d-43dc-a056-6585cb6658b8`；-v 输出：D1 PASS(0.02s)、D2 PASS(0.81s)、D3 3 子用例、D4 3 子用例、D5 2 子用例、D6 2 子用例全 PASS，包内 `ok … 3.159s` |
| 逐轮确定性 | `go test ./pkg/hno/graph -run TestP1R17_ -count=20 -race` | exit 0，`ok … 18.188s`（20/20 通过；D2 的 100 轮 0 容忍在 20×100 = 2000 轮里无一命中，RED 时同一夹具是 166/2000 命中） |
| 包级不回归 | `go test -count=1 -race -v ./pkg/hno/graph/...` | exit 0；55 个 Test 函数 **55 PASS / 0 FAIL**，`DATA RACE` 0 次（既有 49 个全保持，含 `TestP1R13_CancellationIsNotReportedAsStepLimit` 与 B4/步数族、p1r16 的 B8 族） |
| 编译与静态检查 | `go build ./...` ; `go vet ./pkg/hno/graph/...` ; `gofmt -l pkg/hno/graph/` | 全部干净（`build+vet OK`，gofmt 无输出） |
| 结构判据 | `sh /tmp/p1-b9/struct.sh` | `sync_dot_graph=0 sync_dot_scheduler=0 recover_scheduler=1 ctx_checks_in_scheduler=4 loc_nonblank_nontest=647 testfunc_total=55` |

结构判据逐条对账契约完成判据第 4 条：零锁不变（`sync.` 计数 0/0）、单一 `recover(` 不变、
`ctx_checks_in_scheduler` 由 **2 增加到 4**（正是 §1 的两个卫语句，每个各含一次 `ctx.Err()`；
grep 口径把 `ctx.Err()`/`ctx.Done()` 都计入，select 分支自身贡献 2），
LOC 634 → 647（票面 §9 上限 1500），Test 函数 49 → 55（契约第 1 条）。

## 4. 尚未闭环的项（不写成已完成）

- 变异矩阵（κ λ μ ν ξ ο ρ σ τ υ 十条）尚未执行：契约把它列在完成判据第 3 条，属 GREEN 之后的
  验证轮，将由本轮后续阶段逐条实跑并登记指定杀手行与哈希自检。
- `golangci-lint`：环境里没有可执行文件，按契约第 5 条如实登记为**未运行**，不记作通过。
- 契约 `writeBackOwed` 的两处回写（M-4 漏检率数字改为本夹具实测 5.8%~8.3%/轮、母约 §3.3 的
  「取消咨询点」表述）尚未落到文档里，与 B9-TICKET-Wording 一并在收尾阶段处理。
- `pkg/hno/graph` 之外的消费面：`grep -rl "hno/graph" --include="*.go"` 在包外没有任何命中，
  也就是本片的改动目前还不被 `pkg/hno/agent`、`pkg/hno/runner` 消费。因此不对外推结论，
  等 graph 接进 runner 的切片时再实测取消语义的传导。
