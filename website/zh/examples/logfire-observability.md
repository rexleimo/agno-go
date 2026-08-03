# Logfire 可观测性示例

该示例演示如何使用 OpenTelemetry 为 HNO 智能体接入 [Logfire](https://logfire.dev)，以便在观测平台中查看推理内容、token 消耗以及工具执行数据。

## 前置条件

- Go 1.21+
- OpenAI API Key（或任何支持 reasoning 的模型提供商）
- Logfire 写入令牌：`LOGFIRE_WRITE_TOKEN`
- 可选：自定义 OTLP 端点 `LOGFIRE_OTLP_ENDPOINT`（默认 `logfire-eu.pydantic.dev`）

## 运行示例

```bash
export OPENAI_API_KEY=sk-your-key
export LOGFIRE_WRITE_TOKEN=lf_your_token
go run -tags logfire cmd/examples/logfire_observability/main.go
```

> **提示**  
> 示例通过 `logfire` 构建标签隔离 OpenTelemetry 依赖，常规构建不会引入额外包。

## 核心步骤

1. 使用 OTLP/HTTP 导出器配置 TLS 及写入令牌。
2. 运行具备 reasoning 能力的智能体（默认 OpenAI o1 preview）。
3. 记录 span 属性：运行时长、循环次数、token 统计。
4. 发送 `reasoning.complete` 事件，包含思考内容与脱敏文本。

Logfire 中会看到类似的 span：

- `agent.run`
  - 属性：`agent.model`、`agent.provider`、`agent.duration_ms`、`agent.usage.*`
  - 事件：`reasoning.complete`（含 `reasoning.content`、`reasoning.token_count`）

## 参考文档

- [`docs/release/logfire_observability.md`](https://github.com/rexleimo/HNO/blob/main/docs/release/logfire_observability.md) – 详细操作指南（GitHub）。
- [`website/zh/advanced/observability.md`](../advanced/observability.md) – 项目观测方案概览，包括 SSE 事件流。
