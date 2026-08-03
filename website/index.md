---
layout: home

hero:
  name: "Agno-Go"
  text: "High-Performance Multi-Agent Framework"
  tagline: "16x faster than Python | 180ns instantiation | 1.2KB memory per agent"
  image:
    src: /logo.png
    alt: Agno-Go
  actions:
    - theme: brand
      text: Get Started
      link: /guide/quick-start
    - theme: alt
      text: View on GitHub
      link: https://github.com/rexleimo/agno-Go

features:
  - icon: 🚀
    title: Extreme Performance
    details: Agent instantiation in ~180ns with ~1.2KB per agent, delivering a 16× speedup over the Python runtime.

  - icon: 🤖
    title: Production Ready AgentOS
    details: REST server with OpenAPI 3.0, session storage, health checks, structured logging, CORS, request timeouts, and parity endpoints for summaries, reuse, and history filters.

  - icon: 🪄
    title: Session Parity
    details: Share sessions across agents and teams, trigger sync/async summaries, capture cache hits and cancellation reasons, and mirror Python's `stream_events` switches.

  - icon: 🧩
    title: Flexible Architecture
    details: Compose Agents, Teams (4 coordination modes), and Workflows (5 primitives) with inherited defaults, resumable checkpoints, and deterministic orchestration.

  - icon: 🔌
    title: Multi-Provider Models
    details: Ready for OpenAI o-series, Anthropic Claude, Google Gemini, DeepSeek, GLM, ModelScope, Ollama, Cohere, Groq, Together, OpenRouter, LM Studio, Vercel, Portkey, InternLM, and SambaNova.

  - icon: 🔧
    title: Extensible Tooling
    details: Calculator, HTTP, file ops, search, Claude Agent Skills, Tavily Reader/Search, Gmail mark-as-read, Jira worklogs, ElevenLabs voice, PPTX reader, plus MCP connectors.

  - icon: 💾
    title: Knowledge & Caching
    details: ChromaDB integration, batching utilities, ingestion helpers, and response caching to deduplicate identical model calls.

  - icon: 🛡️
    title: Guardrails & Observability
    details: Prompt-injection guard, custom pre/post hooks, media validation, SSE reasoning stream, and Logfire/OpenTelemetry tracing samples.

  - icon: 📦
    title: Easy Deployment
    details: Ship single binaries or use Docker, Compose, and Kubernetes manifests with ready-to-run deployment guides.
---

## Quick Example

Create an AI agent with tools in just a few lines:

```go
package main

import (
    "context"
    "fmt"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
)

func main() {
    // Create model
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    // Create agent with tools
    ag, _ := agent.New(agent.Config{
        Name:     "Math Assistant",
        Model:    model,
        Toolkits: []toolkit.Toolkit{calculator.New()},
    })

    // Run agent
    output, _ := ag.Run(context.Background(), "What is 25 * 4 + 15?")
    fmt.Println(output.Content) // Output: 115
}
```

## Performance Comparison

| Metric | Python Agno | Agno-Go | Improvement |
|--------|-------------|---------|-------------|
| Agent Creation | ~3μs | ~180ns | **16x faster** |
| Memory/Agent | ~6.5KB | ~1.2KB | **5.4x less** |
| Concurrency | GIL limited | Native goroutines | **Unlimited** |

## Why Agno-Go?
### What's New in v1.2.9

- **EvoLink Media Agents** – First-class EvoLink provider under `pkg/hno/providers/evolink` and `pkg/hno/models/evolink/*` for text, image, and video generation, with end-to-end examples in the EvoLink Media Agents docs.
- **Knowledge Upload Chunking** – `POST /api/v1/knowledge/content` supports `chunk_size` and `chunk_overlap` for JSON, `text/plain` (query params), and multipart uploads, and records these values plus `chunker_type` in chunk metadata.
- **AgentOS HTTP Tips** – Updated docs show how to customize health endpoints, rely on `/openapi.yaml` and `/docs`, and use `server.Resync()` after router changes.

### Built for Production

Agno-Go isn't just a framework—it's a complete production system. The included **AgentOS** server provides:

- RESTful API with OpenAPI 3.0 specification
- Session management for multi-turn conversations
- Thread-safe agent registry
- Health monitoring and structured logging
- CORS support and request timeout handling

### KISS Principle

Following the **Keep It Simple, Stupid** philosophy:

- **3 core LLM providers** (not 45+) - OpenAI, Anthropic, Ollama
- **Essential tools** (not 115+) - Calculator, HTTP, File, Search
- **Quality over quantity** - Focus on production-ready features

### Developer Experience

- **Type-Safe**: Go's strong typing catches errors at compile time
- **Fast Builds**: Go's compilation speed enables rapid iteration
- **Easy Deployment**: Single binary with no runtime dependencies
- **Great Tooling**: Built-in testing, profiling, and race detection

## Get Started in 5 Minutes

```bash
# Clone repository
git clone https://github.com/rexleimo/agno-Go.git
cd agno-Go

# Set API key
export OPENAI_API_KEY=sk-your-key-here

# Run example
go run cmd/examples/simple_agent/main.go

# Or start AgentOS server
docker-compose up -d
curl http://localhost:8080/health
```

## What's Included

- **Core Framework**: Agent, Team (4 modes), Workflow (5 primitives)
- **Models**: OpenAI, Anthropic Claude, Ollama, DeepSeek, Gemini, ModelScope
- **Tools**: Calculator (75.6%), HTTP (88.9%), File (76.2%), Search (92.1%)
- **MCP Integration**: Model Context Protocol support for connecting to any MCP server
- **RAG**: ChromaDB integration + OpenAI embeddings
- **AgentOS**: Production HTTP server (65.0% coverage)
- **Examples**: 8 working examples covering all features (Simple Agent, Claude Agent, Ollama Agent, Team Demo, Workflow Demo, RAG Demo, MCP Demo, Logfire Observability)
- **Docs**: Complete guides, API reference, deployment instructions

## Community

- **GitHub**: [rexleimo/agno-Go](https://github.com/rexleimo/agno-Go)
- **Issues**: [Report bugs and request features](https://github.com/rexleimo/agno-Go/issues)
- **Discussions**: [Ask questions and share ideas](https://github.com/rexleimo/agno-Go/discussions)

## License

Agno-Go is released under the [MIT License](https://github.com/rexleimo/agno-Go/blob/main/LICENSE).

Inspired by [Agno (Python)](https://github.com/agno-agi/agno) framework.
