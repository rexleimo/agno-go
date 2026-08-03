# 可观测性与遥测

HNO 提供多层可观测性，便于在生产环境监控推理过程、token 消耗与长会话状态。

## AgentOS SSE 事件流

`POST /api/v1/agents/{id}/run/stream` 返回 Server-Sent Events (SSE)。可通过 `types` 查询参数过滤事件（例如 `types=run_start,reasoning,token,complete`）。

### 事件类型

| 事件 | 说明 |
| --- | --- |
| `run_start` | 输入内容及会话元数据。 |
| `token` | 模型流式输出的 token。 |
| `tool_call` | 工具调用名称、参数、结果。 |
| `reasoning` | 推理片段，包含思考内容、token 统计、脱敏文本、模型/提供商信息。 |
| `complete` | 最终输出、运行耗时、token 汇总（包含 reasoning token）。 |
| `error` | 结构化错误信息。 |

当模型返回 `ReasoningContent` 时会自动触发 `reasoning` 事件，已适配 OpenAI o1/o3/o4、Gemini 2.5 Thinking、开启 `thinking` 的 Claude。

## Logfire 集成

示例 `cmd/examples/logfire_observability` 展示如何通过 OpenTelemetry 向 Logfire 输出追踪数据：

1. 配置 OTLP 端点与写入令牌（`LOGFIRE_WRITE_TOKEN`、`LOGFIRE_OTLP_ENDPOINT`）。
2. 使用 `logfire` 构建标签运行示例：
   ```bash
   go run -tags logfire cmd/examples/logfire_observability/main.go
   ```
3. 示例会记录循环次数、token 统计以及推理文本（包括可选的脱敏内容）。

详细步骤参见 [`docs/release/logfire_observability.md`](https://github.com/rexleimo/HNO/blob/main/docs/release/logfire_observability.md)。

## 下一步

- 将 SSE 事件转发至自有监控平台（Logfire、Elastic、Datadog 等）。
- 基于 reasoning token 统计构建成本看板。
- 借助 OpenTelemetry 钩子为工具执行或终端用户请求补充追踪信息。
