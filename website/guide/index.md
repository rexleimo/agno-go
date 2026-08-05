# What is HNO?

**HNO** is a multi-agent system framework built with Go. It uses Go's concurrency model, static typing, deployment model, and standard tooling; workload-specific performance claims are documented only when a reproducible benchmark is available.

## Key Features

### 🚀 Measured Performance

- Agent-construction benchmark results and the exact environment are recorded in [Performance](/advanced/performance).
- The benchmark uses a local `MockModel` and measures framework allocation, not LLM latency or service throughput.
- No Go-vs-Python speedup or memory ratio is published without an apples-to-apples benchmark.
- **Native Concurrency**: Go goroutines are available for application-level concurrency.

### 🤖 AgentOS HTTP Server

HNO includes **AgentOS**, an HTTP server with:

- RESTful API with OpenAPI 3.0 specification
- Session management for multi-turn conversations
- Thread-safe agent registry
- Health monitoring and structured logging
- CORS support and request timeout handling

### 🧩 Flexible Architecture

Three core abstractions for different use cases:

1. **Agent** - Autonomous AI agents with tool support and memory
2. **Team** - Multi-agent collaboration with 4 coordination modes
   - Sequential, Parallel, Leader-Follower, Consensus
3. **Workflow** - Step-based orchestration with 5 primitives
   - Step, Condition, Loop, Parallel, Router

### 🔌 Multi-Model Support

Built-in support for multiple LLM providers:

- **OpenAI** - GPT-4, GPT-3.5 Turbo, etc.
- **Anthropic** - Claude 3.5 Sonnet, Claude 3 Opus/Sonnet/Haiku
- **Ollama** - Local models (Llama 3, Mistral, CodeLlama, etc.)
- **DeepSeek** - DeepSeek-V2, DeepSeek-Coder
- **Google Gemini** - Gemini Pro, Flash
- **ModelScope** - Qwen, Yi models

### 🔧 Extensible Tools

Following the KISS principle, we provide essential tools with high quality:

- **Calculator** - Basic math operations
- **HTTP** - Make HTTP GET/POST requests
- **File Operations** - Read, write, list, delete with security controls
- **Search** - DuckDuckGo web search

Easy to create custom tools - see [Tools Guide](/guide/tools).

### 💾 RAG & Knowledge

Build intelligent agents with knowledge bases:

- **ChromaDB** - Vector database integration
- **OpenAI Embeddings** - text-embedding-3-small/large support
- Automatic embedding generation and semantic search

See [RAG Demo](/examples/rag-demo) for a complete example.

## Design Philosophy

### KISS Principle

**Keep It Simple, Stupid** - Focus on quality over quantity:

- A small, inspectable core
- Essential tools
- Pluggable storage integrations

This focused approach aims for:
- Better code quality
- Easier maintenance
- Deployable server features; production suitability depends on the workload

### Go Advantages

Why build multi-agent systems with Go?

1. **Performance** - Compiled language, fast execution
2. **Concurrency** - Native goroutines, no GIL
3. **Type Safety** - Catch errors at compile time
4. **Single Binary** - Easy deployment, no runtime dependencies
5. **Great Tooling** - Built-in testing, profiling, race detection

## Use Cases

HNO is perfect for:

- **Production AI Applications** - Deploy with AgentOS HTTP server
- **Multi-Agent Systems** - Coordinate multiple AI agents
- **Application Workflows** - Compose multi-step agent tasks
- **Local AI Development** - Use Ollama for privacy-focused applications
- **RAG Applications** - Build knowledge-based AI assistants

## Quality Metrics

- **Test status**: Run `go test ./...` for the current result
- **Benchmark status**: See the reproducible snapshot in [Performance](/advanced/performance)
- **Documentation**: Guides, API reference, and examples are built with VitePress
- **Deployment**: Docker and deployment material are provided; production suitability depends on the deployment workload

## Next Steps

Ready to get started?

1. [Quick Start](/guide/quick-start) - Build your first agent in 5 minutes
2. [Installation](/guide/installation) - Detailed setup instructions
3. [Core Concepts](/guide/agent) - Learn about Agent, Team, Workflow

## Quick Links

- Embeddings: [OpenAI/VLLM usage](/guide/embeddings)
- Vector Indexing: [Chroma + Redis (optional) + CLI](/advanced/vector-indexing)

## Community

- **GitHub**: [rexleimo/HNO](https://github.com/rexleimo/HNO)
- **Issues**: [Report bugs](https://github.com/rexleimo/HNO/issues)
- **Discussions**: [Ask questions](https://github.com/rexleimo/HNO/discussions)

## License

HNO is released under the [MIT License](https://github.com/rexleimo/HNO/blob/main/LICENSE).

Inspired by the [Agno Python](https://github.com/agno-agi/agno) project. HNO is the current name of this Go project; the repository does not define an official expansion of the name.
