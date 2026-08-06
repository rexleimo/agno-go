---
title: HNO Blog
description: Original engineering notes, benchmark reports, and timely analysis for Go-native AI agent systems.
head:
  - - meta
    - name: keywords
      content: "AI agent blog, agent framework benchmark, Go AI, HNO, LangGraph, Agno"
  - - meta
    - property: og:title
      content: "HNO Blog"
  - - meta
    - property: og:description
      content: "Original engineering notes, benchmark reports, and timely analysis for Go-native AI agent systems."
  - - link
    - rel: canonical
      href: https://hno.rexai.top/blog/
---

# HNO Blog

Short, evidence-led articles about AI agent systems, Go runtime design, benchmarks,
and the engineering ideas behind the news cycle.

## Latest article

### Agent File Tools Need Root-Bound Sandboxes, Not Path Prefix Checks

[Read the full article](/blog/sandboxed-file-io)

An engineering note on why a string path allowlist is not an Agent security
boundary, and how HNO uses explicit read/write capabilities plus Go's
root-bound filesystem handles to contain file-tool operations.

- **Category:** Security engineering
- **Tags:** AI agents, Agent security, sandbox, file I/O, Go
- **Evidence:** Root-relative tool paths, `os.Root` enforcement, escape-regression tests, and documented deployment limits

### Also recent: AI Agent Framework Benchmarks: Why Runtime Overhead Matters More Than Model Latency

[Read the benchmark](/blog/ai-agent-runtime-benchmark)

## How HNO uses timely topics

A timely headline is only the starting point. Every article should add an original
engineering question, a reproducible example, or a measured result that remains
useful after the headline stops trending.

1. **Capture the topic and source it.** Record the source URL and publication time.
2. **Connect it to a real HNO question.** Do not force an unrelated keyword into an article.
3. **Add evidence.** Prefer code, benchmark commands, traces, or a clear limitation.
4. **Link to durable documentation.** Readers should be able to continue to the
   [Sandboxed File I/O guide](/guide/sandboxed-file-io), [Agent guide](/guide/agent),
   [performance report](/advanced/performance), or [system overhead matrix](/advanced/system-overhead).
5. **Update instead of duplicating.** If the same topic evolves, update the original
   article and record what changed.

## Topic areas

- AI agent frameworks and orchestration
- Go concurrency, memory, and deployment
- Model-provider integration and OpenAI-compatible APIs
- MCP, RAG, memory, and observability
- Reproducible performance measurement

## Editorial boundary

HNO is not affiliated with a product merely because an article discusses it. We do
not copy source articles, fabricate benchmark results, or present a remote-model
latency snapshot as a universal framework claim. Trend-driven content still needs
accurate sources, a publication date, and a clear measurement boundary.

Subscribe to the site RSS feed: [/rss.xml](/rss.xml).
