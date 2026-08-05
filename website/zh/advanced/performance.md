# DeepSeek 100 次并发真实模型性能报告

<div class="benchmark-page-marker" aria-hidden="true"></div>

<div class="benchmark-status benchmark-status--ready">
  <div class="benchmark-status__eyebrow">发布状态</div>
  <div class="benchmark-status__title">DeepSeek 固定问答：100 次并发对照已完成</div>
  <div class="benchmark-status__badges">
    <span class="benchmark-badge benchmark-badge--ready">真实 DeepSeek</span>
    <span class="benchmark-badge">每条路径 100 次</span>
    <span class="benchmark-badge">并发度 8</span>
    <span class="benchmark-badge">Provider 前缀缓存</span>
  </div>
  <p><strong>核心结果：</strong>在本次 DeepSeek <code>deepseek-v4-flash</code> 固定问答场景中，HNO 的平均客户端延迟最低。相对于 Direct API，HNO、Agno、LangGraph 的平均倍率分别为 <strong>1.60x</strong>、<strong>1.33x</strong> 和 <strong>1.54x</strong>。</p>
  <p><strong>边界：</strong>这是同一个 Provider、同一个模型、同一个 Prompt，在 8 个请求同时进行时得到的延迟快照，不是 Go、Python、吞吐量或生产容量的通用结论。</p>
</div>

## 本地系统开销摘要

主框架对比使用同一个本地 OpenAI-compatible Stub，并采用 fresh operation 生命周期。每行包含 100 次正式操作、5 次预热、1 ms 固定响应延迟和对应并发度。RSS 是进程树工作集；测得 RPS 是正式测量批次的完成吞吐。

| 并发度 | Framework | 平均 | P95 | 测得 RPS | 成功率 | 峰值 RSS |
| ---: | --- | ---: | ---: | ---: | ---: | ---: |
| 8 | HNO | **1.859 ms** | **2.655 ms** | **4,186.08** | 100/100 | **12.3 MB** |
| 8 | Agno | 61.766 ms | 83.473 ms | 125.66 | 100/100 | 285.0 MB |
| 8 | LangGraph | 30.117 ms | 37.638 ms | 251.95 | 100/100 | 157.9 MB |
| 32 | HNO | **6.703 ms** | **18.637 ms** | **3,627.35** | 100/100 | **16.7 MB** |
| 32 | Agno | 138.678 ms | 241.744 ms | 170.17 | 100/100 | 373.8 MB |
| 32 | LangGraph | 78.370 ms | 129.618 ms | 241.67 | 100/100 | 161.1 MB |

并发度 8 下，HNO 在这个本地协议中的测得批次 RPS 约为 LangGraph 的 16.6 倍、Agno 的 33.3 倍；并发度 32 下约为 15.0 倍和 21.3 倍。这是框架和运行时开销证据，不是远程模型加速。

详见[完整本地系统开销矩阵](/zh/advanced/system-overhead)，其中包含原始文件、资源指标定义、生命周期口径和复现命令。

## 最终结果

记录时间：`2026-08-04T15:33:06Z`。每条路径均使用串行预热 3 次，正式测量 100 次，并发度为 8。

| 路径 | 平均 | 中位数 P50 | P95 | 最小-最大 | 成功率 | 相对 Direct |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Direct API | 2,094.52 ms | 1,707.51 ms | 4,225.19 ms | 905.05-4,592.62 ms | 100/100 | 1.00x |
| HNO | **1,312.71 ms** | **1,288.57 ms** | **1,563.62 ms** | 922.35-1,847.16 ms | 100/100 | **1.60x** |
| Agno | 1,571.34 ms | 1,513.03 ms | 1,988.19 ms | 1,150.89-2,646.64 ms | 100/100 | 1.33x |
| LangGraph | 1,362.09 ms | 1,332.45 ms | 1,753.45 ms | 983.37-1,927.02 ms | 100/100 | 1.54x |

### 如何解释相对倍率

本报告的倍率定义为：

```text
Direct API 平均耗时 / 路径平均耗时
```

以 HNO 为例：

```text
2094.52 / 1312.71 = 1.60x
```

在这次快照中，HNO 的平均客户端延迟比 Direct API 低约 37.3%。这是完整客户端链路的测量，不是把 Go 运行时间与 Python 运行时间直接相除。测量包含：

```text
客户端请求准备 + Provider 网络 + Provider 排队 + 模型生成 + 响应解析
```

并发协议与之前的串行 1000 次快照不同。本次结果表示 8 个请求同时在途时观察到的延迟，不是每秒请求数（RPS）测试，也不能被描述成纯框架运行时加速。

## Prompt Cache 证据

DeepSeek 的 Prompt Cache 由 Provider 自动管理，客户端没有可以强制打开的 `cache=true` 开关。四条路径使用了相同的稳定长前缀和固定问答 Prompt。

Direct API 路径返回：

```text
99/100 个正式请求：prompt_cache_hit_tokens = 1920，prompt_cache_miss_tokens = 13
1/100 个正式请求： prompt_cache_hit_tokens = 0，   prompt_cache_miss_tokens = 1933
```

本次测试中第一个正式请求未命中缓存，后续请求观察到了前缀缓存。当前框架 SDK 的 Runner 结果结构没有暴露相同的 Provider usage 字段，因此 Direct API 数据只作为 Provider 级缓存证据保存，不把它误认为框架独有指标。

## 测试协议

| 条件 | 值 |
| --- | --- |
| Provider | DeepSeek |
| Model | `deepseek-v4-flash` |
| API | OpenAI-compatible |
| Endpoint | `https://api.deepseek.com/v1` |
| Prompt | 固定问答，返回 `REMOTE_MODEL_OK` |
| Temperature | 0 |
| Seed | 42 |
| Max output tokens | 128 |
| Warmup | 每条路径串行 3 次 |
| Measured runs | 每条路径 100 次 |
| Concurrency | 8 个正式请求同时在途 |
| Prompt Cache | Provider 自动前缀缓存 |
| Key | 只从本地读取，不写入结果 |

## 结果边界

这组结果可以支持的表述是：

> 在 DeepSeek `deepseek-v4-flash`、固定问答、稳定前缀、并发度 8、正式 100 次、temperature 0 和 seed 42 的协议下，HNO 的观测平均客户端延迟相对于 Direct API 为 1.60x，Agno 为 1.33x，LangGraph 为 1.54x。

这组结果不能支持：

- 所有 Go 程序都比 Python 快；
- HNO 在所有模型、Provider、Prompt、工具和并发条件下都能达到 1.60x；
- 1.60x 等于纯 Go 语言运行时的性能提升；
- 这就是生产吞吐量或容量；
- 这就是 TTFT 或 TPS；
- 在限流、重试、失败请求或其他时间窗口下仍然保持相同结果。

P95 和最大值显示 Provider、网络和排队存在长尾。因此生产环境还需要单独测试吞吐量、限流、重试、TTFT、TPS、Token 使用量、资源消耗和成本。

## 原始结果

本次并发快照为：

```text
benchmarks/framework_comparison/results/remote_deepseek_v4_flash/latest_100_concurrent.json
benchmarks/framework_comparison/results/remote_deepseek_v4_flash/latest_100_concurrent.md
```

原始样本：

```text
benchmarks/framework_comparison/results/remote_deepseek_v4_flash/direct_simple_100_concurrent.json
benchmarks/framework_comparison/results/remote_deepseek_v4_flash/hno_simple_100_concurrent.json
benchmarks/framework_comparison/results/remote_deepseek_v4_flash/python_simple_100_concurrent.json
```

之前的串行 1000 次文件仍保留在 results 目录中用于历史对照，但本页面的结论不再使用它们。

## 本地框架开销矩阵

如果要看不受 Provider 影响的 HNO、Agno、LangGraph 对比，请看[本地框架与系统开销矩阵](/zh/advanced/system-overhead)。该页面使用同一个本地固定响应 Endpoint，测量并发度 1、8、32 下的 100 次请求、延迟、测得批次 RPS、进程树 RSS 和 CPU 采样。

## 复现

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

本地配置文件已被 Git 忽略。不要把 API Key 写入源码、结果、日志或文档。
