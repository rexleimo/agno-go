# DeepSeek 100-run concurrent real-model performance report

<div class="benchmark-page-marker" aria-hidden="true"></div>

<div class="benchmark-status benchmark-status--ready">
  <div class="benchmark-status__eyebrow">Publication status</div>
  <div class="benchmark-status__title">DeepSeek fixed-answer comparison: 100 concurrent runs complete</div>
  <div class="benchmark-status__badges">
    <span class="benchmark-badge benchmark-badge--ready">Real DeepSeek</span>
    <span class="benchmark-badge">100 runs per path</span>
    <span class="benchmark-badge">Concurrency 8</span>
    <span class="benchmark-badge">Provider prefix cache</span>
  </div>
  <p><strong>Result:</strong> in this DeepSeek <code>deepseek-v4-flash</code> fixed-answer workload, HNO had the lowest mean client latency. Relative to Direct API, the mean ratios were <strong>1.60x</strong> for HNO, <strong>1.33x</strong> for Agno, and <strong>1.54x</strong> for LangGraph.</p>
  <p><strong>Boundary:</strong> this is a same-provider, same-model, same-prompt latency snapshot with eight requests in flight. It is not a universal claim about Go, Python, throughput, or production capacity.</p>
</div>

## Local system-overhead summary

The primary framework comparison is a fresh-operation matrix against the same local OpenAI-compatible stub. Each row uses 100 measured operations, 5 warmups, a 1 ms fixed response delay, and the listed concurrency. RSS is process-tree working set; measured RPS is the completed measured batch throughput.

| Concurrency | Framework | Mean | P95 | Measured RPS | Success | Peak RSS |
| ---: | --- | ---: | ---: | ---: | ---: | ---: |
| 8 | HNO | **1.859 ms** | **2.655 ms** | **4,186.08** | 100/100 | **12.3 MB** |
| 8 | Agno | 61.766 ms | 83.473 ms | 125.66 | 100/100 | 285.0 MB |
| 8 | LangGraph | 30.117 ms | 37.638 ms | 251.95 | 100/100 | 157.9 MB |
| 32 | HNO | **6.703 ms** | **18.637 ms** | **3,627.35** | 100/100 | **16.7 MB** |
| 32 | Agno | 138.678 ms | 241.744 ms | 170.17 | 100/100 | 373.8 MB |
| 32 | LangGraph | 78.370 ms | 129.618 ms | 241.67 | 100/100 | 161.1 MB |

At concurrency 8, HNO measured 16.6x LangGraph's batch RPS and 33.3x Agno's in this local protocol. At concurrency 32, the ratios were 15.0x and 21.3x. This is framework/runtime overhead evidence, not remote model speed.

See the [full local system-overhead matrix](/advanced/system-overhead) for raw files, resource definitions, lifecycle scope, and reproduction.

## Final results

Recorded: `2026-08-04T15:33:06Z`. Each path used 3 serial warmups and 100 measured samples with bounded concurrency 8.

| Path | Mean | Median P50 | P95 | Min-max | Success | Relative to Direct |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Direct API | 2,094.52 ms | 1,707.51 ms | 4,225.19 ms | 905.05-4,592.62 ms | 100/100 | 1.00x |
| HNO | **1,312.71 ms** | **1,288.57 ms** | **1,563.62 ms** | 922.35-1,847.16 ms | 100/100 | **1.60x** |
| Agno | 1,571.34 ms | 1,513.03 ms | 1,988.19 ms | 1,150.89-2,646.64 ms | 100/100 | 1.33x |
| LangGraph | 1,362.09 ms | 1,332.45 ms | 1,753.45 ms | 983.37-1,927.02 ms | 100/100 | 1.54x |

### What the relative mean means

The multiplier is defined as:

```text
Direct API mean duration / path mean duration
```

For HNO:

```text
2094.52 / 1312.71 = 1.60x
```

HNO's mean client latency was about 37.3% lower than the Direct API mean in this snapshot. This is an end-to-end client measurement, not a division of Go runtime time by Python runtime time. It includes:

```text
client request preparation + provider network + provider queueing + model generation + response parsing
```

The concurrent protocol changes the interpretation from the earlier serial 1000-run snapshot. The result shows observed latency for eight in-flight requests; it is not a request-per-second benchmark and should not be presented as a pure framework speedup.

## Prompt-cache evidence

DeepSeek prompt caching is provider-managed. There is no client-side `cache=true` switch to force it. The four paths used the same stable long prefix and the same fixed-answer prompt.

The Direct API path reported:

```text
99/100 measured requests: prompt_cache_hit_tokens = 1920, prompt_cache_miss_tokens = 13
1/100 measured requests:  prompt_cache_hit_tokens = 0,    prompt_cache_miss_tokens = 1933
```

The first measured request was a cache miss in this run; later requests observed the cached prefix. Framework SDK results do not expose the same provider usage fields in their current result structures, so the direct path is retained as provider-level cache evidence rather than treated as a framework-specific metric.

## Test protocol

| Condition | Value |
| --- | --- |
| Provider | DeepSeek |
| Model | `deepseek-v4-flash` |
| API | OpenAI-compatible |
| Endpoint | `https://api.deepseek.com/v1` |
| Prompt | Fixed answer, return `REMOTE_MODEL_OK` |
| Temperature | 0 |
| Seed | 42 |
| Max output tokens | 128 |
| Warmup | 3 serial requests per path |
| Measured runs | 100 per path |
| Concurrency | 8 measured requests in flight |
| Prompt cache | Provider-managed prefix cache |
| Key | Read locally and never stored in results |

## Result boundaries

The supported statement is:

> Under the DeepSeek `deepseek-v4-flash`, fixed-answer, stable-prefix, 100-run protocol with concurrency 8, temperature 0, and seed 42, HNO's observed mean client latency was 1.60x relative to Direct API, Agno was 1.33x, and LangGraph was 1.54x.

This does not prove:

- that every Go program is faster than Python;
- that HNO will be 1.60x faster for every model, provider, prompt, tool, or concurrency level;
- that 1.60x is a pure Go runtime improvement;
- that this is production throughput or capacity;
- that this is TTFT or TPS;
- that the result remains stable under rate limits, retries, failures, or a different time window.

The P95 and maximum values show provider/network tail behavior. Production work still needs separate tests for throughput, rate limits, retries, TTFT, TPS, token usage, resource consumption, and cost.

## Raw results

The checked-in concurrent snapshot is:

```text
benchmarks/framework_comparison/results/remote_deepseek_v4_flash/latest_100_concurrent.json
benchmarks/framework_comparison/results/remote_deepseek_v4_flash/latest_100_concurrent.md
```

Raw samples:

```text
benchmarks/framework_comparison/results/remote_deepseek_v4_flash/direct_simple_100_concurrent.json
benchmarks/framework_comparison/results/remote_deepseek_v4_flash/hno_simple_100_concurrent.json
benchmarks/framework_comparison/results/remote_deepseek_v4_flash/python_simple_100_concurrent.json
```

The earlier serial 1000-run files remain in the results directory for historical comparison; they are not used as the conclusion on this page.

## Local framework overhead matrix

For a provider-independent comparison of HNO, Agno, and LangGraph using the same local fixed-response endpoint, see the [local framework and system overhead matrix](/advanced/system-overhead). It reports 100 samples at concurrency 1, 8, and 32, with request latency, measured batch RPS, peak process-tree RSS, and CPU telemetry.

## Reproduce

```bash
python benchmarks/framework_comparison/remote_fixed_baseline.py \
  --config benchmarks/framework_comparison/remote_model.local.env \
  --warmup 3 --runs 100 --concurrency 8 \
  --output benchmarks/framework_comparison/results/remote_deepseek_v4_flash/direct_simple_100_concurrent.json

go run ./benchmarks/framework_comparison/hno_local_runner \
  -config benchmarks/framework_comparison/remote_model.local.env \
  -warmup 3 -runs 100 -concurrency 8 \
  > benchmarks/framework_comparison/results/remote_deepseek_v4_flash/hno_simple_100_concurrent.json

uv run --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  --with 'langchain-openai' --with 'langchain-core' \
  python benchmarks/framework_comparison/real_local.py \
  --config benchmarks/framework_comparison/remote_model.local.env \
  --scenario simple --warmup 3 --runs 100 --concurrency 8 \
  --output benchmarks/framework_comparison/results/remote_deepseek_v4_flash/python_simple_100_concurrent.json

python benchmarks/framework_comparison/summarize_remote.py
```

The local config file is intentionally ignored by Git. Do not paste the API key into source files, reports, logs, or documentation.
