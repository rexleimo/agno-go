# 切片 32 REFACTOR（P7 第 2 片，母约 G9：run/agent 级 invoke_agent span）

工作项 `work-p7-observability` / 契约 `docs/design/v3-test-scope-p7-g9-agent-span.json`（D1–D11）。

## 1. REFACTOR 结论：无独立重构动作，最终字节 = GREEN 字节

实现即为预演过的最小形状（kernel.go 一个私有 runKernel、两个调用点各一行、测试文件 563 行）；
REFACTOR 阶段零代码改动。最终字节：`kernel.go 1114030e5993cdc0011c67c6eaade56e2af659fb` /
`run.go d0d975b9aa671cbec46e30a0213ef79ce8aeb2cd` / `stream.go 51943b690dcd8d26cd9414c2265de001eda6f672`
（与契约 changedBeforeCommand 登记的候选对照值逐字节同）。最终门禁复测见 green 文档 §1。

## 2. 完整变异矩阵（`scripts/mutation/p7g9s2-agent-span.mjs`，仓库最终字节上复现）

预检 10 条锚唯一性全过；基线三文件哈希即上行最终字节；每条变异 `restored=true`、
`finalCheck identical=true`、B/C 零归属泄漏（C 面剔除 P7G9S2_ 后判红恒空）。整卷
exit 0（`receipt:ea0c8872-2699-4165-9207-a42678ff7af5`；首卷逐行转录
`/tmp/p7g9s2_mutation.txt`，两卷同字节同读数）。

| 变异 | 形态（D 行牙齿） | 命令 A 判定 | 判定 |
|---|---|---|---|
| m-parent-lost | spanCtx 不交给 r.Run（D2/D3/D4/D7 破） | 杀红 D2/D3/D4/D7 四函数（计子场景行 = 5 行）；D1/D5/D6/D8 全绿——互补牙齿的证明成立 | 杀红 |
| m-no-end | 开了从不 End（D1–D8 破） | 杀红 8 函数 = 11 行，唯一绿 D7/abandoned（`-v` 实读：`receipt:a162dbeb-dc8c-4085-896b-fc44d3ec9971` 里 D7/abandoned PASS、D7/cancel FAIL） | 杀红 |
| m-identity-id | agent.name 取 a.ID（D1 破） | 只杀 D1（1 红/11 绿）——单点牙，不可与别行合并 | 杀红 |
| m-runid-blank | run_id 留空（D1/D3 破） | 杀红 D1 + D3 | 杀红 |
| m-double-open | 一次驱动开两条 span | 杀红 8 函数 = 11 行（含 D8 的 SDK 侧正控 :561 读 2），唯一绿 D7/abandoned | 杀红 |
| m-stream-unwired | stream.go 退回直调 r.Run | 杀红 D3/D4/D5/D6/D7 = 7 行（D6 正控拿不到 1 条） | 杀红 |
| m-sync-unwired | run.go 退回直调 r.Run | 杀红 D1/D2/D5/D8 = 5 行（D8 正控随同步缝死亡） | 杀红 |
| m-tracer-guard | 以 IsRecording(ctx) 为发射 guard（被禁捷径） | 杀红 8 函数 = 11 行——恒假谓词导致的静默零 span 可被杀死（S32-STD-1 的实测面） | 杀红 |
| m-explicit-end | defer End 改单返回点显式 End | exit 0 全绿 12 行 | **等价变异（登记确认）** |
| m-status | 失败驱动在父 span 上 RecordError | exit 0 全绿 12 行 | **本片真实覆盖缺口（登记 S32-STATUS-1）** |

行覆盖核对（对契约 completionCriteria 第 3 条）：D1（m-identity-id/m-runid-blank/
m-sync-unwired）、D2（m-parent-lost/m-sync-unwired）、D3（m-runid-blank/
m-stream-unwired）、D4（m-parent-lost/m-stream-unwired）、D5（m-no-end/m-sync-unwired/
m-stream-unwired）、D6（m-double-open/m-stream-unwired/m-tracer-guard）、
D7（m-runid-blank/m-parent-lost/m-stream-unwired）、D8（m-sync-unwired/m-no-end/
m-double-open）——**每行至少被一个变异杀死，无未声明杀、无归属泄漏**。
计数口径披露：m-double-open 实杀 11 行含 D8（其 SDK 侧正控 :560 读 2≠1），与 M3 的
「杀 11 行」census 一致；契约判据文字「m-double-open 杀 D1–D7」是行清单摘要，
以 census 为准，此处如实登记两者差异。

## 3. 契约 completionCriteria 逐条对账

1. redProtocol 两段各留 receipt（`receipt:7aaacbce…` / `receipt:5363712d…`，逐行红因
   落 `v3-red-observation-p7g9s2.md`）；两段之间测试文件零改动（同哈希
   `358403de…`）。✅
2. 既有测试零改动：15 份既有测试文件与 8 份锚逐字节同哈希；改动只落在 changedBefore
   三份且差异只在 sourceSeam 声明范围内（green 文档 §2/§3）。✅
3. 变异矩阵 10 项在仓库最终字节上复现（§2；`receipt:ea0c8872…`）。m-parent-lost 杀
   D2/D3/D4/D7 且 D1/D5 绿；m-identity-id 单杀 D1；等价与缺口各一条实测确认。✅
4. 门禁全 exit 0：绑定命令两形式、`-race -count=30`、邻居 -race、build/vet 三门包、
   gofmt 无输出（green 文档 §1；build `./...` 按编排方兄弟片约束收窄为三门包，偏差已申报）。✅
5. 结构读数逐条实读到值（green 文档 §3 表：站点 1、sync 0×7、go func 0、r.Run 0/0、
   a.runKernel 1/1、stream go func 2、导出 21、t.Parallel 0）。✅
6. LOC：非测试 **1750 ≤ 1757**（+23 全在 kernel.go；run/stream 行中性）；未删任何
   双语注释——runKernel 的双语块正是「span 为何只能开在驱动点」的解释，显式申报的
   新上限再登记按契约第 6 条原文。✅
7. 写回：母约 §10 P7 行（:595）改为 run/agent=已接 + **P7 四级全清**；§3 G9 行（:87）
   改为第 1+2 片已交付、层级闭合、event 级与 status 挂账（S32-EVT-1/S32-STATUS-1）；
   S25-DEFER-1 在切片 25 契约侧的结清登记与 v3-p1-graph-status.md 条目由编排方收口
   （writeBackOwed 第 3/4 条，本片不代改）。✅
8. 零 tracer 代价数字登记进 green 文档 §3（+6 allocs、≈+441…510 B/op，最终字节实测），
   并明确它不是判据、不得为省它引入 guard。✅

## 4. 过程如实登记

- 实现由派发子代理按契约预演候选字节直接落盘（/tmp/g9s2 系列副本与仓库逐字节同源，
  三份 seam 文件与新测试文件逐一 hash-object 对账后才复制），两段制因此零漂移。
- m-no-end 的子场景级验证（D7/abandoned 唯一绿）以手工施加→`-v` 实读→逐字节还原的
  一次性运行取证（`receipt:a162dbeb…`），还原后三文件哈希复账全同。
- 零 tracer 代价在 HEAD 副本与最终字节副本各重测（非沿用预演数字）。

## 5. 审查

review 结论见 `docs/design/v3-p7-g9s2-review-verdict.json`（独立性披露：实现由派发
子代理执行，审查由主代理承担，同会话两层如实登记）。
