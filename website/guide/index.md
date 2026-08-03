# What is HNO?

**HNO** is a high-performance multi-agent system framework built with Go. Inheriting the design philosophy from the Python Agno framework, it leverages Go's concurrency model and performance advantages to build efficient, scalable AI agent systems.

## Key Features

### 🚀 Extreme Performance

- **Agent Instantiation**: ~180ns average (16x faster than Python version)
- **Memory Footprint**: ~1.2KB per agent (5.4x less than Python)
- **Native Concurrency**: Full goroutine support without GIL limitations

### 🤖 Production-Ready

HNO includes **AgentOS**, a production HTTP server with:

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

Built-in support for 6 major LLM providers:

- **OpenAI** - GPT-4, GPT-3.5 Turbo, etc.
- **Anthropic** - Claude 3.5 Sonnet, Claude 3 Opus/Sonnet/Haiku
- **Ollama** - Local models (Llama 3, Mistral, CodeLlama, etc.)
- **DeepSeek** - DeepSeek-V2, DeepSeek-Coder
- **Google Gemini** - Gemini Pro, Flash
- **ModelScope** - Qwen, Yi models

### 🔧 Extensible Tools

Following the KISS principle, we provide essential tools with high quality:

- **Calculator** - Basic math operations (75.6% test coverage)
- **HTTP** - Make HTTP GET/POST requests (88.9% coverage)
- **File Operations** - Read, write, list, delete with security controls (76.2% coverage)
- **Search** - DuckDuckGo web search (92.1% coverage)

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

- **3 core LLM providers** (not 45+)
- **Essential tools** (not 115+)
- **1 vector database** (not 15+)

This focused approach ensures:
- Better code quality
- Easier maintenance
- Production-ready features

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
- **High-Performance Workflows** - Process thousands of requests
- **Local AI Development** - Use Ollama for privacy-focused applications
- **RAG Applications** - Build knowledge-based AI assistants

## Quality Metrics

- **Test Coverage**: 80.8% average across core packages
- **Test Cases**: 85+ tests with 100% pass rate
- **Documentation**: Complete guides, API reference, examples
- **Production-Ready**: Docker, K8s manifests, deployment guides

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

Inspired by [Agno (Python)](https://github.com/agno-agi/agno) framework.
