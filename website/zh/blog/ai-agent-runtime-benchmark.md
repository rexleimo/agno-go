---
title: "AI Agent 框架性能怎么测：为什么运行时开销比模型延迟更重要"
description: "使用相同的本地 Stub，对 HNO、Agno 和 LangGraph 进行 100 次、并发度 1/8/32 的可复现对比。"
date: 2026-08-05
lastUpdated: 2026-08-05
author: HNO Team
category: 性能基准
tags:
  - AI Agent
  - Agent 框架
  - 性能基准
  - Go
  - Python
  - LangGraph
  - Agno
hotTopic: "AI Agent 性能"
head:
  - - meta
    - name: keywords
      content: "AI Agent 性能基准, Agent 框架性能, Go AI 框架, LangGraph 性能, Agno 性能"
  - - meta
    - property: og:type
      content: article
  - - meta
    - property: og:title
      content: "AI Agent 框架性能怎么测：为什么运行时开销比模型延迟更重要"
  - - meta
    - property: og:description
      content: "使用相同的本地 Stub，对 HNO、Agno 和 LangGraph 进行可复现对比。"
  - - meta
    - property: article:published_time
      content: "2026-08-05T00:00:00Z"
  - - link
    - rel: canonical
      href: https://hno.rexai.top/zh/blog/ai-agent-runtime-benchmark
---

# AI Agent 框架性能怎么测：为什么运行时开销比模型延迟更重要

当一个新模型或 Agent 框架成为热点时，很多性能数字会把不同成本混在一起：
网络、Provider 排队、模型生成、响应解析、框架编排和进程启动。端到端请求耗时
当然有价值，但它并不等于框架自身的运行时开销。

这篇文章使用固定的本地协议把问题拆开。它是 HNO 的原创基准记录，不代表 HNO
与 Agno 或 LangGraph 存在任何官方关联。

## 热点标题背后的真正问题

真正有用的问题不是简单问“哪个框架最快”，而是：

> 当模型响应完全相同时，每个框架额外增加了多少工作？

新模型发布时，这个问题尤其重要。如果所有路径使用同一个模型，先用确定性的
本地 Stub 观察编排和运行时开销，就不会被远程 Provider 的波动完全淹没。

## 测试口径

本次矩阵遵循以下规则：

- HNO、Agno 和 LangGraph 使用同一个本地 OpenAI-compatible Stub。
- Stub 在 1 ms 延迟后返回固定的 `LOCAL_MODEL_OK`。
- 每条路径串行预热 5 次，正式测量 100 次。
- 测试并发度 1、8、32。
- 每次操作都会重新创建 client、model、Agent 或 Graph。
- 记录客户端墙钟延迟，同时记录进程 RSS 和 CPU 指标。

这里采用 fresh operation，是为了回答冷启动和生命周期问题，不是长期运行的
Worker 服务容量测试。

## 结果

所有行均成功完成。

### 并发度 1

| Framework | 平均 | P95 | 测得 RPS | 峰值 RSS |
| --- | ---: | ---: | ---: | ---: |
| HNO | **1.583 ms** | **1.675 ms** | **631.71** | **12.2 MB** |
| Agno | 41.632 ms | 61.863 ms | 24.01 | 204.5 MB |
| LangGraph | 7.687 ms | 9.082 ms | 129.68 | 156.3 MB |

### 并发度 8

| Framework | 平均 | P95 | 测得 RPS | 峰值 RSS |
| --- | ---: | ---: | ---: | ---: |
| HNO | **1.859 ms** | **2.655 ms** | **4,186.08** | **12.3 MB** |
| Agno | 61.766 ms | 83.473 ms | 125.66 | 285.0 MB |
| LangGraph | 30.117 ms | 37.638 ms | 251.95 | 157.9 MB |

### 并发度 32

| Framework | 平均 | P95 | 测得 RPS | 峰值 RSS |
| --- | ---: | ---: | ---: | ---: |
| HNO | **6.703 ms** | **18.637 ms** | **3,627.35** | **16.7 MB** |
| Agno | 138.678 ms | 241.744 ms | 170.17 | 373.8 MB |
| LangGraph | 78.370 ms | 129.618 ms | 241.67 | 161.1 MB |

在这个协议中，HNO 的本地编排开销低于两个 Python 路径。这是针对本次生命
周期和工作负载的证据，不代表 HNO 在所有模型、工具循环、记忆后端或生产服务
中都更快。

## 这些数据不能证明什么

这个矩阵没有测量：

- 模型质量或 tokens per second；
- 远程 Provider 延迟和排队；
- 流式行为；
- 工具循环、记忆、Team 或 RAG 检索；
- 长期 Worker 进程的生产容量；
- 把 Python 内存数据当成 Go `B/op`。

远程模型快照请查看[100 次 DeepSeek 性能报告](/zh/advanced/performance)。它包含
模型和网络路径，应该被理解为端到端客户端测量，而不是纯框架排名。

## 如何把热点写成有用的工程内容

新模型或 Agent 框架登上热点时，可以按下面的顺序写，而不是重复标题：

1. 先固定问题：要测模型延迟、框架开销还是服务容量？
2. 让所有实现使用完全相同的工作负载。
3. 记录版本、硬件、并发度、预热次数和生命周期口径。
4. 和汇总结果一起发布原始样本与限制条件。
5. 协议或结果发生变化时，更新原文章，而不是重复创建同义页面。

复现命令和原始文件见[本地系统开销矩阵](/zh/advanced/system-overhead)以及仓库中的
`benchmarks/framework_comparison/` 目录。

## 复现

在仓库根目录执行：

```bash
uv run --with psutil --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  --with 'langchain-openai' --with 'langchain-core' \
  python benchmarks/framework_comparison/local_overhead_matrix.py
```

这是一组特定环境下的快照。做生产决策前，请在自己的机器上重新运行。
