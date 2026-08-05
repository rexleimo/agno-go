# DeepSeek 1000-run cached benchmark

> Completed fixed-answer comparison: Direct API, HNO, Agno, and LangGraph. All runs were serial, used the same `deepseek-v4-flash` model, and used a stable prompt prefix to exercise provider-side prefix caching.

Captured: `2026-08-04T14:40:09.405669+00:00` | Warmup: `3` | Measured runs: `1000`

## Final results

| Path | Mean ms | Median ms | P95 ms | Min-max ms | Success | Speedup vs Direct |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Direct API | 1466.32 | 1421.29 | 1820.42 | 671.77-41608.16 | 1000/1000 (100%) | 1.00x |
| HNO | 1147.70 | 1109.80 | 1657.85 | 592.25-5438.43 | 1000/1000 (100%) | 1.28x |
| Agno | 1560.63 | 1570.97 | 2076.35 | 804.28-5576.80 | 1000/1000 (100%) | 0.94x |
| LangGraph | 1728.53 | 1728.29 | 2473.98 | 672.50-4066.40 | 1000/1000 (100%) | 0.85x |

## Interpretation

HNO mean latency was `1.28x` the Direct API baseline, equivalent to a `-27.8%` lower mean in this controlled snapshot. Agno and LangGraph were `1.06x` and `1.18x` slower than HNO by mean latency comparison, respectively.

The speedup is an end-to-end result for this exact DeepSeek endpoint and workload. It must not be generalized to all models, providers, prompts, or production traffic.

## Prompt cache evidence

The direct baseline reported the following usage on all 1000 measured requests:

```text
prompt_cache_hit_tokens: 1920
prompt_cache_miss_tokens: 13
```

The cache is provider-managed and automatic; there is no client-side `cache=true` switch in this API.

## Raw files

```text
direct_simple_cache_1000.json
hno_simple_cache_1000.json
python_simple_cache_1000.json
latest_cache.json
```
