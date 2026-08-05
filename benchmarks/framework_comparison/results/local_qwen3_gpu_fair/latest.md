# Real local-model framework comparison

> Controlled snapshot: llama.cpp + Qwen3-4B-Q8_0.gguf, CUDA `-ngl 99`, prompt cache off, reasoning off, one server slot.

Captured: `2026-08-04T09:44:38.331681+00:00` | Warmup: `3` | Measured runs: `20`

## Fixed answer

| Framework | Mean ms | Median ms | P95 ms | Min-max ms | Success |
| --- | ---: | ---: | ---: | ---: | ---: |
| hno | 120.32 | 104.27 | 162.13 | 94.77-381.00 | 20/20 (100%) |
| agno | 150.55 | 135.91 | 208.99 | 126.01-251.34 | 20/20 (100%) |
| langgraph | 108.69 | 105.48 | 132.67 | 98.95-154.80 | 20/20 (100%) |

## Tool calling

| Framework | Mean ms | Median ms | P95 ms | Min-max ms | Success |
| --- | ---: | ---: | ---: | ---: | ---: |
| hno | 743.76 | 736.27 | 781.35 | 718.92-799.29 | 20/20 (100%) |
| agno | 790.40 | 792.39 | 835.35 | 743.85-840.94 | 20/20 (100%) |
| langgraph | 777.68 | 765.40 | 810.93 | 756.99-818.96 | 20/20 (100%) |

## Two-turn memory

| Framework | Mean ms | Median ms | P95 ms | Min-max ms | Success |
| --- | ---: | ---: | ---: | ---: | ---: |
| hno | 175.27 | 174.64 | 181.82 | 169.36-187.50 | 20/20 (100%) |
| agno | 225.51 | 228.48 | 239.43 | 205.87-247.23 | 20/20 (100%) |
| langgraph | 191.04 | 191.29 | 198.23 | 181.34-203.69 | 20/20 (100%) |

## Interpretation

These numbers are end-to-end client timings for this controlled local model snapshot. They should not be presented as universal Go-vs-Python or production throughput claims.

Raw per-framework samples are stored beside this file. The JSON also records server configuration, runtime versions, Git state, and source hashes.
