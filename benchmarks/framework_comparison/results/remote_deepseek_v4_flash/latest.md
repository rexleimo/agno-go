# Real DeepSeek remote-model framework comparison

> Provider snapshot: DeepSeek `deepseek-v4-flash`, OpenAI-compatible API, serial execution, temperature 0, seed 42.

Captured: `2026-08-04T12:08:58.973090+00:00` | Warmup: `3` | Measured runs: `20`

## Fixed answer

| Framework | Mean ms | Median ms | P95 ms | Min-max ms | Success |
| --- | ---: | ---: | ---: | ---: | ---: |
| hno | 828.07 | 792.02 | 1043.96 | 505.37-1777.28 | 20/20 (100%) |
| agno | 3676.96 | 1060.51 | 19947.06 | 769.76-23823.89 | 20/20 (100%) |
| langgraph | 5481.74 | 1046.80 | 22450.54 | 637.34-39424.95 | 20/20 (100%) |

## Tool calling

| Framework | Mean ms | Median ms | P95 ms | Min-max ms | Success |
| --- | ---: | ---: | ---: | ---: | ---: |
| hno | 2486.96 | 2446.69 | 3163.28 | 1999.71-3286.22 | 20/20 (100%) |
| agno | 2337.33 | 2330.56 | 2814.08 | 1611.69-3502.25 | 20/20 (100%) |
| langgraph | 2325.97 | 2308.79 | 2875.77 | 1666.68-2951.96 | 20/20 (100%) |

## Two-turn memory

| Framework | Mean ms | Median ms | P95 ms | Min-max ms | Success |
| --- | ---: | ---: | ---: | ---: | ---: |
| hno | 3836.46 | 2596.46 | 8435.73 | 1895.30-18221.08 | 17/20 (85%) |
| agno | 6213.28 | 2304.77 | 8668.19 | 1944.89-77246.03 | 20/20 (100%) |
| langgraph | 3854.66 | 2878.31 | 9332.86 | 2110.58-9388.79 | 20/20 (100%) |

## Interpretation

This is a real external-model snapshot. The latency is end-to-end provider latency, not a universal ranking or isolated framework overhead measurement. HNO memory includes three empty responses and must not be advertised as 20/20.

Raw per-framework samples are stored in this directory. The API key is not stored.
