# 切片 32 RED 观察（P7 第 2 片，母约 G9：run/agent 级 invoke_agent span）

工作项 `work-p7-observability` / 契约 `docs/design/v3-test-scope-p7-g9-agent-span.json`（D1–D11）。
契约 redProtocol 两段制，段间测试文件零改动（`p7g9s2_agent_span_test.go` 两段同为
`358403deb276f7bba618e7822b7552ae5176f3f6`）。本片绑定场景命令逐字：

```bash
go test ./pkg/hno/agent -run TestP7G9S2_ -count=1 -race -v
```

命名说明：契约 redProtocol 登记的观察记录名为 `v3-red-observation-p7g9-agent-span.md`；
本文件即该登记的实现片落笔（沿用切片 25/31 的 `v3-red-observation-*.md` 家族命名，
后缀取切片号 32，与 GREEN/REFACTOR 三件套一致）。

## 1. 第一段（缝的形状，故意不含 span）——逐行红形

第一段字节：`kernel.go df36bdd0eaddbcc49734c49cdf7021011fb4a2c6`（100 行）——runKernel
已存在、签名与返回四元组与最终一致、`run.go`/`stream.go` 两个调用点已各改为经它，
但函数体只有 `return r.Run(ctx, messages)`，无 observability import。

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| 第一段可编译（redProtocol 的前提证明） | `go vet ./pkg/hno/agent` | 0 | `receipt:3bea1e65-9394-4481-9559-f821887cf0e1` |
| 绑定命令（-race -v） | 8 个测试函数全判红 | 1 | `receipt:7aaacbce-0df2-4e1a-b11a-1014acca6c94` |
| 绑定命令裸退码形式 | 8 函数同名 FAIL | 1 | `receipt:581ff388-669b-4267-ab07-6f7f2ebd0ddb` |
| `-v` 读数 `--- FAIL` 计数 | 8（行数读数，非 receipt 记录） | — | 过程转录 `/tmp/p7g9s2_red_v.txt` |
| 整包 census（不带 -run 过滤）非 P7G9S2 判红 | 0 条（第一段对既有测试零附带伤害；非 receipt 记录） | 1（红即本片 8 条） | 过程转录 `/tmp/p7g9s2_red_census.txt` |

逐行红因（`-v` 实读，行号取自测试文件）：

| 行 | 红因类别 | 实读 |
|---|---|---|
| D1 | **行为红** | :256 `invoke_agent spans = 0, want exactly 1 (exported names [chat])`——主干缺席而子 span 在场，恰是切片 25 D11 的 span-present 形态 |
| D2 | **行为红** | :284 同形（exported names [chat execute_tool chat]） |
| D3 | **行为红** | :326 同形 |
| D4 | **行为红** | :368 同形 |
| D5 | **行为红 ×2** | :398（sync）/ :419（stream）两子场景同形；父存在性检查属 Fatal 层前提，包含关系比较在该段从未执行 |
| D6 | **前提红** | :444 `positive control: invoke_agent spans = 0, want 1 before the fail-closed call`——正控先失败，本行在该段无法行使牙齿，如实归类不混称行为红 |
| D7/abandoned | **vacuous 绿** | 该段发不出任何 span，本场景断言 0 条已结束 span，不可证伪（`-v` 里唯一的 `--- PASS` 子场景） |
| D7/cancel | **行为红** | :495 同 D1 形 |
| D8 | **前提红** | :561 `invoke_agent spans under a real SDK tracer = 0, want 1`——SDK 侧正控先失败 |

## 2. 第二段（语义）——全绿

第二段 = 最终字节：span 在 kernel 驱动点之前开、spanCtx 交给 r.Run、End 由驱动拥有
（`kernel.go 1114030e5993cdc0011c67c6eaade56e2af659fb`，117 行）。两段之间仅
`kernel.go` 变动，测试文件零改动。

| 项 | 结果 | receipt |
|---|---|---|
| 绑定命令（-race -v） | 8 个 `--- PASS`、0 个 `--- FAIL`，exit 0 | `receipt:5363712d-86d3-4446-864c-a8f7e412dde7` |
| 绑定命令裸退码形式 | exit 0 | `receipt:ee5fe579-bfe0-4537-8669-1d6da5a3e497` |

D7/abandoned 在第二段转为「驱动仍阻塞、span 仍未结束」的正确语义绿（非 vacuous）；
其牙齿由变异矩阵 m-parent-lost / m-stream-unwired 提供（见 refactor 文档 §2）。

## 3. 门与锚的第一段状态

- `gofmt -l pkg/hno/agent` 两段均无输出。
- 8 份锚 + 8 份 out-of-seam 文件在两段全程与基线逐字节同哈希（收口对账见 green 文档 §3）。
- changedBefore 三份在第一段即已变动（kernel 100 行版 + run/stream 各一行调用点），
  第二段只再动 kernel.go 一份。
