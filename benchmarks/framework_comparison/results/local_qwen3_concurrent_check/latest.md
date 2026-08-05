# Framework comparison snapshot

Recorded: `2026-08-04T16:09:01.467861+00:00`
Revision: `60778ef0690c`
Worktree: `dirty` (source hashes are in `latest.json`)
Platform: `Windows AMD64`
CPU: `12th Gen Intel(R) Core(TM) i5-12400F`
Go: `go version go1.26.4 windows/amd64`
Python: `3.14.5`
Packages: `agno==2.8.6`, `langgraph==1.2.10`

All measurements use local dummy objects. No LLM, network, database, or token generation is involved.

## HNO and Agno: minimal Agent construction

| Operation | Framework | Mean ns/op | Median ns/op | Min-max ns/op |
| --- | --- | ---: | ---: | ---: |
| Agent creation | HNO | 250.7 | 250.1 | 243.8-263.1 |
| Agent creation | Agno | 6896.4 | 5850.0 | 5660.0-12912.0 |
| Agent creation with one tool | HNO | 267.3 | 267.8 | 258.8-273.1 |
| Agent creation with one tool | Agno | 6056.1 | 6062.0 | 5913.0-6243.0 |

Within this construction-only workload, the Agno/HNO mean ratio is **27.5x** for the minimal Agent and **22.7x** with one tool. This ratio is not an end-to-end agent or service speedup.

## HNO and Agno: fresh local dummy run

This operation constructs a fresh Agent and performs one `ping` run with a fixed local response.

| Framework | Mean ns/op | Median ns/op | Min-max ns/op |
| --- | ---: | ---: | ---: |
| HNO | 5638.7 | 5593.5 | 5211.0-6304.0 |
| Agno | 165342.6 | 160501.0 | 154214.0-208476.0 |

Within this fixed local run, the Agno/HNO mean ratio is **29.3x**. It excludes real model, network, database, and token-generation latency.

HNO's Go `B/op` and Python's `timeit` time are not memory equivalents. The HNO benchmark uses the checked-in `MockModel`; Agno uses a local `Model` subclass with the same no-network intent.

## LangGraph: separate operation

LangGraph is a graph/orchestration library rather than a direct Agent object equivalent. Its result below is therefore reported separately:

- Minimal `StateGraph` build + compile median: **361943.5 ns/op**
- Observed range: **353275.0-399506.0 ns/op**

Do not turn these values into a universal framework ranking. The next fair comparison would need the same application semantics, tool loop, state, and output contract in every framework.

## Reproduce

```bash
uv run --with 'agno==2.8.6' --with 'langgraph==1.2.10' python benchmarks/framework_comparison/compare.py
```
