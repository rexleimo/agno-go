# 切片 31 第二段 REFACTOR：变异矩阵与牙齿归属

执行体：`scripts/mutation/p4s31-tasks-producer.mjs`（`a5488069e66a9ad53d4568711847d8f5508e79b5`）。
完整矩阵在**最终字节**上 exit 0、0 违规（`receipt:5fa2e070-4258-4382-a812-6f11745faee7`；
stdout 明文 `/tmp/s31/final/matrix-final.txt`；每条变异的原始 `-v` 输出落 `/tmp/s31-matrix/*.<stamp>.txt`）。

## 1. 注入不碰仓库工作树（与切片 30 执行体唯一的纪律差异）

切片 30 的执行体改 `pkg/hno/graph/scheduler.go` 再复原并逐条 `git hash-object` 复核；那条纪律
在当时成立，但本轮现场是**兄弟代理在同一工作树并发提交并移动文件**（HEAD 在谈判期间从
`b44b588` 推进到 `4d58ea5`，`pkg/hno/graph/` 里新增过文件）。因此本片执行体改成：

1. 每条变异前把源码整份复制到 `$S31_MUT_WORK`（排除 `.git/website/node_modules/bin/dist/.rex-harness`），
   并断言副本的两个注入目标与仓库同哈希（`verifyBaseline`）。
2. 用 `go -C <副本>` 跑契约绑定的**同一条**场景命令（参数逐字未改）。
3. 跑完整份重拷；收尾再断言「副本与仓库同字节 = true」。

实测收尾行：`finalCheck 副本与仓库同字节=true 工作树从未被注入=true`。
绑定命令在仓库最终字节上的收口证据另由 `receipt:d5b15e9f` 独立给出，矩阵不替代它。

## 2. 九条变异与逐条接手行（矩阵实际输出，非叙述）

13 个接手行 = D1,D2,D3,D4,D5/err,D5/lookalike,D6,D7,D8/swap,D8/dup,D8/none,D8/debug,D9。

| 变异 | 注入 | exit | 接手行（实测） | 主杀（契约声明） | 额外红（本片补登记） |
|---|---|---|---|---|---|
| `m-gate` | `enabled` 恒真，按族门控摘掉 | 1 | D3,D4,D5/err,D5/lookalike,D6,D7,D8/dup,D8/none | **D3,D7** | D4,D5×2,D6,D8/dup,D8/none |
| `m-degrade` | 未接线模式 `continue`（fail-closed 摘掉） | 1 | D2,D8/debug | **D2** | D8/debug |
| `m-sniff` | `strings.Contains(res.message,"error")` 分类 | 1 | D5/lookalike,D6 | **D5/lookalike** | D6 |
| `m-sniff-prefix` | `strings.HasPrefix(res.message,"tool execution error:")` | **0** | — | (等价变异，不充当牙齿) | — |
| `m-order` | `started` 挪到批次汇合之后，与结束事件同批 | 1 | D3,D4,D6,D8/dup,D8/swap | **D3,D6** | D4,D8/swap,D8/dup |
| `m-skip` | `started` 改挂模型回合（绕过 `ToolCallLimit` 截断） | 1 | D9 | **D9** | — |
| `m-payload-id` | `node` 填成 `call.ID` | 1 | D4,D6 | **D4,D6** | — |
| `m-payload-agent` | `node` 填成 `e.agentID` | 1 | D4,D6 | **D4,D6** | — |
| `m-dup` | 模式选择退化成不过滤列表，Messages 门按条目发射 | 1 | D8/dup | **D8/dup** | — |

契约 completionCriteria 2 点名的七条（m-gate / m-degrade / m-sniff / m-order / m-skip /
m-payload / m-dup）**逐条杀红**，其中 m-payload 拆成两段各证一次。

## 3. 等价变异：从「登记」升级为「实测」

`m-sniff-prefix` exit 0，13 条接手行全绿 ⇒ 契约 M2 登记的那条等价变异**由测量确认**，
不再是起草时的推断。同时测出了它成立的**根因**（这是登记时没有的信息）：

- 成功结果经 `toolkit.FormatResult` 是 JSON 串，文本首字符是引号，因此
  `HasPrefix(message, "tool execution error:")` 对 lookalike 工具返回的
  `tool execution error: handler exploded` 判假 —— 与 `toolResult.err` 判据同解。
- 反过来，`m-sniff`（Contains）之所以被杀，正是因为 D5 第二段专门造了那句长得很像的**成功**文本。
- 前缀式嗅探真正的破绽在 **not-found / 参数解析失败 / 参数校验失败** 三个失败位
  （`tool_executor.go:106-132`：文本分别是 `function %s not found…`、`failed to parse arguments: …`、
  校验错误原文），它们都不以该前缀开头，会被嗅探实现判成 `node_completed`。
  **本片的 D 行没有覆盖这三处**，所以这条变异在当前测试面上不可杀。
  处置：不在本片补 D 段（契约 completionCriteria 1 钉了「两段之间测试文件零改动」，
  且新增判据要回契约改 rows 表），改为把缺口交状态文档 §5 登记 `S31-GAP-1`，
  后续片给 D5 加第三个失败位子例即可把它从「等价」变成「杀得动」。

## 4. 判定脚本自己也要有牙齿（本片踩到的第二课）

矩阵第一版报出 **4 条违规**，四条都是**脚本自己错了**，没有一条是「测试没牙齿」：
`m-degrade` 记 `BUILD_FAILED`，`m-gate`/`m-sniff`/`m-order` 各记一次 `ATTRIBUTION_LEAK`
（baseline `3083f707…` 那轮，`/tmp/s31-matrix/*.[T113652]*` 三个 stamp 文件即现场）。

1. `m-degrade` 报 `BUILD_FAILED`：撤掉 `Errorf` 之后 `fmt` 变成未使用导入，变异根本编译不过。
   修法是让注入同时撤掉 `"fmt"` 那一行 import —— 这不是给变异放水，而是「摘掉 fail-closed」这个
   意图在 Go 里的最小可编译形态。
2. 三条 `ATTRIBUTION_LEAK`：我把若干行声明成了「应当绿」，实际它们按绝对序列同时判红。
   这不是测试的错，是我的归属表写漏了；而**如果**当时把 leak 当成噪声忽略，
   就会把「牙齿比声明更密」这件好事记成脚本失败。
3. 归属表覆盖不全：第一版每条变异只列了部分行，「没列出的行」既不算杀也不算绿，等于允许脚本
   对不上的时候沉默。

修法三条全部进代码，不再靠散文提醒：

- **覆盖完整性强制**：每条变异必须把 13 行全部署名为「主杀 / 额外红 / 应当绿」，
  缺一条 `--check` 直接抛错（`的接手行没覆盖全部判据`）。
- **额外红单列 `alsoKills` 并要求复现**：登记的额外红若没再现 ⇒ `ALSO_KILL_DRIFT`；
  未署名的判红 ⇒ `UNDECLARED_KILL`。两者都计违规、都 exit 1。
- **三条反空转分型保留**：`BUILD_FAILED` / `CRASH_OR_RACE`（含 `DATA RACE`）/
  `NO_ASSERTION_RED`（非零退出但无判红行）一律**不计牙齿**；等价变异若反而杀红 ⇒
  `EQUIVALENCE_REGISTRATION_DRIFT`。

`--check` 在当前字节上 exit 0：`preflight ok: 9 条变异，锚在仓库当前字节上唯一命中，接手行声明完整`
—— 锚唯一性是一条命令的事后校验，不是「我读过代码」。

## 5. 未列进矩阵的两条如实说明

- `m-order` 在单调用场景（D5 两段）上不可区分：一个 `started` 一个结束事件的序列，挪到汇合之后
  仍是同样两行。所以它的牙齿来自 D3/D6/D8 的多调用绝对序列，而**不是**来自 D5；这条归属写死在
  `expectGreen` 里，避免以后有人以为改 D5 就能守住次序。
- `m-skip` 在单批次场景（D1/D3/D6）上也不可区分：`started` 挂在回合级还是执行器级，同批次内
  仍是「先两个 started 后两个结束」。它的唯一牙齿是 D9 的 `ToolCallLimit` 截断静默 ——
  这也是本片为什么必须有 D9 这一行。
