# 🎉 Agno-Go v1.0.0 - Initial Production Release

**Agno-Go** is a high-performance multi-agent system framework built with Go, designed for building production-ready AI agent systems.

## 🚀 Installation

```bash
go get github.com/rexleimo/agno-go@v1.0.0
```

## ✨ Key Features

### Core Agent System
- **Agent** - Single autonomous agent with tool support (74.7% test coverage)
- **Team** - Multi-agent collaboration with 4 coordination modes (92.3% coverage)
  - Sequential, Parallel, LeaderFollower, Consensus
- **Workflow** - Step-based orchestration with 5 primitives (80.4% coverage)
  - Step, Condition, Loop, Parallel, Router

### LLM Providers (6 providers)
- ✅ **OpenAI** - GPT-4, GPT-3.5, GPT-4 Turbo
- ✅ **Anthropic** - Claude 3.5 Sonnet, Claude 3 Opus/Sonnet/Haiku
- ✅ **Ollama** - Local model support (llama3, mistral, etc.)
- ✅ **DeepSeek** - DeepSeek-V2, DeepSeek-Coder
- ✅ **Google Gemini** - Gemini Pro, Flash
- ✅ **ModelScope** - Qwen, Yi models

### Built-in Tools
- **Calculator** - Basic math operations (75.6% coverage)
- **HTTP** - GET/POST requests (88.9% coverage)
- **File** - File operations with safety controls (76.2% coverage)
- **Search** - DuckDuckGo web search (92.1% coverage)

### Storage & Knowledge
- **Memory** - In-memory conversation storage with auto-truncation (93.1% coverage)
- **ChromaDB** - Vector database integration for RAG applications
- **Embeddings** - OpenAI embeddings support (text-embedding-3-small/large)

### AgentOS - Production Server
- **RESTful API** - Full-featured HTTP server with session management
- **Agent Registry** - Thread-safe agent management
- **OpenAPI 3.0** - Complete API specification
- **Deployment** - Docker, Docker Compose, Kubernetes manifests included

## 📊 Performance

- **Agent instantiation**: ~180ns (16x faster than Python version)
- **Memory footprint**: ~1.2KB per agent
- **Test coverage**: 80.8% average across core packages

## 🧪 Quality Assurance

- ✅ **85+ test cases** with 100% pass rate
- ✅ All core packages exceed 70% coverage target
- ✅ Types package: **100% coverage** ⭐
- ✅ Comprehensive integration tests
- ✅ Performance benchmarks

## 📚 Documentation

- 📖 [Quick Start Guide](https://github.com/rexleimo/agno-Go#-quick-start)
- 📖 [API Documentation](https://pkg.go.dev/github.com/rexleimo/agno-go)
- 📖 [Deployment Guide](https://github.com/rexleimo/agno-Go/blob/main/website/advanced/deployment.md)
- 📖 [Architecture Overview](https://github.com/rexleimo/agno-Go/blob/main/website/advanced/architecture.md)
- 📖 [Performance Benchmarks](https://github.com/rexleimo/agno-Go/blob/main/website/advanced/performance.md)
- 📖 [Examples](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples)

## 🎯 Quick Start Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
    // Create model
    model, err := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create agent with tools
    ag, err := agent.New(agent.Config{
        Name:     "Assistant",
        Model:    model,
        Toolkits: []toolkit.Toolkit{calculator.New()},
    })
    if err != nil {
        log.Fatal(err)
    }

    // Run agent
    output, err := ag.Run(context.Background(), "What is 25 * 4 + 15?")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output.Content) // Output: 115
}
```

## 📦 Example Programs

See [cmd/examples/](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples):
- `simple_agent` - Basic agent with OpenAI and calculator tools
- `claude_agent` - Anthropic Claude integration with tools
- `ollama_agent` - Local model support with Ollama
- `team_demo` - Multi-agent collaboration demo
- `workflow_demo` - Workflow orchestration demo
- `rag_demo` - RAG pipeline with ChromaDB

## 🔧 Technical Details

**Requirements:**
- Go 1.21 or later
- LLM API keys (OpenAI, Anthropic, etc.)

**Optional Dependencies:**
- PostgreSQL 15+ (for persistent session storage)
- Redis 7+ (for caching)
- ChromaDB (for vector storage/RAG)
- Ollama (for local models)

## 🎯 Design Philosophy

Agno-Go follows the **KISS principle** (Keep It Simple, Stupid):
- Focus on quality over quantity
- Clear, maintainable code
- Comprehensive testing
- Production-ready from day one

## 🙏 Acknowledgments

Agno-Go is inspired by and compatible with the design philosophy of [Agno](https://github.com/agno-agi/agno) - Python multi-agent framework.

## 📝 Full Changelog

See [CHANGELOG.md](https://github.com/rexleimo/agno-Go/blob/main/CHANGELOG.md) for complete release notes.

## 🔗 Links

- **GitHub**: https://github.com/rexleimo/agno-go
- **Go Packages**: https://pkg.go.dev/github.com/rexleimo/agno-go
- **Issues**: https://github.com/rexleimo/agno-go/issues
- **Discussions**: https://github.com/rexleimo/agno-go/discussions

---

**Full Changelog**: https://github.com/rexleimo/agno-Go/commits/v1.0.0
