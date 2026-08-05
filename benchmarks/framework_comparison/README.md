# Cross-framework benchmark

This directory contains a reproducible, deliberately scoped comparison between
HNO (Go), Agno (Python), and LangGraph (Python).

## What is compared

- **HNO vs Agno Agent construction**: create a minimal Agent with a local dummy
  model, and create one with a single local `add` tool.
- **HNO vs Agno fresh local run**: construct a fresh Agent and run `ping` once
  against a fixed local response.
- **LangGraph**: build and compile a minimal `StateGraph`. LangGraph does not
  expose the same Agent object operation, so this result is reported separately
  and is not used as an Agent-construction ranking.

No test calls an LLM, network service, database, or token generator.

## Versions used for the current snapshot

```text
agno==2.8.6
langgraph==1.2.10
```

The exact Python dependency environment, Go version, revision, worktree state,
source hashes, raw samples, means, medians, and ranges are in:

- `results/latest.json`
- `results/latest.md`

The generated files are a measurement snapshot, not a guarantee for other
machines or future versions.

## Reproduce

From the repository root, use `uv` to create an isolated environment and run
without modifying the project's Go or Node dependencies:

```bash
uv run --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  python benchmarks/framework_comparison/compare.py --repeat 20 --number 100
```

The script also runs the checked-in Go benchmarks:

```bash
go test -run='^$' -bench='BenchmarkAgentCreation|BenchmarkAgentFreshRunLocalDummy' \
  -benchmem -count=10 ./pkg/hno/agent/
```

On Windows, run the commands from the repository root. `uv` must be installed;
Python 3.11 was used for the published snapshot.

## Interpretation rules

- Report mean, median, and min/max together; do not publish a single favorable
  run as a universal ratio.
- Go `B/op` and Python `timeit` duration are different measurement systems and
  must not be presented as equivalent memory usage.
- The local dummy run measures framework orchestration overhead only. It says
  nothing about provider latency, model quality, tokens per second, or production
  throughput.
- A fair end-to-end comparison still needs the same model provider, prompt,
  tools, output contract, concurrency, timeout, and hardware in every framework.

## Real remote fixed-answer run

The fixed-answer remote benchmark uses bounded concurrency so the measured
request count stays exact while several requests are in flight:

```bash
python benchmarks/framework_comparison/remote_fixed_baseline.py \
  --runs 100 --concurrency 8 \
  --output benchmarks/framework_comparison/results/remote_deepseek_v4_flash/direct_simple_100_concurrent.json

go run ./benchmarks/framework_comparison/hno_local_runner \
  -config benchmarks/framework_comparison/remote_model.local.env \
  -runs 100 -concurrency 8 \
  > benchmarks/framework_comparison/results/remote_deepseek_v4_flash/hno_simple_100_concurrent.json
```

`real_local.py` accepts `--config benchmarks/framework_comparison/remote_model.local.env`
plus the same `--runs 100 --concurrency 8` arguments for the Agno and LangGraph
paths. Warmups remain serial; measured samples are bounded by the requested
concurrency. The resulting latency is client wall-clock time and includes
provider/network/queueing effects, so it must not be compared with the local
dummy benchmark as a runtime-only speedup.

After the three raw files exist, generate the checked-in comparison snapshot:

```bash
python benchmarks/framework_comparison/summarize_remote.py
```

This writes `latest_100_concurrent.json` and `latest_100_concurrent.md` without
including the API key.

## AgentOS HTTP service probe

With the example server running on port 8080, measure the HTTP layer without
calling a model:

```bash
python benchmarks/framework_comparison/http_service_probe.py \
  --url http://127.0.0.1:8080/health --runs 100 --concurrency 8
```

The probe measures client wall-clock time and therefore includes the Python
HTTP client and connection overhead. It is useful for separating service
routing cost from remote model latency; it is not a server-only CPU benchmark.

## Local framework overhead matrix

For a provider-independent comparison, run all three paths against the same
deterministic local OpenAI-compatible stub (default response delay 1 ms). This records request latency,
measured RPS, peak RSS, CPU time, and process wall time at concurrency 1, 8,
and 32. Each row uses 100 measured requests and 5 warmups:

```bash
uv run --with psutil --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  --with 'langchain-openai' --with 'langchain-core' \
  python benchmarks/framework_comparison/local_overhead_matrix.py
```

Raw per-process samples and the report are written to
`benchmarks/framework_comparison/results/local_stub_matrix/`. The stub removes
remote provider variance; it does not measure model quality, tool loops,
streaming, or production capacity. RSS includes process-tree runtime/import overhead and is
reported separately from per-request allocations. The matrix uses a fresh-operation lifecycle for all three paths; a warm steady-state matrix is a separate protocol.
