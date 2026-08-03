# Logfire Observability

This example instruments an HNO agent with OpenTelemetry and forwards spans to [Logfire](https://logfire.dev). It is useful when you want to correlate reasoning content, token usage, and tool activity inside your observability platform.

## Requirements

- Go 1.21+
- OpenAI API key (or another reasoning-capable provider)
- Logfire write token (`LOGFIRE_WRITE_TOKEN`)
- Optional: custom OTLP endpoint (`LOGFIRE_OTLP_ENDPOINT` defaults to `logfire-eu.pydantic.dev`)

## Run

```bash
export OPENAI_API_KEY=sk-your-key
export LOGFIRE_WRITE_TOKEN=lf_your_token
go run -tags logfire cmd/examples/logfire_observability/main.go
```

> **Note**  
> The example is guarded by the `logfire` build tag to avoid pulling OpenTelemetry dependencies during normal builds.

## What the Example Does

1. Configures an OTLP/HTTP exporter with TLS and the Logfire write token.
2. Runs a reasoning-capable agent (OpenAI o1 preview by default).
3. Records span attributes for runtime, loop count, and token usage.
4. Emits a `reasoning.complete` span event containing the reasoning snippet (and optional redacted content).

In Logfire you will see spans similar to:

- `agent.run`
  - Attributes: `agent.model`, `agent.provider`, `agent.duration_ms`, `agent.usage.*`
  - Events: `reasoning.complete` with `reasoning.content`, `reasoning.token_count`

## Related Docs

- [`docs/release/logfire_observability.md`](https://github.com/rexleimo/HNO/blob/main/docs/release/logfire_observability.md) – deep dive guide (GitHub).
- [`website/advanced/observability.md`](../advanced/observability.md) – overview of the observability stack, including the SSE event stream.
