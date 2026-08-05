# 本地框架与系统开销对比矩阵

<div class="benchmark-page-marker" aria-hidden="true"></div>

<div class="benchmark-status benchmark-status--ready">
  <div class="benchmark-status__eyebrow">测量状态</div>
  <div class="benchmark-status__title">HNO、Agno、LangGraph：本地固定模型矩阵已完成</div>
  <div class="benchmark-status__badges">
    <span class="benchmark-badge benchmark-badge--ready">同一个本地 Stub</span>
    <span class="benchmark-badge">每行 100 次</span>
    <span class="benchmark-badge">并发度 1 / 8 / 32</span>
    <span class="benchmark-badge">RSS + CPU 采样</span>
  </div>
  <p><strong>结果：</strong>去掉远程 Provider 波动后，在本次固定单请求场景中，HNO 的请求开销更低、测得的批次 RPS 更高，观测到的进程树 RSS 也低于两个 Python 对照。</p>
  <p><strong>边界：</strong>这是针对确定性本地 HTTP Stub 的框架和运行时开销测量，不是模型质量、生产容量或“某种语言永远更快”的通用结论。</p>
</div>

## 测试协议

记录时间：`2026-08-05T02:09:34Z`。

| 条件 | 值 |
| --- | --- |
| Framework | HNO、Agno、LangGraph |
| Model Endpoint | 同一个本地 OpenAI-compatible Stub |
| Model ID | `stub-model` |
| 返回内容 | `LOCAL_MODEL_OK` |
| Stub 响应延迟 | 1 ms |
| Warmup | 每个框架/并发档位 5 次 |
| Measured runs | 每个框架/并发档位 100 次 |
| Concurrency | 1、8、32 |
| 环境 | Windows AMD64、Go 1.26.4、Python 3.14.5 |
| Lifecycle | Fresh operation：每次操作包含 client/model/Agent/Graph 初始化 |
| 指标 | 请求延迟、批次 RPS、成功率、峰值 RSS、CPU 时间、进程总耗时 |

所有框架使用相同的 Endpoint 和响应协议。本矩阵是 fresh operation 测量：每次操作所属的初始化开销都一致纳入，不把复用的 HNO client 和每次新建的 Python 对象混在一起。Agno 与 LangGraph 分别在独立进程中运行，因此 RSS 和 CPU 没有混在一起。

## 延迟、吞吐和资源

### 并发度 1

| Framework | 平均 ms | P50 ms | P95 ms | 测得 RPS | 成功率 | 峰值 RSS MB | CPU s | 进程 wall s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| HNO | **1.583** | **1.528** | **1.675** | **631.71** | 100/100 | **12.2** | 0.047 | **0.220** |
| Agno | 41.632 | 36.813 | 61.863 | 24.01 | 100/100 | 204.5 | 5.906 | 6.905 |
| LangGraph | 7.687 | 7.521 | 9.082 | 129.68 | 100/100 | 156.3 | 2.844 | 3.240 |

### 并发度 8

| Framework | 平均 ms | P50 ms | P95 ms | 测得 RPS | 成功率 | 峰值 RSS MB | CPU s | 进程 wall s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| HNO | **1.859** | **1.596** | **2.655** | **4,186.08** | 100/100 | **12.3** | 0.109 | **0.066** |
| Agno | 61.766 | 59.820 | 83.473 | 125.66 | 100/100 | 285.0 | 6.922 | 3.445 |
| LangGraph | 30.117 | 30.089 | 37.638 | 251.95 | 100/100 | 157.9 | 2.672 | 2.787 |

### 并发度 32

| Framework | 平均 ms | P50 ms | P95 ms | 测得 RPS | 成功率 | 峰值 RSS MB | CPU s | 进程 wall s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| HNO | **6.703** | **3.091** | **18.637** | **3,627.35** | 100/100 | **16.7** | 0.094 | **0.105** |
| Agno | 138.678 | 129.345 | 241.744 | 170.17 | 100/100 | 373.8 | 7.469 | 3.383 |
| LangGraph | 78.370 | 74.864 | 129.618 | 241.67 | 100/100 | 161.1 | 2.766 | 2.807 |

\* HNO 进程很短，Windows 进程 CPU 时间有较粗粒度；CPU 只作为辅助采样，主要结论看延迟、RPS 和 RSS。

## 这组数据说明什么

在并发度 8 下，HNO 的测得批次 RPS 约为 LangGraph 的 **16.6 倍**、Agno 的 **33.3 倍**。并发度 32 下，HNO 约为 LangGraph 的 **15.0 倍**、Agno 的 **21.3 倍**。

并发度 32 时，观测到的峰值 RSS 为：

```text
HNO        16.7 MB
LangGraph 161.1 MB
Agno       373.8 MB
```

这些是包含运行时和 import 状态的进程树观测值，不是每个请求的分配量。

这与 HNO 的设计目标一致：尽量减少编排和 HTTP 客户端开销，把预算留给真正的模型工作。但它不表示 HNO 能让远程模型本身更快生成 Token。

## 指标定义

- **平均 / P50 / P95：** 每次请求的客户端 wall-clock 样本，包含框架准备和本地 HTTP 请求。
- **测得 RPS：** `100 / 正式测量批次耗时`，不包含预热，但包含框架请求路径。
- **峰值 RSS：** 框架进程树的最大常驻内存，包括启动和 import 开销。
- **CPU s：** 进程树观测到的 user + system CPU 时间；Windows 短进程可能有粗粒度误差。
- **进程 wall s：** 包含解释器或二进制启动、预热和正式测量。

## 限制和正确解读

本矩阵没有测量：

- 模型质量和 Token 生成速度；
- 远程 Provider 排队和限流；
- 工具循环、Memory、Streaming、Team、Workflow；
- 带认证、持久化、TLS、观测开销的生产 RPS；
- 每请求分配量或长期堆行为；
- 所有部署模式下完全等价的 Python/Go GC 调优；
- client、Agent、Graph 只创建一次的 warm steady-state 调用。

这组数据可以支持的表述是：

> 在声明的本地固定响应协议下，HNO 在这台机器上比 Agno 和 LangGraph 观测到更低的编排延迟、更高的测得批次 RPS 和更低的进程树 RSS。

不能把它写成：

> HNO 在所有模型、所有 Provider、所有 Python 框架和所有生产环境中都更快。

## 原始结果与复现

原始样本和汇总报告：

```text
benchmarks/framework_comparison/results/local_stub_matrix/latest.json
benchmarks/framework_comparison/results/local_stub_matrix/latest.md
```

单个原始文件按框架和并发度命名，例如：

```text
hno_simple_c8.json
agno_simple_c8.json
langgraph_simple_c8.json
```

复现命令：

```bash
uv run --with psutil --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  --with 'langchain-openai' --with 'langchain-core' \
  python benchmarks/framework_comparison/local_overhead_matrix.py \
  --runs 100 --warmup 5 --concurrencies 1,8,32 --delay-ms 1
```

远程 DeepSeek 报告仍然是独立的端到端 Provider 快照，不能和这份本地框架开销矩阵混成一个结论。
