# Local fixed-model overhead matrix

Captured: `2026-08-05T02:09:34.855179+00:00`
Model: `stub-model` | Stub delay: `1.0 ms`
Measured runs: `100` per framework/concurrency | Warmup: `5`
Lifecycle: `fresh operation: client/model/Agent/Graph setup is included per operation`
Environment: `Windows AMD64` | Python `3.14.5` | `go version go1.26.4 windows/amd64`

All frameworks used the same local OpenAI-compatible stub and returned the same fixed response. The measured latency excludes warmup but includes each framework's request preparation and HTTP client path.

## Concurrency 1

| Framework | Mean ms | P50 ms | P95 ms | Measured RPS | Success | Peak RSS MB | CPU s | Process wall s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| HNO | 1.583 | 1.528 | 1.675 | 631.71 | 100/100 | 12.2 | 0.047 | 0.220 |
| Agno | 41.632 | 36.813 | 61.863 | 24.01 | 100/100 | 204.5 | 5.906 | 6.905 |
| Langgraph | 7.687 | 7.521 | 9.082 | 129.68 | 100/100 | 156.3 | 2.844 | 3.240 |

## Concurrency 8

| Framework | Mean ms | P50 ms | P95 ms | Measured RPS | Success | Peak RSS MB | CPU s | Process wall s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| HNO | 1.859 | 1.596 | 2.655 | 4186.08 | 100/100 | 12.3 | 0.109 | 0.066 |
| Agno | 61.766 | 59.820 | 83.473 | 125.66 | 100/100 | 285.0 | 6.922 | 3.445 |
| Langgraph | 30.117 | 30.089 | 37.638 | 251.95 | 100/100 | 157.9 | 2.672 | 2.787 |

## Concurrency 32

| Framework | Mean ms | P50 ms | P95 ms | Measured RPS | Success | Peak RSS MB | CPU s | Process wall s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| HNO | 6.703 | 3.091 | 18.637 | 3627.35 | 100/100 | 16.7 | 0.094 | 0.105 |
| Agno | 138.678 | 129.345 | 241.744 | 170.17 | 100/100 | 373.8 | 7.469 | 3.383 |
| Langgraph | 78.370 | 74.864 | 129.618 | 241.67 | 100/100 | 161.1 | 2.766 | 2.807 |

## Resource metric definitions

- `Peak RSS MB` is the maximum resident set observed for the framework process tree, including interpreter/runtime startup.
- `CPU s` is user plus system CPU time observed for that process tree.
- `Process wall s` includes process startup, warmups, and measured requests; `Measured RPS` uses only the measured batch elapsed time.
- The stub is shared infrastructure and is not attributed to any framework.

## Limits

- The local stub removes remote provider variance but is not a model-quality or provider benchmark.
- The same HTTP endpoint and fixed response are used by every framework.
- Peak RSS includes runtime and import overhead; it is not an allocation-per-request measurement.
- Agno and LangGraph are run in separate processes so their RSS and CPU values are not combined.
- This measures the simple one-request scenario; tool loops, memory, streaming, and teams need separate rows.
