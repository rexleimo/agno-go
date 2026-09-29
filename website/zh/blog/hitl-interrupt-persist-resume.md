---
title: "图的 HitL：把人放回回路 —— Interrupt、持久化与跨进程 Resume"
description: "v3 引擎如何把节点的 RequestInterrupt 变成可持久化的挂起，配 schema 校验、幂等恢复——以及为什么挂起绝不能是半截 Result。"
date: 2026-09-29
lastUpdated: 2026-09-29
author: HNO Team
category: Go engineering
tags:
  - Go
  - HITL
  - 图
  - 持久化
  - Agent 工作流
---

每个 Agent 框架都承诺「暂停等人工审批」。多数实现成阻塞 goroutine 的回调，
或一个调用方必须自行理解的半成品结果。这两种形态都在同一个地方崩掉：
进程重启，或者审批人去吃午饭。

这篇讲 v3 引擎的第三种落地：挂起是持久化状态，恢复带 schema 校验且幂等，
跨进程路径经得起审批场景唯一假设的东西——时间。

## 会失败的两个形状：半截结果与阻塞回调

阻塞回调把 goroutine 和 context 扣作人质，人多久审批就扣多久；取消运行后
审批问题直接蒸发。半截结果返回一个带标志位的 `Result`——但只走过半张图的
`Result`，恰恰是构建期校验工作用二十一个切片清除的那类「来自调用方没写过
的图的结论」。

所以引擎划了硬线：**挂起的运行不是 `Result`。** `Run` 返回 `(nil, err)`，
`err` 满足 `errors.Is(err, ErrSuspended)`，`errors.As` 取出 `*Suspension`，
携带全部待答中断与挂起前完成的节点。调用方不可能把等待中的运行误当完成——
错误链的类型系统让这个区分无法忽视。

## RequestInterrupt：节点侧最小面

节点不需要新接口、不改签名。需要人的节点返回哨兵：

```go
return nil, graph.RequestInterrupt(graph.Interrupt{
    InterruptID: "approve-release",
    ResponseSchema: map[string]any{
        "type":     "object",
        "required": []string{"approved"},
        "properties": map[string]any{
            "approved": map[string]any{"type": "boolean"},
        },
    },
    Mode: graph.ResumeRerun,
})
```

两个细节承载大部分语义。其一，`RequestInterrupt` 是哨兵错误，重试包络刻意
**不吞它**——`MaxAttempts: 5` 的节点一旦中断，恰执行一次然后挂起。其二，
每个 `Interrupt` 带自己的 `Mode`：`ResumeRerun` 让节点带着响应重跑（经
`InterruptResponse(ctx, id)` 取）；`ResumeHandoff` 不再执行——响应直接成为
它的输出流向后继。审批流要交接；「按反馈重新生成」要重入。按中断逐个选，
不按图选。

## Resume：校验过、幂等、对失败诚实

`Resume(ctx, responses)` 强制三条语义，每条都有变异测试过的担保：

1. **schema 校验。** 响应按中断的 schema 校验——诚实的子集（`type`、
   `required`、`properties.<name>.type`），不是假装的完整 JSON Schema。坏
   响应返回 `ErrInvalidResponse` 且挂起原样保留。调用方修正答案重试即可；
   什么都没消耗。
2. **幂等。** 双重恢复与从未挂起同答 `ErrNothingToResume`。诱人的捷径——
   返回空 `Result`——正是 join 屏障工作批评过的「静默无结论」形状，所以
   点名拒绝。
3. **按中断双档。** 一次 Resume 可以同时回答一个重入中断和一个交接中断；
   并行扇出可以同时挂起两支、一次恢复两支。

## Durability：挂起就是检查点数据

挂起骑在上一版引擎发布的 durability 缝上：挂起时经同一个 `Checkpointer`
（按图声明的 Sync/Async/Exit 档）提交 `EntryInterrupt` 条目，与已完成节点
条目并列。sink 失败不会掩盖挂起——两个错误都能 `errors.Is`。步数预算跨
恢复累计，因为中断了六个小时的运行仍然是一次逻辑运行。

跨进程恢复由会话侧车（`pkg/hno/session/sidecar`）加一个小桥接完成：把挂起
捕获进存储，全程不触碰会话对外 JSON——对 marshal 锚逐字节验证过——新进程
按声明重建图、装回存储的挂起态后调 `Resume`。端到端测试在双档下走完
中断→快照→「重启」→恢复。

## 变异矩阵怎么说

本片 13 条变异矩阵杀掉每一条恢复担保：摘掉捕获，中断变普通错误；摘掉
schema 校验，坏响应消耗掉挂起；反转重试守卫，把重试预算烧在一个没人回答
的问题上；每次恢复重置预算，把六小时审批变成崭新的 1000 步额度。两条等价
变异如实登记——包括一个覆盖率缺口（停止态 span），已排队跟进。

## 代码入口

- 指南：[Human-in-the-Loop](/zh/guide/human-in-the-loop)
- API：[Graph — HITL](/zh/api/graph)
- 持久化缝：[Graph Durability](/zh/advanced/graph-durability)
- 证据：`docs/design/v3-p5-graph-g7-green.md`、`docs/design/v3-p5-s28-green.md`
  及其引用的切片契约。
