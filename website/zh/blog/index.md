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

## 最新文章

### AI Agent 框架性能怎么测：为什么运行时开销比模型延迟更重要

[阅读完整文章](/zh/blog/ai-agent-runtime-benchmark)

使用同一个本地 OpenAI-compatible Stub，对 HNO、Agno 和 LangGraph 进行可复现
对比：每组 100 次正式操作，并发度 1、8、32。文章解释了为什么看到某个新模型
或 Agent 框架登上热点时，不能直接把宣传中的模型速度当成框架性能。

- **分类：**性能基准
- **标签：**AI Agent、框架性能、Go、Python、LangGraph、Agno
- **证据：**5 次预热、100 次正式请求、相同的本地 OpenAI-compatible Stub

## HNO 如何使用热点

热点只是选题入口，不是复制内容的理由。每篇文章都应该回答一个真实的工程
问题，并留下代码、数据、复现命令或明确的限制条件。

1. **记录热点来源。** 保存来源 URL 和发布时间。
2. **连接到真实的 HNO 问题。** 不为了关键词强行写无关内容。
3. **补充证据。** 优先使用代码、benchmark、trace 或可复现的测量。
4. **链接到长期文档。** 读者可以继续查看 [Agent 指南](/zh/guide/agent)、
   [性能报告](/zh/advanced/performance) 或[系统开销矩阵](/zh/advanced/system-overhead)。
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
