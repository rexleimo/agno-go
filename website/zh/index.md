---
layout: home

hero:
  name: "HNO"
  text: "Go 原生多智能体框架"
  tagline: "以明确、可复现的证据描述 Go 实现，不把未经验证的性能估算当成事实。"
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/guide/quick-start
    - theme: alt
      text: 在 GitHub 上查看
      link: https://github.com/rexleimo/agno-go

features:
  - title: 用测量代替猜测
    details: 仓库包含 Agent 创建 benchmark。性能页记录了命令、环境、结果范围和限制条件。

  - title: Provider 适配器
    details: Provider 实现通过 Model 接口提供。当前源码包含 17 个顶层 Provider 包；这是代码清单，不是兼容性或延迟保证。

  - title: 共享编排组件
    details: Agent、Team、Workflow 在 Go 源码中共享执行组件和工具分发抽象。

  - title: 可观测性集成
    details: 提供 OpenTelemetry 和结构化运行时埋点。启用埋点后的开销取决于部署配置，必须针对目标服务测量。

  - title: Skills、MCP 与记忆
    details: Agent Skills、MCP 桥接、可插拔记忆和会话存储作为可选框架能力提供。

  - title: 诚实的协议覆盖说明
    details: 自动化测试覆盖适配器和请求/响应映射。真机 Provider 验证需要凭据；测试响应不会被描述成线上证据。

---

## 本地系统开销矩阵

主框架对比使用同一个确定性的本地 OpenAI-compatible Stub，比较 HNO、Agno
和 LangGraph。每行使用 100 次正式操作、5 次预热、fresh operation 生命周期
以及相同的 1 ms Stub 响应延迟。RSS 是进程树工作集；RPS 是正式测量批次的
吞吐，不是生产容量承诺。

| 并发度 | Framework | 平均 | P95 | 测得 RPS | 成功率 | 峰值 RSS |
| ---: | --- | ---: | ---: | ---: | ---: | ---: |
| 8 | HNO | **1.859 ms** | **2.655 ms** | **4,186.08** | 100/100 | **12.3 MB** |
| 8 | Agno | 61.766 ms | 83.473 ms | 125.66 | 100/100 | 285.0 MB |
| 8 | LangGraph | 30.117 ms | 37.638 ms | 251.95 | 100/100 | 157.9 MB |
| 32 | HNO | **6.703 ms** | **18.637 ms** | **3,627.35** | 100/100 | **16.7 MB** |
| 32 | Agno | 138.678 ms | 241.744 ms | 170.17 | 100/100 | 373.8 MB |
| 32 | LangGraph | 78.370 ms | 129.618 ms | 241.67 | 100/100 | 161.1 MB |

在并发度 8 下，HNO 在这个固定本地协议中的测得批次 RPS 约为 LangGraph
的 16.6 倍、Agno 的 33.3 倍；并发度 32 下约为 15.0 倍和 21.3 倍。这些
是本地编排和运行时观测，不是远程模型加速。

完整的[本地系统开销矩阵](/zh/advanced/system-overhead)包含原始 JSON、
资源指标定义和复现命令。

## 远程模型附录

远程 DeepSeek 测试是独立的端到端快照：每条路径 100 次正式请求，并发度 8。
它包含网络、Provider 排队和模型生成时间，因此不是纯框架 benchmark。

| 路径 | 平均 | P95 | 成功率 | 相对平均值 |
| --- | ---: | ---: | ---: | ---: |
| Direct API | 2,094.52 ms | 4,225.19 ms | 100/100 | 1.00x |
| HNO | **1,312.71 ms** | **1,563.62 ms** | 100/100 | **1.60x** |
| Agno | 1,571.34 ms | 1,988.19 ms | 100/100 | 1.33x |
| LangGraph | 1,362.09 ms | 1,753.45 ms | 100/100 | 1.54x |

相对倍率定义为 `Direct API 平均耗时 / 路径平均耗时`。这组数据只能说明
本次快照中端到端延迟接近且 HNO 略有优势，不能作为远程生产性能的普遍结论。
详见[远程性能报告](/zh/advanced/performance)。

## HNO 博客最新文章

### AI Agent 框架性能怎么测：为什么运行时开销比模型延迟更重要

[阅读完整文章](/zh/blog/ai-agent-runtime-benchmark)

当一个模型或 Agent 框架成为热点时，先把模型延迟和框架开销分开，再讨论性能
结论。本文使用同一个本地 OpenAI-compatible Stub、5 次预热、100 次正式操作，
测试并发度 1、8、32。

更多内容见 [HNO 博客](/zh/blog/)，也可以订阅 [RSS](/rss.xml)。

## 为什么是 Go，为什么是 HNO

**为什么使用 Go？** Go 是实现选择，工程原因包括编译后的部署产物、内置
并发模型、静态类型、标准 HTTP/JSON 库，以及一流的测试和性能分析工具。
这些是设计理由，不代表任何工作负载都固定快多少。

**为什么叫 HNO？** HNO 是当前项目名称。仓库没有定义这个名称的正式全称，
所以本站不擅自编造。Go module 路径仍然是 `github.com/rexleimo/agno-go`；
HNO 是项目身份，不是标准化模型、协议或性能指标。

**证据规则：** 实测结果附命令、版本、环境、平均值、中位数和范围。Go 的分配
字节不能当作 Python 内存。真实 LLM、生产容量和跨框架结论需要相同 Provider、
相同工作负载的独立实验。
