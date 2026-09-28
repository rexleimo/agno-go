# 切片 28 REFACTOR（母约 §9 G7 第 2 片：会话事件侧车 + 派生视图 + 跨进程恢复）

工作项 `work-p5-session-sidecar` / 契约 `docs/design/v3-test-scope-p1-graph-slice28.json`（D1–D10）。

## 1. REFACTOR 结论：无独立重构动作，最终字节 = 修复后字节

接手（子代理 GREEN 中途被取消）后只补了 `PendingInterrupts` 的 6 行展平接线，实现已是最小形状；
REFACTOR 阶段零代码改动。最终字节：`sidecar.go a773337c…`、`bridge.go 1afa3f13…`、
`restore.go feebb799…`（变异矩阵基线口径；sidecar/bridge 文件级 git hash-object 见 green 文档 §2）。
最终门禁复测：绑定命令 exit 0（`receipt:2a7761b3…`）、邻近面 `receipt:66e68495…`、
整包 `-count=5 -race` `receipt:3c595bef…`。

## 2. 完整变异矩阵（`scripts/mutation/p5s28-sidecar.mjs`，仓库内可复现）

预检 12 条（m1–m10 + e1/e2）锚唯一性全过；基线三文件
`bridge.go 1afa3f13…` / `sidecar.go a773337c…` / `restore.go feebb799…`；
每条变异 `restored=true`、`finalCheck identical=true`。

| 变异 | 形态 | 命令 A | 判定 |
|---|---|---|---|
| m1 | 挂起记录写失败静默吞（D9 fail-closed 破） | 杀红 FailureSemantics | 杀红 |
| m2 | 事件写失败升级为阻塞（D9 fail-open 破） | 杀红 FailureSemantics | 杀红 |
| m3 | 半截记录放行（D2 fail-closed 破） | 杀红 InvalidRecordRejected | 杀红 |
| m4 | Install 覆盖活挂起（生命周期破） | 杀红 CrossProcessResumeRoundTrip / ResuspendChainLifecycle | 杀红 |
| m5 | 挂起记录部分接受 | 杀红 CrossProcessResumeRoundTrip | 杀红 |
| m6 | 成功后不清除挂起记录（D8 破） | 杀红 ResuspendChainLifecycle（B 零误伤） | 杀红 |
| m7 | RestorePending 校验摘掉 | 杀红 CrossProcessResumeRoundTrip | 杀红 |
| m8 | 快照水合丢缺省字段 | 杀红 SnapshotRestart | 杀红 |
| m9 | PendingInterrupts 退回 `return nil, nil`（接手时的存根形态） | 杀红 CrossProcessResumeRoundTrip / StoreRoundTripAndViews | **杀红 —— 正是接手时两条红的形态复现** |
| m10 | sink 失败吞挂起（D13 族跨片） | 杀红 FailureSemantics | 杀红 |
| e1/e2 | 等价变异（登记为观察上限） | 全绿 | 等价 |

B 侧（先验家族命令）在 m1/m2/m5/m7/m10 出现的 `TestP1R17_*` 判红为 S24-STD-2 已知环境性复发
（本轮宿主负载高），字节复核未变，登记不修。

## 3. 契约 completionCriteria 逐条对账

1. redProtocol：阶段 1 receipt 因子代理取消未及落盘（如实披露于 red 文档 §1）；
   中间态 `receipt:531c79d9…`（6 过/2 挂，D3 行真实红）+ 契约 M2 的 21 观察面副本实测
   共同充当红证据；修复后全绿 `receipt:2a7761b3…`；两段之间测试文件零改动。⚠️ 满足（带披露）。
2. 牙齿由变异证明：契约点名的形态（事件写入改道 Update、挂起失败静默吞、事件失败升级阻塞、
   成功不清除）分别由 m1/m2/m6/m4 覆盖，另补 m3/m5/m7/m8/m9/m10；e1/e2 等价登记。✅
3. 既有全部测试文件零改动；8 个边界锚逐字节全同（green 文档 §3）。✅
4. 门禁全 exit 0：build/vet/gofmt + 绑定命令 + 邻近面（agent/runner/session-contract/session）。✅
5. 结构判据：sidecar 18 / bridge 3 导出、新包 LOC 380（≤450）、边界改动文件数 0、
   `sync.` 仅 MemorySidecar 自有互斥（契约允许）。✅
6. 绑定命令最终字节重跑 `receipt:2a7761b3…`；-race 无 DATA RACE。✅
7. 写回：母约 §9/§10 P5 行标注第 2 片交付；D1 裁决材料补落地指针；状态文档由编排方收口。✅
8. 防碰巧绿：绑定 `-count=30 -race`（`receipt:b09c9acb…`）+ 条件性入口落进 graph 后的
   整包 `-count=5 -race`（`receipt:3c595bef…`）各一次 exit 0。✅

## 4. 过程如实登记（子代理取消事件）

本片实现原派发子代理，在 GREEN 中途被系统取消（无返回）。取消时已落地：全部四个实现/测试文件、
条件性入口 restore.go、变异脚本；未落地：RED/收尾 receipt、变异矩阵执行、证据文档、写回。
主代理接手后：取中间态回执（`531c79d9…`）→ 诊断两处失败同源于 `PendingInterrupts` 存根 →
补 6 行展平 → 全量门禁/变异/写回。教训并入 S27 家族：**长任务子代理可能被随时取消，
派发前必须把「已落盘状态可 salvage」作为前提设计**（本片的契约+预演副本使 salvage 成本仅 6 行）。

## 5. 审查

review 结论见 `docs/design/v3-p5-s28-review-verdict.json`（作者自查 + 编排方复核，
独立性披露同前序切片）。
