---
title: "在 Go 里造一个 LangGraph：一个零锁图引擎的设计与实证"
description: "HNO v3 图执行器如何在不加一把锁的前提下跑 DAG、条件路由与汇聚屏障 —— 以及让这个断言可审计的构建期校验清单、运行期护栏与变异矩阵。"
date: 2026-09-28
lastUpdated: 2026-09-28
author: HNO Team
category: Go engineering
tags:
  - Go
  - 并发
  - 图引擎
  - LangGraph
  - DAG
  - Agent 框架
head:
  - - meta
    - name: keywords
      content: "零锁图引擎, Go DAG 调度器, 单消费者并发模型, LangGraph Go 替代, Agent 编排框架, 构建期图校验"
  - - meta
    - property: og:type
      content: article
  - - meta
    - property: og:title
      content: "在 Go 里造一个 LangGraph：一个零锁图引擎的设计与实证"
  - - meta
    - property: og:description
      content: "一个没有锁的 Go 控制流图引擎：构建期校验清单、运行期护栏，以及变异测试养出的信任。"
  - - meta
    - property: article:published_time
      content: "2026-09-28T00:00:00Z"
  - - link
    - rel: canonical
      href: https://hno.rexai.top/zh/blog/zero-lock-graph-engine
---

# 在 Go 里造一个 LangGraph：一个零锁图引擎的设计与实证

每个 Agent 编排框架最终都会长出一个图执行器：节点是步骤、边是控制流、条件做
分支、Join 做并行汇聚。LangGraph 在 Python 世界把这个形状带火了。当我们开始写
HNO 的 v3 图引擎时，有意思的问题不是抄哪些原语 —— 而是一个并发问题：
**锁在哪里？**

这是一篇 HNO 的原创工程笔记。HNO 与 LangGraph、Agno 没有任何关联；我们提到
它们只作为设计先例与基准邻居，仅此而已。

## 默认形状，以及锁为什么总是默认选项

图执行器是一个繁忙的共享对象。四个分支并行跑的时候，总得有什么东西维护待派
队列、运行计数、哪些 Join 前驱已经到齐、结果表。常规答案是在这些状态外面套一
把（或几把）互斥锁，因为任何生产者 goroutine 都可能在任何时刻改它们中的任何
一个。

锁是正确的、广为理解的、乏味的 —— 褒义的乏味。但它买正确性的方式是把对一批
状态的访问串行化，而细看之下，这批状态并不是真的被所有人共享读写。

## 重新表述：锁保护的到底是什么

锁保护的是**可变状态的独占权**。它不保护时间片、goroutine 生命周期，也不保护
channel —— 那些是 Go 原生给你的。于是设计问题变成：*调度器状态里，到底有多少
真正需要多写者访问？*

我们的回答是：一个都没有。这就指向了另一种架构。

## 单消费者模型

HNO 的图引擎（致谢：模型借鉴 adk-go 的 workflow scheduler）把执行拆成两个角
色：

- **生产者 goroutine** 跑节点激活。每个生产者执行节点代码 —— 重试、超时、缓
  存查询全在内 —— 然后向 channel 发送恰好一条完成项。生产者从不写调度器状
  态。panic 的生产者在节点帧之上被恢复，仍通过队列汇报。
- **唯一消费者**是调用 `Run` 的那个 goroutine。它是完成队列的唯一读者，也是
  pending 列表、运行计数、Join 屏障表、步数账本与结果的唯一写者。

只有一个 goroutine 触碰的状态不需要锁。不是「一把便宜的锁」，是不需要锁。这个
断言可以对着引擎源码一行审计：

```bash
grep -o 'sync\.' pkg/hno/graph/graph.go pkg/hno/graph/scheduler.go \
  pkg/hno/graph/policy.go pkg/hno/graph/durability.go | wc -l
# 0
```

而「没有 `sync.`」只是没有证据，不是证据的缺失 —— 所以整包图引擎测试（含取
消归一、Join 屏障、重复注册、策略、持久化的并发专项）在 `-race` 下全绿。各切
片的回执都在 `docs/design/v3-p1-graph-status.md` 在册。

唯一刻意的例外形态：Async 持久化档跑一个冲刷 goroutine，它与引擎只共享一条
待提交 channel。channel 交接不是共享可变状态，不变量在那里也成立。

## 构建期校验清单：错就错在 Validate

静默的图 bug 比崩溃更糟：一条声明错了的边悄悄产出 `err == nil` 加一个缺失的
输出，那是调用方从没写过的结论。所以 `Validate()` 在任何节点执行前跑完整清
单，而且每条拒绝都是一句你能归因到自己那行代码的文案：

| 拒绝项 | 错误文案（逐字） |
|---|---|
| 非正步数上限 | `graph: WithStepLimit(0) is not a usable step limit: a positive number of steps is required` |
| 入口不存在 | `graph: entry node "..." does not exist` |
| 输出节点不存在 | `graph: output node "..." does not exist` |
| 悬空边端点 | `graph: unconditional edge from node "..." does not exist` |
| 条件边缺谓词 | `graph: conditional edge from "..." to "..." declares no predicate` |
| 不可路由的边种类 | `graph: ... edge from "..." to "..." is declared but not routable by this engine` |
| Join 前驱不足 | `graph: join target "..." must declare at least 2 distinct predecessors, got 1` |
| 节点不可达 | `graph: unreachable node "...": not reachable from entry "..." through any declared edge` |
| 不可断开的环 | `graph: unconditional cycle "a" -> "b" -> "a": every edge on it is always taken, so Run cannot terminate` |
| 重复节点名 | `graph: node name "..." is declared more than once` |

三条值得展开：

- **环是合法的。** 只有「每条边都必定被走」的环被拒绝，因为那种环不可能终止。
  任何一条条件边能断开的环都过校验 —— 运行期由步数阀兜底。
- **可归因文案是特性。** 所有节点在同一行代码用常量文案报类型不符是调试死路；
  这里每条错误点名出问题的声明。长环渲染成开头点名 + 省略号 + 规模数字，
  20000 个节点的环不会把 260 KB 倒进你的 issue。
- **重复注册是记账，不是 panic。** `AddNode` 返回 `*Graph`，builder 上不存在
  错误通道。同名再注册仍在表里覆盖（进行中的 Run 持有拓扑快照），但撞名被记
  账、`Validate()` 随后拒绝。记账 + 复查是这个 API 形状下唯一诚实的增量检查。

非致命关切走独立、只读的 `Warnings()` —— 今天唯一一条，提醒你含环图正跑在一个
你从没选过的默认步数预算 1000 上。危险和建议是两条通道；混在一起等于替调用方
决定这张图不能跑。

## 运行期护栏

校验看不见的东西 —— 谓词、节点行为、进程信号 —— 由运行期带着阀门，全部由唯
一消费者执行：

1. **步数阀**（默认 1000）。计步发生在把激活交给节点*之前*：会让步数越界的激
   活根本不启动，副作用不会发生。错误包装 `graph.ErrStepLimitExceeded` 哨兵，
   供 `errors.Is`。
2. **panic 屏障。** 节点 panic 变成
   `graph: node "..." panicked while running: ...`，在节点自己的 `defer` 跑完
   后恢复，像任何错误一样经队列汇报。
3. **取消归一。** 调用方取消与节点错误同时成立时，`Run` 返回 `ctx.Err()` ——
   运行报告的是谁叫停了它，而不是它恰好死在干什么。
4. **并发上限 FIFO。** `0` 表示不限是守卫自身的语义；槽位打满就按序排队，每完
   成一个放行下一个。
5. **Join 屏障。** 汇聚目标在全部声明前驱完成后激活一次，收到以前驱名为键的
   `map[string]any`。「凑齐」按键集合覆盖判定，不按计数器算术 —— 集合就是调用
   方在结果里看到的那个形状。
6. **快照隔离。** `Run` 执行的是入口时通过 `Validate()` 的那份拓扑；节点里一
   次随手 `AddEdge` 偷运不进进行中的执行。

## 工程纪律怎么长出信任：变异矩阵

「测试都过了」是弱陈述；「实现不对时测试只能红」才是强陈述。这个引擎的每个切
片都按两段制交付 —— 先对着未接线的实现写行为测试（RED，回执在册），接线后用
逐字节相同的测试复跑（GREEN，回执在册）—— 然后过**变异矩阵**：刻意按指定方
式破坏实现，要求套件抓住每一个。

证据文档里的杀红记录：

| 切片 | 变异 | 结果 |
|---|---|---|
| 节点策略（重试/超时/缓存/追踪） | 8 条 + 1 等价 | 8 条杀红；e1（失败路径 trace 发射）登记为等价 |
| 流协议（六事件类型） | m1–m6 | 全部杀红 |
| 持久化（三档 + fail-closed sink） | m1–m8 + e1/e2 | 全部杀红；等价项连同量得的观察上限一并登记 |
| 观测接线 | m1–m9 + e1/e2 | 全部杀红；e2 顺带量得一种缺陷形态的观察上限并登记 |
| 重复名拒绝 | m1–m3 + m4 | m1–m3 杀红；m4 等价 |

两个值得复述的例子。来自变异记录：摘掉事件解码里「精确匹配先于模糊归一化」的
次序，旧的子串匹配器就会把 `node_completed` 吞进完成事件（协议片变异 m2，杀
红）。来自调度器里钉住的设计理由：步数记账刻意发生在把激活交给节点*之前*，因
为按「已完成数」计步会先把一整批并行激活放出限额再道歉 —— 召不回的副作用不是
阀门。变异矩阵就是让这类理由不烂成没人复查的注释的手段。

## 相邻基准（先读边界）

入库的框架对比矩阵 —— 同一本地 stub、每次操作新建对象 —— 度量 HNO 相对 Agno
与 LangGraph 的运行时开销：

| 并发 | HNO 均值 | HNO RPS | HNO RSS | LangGraph 均值 | Agno 均值 |
|---|---:|---:|---:|---:|---:|
| 1 | 1.583 ms | 631.71 | 12.2 MB | 7.687 ms | 41.632 ms |
| 8 | 1.859 ms | 4,186.08 | 12.3 MB | 30.117 ms | 61.766 ms |
| 32 | 6.703 ms | 3,627.35 | 16.7 MB | 78.370 ms | 138.678 ms |

边界，从原基准文章照实重述：这是冷启动、生命周期口径的协议（每次操作新建
client/model/agent、stub 固定响应、5 次预热 + 100 次度量）。它走的是 agent 运
行路径 —— **不**单独度量图引擎，对模型质量、远程延迟、流式、工具循环、稳态
服务容量一概不作证。在这里引用它，是因为它与本引擎是同一种工程姿态 —— 量你
真正增加的开销 —— 施加在这个运行时上。复现命令见
[基准文章](/zh/blog/ai-agent-runtime-benchmark)。

## 这证明不了什么

诚实小节，与 HNO 每篇基准笔记同款：

- **单消费者模型不是万能的。** 它适合「状态天然是一条只有一个排空者的消息队
  列」的调度器。许多 goroutine 必须读写同一结构的工作负载仍是锁（或 channel）
  的问题，硬套这个形状是教条不是设计。
- **零锁不是零成本。** 一切仍经一个消费者汇合；病态拓扑可以让那个 goroutine
  成为瓶颈。我们认为这是单写者状态的可调试性换来的一笔好交易 —— 但它是一笔
  交易。
- **上面的基准是生命周期口径**，且没有单独度量图执行。
- **动态扇出（`Send`）写作时尚未交付**，此后已落地：`SenderFunc` 扇出加显式
  `AddJoinSend(source, target)` 汇聚声明，见[图引擎指南](/zh/guide/graph-engine)；
  另写一篇的计划照旧。同样，从持久化检查点恢复/重放已设计未落地。
- **一个运行期形状仍开放**：Join 屏障的前驱始终凑不齐且不再有待派激活时，当前
  交出的是 `err == nil` 加空输出，而不是一句诊断。它登记为未裁决项，我们宁可
  先告诉你，也不想让你在生产里自己发现。

## 复现

在仓库根目录：

```bash
# 关于引擎的断言：
grep -o 'sync\.' pkg/hno/graph/graph.go pkg/hno/graph/scheduler.go \
  pkg/hno/graph/policy.go pkg/hno/graph/durability.go | wc -l   # 期望 0
go test ./pkg/hno/graph/... -race -count=1

# 基准矩阵：
uv run --with psutil --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  --with 'langchain-openai' --with 'langchain-core' \
  python benchmarks/framework_comparison/local_overhead_matrix.py
```

上手图引擎看[图引擎指南](/zh/guide/graph-engine)；签名在
[Graph API 参考](/zh/api/graph)。

## 延伸阅读

- [下一篇：节点策略四合一 —— 一次 variadic 设计](/zh/blog/node-option-variadic-design)
- [AI Agent 框架性能基准](/zh/blog/ai-agent-runtime-benchmark)
- [全部博客文章](/zh/blog/)
