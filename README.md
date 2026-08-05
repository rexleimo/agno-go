# Agno-Go

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

[![Release](https://img.shields.io/badge/release-v1.2.9-blue.svg)](CHANGELOG.md)

**Agno-Go** is a multi-agent framework written in Go. It keeps the KISS philosophy of the Agno project while using Go’s concurrency, static typing, deployment, and standard tooling strengths.

---

## Feature Highlights

- **🚀 Measured performance** – checked-in Go benchmarks report allocation ranges for agent construction; see the [performance report](website/advanced/performance.md) for the exact environment and limitations.
- **🤖 AgentOS HTTP server** – REST server (OpenAPI 3.0), session storage, health checks, structured logging, CORS, request timeouts, and parity endpoints for summaries, reuse, and history filters.
- **🪄 Session parity** – shared sessions across agents/teams, async + sync summaries, run metadata with cache hits and cancellation reasons, and `stream_events` flags matching the Python runtime.
- **🧩 Flexible architecture** – build with Agents, Teams (4 coordination modes), or Workflows (5 primitives) and mix freely; teams inherit/default models and workflows resume from snapshots.
- **🔌 Multi-provider models** – OpenAI (incl. o-series reasoning), Anthropic Claude, Google Gemini, DeepSeek, GLM, ModelScope, Ollama, Cohere, Groq, Together, OpenRouter, LM Studio, Vercel, Portkey, InternLM, SambaNova.
- **🔧 Extensible tooling** – calculator, HTTP, file operations, search, Claude Agent Skills, Tavily Reader, PPTX reader, Jira worklogs, Gmail mark-as-read, ElevenLabs speech, plus an SDK for bespoke toolkits or MCP connectors.
- **💾 Knowledge & RAG** – ChromaDB integration, batching utilities, response caching helpers, and ingestion helpers.
- **🛡️ Guardrails & hooks** – prompt-injection guard, custom pre/post hooks, media validation, graceful degradation.
- **📊 Observability** – rich SSE event stream with reasoning snapshots, Logfire / OpenTelemetry sample included.

External service toolkits use live provider APIs and never return built-in sample
records. Set `YOUTUBE_API_KEY` for YouTube Data API and `TAVILY_API_KEY` for
Tavily web search. Bitbucket, Confluence, and Airflow require their endpoint
and credentials in the tool call; Hacker News uses its public Firebase and
Algolia APIs. Missing credentials or provider failures are returned as errors.

---

## Getting Started

```bash
go get github.com/rexleimo/agno-go
```

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	model, _ := openai.New("gpt-4o-mini", openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
	})

	ag, _ := agent.New(agent.Config{
		Name:     "Math Assistant",
		Model:    model,
		Toolkits: []toolkit.Toolkit{calculator.New()},
	})

	output, _ := ag.Run(context.Background(), "What is 25 * 4 + 15?")
	fmt.Println(output.Content)
}
```

Run the production server with Docker:

```bash
docker compose up -d
curl http://localhost:8080/health
```

### AgentOS HTTP tips

- Override the default `GET /health` path via `Config.HealthPath` or attach your
  own handlers with `server.GetHealthRouter("/health-check").GET("", customHandler)`.
- `/openapi.yaml` always serves the current OpenAPI document and `/docs` hosts a
  self-contained Swagger UI bundle. Call `server.Resync()` after hot-swapping
  routers to remount the documentation routes.
- Sample probes:
  ```bash
  curl http://localhost:8080/health-check
  curl http://localhost:8080/openapi.yaml | head -n 5
  ```

---

## Documentation

| Resource | Link |
| --- | --- |
| Guides | https://rexleimo.github.io/agno-Go/guide/ |
| API Reference | https://rexleimo.github.io/agno-Go/api/ |
| Advanced Topics | https://rexleimo.github.io/agno-Go/advanced/ |
| Examples | https://rexleimo.github.io/agno-Go/examples/ |
| Release Notes | https://rexleimo.github.io/agno-Go/release-notes |
| Internal / WIP Docs | [`docs/`](docs/) |

## What's New in v1.2.9

- **EvoLink Media Agents** – First-class EvoLink provider under `pkg/hno/providers/evolink` and `pkg/hno/models/evolink/*` for text, image, and video generation, with example workflows in `website/examples/evolink-media-agents.md`.
- **Knowledge Upload Chunking** – `POST /api/v1/knowledge/content` now accepts `chunk_size` and `chunk_overlap` (JSON, `text/plain` query params, multipart form fields) and records these values plus `chunker_type` in stored chunk metadata.
- **AgentOS HTTP Tips in Docs** – The AgentOS API page now documents how to customize health endpoints, rely on `/openapi.yaml` and `/docs`, and when to call `server.Resync()` after router changes.

## Session Runtime & Storage Parity

- **Session reuse & history:** `POST /api/v1/sessions/{id}/reuse` shares conversations between agents, teams, and workflows, while `GET /api/v1/sessions/{id}/history?num_messages=N&stream_events=true` mirrors Python-style pagination and SSE toggles.
- **Summaries:** `GET`/`POST /api/v1/sessions/{id}/summary` trigger synchronous or async summaries via `session.SummaryManager`, persisting the latest snapshot on completion.
- **Run metadata:** responses include `runs[*].cache_hit`, `runs[*].status`, timestamps, and cancellation reasons to power audits and resumptions.
- **Pluggable stores:** choose Postgres, MongoDB, or SQLite adapters with identical JSON contracts; fall back to in-memory storage for tests.
- **Response caching:** enable the built-in cache to deduplicate identical model calls across runs.

```go
db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))
store, _ := postgres.NewStorage(db, postgres.WithSchema("agentos"))

summaryModel, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: os.Getenv("OPENAI_API_KEY")})
summary := session.NewSummaryManager(
    session.WithSummaryModel(summaryModel),
    session.WithSummaryTimeout(45*time.Second),
)

server, _ := agentos.NewServer(&agentos.Config{
    Address:        ":8080",
    SessionStorage: store,
    SummaryManager: summary,
})

agent, _ := agent.New(agent.Config{
    Name:        "Cached Assistant",
    Model:       summaryModel,
    EnableCache: true,
})
```

`docs/README.md` explains the split between the public site (`website/`) and internal design notes (`docs/`).

---

### Knowledge upload chunking

`POST /api/v1/knowledge/content` now accepts `chunk_size` and `chunk_overlap`
in both JSON and multipart form uploads. Provide them as query parameters for
`text/plain` requests or as form fields (`chunk_size=2000&chunk_overlap=250`) when
streaming files. Both values propagate into the reader metadata, so downstream
pipelines can inspect how documents were segmented.

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/content \
  -F file=@docs/guide.md \
  -F chunk_size=1800 \
  -F chunk_overlap=200 \
  -F metadata='{"source_url":"https://example.com/guide"}'
```

Each stored chunk automatically records `chunk_size`, `chunk_overlap`, and the
`chunker_type` used—mirroring the AgentOS Python responses.

---

## Observability & Reasoning

- **SSE Event Stream** – `POST /api/v1/agents/{id}/run/stream?types=run_start,reasoning,token,complete` emits structured events. `reasoning` events carry token counts, redacted transcripts, and provider metadata; `complete` events summarise the run.
- **Logfire Integration** – `cmd/examples/logfire_observability` shows how to export spans with OpenTelemetry (build with `-tags logfire`). Detailed walkthrough: [`docs/release/logfire_observability.md`](docs/release/logfire_observability.md).

---

### Anthropic Claude betas & context management

Set `anthropic.Config.Betas` to opt into long-context beta deployments and use
`anthropic.Config.ContextManagement` (or `req.Extra["context_management"]`) to
attach `applied_edits` and other context-management hints. The Go client merges
config-level and per-request metadata, and surfaced `context_management` payloads
end up in `RunOutput.Metadata`, so tool builders can inspect `applied_edits`
directly.

```go
model, _ := anthropic.New("claude-3-5-sonnet", anthropic.Config{
    APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
    Betas:   []string{"context-1m-2025-08-07"},
    ContextManagement: map[string]interface{}{"applied_edits": []string{"trim_history"}},
})
```

---

## Example Catalogue

| Example | Highlights | Run |
| --- | --- | --- |
| **Simple Agent** (`cmd/examples/simple_agent/`) | GPT‑4o mini, calculator toolkit, single agent | `go run cmd/examples/simple_agent/main.go` |
| **Claude Agent** (`cmd/examples/claude_agent/`) | Anthropic Claude 3.5, HTTP + calculator tools | `go run cmd/examples/claude_agent/main.go` |
| **Ollama Agent** (`cmd/examples/ollama_agent/`) | Local Llama 3 via Ollama, file operations | `go run cmd/examples/ollama_agent/main.go` |
| **Team Demo** (`cmd/examples/team_demo/`) | 4 coordination modes, researcher + writer workflow | `go run cmd/examples/team_demo/main.go` |
| **Workflow Demo** (`cmd/examples/workflow_demo/`) | Step / condition / loop / parallel orchestration | `go run cmd/examples/workflow_demo/main.go` |
| **RAG Demo** (`cmd/examples/rag_demo/`) | ChromaDB, embeddings, document Q&A | `go run cmd/examples/rag_demo/main.go` |
| **Reasoning Demo** (`examples/reasoning/`) | OpenAI o1 / Gemini 2.5 thinking extraction | `go run examples/reasoning/main.go` |
| **Logfire Observability** (`cmd/examples/logfire_observability/`) | OpenTelemetry spans + reasoning metadata | `go run -tags logfire cmd/examples/logfire_observability/main.go` |

More details live in the [Examples documentation](website/examples/index.md).

---

## Development & Contribution

1. Read [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) for tooling, linting, and testing workflow.
2. Docs: follow the structure in [`docs/README.md`](docs/README.md) and update the VitePress site (`website/`) when promoting features.
3. Run targeted tests, then `go test ./...` (with `GOCACHE=$(pwd)/.gocache` when sandboxed).
4. Submit PRs with lint/test evidence. Adhere to Conventional Commits.

For ongoing changes and release scope, see [`CHANGELOG.md`](CHANGELOG.md) and the VitePress site’s release notes (`website/release-notes.md`).

---

## License

MIT © [Contributors](https://github.com/rexleimo/agno-Go/graphs/contributors). Inspired by the [Agno (Python)](https://github.com/agno-agi/agno) framework.
