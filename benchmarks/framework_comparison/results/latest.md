# Framework comparison snapshot

Recorded: `2026-08-04T04:32:17.955276+00:00`
Revision: `60778ef0690c`
Worktree: `dirty` (source hashes are in `latest.json`)
Platform: `Windows AMD64`
CPU: `12th Gen Intel(R) Core(TM) i5-12400F`
Go: `go version go1.26.4 windows/amd64`
Python: `3.11.15`
Packages: `agno==2.8.6`, `langgraph==1.2.10`

All measurements use local dummy objects. No LLM, network, database, or token generation is involved.

## HNO and Agno: minimal Agent construction

| Operation | Framework | Mean ns/op | Median ns/op | Min-max ns/op |
| --- | --- | ---: | ---: | ---: |
| Agent creation | HNO | 257.8 | 256.3 | 247.7-275.0 |
| Agent creation | Agno | 5712.6 | 5580.0 | 5251.6-6951.2 |
| Agent creation with one tool | HNO | 291.4 | 287.5 | 277.8-310.7 |
| Agent creation with one tool | Agno | 5993.1 | 5868.6 | 5466.8-7661.3 |

Within this construction-only workload, the Agno/HNO mean ratio is **22.2x** for the minimal Agent and **20.6x** with one tool. This ratio is not an end-to-end agent or service speedup.

## HNO and Agno: fresh local dummy run

This operation constructs a fresh Agent and performs one `ping` run with a fixed local response.

| Framework | Mean ns/op | Median ns/op | Min-max ns/op |
| --- | ---: | ---: | ---: |
| HNO | 6402.2 | 6130.0 | 5549.0-7810.0 |
| Agno | 179683.5 | 176684.0 | 159056.0-209911.0 |

Within this fixed local run, the Agno/HNO mean ratio is **28.1x**. It excludes real model, network, database, and token-generation latency.

HNO's Go `B/op` and Python's `timeit` time are not memory equivalents. The HNO benchmark uses the checked-in `MockModel`; Agno uses a local `Model` subclass with the same no-network intent.

## LangGraph: separate operation

LangGraph is a graph/orchestration library rather than a direct Agent object equivalent. Its result below is therefore reported separately:

- Minimal `StateGraph` build + compile median: **321801.0 ns/op**
- Observed range: **302730.0-385255.0 ns/op**

Do not turn these values into a universal framework ranking. The next fair comparison would need the same application semantics, tool loop, state, and output contract in every framework.

## Reproduce

```bash
uv run --with 'agno==2.8.6' --with 'langgraph==1.2.10' python benchmarks/framework_comparison/compare.py
```
