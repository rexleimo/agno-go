# DeepSeek fixed-answer concurrent benchmark

Captured: `2026-08-04T15:47:24.214109+00:00` | Warmup: `3` | Measured runs: `100` | Concurrency: `8`
Model: `deepseek-v4-flash` | Endpoint: `https://api.deepseek.com`

All paths used the same remote model and fixed-answer prompt. Each path measured 100 samples with bounded concurrency 8.

| Path | Mean ms | Median ms | P95 ms | Min-max ms | Success | Relative mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Direct | 2094.52 | 1707.51 | 4225.19 | 905.05-4592.62 | 100/100 (100%) | 1.00x |
| Hno | 1312.71 | 1288.57 | 1563.62 | 922.35-1847.16 | 100/100 (100%) | 1.60x |
| Agno | 1571.34 | 1513.03 | 1988.19 | 1150.89-2646.64 | 100/100 (100%) | 1.33x |
| Langgraph | 1362.09 | 1332.45 | 1753.45 | 983.37-1927.02 | 100/100 (100%) | 1.54x |

`Relative mean` is `Direct API mean / path mean`; it is not a pure framework runtime speedup.
Concurrent client latency includes provider/network/queueing effects and must not be interpreted as RPS.

## Raw inputs

- `direct_simple_100_concurrent.json`
- `hno_simple_100_concurrent.json`
- `python_simple_100_concurrent.json`

## Limits

- All paths use the same provider, model, prompt, temperature, seed, and output limit.
- Warmups are serial; measured requests use bounded concurrency.
- The result is a latency snapshot, not a throughput, TTFT, TPS, or cost benchmark.
- Provider queueing, rate limits, and network conditions are included in client wall-clock time.
