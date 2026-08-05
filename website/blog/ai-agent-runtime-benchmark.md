---
title: "AI Agent Framework Benchmarks: Why Runtime Overhead Matters More Than Model Latency"
description: "A reproducible local-stub benchmark comparing HNO, Agno, and LangGraph at 100 requests and concurrency 1, 8, and 32."
date: 2026-08-05
lastUpdated: 2026-08-05
author: HNO Team
category: Benchmarks
tags:
  - AI agents
  - agent frameworks
  - benchmark
  - Go
  - Python
  - LangGraph
  - Agno
hotTopic: "AI agent performance"
head:
  - - meta
    - name: keywords
      content: "AI agent benchmark, agent framework performance, Go AI framework, LangGraph benchmark, Agno benchmark"
  - - meta
    - property: og:type
      content: article
  - - meta
    - property: og:title
      content: "AI Agent Framework Benchmarks: Why Runtime Overhead Matters More Than Model Latency"
  - - meta
    - property: og:description
      content: "A reproducible local-stub benchmark comparing HNO, Agno, and LangGraph."
  - - meta
    - property: article:published_time
      content: "2026-08-05T00:00:00Z"
  - - link
    - rel: canonical
      href: https://hno.rexai.top/blog/ai-agent-runtime-benchmark
---

# AI Agent Framework Benchmarks: Why Runtime Overhead Matters More Than Model Latency

When a new model or agent framework becomes a hot topic, performance claims often
collapse several different costs into one number. A remote request may include
network time, provider queueing, model generation, response parsing, orchestration,
and process startup. Those are useful end-to-end facts, but they are not the same
as framework runtime overhead.

This article uses a controlled local protocol to separate the question. It is an
original HNO benchmark note, not an endorsement of or affiliation with Agno or
LangGraph.

## The question behind the headline

The useful question is not simply “which framework is fastest?” It is:

> How much work does each framework add when the model response is held constant?

That question matters when a new model release is trending. If the model is the
same for every path, a local deterministic stub lets us observe orchestration and
runtime costs before remote provider variance dominates the result.

## Protocol

The checked-in matrix uses the following rules:

- HNO, Agno, and LangGraph use the same local OpenAI-compatible stub.
- The stub returns a fixed `LOCAL_MODEL_OK` response after a 1 ms delay.
- Each path performs 5 serial warmups and 100 measured operations.
- Concurrency is tested at 1, 8, and 32.
- A fresh client/model/Agent/Graph is created for every operation.
- Samples record client wall-clock latency, while the process records RSS and CPU telemetry.

The fresh-operation rule is intentional. It answers a cold/lifecycle question, not
a steady-state server-capacity question.

## Results

All rows completed successfully.

### Concurrency 1

| Framework | Mean | P95 | Measured RPS | Peak RSS |
| --- | ---: | ---: | ---: | ---: |
| HNO | **1.583 ms** | **1.675 ms** | **631.71** | **12.2 MB** |
| Agno | 41.632 ms | 61.863 ms | 24.01 | 204.5 MB |
| LangGraph | 7.687 ms | 9.082 ms | 129.68 | 156.3 MB |

### Concurrency 8

| Framework | Mean | P95 | Measured RPS | Peak RSS |
| --- | ---: | ---: | ---: | ---: |
| HNO | **1.859 ms** | **2.655 ms** | **4,186.08** | **12.3 MB** |
| Agno | 61.766 ms | 83.473 ms | 125.66 | 285.0 MB |
| LangGraph | 30.117 ms | 37.638 ms | 251.95 | 157.9 MB |

### Concurrency 32

| Framework | Mean | P95 | Measured RPS | Peak RSS |
| --- | ---: | ---: | ---: | ---: |
| HNO | **6.703 ms** | **18.637 ms** | **3,627.35** | **16.7 MB** |
| Agno | 138.678 ms | 241.744 ms | 170.17 | 373.8 MB |
| LangGraph | 78.370 ms | 129.618 ms | 241.67 | 161.1 MB |

In this protocol, the HNO process adds less local orchestration overhead than the
Python paths. That is useful evidence for this lifecycle and workload. It is not a
claim that HNO is faster for every model, tool loop, memory backend, or production
service.

## What this does not prove

This matrix does not measure:

- model quality or tokens per second;
- remote provider latency or queueing;
- streaming behavior;
- tool loops, memory, teams, or RAG retrieval;
- production capacity under a long-lived worker pool;
- Python allocation bytes as if they were Go `B/op`.

For the separate remote-model snapshot, see the [100-run DeepSeek performance report](/advanced/performance).
It includes the model and network path, so it must be read as an end-to-end client
measurement rather than a pure framework ranking.

## How to turn a hot topic into useful engineering content

When a new model or Agent framework is trending, use this sequence instead of
repeating the headline:

1. Freeze the question: model latency, framework overhead, or service capacity?
2. Make the workload identical across implementations.
3. Record versions, hardware, concurrency, warmups, and lifecycle semantics.
4. Publish raw samples and limitations with the summary.
5. Update the same article when the protocol or result changes.

The reproduction command and raw files are documented in the
[local system overhead matrix](/advanced/system-overhead) and the repository's
`benchmarks/framework_comparison/` directory.

## Reproduce it

From the repository root:

```bash
uv run --with psutil --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  --with 'langchain-openai' --with 'langchain-core' \
  python benchmarks/framework_comparison/local_overhead_matrix.py
```

The benchmark is a snapshot. Re-run it on your own machine before making a
production decision.
