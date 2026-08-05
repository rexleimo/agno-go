---
layout: home

hero:
  name: "HNO"
  text: "Go-Native Multi-Agent Framework"
  tagline: "A Go implementation with explicit, reproducible evidence instead of unqualified performance claims."
  actions:
    - theme: brand
      text: Get Started
      link: /guide/quick-start
    - theme: alt
      text: View on GitHub
      link: https://github.com/rexleimo/agno-go

features:
  - title: Measured, not guessed
    details: The repository contains Go benchmarks for agent construction. The Performance page records the command, environment, ranges, and limitations.

  - title: Provider adapters
    details: Provider implementations are exposed behind the Model interface. The current source contains 17 top-level provider packages; this is an inventory, not a compatibility or latency guarantee.

  - title: Shared orchestration components
    details: Agent, Team, and Workflow share execution components and tool dispatch abstractions in the Go source.

  - title: Observability integration
    details: OpenTelemetry and structured runtime instrumentation are available. Deployment overhead depends on which instrumentation is enabled and must be measured for the target service.

  - title: Skills, MCP, and memory
    details: Agent Skills, MCP bridging, pluggable memory, and session storage are implemented as opt-in framework capabilities.

  - title: Honest protocol coverage
    details: Automated tests cover adapters and request/response mappings. Live-provider validation requires credentials; test responses are not presented as live evidence.

---

## Local system overhead matrix

The primary framework comparison uses the same deterministic local
OpenAI-compatible stub for HNO, Agno, and LangGraph. Each row uses 100 measured
operations, five warmups, fresh-operation lifecycle semantics, and the same 1 ms
stub response delay. RSS is process-tree working set; RPS is the measured batch
throughput, not a production-capacity promise.

| Concurrency | Framework | Mean | P95 | Measured RPS | Success | Peak RSS |
| ---: | --- | ---: | ---: | ---: | ---: | ---: |
| 8 | HNO | **1.859 ms** | **2.655 ms** | **4,186.08** | 100/100 | **12.3 MB** |
| 8 | Agno | 61.766 ms | 83.473 ms | 125.66 | 100/100 | 285.0 MB |
| 8 | LangGraph | 30.117 ms | 37.638 ms | 251.95 | 100/100 | 157.9 MB |
| 32 | HNO | **6.703 ms** | **18.637 ms** | **3,627.35** | 100/100 | **16.7 MB** |
| 32 | Agno | 138.678 ms | 241.744 ms | 170.17 | 100/100 | 373.8 MB |
| 32 | LangGraph | 78.370 ms | 129.618 ms | 241.67 | 100/100 | 161.1 MB |

At concurrency 8, HNO measured 16.6x LangGraph's batch RPS and 33.3x Agno's
under this fixed local protocol. At concurrency 32, the ratios were 15.0x and
21.3x. These are local orchestration and runtime observations, not remote model
speedups.

See the full [local system overhead matrix](/advanced/system-overhead),
raw JSON, resource definitions, and reproduction command.

## Remote model appendix

The remote DeepSeek run is a separate end-to-end snapshot using 100 measured
requests at concurrency 8. It includes network, Provider queueing, and model
generation, so it is not a pure framework benchmark.

| Path | Mean | P95 | Success | Relative mean |
| --- | ---: | ---: | ---: | ---: |
| Direct API | 2,094.52 ms | 4,225.19 ms | 100/100 | 1.00x |
| HNO | **1,312.71 ms** | **1,563.62 ms** | 100/100 | **1.60x** |
| Agno | 1,571.34 ms | 1,988.19 ms | 100/100 | 1.33x |
| LangGraph | 1,362.09 ms | 1,753.45 ms | 100/100 | 1.54x |

The relative mean is `Direct API mean / path mean`. This snapshot supports
parity with a small observed latency advantage, not a universal claim about
remote production performance. See the [remote performance report](/advanced/performance).

## Latest from the HNO Blog

### AI Agent Framework Benchmarks: Why Runtime Overhead Matters More Than Model Latency

[Read the full article](/blog/ai-agent-runtime-benchmark)

When a model or Agent framework becomes a hot topic, separate model latency from
framework overhead before repeating a performance claim. This article uses the
same local OpenAI-compatible stub, 5 warmups, 100 measured operations, and
concurrency 1, 8, and 32.

Read more in the [HNO Blog](/blog/) or subscribe to the [RSS feed](/rss.xml).

## Why Go, why HNO

**Why Go?** Go is the implementation choice for compiled deployment artifacts,
built-in concurrency, static typing, the standard HTTP/JSON library, and
first-party testing and profiling tools. Those are design reasons, not proof of
a fixed speedup for every workload.

**Why HNO?** HNO is the current project name. The repository does not define an
official expansion of the name, so this site does not invent one. The Go module
path remains `github.com/rexleimo/agno-go`; HNO is a project identity, not a
standardized model, protocol, or performance metric.

**Evidence policy:** measured results include the command, versions, environment,
mean, median, and range. Go allocation bytes are not treated as Python memory.
Real LLM, production-capacity, and cross-framework claims require a separate
same-provider, same-workload experiment.
