# Advanced Topics

Deep dive into advanced concepts, performance optimization, deployment strategies, and testing best practices for HNO.

## Overview

This section covers advanced topics for developers who want to:

- 🏗️ **Understand the architecture** - Learn the core design principles and patterns
- ⚡ **Measure performance** - Reproduce the checked-in Go benchmarks on your target environment
- 🚀 **Deploy to production** - Best practices for production deployments
- 🧪 **Test effectively** - Comprehensive testing strategies and tools

## Core Topics

### [Architecture](/advanced/architecture)

Learn about HNO's modular architecture and design philosophy:

- Core interfaces (Model, Toolkit, Memory)
- Abstraction patterns (Agent, Team, Workflow)
- Go concurrency model integration
- Error handling strategies
- Package organization

**Key concepts**: Clean architecture, dependency injection, interface design

### [Performance](/advanced/performance)

Understand measured performance characteristics and benchmark methodology:

- Reproducible agent-construction benchmarks with environment details
- Concurrency and parallelism
- Benchmarking tools and methodologies
- Explicit limits on what is not yet benchmarked

**Key metrics**: `ns/op`, `B/op`, allocations, and workload-specific throughput

### [Deployment](/advanced/deployment)

Production deployment strategies and best practices:

- AgentOS HTTP server setup
- Container deployment (Docker, Kubernetes)
- Configuration management
- Monitoring and observability
- Scaling strategies
- Security considerations

**Key technologies**: Docker, Kubernetes, Prometheus, distributed tracing

### [Testing](/advanced/testing)

Comprehensive testing approaches for multi-agent systems:

- Unit testing patterns
- Integration testing with mocks
- Performance benchmarking
- Test coverage requirements (>70%)
- CI/CD integration
- Testing tools and utilities

**Key tools**: Go testing, testify, benchmarking, coverage reports

## Quick Links

### Vector Indexing

```bash
# Create or drop vector collections (Chroma by default)
go run ./cmd/vectordb_migrate --action up --provider chroma --collection mycol \
  --chroma-url http://localhost:8000 --distance cosine

# Redis provider (optional; build with tag)
go run -tags redis ./cmd/vectordb_migrate --action up --provider redis \
  --collection mycol --chroma-url localhost:6379
```

[See release notes →](/release-notes#version-128-2025-11-10)

### Performance Benchmarks

```bash
# Run the reproducible agent-construction snapshot
go test -run='^$' -bench='BenchmarkAgentCreation' -benchmem -count=10 ./pkg/hno/agent/

# Generate CPU profile
go test -bench=. -cpuprofile=cpu.out ./pkg/hno/agent/
```

[See detailed performance metrics →](/advanced/performance)

[See local framework and system overhead matrix →](/advanced/system-overhead)

### Production Deployment

```bash
# Build AgentOS server
cd pkg/agentos && go build -o agentos

# Run with Docker
docker build -t agno-go-agentos .
docker run -p 8080:8080 -e OPENAI_API_KEY=$OPENAI_API_KEY agno-go-agentos
```

[See deployment guide →](/advanced/deployment)

### Testing Coverage

Current test coverage by package:

| Package | Coverage | Status |
|---------|----------|--------|
| types | 100.0% | ✅ Excellent |
| memory | 93.1% | ✅ Excellent |
| team | 92.3% | ✅ Excellent |
| toolkit | 91.7% | ✅ Excellent |
| workflow | 80.4% | ✅ Good |
| agent | 74.7% | ✅ Good |

[See testing guide →](/advanced/testing)

## Design Principles

### KISS (Keep It Simple, Stupid)

HNO embraces simplicity:

- **Focused scope**: 3 LLM providers (OpenAI, Anthropic, Ollama) instead of 8+
- **Essential tools**: 5 core tools instead of 15+
- **Clear abstractions**: Agent, Team, Workflow
- **Minimal dependencies**: Standard library first

### Performance First

Go's concurrency model enables:

- Native goroutine support for parallel execution
- No GIL (Global Interpreter Lock) limitations
- Efficient memory management
- Compile-time optimizations

### Production Ready

Built for real-world deployments:

- Comprehensive error handling
- Context-aware cancellation
- Structured logging
- OpenTelemetry integration
- Health checks and metrics

## Contributing

Interested in contributing to HNO? Check out:

- [Architecture documentation](/advanced/architecture) - Understand the codebase
- [Testing guide](/advanced/testing) - Learn testing standards
- [GitHub repository](https://github.com/rexleimo/HNO) - Submit PRs
- [Development guide](https://github.com/rexleimo/HNO/blob/main/CLAUDE.md) - Development setup

## Additional Resources

### Documentation

- [Go package documentation](https://pkg.go.dev/github.com/rexleimo/HNO)
- [Python HNO framework](https://github.com/agno-agi/agno) (inspiration)
- [VitePress documentation source](https://github.com/rexleimo/HNO/tree/main/website)

### Community

- [GitHub Issues](https://github.com/rexleimo/HNO/issues)
- [GitHub Discussions](https://github.com/rexleimo/HNO/discussions)
- [Release Notes](/release-notes)

## Next Steps

1. 📖 Start with [Architecture](/advanced/architecture) to understand core design
2. ⚡ Learn about [Performance](/advanced/performance) optimization techniques
3. 🚀 Review [Deployment](/advanced/deployment) strategies for production
4. 🧪 Master [Testing](/advanced/testing) best practices

---

**Note**: This section assumes familiarity with basic HNO concepts. If you're new, start with the [Guide](/guide/) section.
