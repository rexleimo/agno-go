# 테스팅

HNO 개발을 위한 종합적인 테스트 가이드입니다.

---

## 개요

HNO는 코드베이스 전체에서 **80.8% 테스트 커버리지**를 통해 높은 품질을 유지합니다. 이 가이드는 테스트 표준, 패턴 및 모범 사례를 다룹니다.

### 테스트 커버리지 현황

| 패키지 | 커버리지 | 상태 |
|---------|----------|--------|
| types | 100.0% | ✅ 우수 |
| memory | 93.1% | ✅ 우수 |
| team | 92.3% | ✅ 우수 |
| toolkit | 91.7% | ✅ 우수 |
| http | 88.9% | ✅ 양호 |
| workflow | 80.4% | ✅ 양호 |
| file | 76.2% | ✅ 양호 |
| calculator | 75.6% | ✅ 양호 |
| agent | 74.7% | ✅ 양호 |
| anthropic | 50.9% | 🟡 개선 필요 |
| openai | 44.6% | 🟡 개선 필요 |
| ollama | 43.8% | 🟡 개선 필요 |

---

## 테스트 실행

### 모든 테스트

```bash
# 커버리지와 함께 모든 테스트 실행
make test

# 동일:
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

### 특정 패키지

```bash
# agent 패키지 테스트
go test -v ./pkg/hno/agent/...

# 커버리지와 함께 테스트
go test -v -cover ./pkg/hno/agent/...
```

### 특정 테스트

```bash
# 특정 테스트 함수 실행
go test -v -run TestAgentRun ./pkg/hno/agent/

# 패턴과 일치하는 테스트 실행
go test -v -run TestAgent.* ./pkg/hno/agent/
```

### 커버리지 보고서

```bash
# HTML 커버리지 보고서 생성
make coverage

# 브라우저에서 coverage.html 열림
# 라인별 커버리지 표시
```

---

## 테스트 표준

### 커버리지 요구사항

- **핵심 패키지** (agent, team, workflow): >70% 커버리지
- **유틸리티 패키지** (types, memory, toolkit): >80% 커버리지
- **새로운 기능**: 테스트 포함 필수
- **버그 수정**: 회귀 테스트 포함 필수

### 테스트 구조

모든 패키지는 다음을 포함해야 함:
- 소스 파일과 나란히 `*_test.go` 파일
- 모든 공개 함수의 단위 테스트
- 복잡한 워크플로우의 통합 테스트
- 성능 중요 코드의 벤치마크 테스트

---

## 단위 테스트 작성

### 기본 단위 테스트

```go
package agent

import (
    "context"
    "testing"

    "github.com/rexleimo/agno-go/pkg/hno/models"
    "github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestAgentRun(t *testing.T) {
    // 모의 모델 생성
    model := &MockModel{
        InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
            return &types.ModelResponse{
                Content: "test response",
            }, nil
        },
    }

    // 에이전트 생성
    agent, err := New(Config{
        Name:  "test-agent",
        Model: model,
    })
    if err != nil {
        t.Fatalf("Failed to create agent: %v", err)
    }

    // 에이전트 실행
    output, err := agent.Run(context.Background(), "test input")
    if err != nil {
        t.Fatalf("Run failed: %v", err)
    }

    // 출력 검증
    if output.Content != "test response" {
        t.Errorf("Expected 'test response', got '%s'", output.Content)
    }
}
```

### 테이블 기반 테스트

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

### 에러 처리 테스트

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

## 모킹

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

## 벤치마크 테스트

### 기본 벤치마크

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

### 벤치마크 실행

```bash
# 모든 벤치마크 실행
go test -bench=. ./pkg/hno/agent/

# 특정 벤치마크 실행
go test -bench=BenchmarkAgentCreation ./pkg/hno/agent/

# 메모리 할당 통계와 함께
go test -bench=. -benchmem ./pkg/hno/agent/

# 정확성을 위한 여러 실행
go test -bench=. -benchtime=10s -count=5 ./pkg/hno/agent/
```

### 벤치마크 출력

```
BenchmarkAgentCreation-8    5623174    180.1 ns/op    1184 B/op    14 allocs/op
```

해석:
- 5,623,174회 반복 실행
- 작업당 180.1 나노초
- 작업당 1,184바이트 할당
- 작업당 14회 할당

---

## 통합 테스트

### 실제 LLM 테스트

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

    // 에이전트가 계산기를 사용했는지 검증
    if !strings.Contains(output.Content, "425") {
        t.Errorf("Expected answer to contain 425, got: %s", output.Content)
    }
}
```

통합 테스트 실행:

```bash
# 통합 테스트만 실행
go test -tags=integration ./...

# 통합 테스트 건너뛰기 (기본값)
go test ./...
```

---

## 테스트 헬퍼

### 공통 테스트 유틸리티

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

## 지속적 통합

### GitHub Actions

모든 푸시 및 풀 리퀘스트에서 자동으로 테스트 실행:

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

## 모범 사례

### 1. 테스트 명명

```go
// 좋음 ✅
func TestAgentRun(t *testing.T)
func TestAgentRun_WithTools(t *testing.T)
func TestAgentRun_EmptyInput_ReturnsError(t *testing.T)

// 나쁨 ❌
func Test1(t *testing.T)
func TestStuff(t *testing.T)
```

### 2. t.Helper() 사용

헬퍼 함수 표시:

```go
func createAgent(t *testing.T) *Agent {
    t.Helper() // 스택 추적이 이 함수를 건너뜀
    // ...
}
```

### 3. 리소스 정리

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

### 4. 하위 테스트 사용

```go
func TestAgent(t *testing.T) {
    t.Run("creation", func(t *testing.T) { /* ... */ })
    t.Run("run", func(t *testing.T) { /* ... */ })
    t.Run("clear memory", func(t *testing.T) { /* ... */ })
}
```

### 5. 병렬 테스트

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

## 문제 해결

### 경쟁 검출기로 테스트 실패

```bash
# 경쟁 검출기와 함께 실행
go test -race ./...

# 경쟁이 감지되면 코드 수정
# 경쟁 검출기 비활성화하지 마세요
```

### 커버리지가 업데이트되지 않음

```bash
# 캐시 정리 후 재실행
go clean -testcache
make test
```

### 느린 테스트

```bash
# 느린 테스트 찾기
go test -v ./... | grep PASS

# 타임아웃 추가
go test -timeout 30s ./...
```

---

## 다음 단계

- [아키텍처](/advanced/architecture) 리뷰하여 디자인 패턴 학습
- [성능](/advanced/performance) 확인하여 벤치마크 이해
- [배포](/advanced/deployment) 읽고 프로덕션 설정 학습
- [기여 가이드](https://github.com/rexleimo/HNO/blob/main/CONTRIBUTING.md) 탐색

---

## 리소스

- [Go Testing](https://golang.org/pkg/testing/)
- [Table Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [Test Coverage](https://go.dev/blog/cover)
- [Benchmarking](https://pkg.go.dev/testing#hdr-Benchmarks)
