# Testing

Comprehensive testing guide for Agno-Go development.

---

## Overview

Agno-Go maintains high quality through comprehensive testing with **80.8% test coverage** across the codebase. This guide covers testing standards, patterns, and best practices.

### Test Coverage Status

| Package | Coverage | Status |
|---------|----------|--------|
| types | 100.0% | ✅ Excellent |
| memory | 93.1% | ✅ Excellent |
| team | 92.3% | ✅ Excellent |
| toolkit | 91.7% | ✅ Excellent |
| http | 88.9% | ✅ Good |
| workflow | 80.4% | ✅ Good |
| file | 76.2% | ✅ Good |
| calculator | 75.6% | ✅ Good |
| agent | 74.7% | ✅ Good |
| anthropic | 50.9% | 🟡 Needs improvement |
| openai | 44.6% | 🟡 Needs improvement |
| ollama | 43.8% | 🟡 Needs improvement |

---

## Running Tests

### All Tests

```bash
# Run all tests with coverage
make test

# Equivalent to:
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

### Specific Package

```bash
# Test agent package
go test -v ./pkg/hno/agent/...

# Test with coverage
go test -v -cover ./pkg/hno/agent/...
```

### Specific Test

```bash
# Run specific test function
go test -v -run TestAgentRun ./pkg/hno/agent/

# Run tests matching pattern
go test -v -run TestAgent.* ./pkg/hno/agent/
```

### Coverage Report

```bash
# Generate HTML coverage report
make coverage

# Opens coverage.html in browser
# Shows line-by-line coverage
```

---

## Testing Standards

### Coverage Requirements

- **Core packages** (agent, team, workflow): >70% coverage
- **Utility packages** (types, memory, toolkit): >80% coverage
- **New features**: Must include tests
- **Bug fixes**: Must include regression tests

### Test Structure

Every package should have:
- `*_test.go` files alongside source files
- Unit tests for all public functions
- Integration tests for complex workflows
- Benchmark tests for performance-critical code

---

## Writing Unit Tests

### Basic Unit Test

```go
package agent

import (
    "context"
    "testing"

    "github.com/rexleimo/agno-go/pkg/hno/models"
    "github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestAgentRun(t *testing.T) {
    // Create mock model
    model := &MockModel{
        InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
            return &types.ModelResponse{
                Content: "test response",
            }, nil
        },
    }

    // Create agent
    agent, err := New(Config{
        Name:  "test-agent",
        Model: model,
    })
    if err != nil {
        t.Fatalf("Failed to create agent: %v", err)
    }

    // Run agent
    output, err := agent.Run(context.Background(), "test input")
    if err != nil {
        t.Fatalf("Run failed: %v", err)
    }

    // Verify output
    if output.Content != "test response" {
        t.Errorf("Expected 'test response', got '%s'", output.Content)
    }
}
```

### Table-Driven Tests

```go
func TestCalculatorAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     float64
        expected float64
    }{
        {"positive numbers", 5.0, 3.0, 8.0},
        {"negative numbers", -5.0, -3.0, -8.0},
        {"mixed signs", 5.0, -3.0, 2.0},
        {"with zero", 5.0, 0.0, 5.0},
        {"decimals", 1.5, 2.3, 3.8},
    }

    calc := New()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := calc.add(map[string]interface{}{
                "a": tt.a,
                "b": tt.b,
            })
            if err != nil {
                t.Fatalf("add failed: %v", err)
            }

            if result != tt.expected {
                t.Errorf("Expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

### Error Handling Tests

```go
func TestAgentInvalidConfig(t *testing.T) {
    tests := []struct {
        name   string
        config Config
        errMsg string
    }{
        {
            name:   "nil model",
            config: Config{Name: "test"},
            errMsg: "model is required",
        },
        {
            name:   "empty name with nil memory",
            config: Config{Model: &MockModel{}, Memory: nil},
            errMsg: "", // Should succeed with defaults
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := New(tt.config)
            if tt.errMsg == "" {
                if err != nil {
                    t.Errorf("Expected no error, got: %v", err)
                }
            } else {
                if err == nil {
                    t.Error("Expected error, got nil")
                } else if !strings.Contains(err.Error(), tt.errMsg) {
                    t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
                }
            }
        })
    }
}
```

---

## Mocking

### Mock Model

```go
type MockModel struct {
    InvokeFunc       func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error)
    InvokeStreamFunc func(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error)
}

func (m *MockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
    if m.InvokeFunc != nil {
        return m.InvokeFunc(ctx, req)
    }
    return &types.ModelResponse{Content: "mock response"}, nil
}

func (m *MockModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
    if m.InvokeStreamFunc != nil {
        return m.InvokeStreamFunc(ctx, req)
    }
    return nil, nil
}

func (m *MockModel) GetProvider() string { return "mock" }
func (m *MockModel) GetID() string       { return "mock-model" }
```

### Mock Toolkit

```go
type MockToolkit struct {
    *toolkit.BaseToolkit
    callCount int
}

func NewMockToolkit() *MockToolkit {
    t := &MockToolkit{
        BaseToolkit: toolkit.NewBaseToolkit("mock"),
    }

    t.RegisterFunction(&toolkit.Function{
        Name:        "mock_function",
        Description: "Mock function for testing",
        Handler:     t.mockHandler,
    })

    return t
}

func (t *MockToolkit) mockHandler(args map[string]interface{}) (interface{}, error) {
    t.callCount++
    return "mock result", nil
}
```

---

## Benchmark Tests

### Basic Benchmark

```go
func BenchmarkAgentCreation(b *testing.B) {
    model := &MockModel{}

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, err := New(Config{
            Name:  "benchmark-agent",
            Model: model,
        })
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./pkg/hno/agent/

# Run specific benchmark
go test -bench=BenchmarkAgentCreation ./pkg/hno/agent/

# With memory allocation stats
go test -bench=. -benchmem ./pkg/hno/agent/

# Multiple runs for accuracy
go test -bench=. -benchtime=10s -count=5 ./pkg/hno/agent/
```

### Benchmark Output

```
BenchmarkAgentCreation-8    5623174    180.1 ns/op    1184 B/op    14 allocs/op
```

Interpretation:
- Ran 5,623,174 iterations
- 180.1 nanoseconds per operation
- 1,184 bytes allocated per operation
- 14 allocations per operation

---

## Integration Tests

### Testing with Real LLMs

```go
// +build integration

func TestAgentWithRealOpenAI(t *testing.T) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }

    model, err := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })
    if err != nil {
        t.Fatal(err)
    }

    agent, _ := New(Config{
        Model:    model,
        Toolkits: []toolkit.Toolkit{calculator.New()},
    })

    output, err := agent.Run(context.Background(), "What is 25 * 17?")
    if err != nil {
        t.Fatal(err)
    }

    // Verify agent used calculator
    if !strings.Contains(output.Content, "425") {
        t.Errorf("Expected answer to contain 425, got: %s", output.Content)
    }
}
```

Run integration tests:

```bash
# Run only integration tests
go test -tags=integration ./...

# Skip integration tests (default)
go test ./...
```

---

## Test Helpers

### Common Test Utilities

```go
// test_helpers.go

package testutil

import (
    "testing"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models"
)

func CreateTestAgent(t *testing.T) *agent.Agent {
    t.Helper()

    model := &MockModel{}
    ag, err := agent.New(agent.Config{
        Name:  "test-agent",
        Model: model,
    })
    if err != nil {
        t.Fatalf("Failed to create test agent: %v", err)
    }

    return ag
}

func AssertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
}

func AssertError(t *testing.T, err error, expectedMsg string) {
    t.Helper()
    if err == nil {
        t.Fatal("Expected error, got nil")
    }
    if !strings.Contains(err.Error(), expectedMsg) {
        t.Fatalf("Expected error containing '%s', got: %v", expectedMsg, err)
    }
}
```

---

## Continuous Integration

### GitHub Actions

Tests run automatically on every push and pull request:

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go test -v -race -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

---

## Best Practices

### 1. Test Naming

```go
// Good ✅
func TestAgentRun(t *testing.T)
func TestAgentRun_WithTools(t *testing.T)
func TestAgentRun_EmptyInput_ReturnsError(t *testing.T)

// Bad ❌
func Test1(t *testing.T)
func TestStuff(t *testing.T)
```

### 2. Use t.Helper()

Mark helper functions:

```go
func createAgent(t *testing.T) *Agent {
    t.Helper() // Stack traces skip this function
    // ...
}
```

### 3. Clean Up Resources

```go
func TestWithTempFile(t *testing.T) {
    f, err := os.CreateTemp("", "test-*")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(f.Name()) // Clean up

    // Test code...
}
```

### 4. Use Subtests

```go
func TestAgent(t *testing.T) {
    t.Run("creation", func(t *testing.T) { /* ... */ })
    t.Run("run", func(t *testing.T) { /* ... */ })
    t.Run("clear memory", func(t *testing.T) { /* ... */ })
}
```

### 5. Parallel Tests

```go
func TestParallel(t *testing.T) {
    tests := []struct{ /* ... */ }{}

    for _, tt := range tests {
        tt := tt // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // Run in parallel
            // Test code...
        })
    }
}
```

---

## Troubleshooting

### Tests Failing with Race Detector

```bash
# Run with race detector
go test -race ./...

# If race detected, fix the code
# Don't disable race detector
```

### Coverage Not Updating

```bash
# Clean cache and rerun
go clean -testcache
make test
```

### Slow Tests

```bash
# Find slow tests
go test -v ./... | grep PASS

# Add timeout
go test -timeout 30s ./...
```

---

## Next Steps

- Review [Architecture](/advanced/architecture) for design patterns
- Check [Performance](/advanced/performance) for benchmarks
- Read [Deployment](/advanced/deployment) for production setup
- Explore [Contributing Guide](https://github.com/rexleimo/agno-Go/blob/main/CONTRIBUTING.md)

---

## Resources

- [Go Testing](https://golang.org/pkg/testing/)
- [Table Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [Test Coverage](https://go.dev/blog/cover)
- [Benchmarking](https://pkg.go.dev/testing#hdr-Benchmarks)
