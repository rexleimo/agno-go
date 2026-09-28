# 切片 30 第二段 REFACTOR：变异矩阵与牙齿归属

契约 `docs/design/v3-test-scope-p1-graph-slice30.json` 的 D6（反空转绿）与 redProtocol 第二段。
执行体已入库：`scripts/mutation/p1r17-cancel-window.mjs`（346 行，node）。
第一段与第二段之间测试文件零改动（第一段收口哈希 `88d3cf15…` 在本段结束时仍是它，见 §4）。

```bash
node scripts/mutation/p1r17-cancel-window.mjs --check   # 预检：10 条锚唯一命中 + 接手行声明完整
node scripts/mutation/p1r17-cancel-window.mjs           # 完整矩阵
```

矩阵回执：`receipt:dcc7911c-a203-46d4-bbfa-680183a26ff6`（exit=0，违规 0）。
每条变异的原始 `-v` 输出落盘 `/tmp/r17-matrix/<id>.<stamp>.txt`，路径逐条打印在矩阵行里（契约 D8）。

## 1. 十条变异与逐条接手行（矩阵实际输出）

| id | 注入（`scheduler.go` 临时态） | 契约归属 | 实测接手行 | 判定 |
|---|---|---|---|---|
| m1 | τ 删掉派发侧取消判据 ③（`:243-245`） | `HardD1 ← τ` | `D1hard` | ok |
| m2 | ο 把步数预算判据抢到取消判据之前 | `HardD1 ← ο`（升级登记） | `D1hard` | ok |
| m3 | λ/L_snap 让 ② 读 `consume` 入口快照（恒 nil） | `HardD2 ← L_snap` | `D2hard` | ok |
| m4 | κ 两条守卫全删（②+③） | `HardD1`/`HardD2` 复合 | `D1hard,D2hard` | ok |
| m5 | 删掉 select 的取消分支 ①（`:149-150`） | `HardD5b ← 删 ①` | `D5a,D5b` | ok |
| m6 | μ 一切结论先交 `ctx.Err()`（`:168` 换成 `s.ctx.Err()`） | 反向牙齿 → D6 | `D6biz`（加固行全绿） | ok |
| m7 | ξ 未取消时把安全阀报成取消（`:247` 换成 `context.Canceled`） | 反向牙齿 → D6 | `D6valve`（加固行全绿） | ok |
| m8 | 取消报成功（① 交出 `s.result, nil`） | `HardD2`/`HardD5b` | `D2hard,D5b`（+ `D4` 三出口、`D5a` 额外红） | ok |
| m9 | 整条删除 ②（`:155-157`）——票面 §13.3 第 1 条原来的「不可达形态」 | `HardD2 ← 删 ②` | `D2hard` | ok |
| m11 | σ 把取消判据提到派发循环之外（取消后继续放出积压） | **等价变异**，不充当牙齿 | 无（exit=0 全绿） | ok |

≥8 条 ✓（10 条）；每行 ≥1 条 solo 杀手 ✓：`HardD1 ← {m1, m2, m4}`、`HardD2 ← {m3, m9, m4, m8}`、
`HardD5b ← {m5, m8}`；反向牙齿 `D6 ← {m6, m7}` 由原始 D6 子例接手、**加固行判绿**（契约 D4 的
「加固行不得接手它们，接手即说明判据被改坏」是脚本里的 `expectGreen`，不是报告里的叙述）。

## 1.1 solo 杀手的 5 次独立执行（契约 completionCriteria #3 的计数口径）

单次矩阵只证明「能杀」；契约要的是 5/5。逐条独立复跑（`bash /tmp/r30/teeth5.sh 5`，
**`receipt:f0824b09-cff2-4597-adbe-f3f170734de0`** exit=0，原始日志 `/tmp/r30/teeth5.<id>.<i>.log`）：

| 变异 | 接手行 | 独立执行 | 结果 |
|---|---|---|---|
| m1 τ | `HardD1` | 5 | **5/5 杀红** |
| m2 ο | `HardD1` | 5 | **5/5 杀红** |
| m3 λ/L_snap | `HardD2` | 5 | **5/5 杀红** |
| m5 删 ① | `HardD5b` | 5 | **5/5 杀红** |
| m8 取消报成功 | `HardD2`+`HardD5b` | 5 | **5/5 杀红** |
| m9 删 ② | `HardD2` | 5 | **5/5 杀红** |

计数取自脚本自身的判定：`expectKill` 未命中 ⇒ `NO_TEETH(空转)` ⇒ 退出码非零；
所以「退出 0 的次数」就是「杀掉声明行的次数」，不是另一个人对着日志数的。
30 次注入跑完后 `scheduler.go` 仍是 `dfd6c1ba1c2a9c35bf4ccb613ee986dd80a7c736`（表末行实测）。

## 2. 三条反空转自查是脚本行为，不是收口叙述

- **守卫红不计牙齿**：脚本对每条变异扫 `GUARD_WORDS`（「本行前提不成立」「先修夹具」「停靠点没建立」
  「槽位没打满」「积压没建立」「没有后继节点开始运行」…），命中即判 `PREMISE_RED` 并计违规。
  本次 10 条全部 0 命中 ⇒ 上表的接手行都是判据行。
- **solo 杀手必须杀到它声明的那一行**：`expectKill` 未命中 ⇒ `NO_TEETH(空转)` + 退出码 1。
  m9 这次能杀 `D2hard`，就是契约要求的「`R17-GAP-1` 闭合」证据形态。
- **等价变异不许杀红**：m11 的 `expectGreen` 覆盖全部登记行；若杀红则判
  `EQUIVALENCE_REGISTRATION_DRIFT`（那会推翻票面 `R17-GAP-2` 的等价登记）。本次 exit=0，登记成立。

## 3. 本段自纠：矩阵第一版是错的，错在「每条变异都 exit 1」没人追问

第一次全矩阵跑出 10 条 exit=1、m11 也「全绿 ok」——**当时脚本判定为 ok 是假绿**：
`run()` 里把 `BOUND.args`（已含 `"test"`）又前置了一次 `"test"`，于是命令实际是
`go test test ./pkg/hno/graph …`，多出的包模式让 go 打了 `FAIL test [setup failed]` 并非零退出，
而真正的测试仍照常跑完并打印 PASS。两个后果都进了脚本：

1. `buildFailed` 的识别式只认 `[build failed]`，不认 `[setup failed]` ⇒ 这类环境红被当成普通非零退出放过；
2. m11「exit=1 却没有任何判红行」被 `expectKill=[]` 的逻辑判成 ok ⇒ 等价变异的「全绿」其实从未被验证。

修法（已入库）：`run(BOUND.args)` 去掉重复前置；`buildFailed` 认 `[setup failed]`；
新增 `NO_ASSERTION_RED` —— 场景命令非零退出且无判红行 ⇒ 崩溃/构建/环境红，**不计牙齿**并计违规；
每条变异把原始输出落盘并在行尾打印 `raw=`。修后 m11 才是真的 `exit=0`，其余 9 条的接手行不变。
这条登记进状态文档，属 `S24-STD-2` 同族的「分型不混」教训：**判定脚本自己也要有牙齿**。

## 4. 字节复核

- 矩阵前后 `scheduler.go` 均为 `dfd6c1ba1c2a9c35bf4ccb613ee986dd80a7c736`（脚本对每条变异都做一次
  `git hash-object` 自检，漂移即 exit 2；末行 `finalCheck scheduler.go identical=true`）。
- 7 份非测试 `.go` 与 `baselineBytes.anchors` 逐位全同（`receipt:dab7c96f-1d08-4474-9bfc-083052badd52`）。
- 测试文件仍是第一段收口时的 `88d3cf15e5a555eedb827d1221331dbac46816fd`（两段之间零改动）。
- 50 份原始输出文件（16 轮负载 + 12 轮 A/B + 10 条矩阵 + `-count=30 -v`）里 **0 条 `DATA RACE`**。

## 5. 矩阵写回票面/状态文档的三条（契约 writeBackOwed 的实测部分）

1. `R17-GAP-1` 由本片闭合：「整条删除 ②」从不可达形态升级为 `HardD2` 的 solo 杀手（m9 实测杀红，
   m3 从快照侧独立杀红），落点见 §1。
2. 票面 §13.1 末条「次序不是判据」需要限定：在**阀未到期**的形状下次序不可观察（原实测结论保留），
   在**阀与取消同处一个循环头**的形状下 ο（m2）可观察并成为 `HardD1` 的杀手。
   `scheduler.go:237-240` 的注释同批核对——本片不改字节，注释更新登记为未来某片。
3. `R17-GAP-2`（σ 等价）继续挂账：m11 在加固形下仍全绿，说明「取消之后不得再放出积压」这一条
   在本片可构造的位置上**没有**可观察差异；`R17-GAP-3`（超时形状）本片未触及。
