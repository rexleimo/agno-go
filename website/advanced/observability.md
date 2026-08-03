# Observability & Telemetry

HNO exposes several layers of runtime visibility so that production operators can monitor reasoning-heavy workloads, token consumption, and long-running sessions.

## AgentOS SSE Stream

`POST /api/v1/agents/{id}/run/stream` returns a Server-Sent Events (SSE) stream. Use the optional `types` query parameter to filter events (e.g. `types=run_start,token,reasoning,complete`).

### Event Types

| Event | Description |
| --- | --- |
| `run_start` | Input payload and session metadata. |
| `token` | Individual streaming tokens returned by the model. |
| `tool_call` | Tool execution metadata (name, arguments, result). |
| `reasoning` | Reasoning snapshots with content, token counts, redacted text, model/provider IDs. |
| `complete` | Final output, duration, aggregated usage (prompt/completion/reasoning tokens). |
| `error` | Structured error object with code and details. |

Reasoning events are emitted whenever the model returns `ReasoningContent`. This works out of the box for OpenAI o1/o3/o4, Gemini 2.5 Thinking, and Claude models with `thinking` enabled.

## Logfire Integration

`cmd/examples/logfire_observability` demonstrates how to export traces to Logfire using OpenTelemetry. Highlights:

1. Configure the OTLP endpoint and write token (`LOGFIRE_WRITE_TOKEN`, `LOGFIRE_OTLP_ENDPOINT`).
2. Build with the `logfire` tag to include the OpenTelemetry exporter:
   ```bash
   go run -tags logfire cmd/examples/logfire_observability/main.go
   ```
3. The example records span metadata such as loop count, aggregated token usage, and reasoning excerpts (with optional redacted content).

Full step-by-step instructions live in [`docs/release/logfire_observability.md`](https://github.com/rexleimo/HNO/blob/main/docs/release/logfire_observability.md).

## Next Steps

- Forward SSE events to your telemetry backend (Logfire, Elastic, Datadog).
- Combine reasoning token metrics with cost dashboards to monitor spend.
- Use the OpenTelemetry hooks to instrument tool execution or end-user request latency.
