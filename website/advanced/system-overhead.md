# Local framework and system overhead matrix

<div class="benchmark-page-marker" aria-hidden="true"></div>

<div class="benchmark-status benchmark-status--ready">
  <div class="benchmark-status__eyebrow">Measurement status</div>
  <div class="benchmark-status__title">HNO vs Agno vs LangGraph: local fixed-model matrix complete</div>
  <div class="benchmark-status__badges">
    <span class="benchmark-badge benchmark-badge--ready">Same local stub</span>
    <span class="benchmark-badge">100 samples per row</span>
    <span class="benchmark-badge">Concurrency 1 / 8 / 32</span>
    <span class="benchmark-badge">RSS + CPU telemetry</span>
  </div>
  <p><strong>Result:</strong> after removing remote provider variance, HNO showed lower request overhead, higher measured batch RPS, and a smaller observed process-tree RSS than both Python paths in this fixed one-request workload.</p>
  <p><strong>Boundary:</strong> this is a framework and runtime-overhead measurement against a deterministic local HTTP stub. It is not a model-quality, production-capacity, or universal language-speed claim.</p>
</div>

## Protocol

Recorded: `2026-08-05T02:09:34Z`.

| Condition | Value |
| --- | --- |
| Frameworks | HNO, Agno, LangGraph |
| Model endpoint | Same local OpenAI-compatible stub |
| Model ID | `stub-model` |
| Response | `LOCAL_MODEL_OK` |
| Stub response delay | 1 ms |
| Warmups | 5 per framework/concurrency |
| Measured runs | 100 per framework/concurrency |
| Concurrency | 1, 8, and 32 |
| Environment | Windows AMD64, Go 1.26.4, Python 3.14.5 |
| Lifecycle | Fresh operation: client/model/Agent/Graph setup included per operation |
| Measured metrics | Per-request latency, measured batch RPS, success rate, peak RSS, CPU time, process wall time |

Every framework used the same endpoint and response contract. The matrix is a fresh-operation measurement: setup that belongs to the operation is included consistently rather than silently mixing a shared HNO client with freshly built Python objects. Agno and LangGraph ran in separate processes so their RSS and CPU values were not combined.

## Latency, throughput, and resources

### Concurrency 1

| Framework | Mean ms | P50 ms | P95 ms | Measured RPS | Success | Peak RSS MB | CPU s | Process wall s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| HNO | **1.583** | **1.528** | **1.675** | **631.71** | 100/100 | **12.2** | 0.047 | **0.220** |
| Agno | 41.632 | 36.813 | 61.863 | 24.01 | 100/100 | 204.5 | 5.906 | 6.905 |
| LangGraph | 7.687 | 7.521 | 9.082 | 129.68 | 100/100 | 156.3 | 2.844 | 3.240 |

### Concurrency 8

| Framework | Mean ms | P50 ms | P95 ms | Measured RPS | Success | Peak RSS MB | CPU s | Process wall s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| HNO | **1.859** | **1.596** | **2.655** | **4,186.08** | 100/100 | **12.3** | 0.109 | **0.066** |
| Agno | 61.766 | 59.820 | 83.473 | 125.66 | 100/100 | 285.0 | 6.922 | 3.445 |
| LangGraph | 30.117 | 30.089 | 37.638 | 251.95 | 100/100 | 157.9 | 2.672 | 2.787 |

### Concurrency 32

| Framework | Mean ms | P50 ms | P95 ms | Measured RPS | Success | Peak RSS MB | CPU s | Process wall s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| HNO | **6.703** | **3.091** | **18.637** | **3,627.35** | 100/100 | **16.7** | 0.094 | **0.105** |
| Agno | 138.678 | 129.345 | 241.744 | 170.17 | 100/100 | 373.8 | 7.469 | 3.383 |
| LangGraph | 78.370 | 74.864 | 129.618 | 241.67 | 100/100 | 161.1 | 2.766 | 2.807 |

\* Very short Go process CPU samples can be below the Windows process-time reporting resolution. CPU time is retained as telemetry, while latency, RPS, and RSS are the primary observations for this short matrix.

## What this demonstrates

At concurrency 8, HNO's measured batch RPS was about 16.6x LangGraph and 33.3x Agno in this fixed local workload. At concurrency 32, HNO was about 15.0x LangGraph and 21.3x Agno by measured batch RPS.

The observed peak RSS at concurrency 32 was 16.7 MB for HNO, 161.1 MB for LangGraph, and 373.8 MB for Agno. These are process-tree observations that include runtime/import state, not allocation-per-request numbers.

The result is consistent with HNO's design goal: keep orchestration and HTTP-client overhead small so the application can spend more of its budget on model work. It does not mean HNO will make a remote model generate tokens faster.

## Metric definitions

- **Mean / P50 / P95:** per-request client wall-clock samples. They include framework preparation and the local HTTP request.
- **Measured RPS:** `100 / measured batch elapsed`; warmups are excluded, while framework request work is included.
- **Peak RSS:** maximum resident set observed across the framework process tree, including startup/import overhead.
- **CPU s:** user plus system CPU time observed across the process tree. Very short Windows processes can report coarse CPU values.
- **Process wall s:** includes interpreter or binary startup, warmups, and measured work.

## Limits and fair interpretation

This matrix does not measure:

- model quality or token generation speed;
- remote provider queueing or rate limits;
- tool loops, memory retrieval, streaming, teams, or workflows;
- production RPS under authentication, persistence, TLS, or observability load;
- allocations per request or long-lived heap behavior;
- equivalent Python and Go garbage-collector tuning under every deployment mode;
- warm steady-state invocation with clients, Agents, or graphs created once;

The fair claim is:

> Under the stated local fixed-response protocol, HNO showed lower orchestration latency, higher measured batch RPS, and lower observed process-tree RSS than Agno and LangGraph on this machine.

The claim is not:

> HNO is universally faster than every Python framework or every model provider.

## Raw results and reproduction

Raw per-process samples and the generated report:

```text
benchmarks/framework_comparison/results/local_stub_matrix/latest.json
benchmarks/framework_comparison/results/local_stub_matrix/latest.md
```

The individual raw files are named by framework and concurrency, for example:

```text
hno_simple_c8.json
agno_simple_c8.json
langgraph_simple_c8.json
```

Reproduce with:

```bash
uv run --with psutil --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  --with 'langchain-openai' --with 'langchain-core' \
  python benchmarks/framework_comparison/local_overhead_matrix.py \
  --runs 100 --warmup 5 --concurrencies 1,8,32 --delay-ms 1
```

The remote DeepSeek report remains a separate end-to-end provider snapshot. It should not be mixed with this local framework-overhead matrix.
