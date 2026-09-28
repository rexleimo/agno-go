---
title: HNO 博客
description: 面向 Go 原生 AI Agent 系统的原创工程文章、性能报告和热点技术分析。
head:
  - - meta
    - name: keywords
      content: "AI Agent 博客, Agent 框架性能, Go AI, HNO, LangGraph, Agno"
  - - meta
    - property: og:title
      content: "HNO 博客"
  - - meta
    - property: og:description
      content: "面向 Go 原生 AI Agent 系统的原创工程文章、性能报告和热点技术分析。"
  - - link
    - rel: canonical
      href: https://hno.rexai.top/zh/blog/
---

# HNO 博客

围绕 AI Agent、Go 运行时、框架性能和模型生态写一些有证据的原创文章，
把热点当作入口，把可复现的工程内容留下来。

## 文章列表

### 在 Go 里造一个 LangGraph：一个零锁图引擎的设计与实证

[阅读完整文章](/zh/blog/zero-lock-graph-engine)

HNO v3 图执行器如何在不加一把锁的前提下跑 DAG、条件路由与汇聚屏障 —— 以及让
这个断言可审计的构建期校验清单、运行期护栏与变异矩阵。

- **分类：**Go engineering
- **标签：**Go、并发、图引擎、LangGraph、DAG、Agent 框架

### 节点策略四合一：Retry/Timeout/Cache/Trace 的一次 variadic 设计

[阅读完整文章](/zh/blog/node-option-variadic-design)

用一条变参 `AddNode` 选项缝把重试、超时、缓存、追踪挂到图节点上：零值即语
义、fail-closed 缓存、零锁调度器里的串行 trace hook。

- **分类：**API design
- **标签：**Go、API 设计、重试、缓存、超时、可观测性

### 一次性代码沙盒：无惧运行 LLM 生成的代码

[阅读完整文章](/zh/blog/code-execution-sandbox)

HNO 如何通过一次性 Podman/Docker 容器或显式 E2B Cloud VM 运行 agent 生成的代码，默认断网、资源有界、provider fail-closed。

- **分类：**安全工程
- **标签：**AI Agent、沙盒、代码执行、Podman、E2B、Go

### Agent 文件工具不能只靠路径前缀：HNO 如何用根句柄做沙盒

[阅读完整文章](/zh/blog/sandboxed-file-io)

一篇工程文章，说明为什么字符串路径白名单不是 Agent 安全边界，以及 HNO 如何结合
显式读写能力和 Go 的根句柄绑定文件系统操作来约束文件工具。

- **分类：**安全工程
- **标签：**AI Agent、Agent 安全、沙盒、文件 I/O、Go

### AI Agent 框架性能怎么测：为什么运行时开销比模型延迟更重要

[阅读性能基准](/zh/blog/ai-agent-runtime-benchmark)

## HNO 如何使用热点

热点只是选题入口，不是复制内容的理由。每篇文章都应该回答一个真实的工程
问题，并留下代码、数据、复现命令或明确的限制条件。

1. **记录热点来源。** 保存来源 URL 和发布时间。
2. **连接到真实的 HNO 问题。** 不为了关键词强行写无关内容。
3. **补充证据。** 优先使用代码、benchmark、trace 或可复现的测量。
4. **链接到长期文档。** 读者可以继续查看[沙盒化文件 I/O 指南](/zh/guide/sandboxed-file-io)、
   [Agent 指南](/zh/guide/agent)、[性能报告](/zh/advanced/performance) 或[系统开销矩阵](/zh/advanced/system-overhead)。
5. **持续更新而不是重复发文。** 同一热点出现新进展时，更新原文章并记录变化。

## 主题方向

- AI Agent 框架与编排
- Go 并发、内存和部署
- 模型 Provider 与 OpenAI-compatible API
- MCP、RAG、记忆和可观测性
- 可复现的性能测量

## 编辑边界

讨论某个产品不代表 HNO 与它存在官方关联。我们不复制来源文章、不编造基准
结果，也不会把远程模型端到端延迟快照写成通用的框架结论。热点文章仍然要有
准确来源、发布时间和清楚的测量边界。

订阅站点 RSS：[/rss.xml](/rss.xml)。
